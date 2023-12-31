package server

import (
	"io"
	"log"
	"sync"

	"google.golang.org/grpc"

	pb "github.com/yutopp/koya/pkg/proto/api/v1"
	"github.com/yutopp/koya/pkg/service/executor"
)

// Server implements the KoyaServiceServer interface
type Server struct {
	pb.UnimplementedKoyaServiceServer
}

var _ pb.KoyaServiceServer = (*Server)(nil)

func Register(s *grpc.Server, srv *Server) {
	pb.RegisterKoyaServiceServer(s, srv)
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) RunOneshot(request *pb.RunOneshotRequest, stream pb.KoyaService_RunOneshotServer) error {
	e := executor.NewSandboxRunner()

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	task := &executor.RunTask{
		Image: "alpine",
		Cmd:   request.Code,

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
	handle, err := e.Run(stream.Context(), task)
	if err != nil {
		return err
	}

	var ioWg sync.WaitGroup
	ioWg.Add(2) // stdout, stderr
	go redirect(&ioWg, stdoutR, func(buf []byte) bool {
		outVal := &pb.Output{
			Kind:   0, // stdout
			Buffer: buf,
		}
		if err := stream.Send(&pb.RunOneshotResponse{Response: &pb.RunOneshotResponse_Output{Output: outVal}}); err != nil {
			return false
		}

		return true
	})
	go redirect(&ioWg, stderrR, func(buf []byte) bool {
		outVal := &pb.Output{
			Kind:   1, // stderr
			Buffer: buf,
		}
		if err := stream.Send(&pb.RunOneshotResponse{Response: &pb.RunOneshotResponse_Output{Output: outVal}}); err != nil {
			return false
		}

		return true
	})
	ioWg.Wait()

	select {
	case <-stream.Context().Done():
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
}
