package cli

import (
	"fmt"
	"log"
	"net"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	v1 "github.com/yutopp/koya/pkg/server"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		port := 50051

		log.Printf("start server on port %d", port)
		if err := run(port); err != nil {
			log.Panicf("failed to run server: %s", err)
		}
	},
}

func run(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()

	srv := v1.NewServer()
	v1.Register(grpcServer, srv)

	return grpcServer.Serve(lis)
}
