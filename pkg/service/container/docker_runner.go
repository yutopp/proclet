package container

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"
)

type DockerRunner struct {
}

type RunTask struct {
	Image    string
	ShellCmd string

	UID         int
	GID         int
	HomeHostDir string

	Stdin  io.Reader
	Stdout io.WriteCloser
	Stderr io.WriteCloser

	Limits ResourceLimits
}

type ResourceLimits struct {
	Core    int64
	Nofile  int64
	NProc   int64 // NOTE: per-user limit
	MemLock int64
	CPUTime int64 // sec
	Memory  int64 // bytes
	FSize   int64
}

func NewDockerRunner() *DockerRunner {
	return &DockerRunner{}
}

type Handle struct {
	DoneCh chan interface{}
}

func makeULimit(name string, lim int64) *units.Ulimit {
	return &units.Ulimit{
		Name: name,
		Soft: lim,
		Hard: lim,
	}
}

func (e *DockerRunner) Run(ctx context.Context, task *RunTask) (*Handle, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker client")
	}

	log.Println("create")

	containerHomeDir := "/home/proclet"
	hostConfig := &container.HostConfig{
		AutoRemove:     true,
		ReadonlyRootfs: true,
		Privileged:     false,
		Resources: container.Resources{
			Memory: task.Limits.Memory, // bytes
			Ulimits: []*units.Ulimit{
				makeULimit("core", task.Limits.Core),
				makeULimit("nofile", task.Limits.Nofile),
				makeULimit("nproc", task.Limits.NProc), // NOTE: per-user limit
				makeULimit("memlock", task.Limits.MemLock),
				makeULimit("cpu", task.Limits.CPUTime),
				// makeULimit("as", task.Limits.Memory), disabled by docker
				makeULimit("fsize", task.Limits.FSize),
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: task.HomeHostDir,
				Target: containerHomeDir,
			},
		},
	}
	// log.Printf("hostConfig: %+v", hostConfig)

	stopTimeout := 3 // sec
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:       task.Image,
		Cmd:         []string{"/bin/sh", "-c", task.ShellCmd},
		StopSignal:  "SIGKILL",
		StopTimeout: &stopTimeout,
		User:        fmt.Sprintf("%d:%d", task.UID, task.GID),
		WorkingDir:  containerHomeDir,
	}, hostConfig, nil, nil, "")
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
				switch {
				case errors.Is(err, context.Canceled):
					// ignore
				default:
					log.Println("err(status): ", err)
				}
				return
			}

			// log.Printf("read: %+v", stats)
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
			log.Printf("ctx done: %+v", ctx.Err())

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
		t := time.NewTimer(time.Duration(task.Limits.CPUTime+extensionSec) * time.Second)
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
