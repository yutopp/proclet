package executor

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"
)

type SandboxRunner struct {
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
	Output chan []byte
	RespCh <-chan container.WaitResponse
	ErrCh  <-chan error
	DoneCh chan struct{}
}

func (e *SandboxRunner) Run(ctx context.Context, code string) (*Handle, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	resourceLimit := ResourceLimits{
		Memory:     6 * 1024 * 1024, // 6MiB
		MemorySoft: 6 * 1024 * 1024, // 4MiB
		CPUCore:    250000000,       // 0.25 core
		PIDNum:     10,              // 10 processes
		TimeoutSec: 1,
	}

	log.Println("create")

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:       "alpine",
		Cmd:         []string{"/bin/sh", "-c", "ulimit -a; sleep 3; echo monyomonyo"},
		StopSignal:  "SIGKILL",
		StopTimeout: &resourceLimit.TimeoutSec,
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
					Soft: int64(resourceLimit.TimeoutSec),
					Hard: int64(resourceLimit.TimeoutSec),
				},
			},
		},
	}, nil, nil, "")
	if err != nil {
		return nil, err
	}

	containerID := resp.ID

	handle := &Handle{}
	handle.Output = make(chan []byte)
	handle.DoneCh = make(chan struct{})
	//handle.RespCh = respCh
	//handle.ErrCh = errCh

	log.Println("stats")

	statsResp, err := cli.ContainerStats(ctx, containerID, true)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	log.Println("go")

	go func() {
		defer hijack.Close()
		defer hijack.Conn.Close()

		_, err := stdcopy.StdCopy(log.Writer(), log.Writer(), hijack.Reader)
		if err != nil {
			log.Println("err(hijack): ", err)
			return
		}
		log.Println("done(hijack): ", err)
	}()

	log.Println("start")

	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return nil, err
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
		case err := <-errCh:
			log.Printf("err: %+v", err)
		}
	}()
	handle.RespCh = respCh
	handle.ErrCh = errCh

	// Realtime checking apart from cgroup limits to prevent sleep() function running infinite.
	go func() {
		const extensionSec = 3
		t := time.NewTimer(time.Duration(resourceLimit.TimeoutSec+extensionSec) * time.Second)
		defer t.Stop()

		select {
		case <-handle.DoneCh:
			log.Println("done")
		case <-t.C:
			immidiate := 0
			err := cli.ContainerStop(ctx, containerID, container.StopOptions{
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
