package server

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"

	"connectrpc.com/connect"

	apiv1pb "github.com/yutopp/koya/pkg/proto/api/v1"
	apiv1connect "github.com/yutopp/koya/pkg/proto/api/v1/v1connect"
	"github.com/yutopp/koya/pkg/service/executor"
)

// Server implements the KoyaServiceServer interface
type Server struct{}

var _ apiv1connect.KoyaServiceHandler = (*Server)(nil)

func Register(mux *http.ServeMux, srv *Server) {
	path, handler := apiv1connect.NewKoyaServiceHandler(srv)
	mux.Handle(path, handler)
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) RunOneshot(
	ctx context.Context,
	req *connect.Request[apiv1pb.RunOneshotRequest],
	stream *connect.ServerStream[apiv1pb.RunOneshotResponse],
) error {
	e := executor.NewSandboxRunner()

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	task := &executor.RunTask{
		Image: "alpine",
		Cmd:   req.Msg.Code,

		Stdin:  nil,
		Stdout: stdoutW,
		Stderr: stderrW,

		Limits: executor.ResourceLimits{
			Memory:     6 * 1024 * 1024, // 6MiB
			MemorySoft: 6 * 1024 * 1024, // 4MiB
			CPUCore:    250000000,       // 0.25 core
			PIDNum:     10,              // 10 processes
			TimeoutSec: 1,
		},
	}
	handle, err := e.Run(ctx, task)
	if err != nil {
		return err
	}

	var ioWg sync.WaitGroup
	ioWg.Add(2) // stdout, stderr
	go redirect(&ioWg, stdoutR, func(buf []byte) bool {
		outVal := &apiv1pb.Output{
			Kind:   0, // stdout
			Buffer: buf,
		}
		if err := stream.Send(&apiv1pb.RunOneshotResponse{Response: &apiv1pb.RunOneshotResponse_Output{Output: outVal}}); err != nil {
			return false
		}

		return true
	})
	go redirect(&ioWg, stderrR, func(buf []byte) bool {
		outVal := &apiv1pb.Output{
			Kind:   1, // stderr
			Buffer: buf,
		}
		if err := stream.Send(&apiv1pb.RunOneshotResponse{Response: &apiv1pb.RunOneshotResponse_Output{Output: outVal}}); err != nil {
			return false
		}

		return true
	})
	ioWg.Wait()

	select {
	case <-ctx.Done():
		return nil

	case out, ok := <-handle.DoneCh:
		if !ok {
			log.Println("done ch closed")
			return nil
		}
		log.Printf("done: %+v", out)
	}

	log.Println("rpc finished")

	return nil
}

func redirect(wg *sync.WaitGroup, pipe *io.PipeReader, callback func([]byte) bool) {
	defer wg.Done()

	buf := make([]byte, 1024)
	for {
		n, err := pipe.Read(buf)
		if err != nil {
			log.Printf("stderr read err: %+v", err)
			return
		}

		if !callback(buf[:n]) {
			return
		}
	}
}

type Language struct {
	ID       string
	ShowName string
	Envs     []LanguageEnv
}

type LanguageEnv struct {
	ID string
}
