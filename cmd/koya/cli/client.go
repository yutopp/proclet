package cli

import (
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	pbv1 "github.com/yutopp/koya/pkg/proto/api/v1"
)

func init() {
	rootCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use: "client",
	Run: func(cmd *cobra.Command, args []string) {
		port := 50051

		conn, err := grpc.Dial(
			fmt.Sprintf("localhost:%d", port),
			grpc.WithInsecure(),
		)
		if err != nil {
			log.Panicf("failed to connect: %s", err)
		}
		defer conn.Close()

		c := pbv1.NewKoyaServiceClient(conn)

		ctx := cmd.Context()
		runC, err := c.Run(ctx)
		if err != nil {
			log.Panicf("failed to run: %s", err)
		}

		err = runC.Send(&pbv1.Request{Code: "hello"})
		if err != nil {
			log.Panicf("failed to send: %s", err)
		}

		for {
			res, err := runC.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Panicf("failed to recv: %s", err)
			}

			log.Printf("stdout: %s", res.GetStdout())
		}
	},
}
