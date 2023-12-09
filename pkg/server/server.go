package server

import (
	"google.golang.org/grpc"

	pb "github.com/yutopp/koya/pkg/proto/api/v1"
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

func (s *Server) Run(stream pb.KoyaService_RunServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	code := req.GetCode()
	if err := stream.Send(&pb.Response{Stdout: code}); err != nil {
		return err
	}

	return nil
}
