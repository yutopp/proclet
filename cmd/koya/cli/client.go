package cli

import (
	"errors"
	"io"
	"log"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	pbv1 "github.com/yutopp/koya/pkg/proto/api/v1"
)

var addr string

func init() {
	clientCmd.Flags().StringVarP(&addr, "addr", "a", "localhost:50051", "server address")

	rootCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use: "client",
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			log.Panicf("failed to connect: %s", err)
		}
		defer conn.Close()

		c := pbv1.NewKoyaServiceClient(conn)

		ctx := cmd.Context()
		runC, err := c.RunOneshot(ctx, &pbv1.RunOneshotRequest{})
		if err != nil {
			log.Panicf("failed to run: %s", err)
		}

		for {
			res, err := runC.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Panicf("failed to recv: %s", err)
			}

			switch res := res.Response.(type) {
			case *pbv1.RunOneshotResponse_Output:
				// res.Output.Kind
				log.Printf("output: %s", res.Output.Buffer)
			}
		}
	},
}
