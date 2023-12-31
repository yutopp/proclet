package executor

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"
)

type SandboxRunner struct {
}

type RunTask struct {
	Image string
	Cmd   string

	Stdin  io.Reader
	Stdout io.WriteCloser
	Stderr io.WriteCloser

	Limits ResourceLimits
}

type ResourceLimits struct {
	Memory     int64 // bytes
	MemorySoft int64 // bytes
	CPUCore    int64 // 1/1000000000 core
	PIDNum     int64
	TimeoutSec int
}

func NewSandboxRunner() *SandboxRunner {
	return &SandboxRunner{}
}

type Handle struct {
	DoneCh chan interface{}
}

func (e *SandboxRunner) Run(ctx context.Context, task *RunTask) (*Handle, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker client")
	}

	log.Println("create")

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:       task.Image,
		Cmd:         []string{"/bin/sh", "-c", task.Cmd},
		StopSignal:  "SIGKILL",
		StopTimeout: &task.Limits.TimeoutSec,
	}, &container.HostConfig{
		AutoRemove:     true,
		ReadonlyRootfs: true,
		Privileged:     false,
		Resources: container.Resources{
			Ulimits: []*units.Ulimit{
				{
					Name: "nofile",
					Soft: 10,
					Hard: 10,
				},
				{
					Name: "cpu",
					Soft: int64(task.Limits.TimeoutSec),
					Hard: int64(task.Limits.TimeoutSec),
				},
			},
		},
	}, nil, nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create container")
	}

	containerID := resp.ID

	handle := &Handle{}
	handle.DoneCh = make(chan interface{})

	log.Println("stats")

	statsResp, err := cli.ContainerStats(ctx, containerID, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get container stats")
	}

	go func() {
		defer statsResp.Body.Close()

		for {
			var stats types.StatsJSON
			err = json.NewDecoder(statsResp.Body).Decode(&stats)
			if err != nil {
				log.Println("err(status): ", err)
				break
			}

			log.Printf("read: %+v", stats)
		}
	}()

	log.Println("attach")

	hijack, err := cli.ContainerAttach(ctx, containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to attach container")
	}

	log.Println("go")

	go func() {
		defer hijack.Close()
		defer hijack.Conn.Close()
		defer task.Stdout.Close()
		defer task.Stderr.Close()

		_, err := stdcopy.StdCopy(task.Stdout, task.Stderr, hijack.Reader)
		if err != nil {
			log.Println("err(hijack): ", err)
			return
		}
		log.Println("done(hijack): ", err)
	}()

	log.Println("start")

	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return nil, errors.Wrap(err, "failed to start container")
	}

	log.Println("wait")

	respCh, errCh := cli.ContainerWait(ctx, containerID, "")
	go func() {
		defer close(handle.DoneCh)

		select {
		case <-ctx.Done():
			log.Println("ctx done")

		case resp := <-respCh:
			log.Printf("resp: %+v", resp)
			handle.DoneCh <- resp

		case err := <-errCh:
			log.Printf("err: %+v", err)
			handle.DoneCh <- err
		}
	}()

	// Realtime checking apart from cgroup limits to prevent sleep() function running infinite.
	stopCtx := context.WithoutCancel(ctx)
	go func() {
		const extensionSec = 3
		t := time.NewTimer(time.Duration(task.Limits.TimeoutSec+extensionSec) * time.Second)
		defer t.Stop()

		select {
		case <-handle.DoneCh:
			log.Println("done")
		case <-t.C:
			immidiate := 0
			err := cli.ContainerStop(stopCtx, containerID, container.StopOptions{
				Timeout: &immidiate,
				Signal:  "SIGKILL",
			})
			if err != nil {
				log.Println("err: ", err)
			}
			log.Println("timeout")
		}
	}()

	log.Println("exited")

	return handle, nil
}
