package server

import (
	"log"

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
	handle, err := e.Run(stream.Context(), "ababab")
	if err != nil {
		return err
	}

	for {
		select {
		case <-stream.Context().Done():
			return nil

		case out := <-handle.Output:
			if err := stream.Send(&pb.RunOneshotResponse{Stdout: string(out)}); err != nil {
				return err
			}

		case resp := <-handle.RespCh:
			log.Printf("resp: %+v", resp)

		case err := <-handle.ErrCh:
			log.Printf("err: %+v", err)
		}
	}

	log.Println("rpc")

	return nil
}
