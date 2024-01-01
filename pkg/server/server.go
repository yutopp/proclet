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
	resourceLimit := executor.ResourceLimits{
		Core:    0,                // Process can NOT create CORE file
		Nofile:  512,              // Process can open 512 files
		NProc:   30,               // Process can create processes to 30
		MemLock: 1024,             // Process can lock 1024 Bytes by mlock(2)
		CPUTime: 5,                // sec
		Memory:  10 * 1024 * 1024, // bytes
		FSize:   5 * 1024 * 1024,  // Process can writes a file only 5MiB
	}
	task := &executor.RunTask{
		Image: "alpine",
		Cmd:   req.Msg.Code,

		Stdin:  nil,
		Stdout: stdoutW,
		Stderr: stderrW,

		Limits: resourceLimit,
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
