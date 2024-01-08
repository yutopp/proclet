package cli

import (
	"log"
	"net/http"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	apiv1pb "github.com/yutopp/koya/pkg/proto/api/v1"
	apiv1connect "github.com/yutopp/koya/pkg/proto/api/v1/v1connect"
)

var addr string

func init() {
	clientCmd.Flags().StringVarP(&addr, "addr", "a", "http://localhost:9000", "server address")

	rootCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use: "client",
	Run: func(cmd *cobra.Command, args []string) {
		c := apiv1connect.NewKoyaServiceClient(http.DefaultClient, addr, connect.WithGRPC())

		ctx := cmd.Context()
		stream, err := c.RunOneshot(ctx, connect.NewRequest(&apiv1pb.RunOneshotRequest{
			LanguageId:  "test-lang",
			ProcessorId: "a-latest",
			TaskId:      "run",

			Files: []*apiv1pb.File{
				{
					Path:    "main.sh",
					Content: []byte("ulimit -a; uname -a; whoami; sleep 5; echo hello"),
				},
			},
		}))
		if err != nil {
			log.Panicf("failed to run: %s", err)
		}

		for stream.Receive() {
			res := stream.Msg().GetResponse()
			switch res := res.(type) {
			case *apiv1pb.RunOneshotResponse_Output:
				// res.Output.Kind
				log.Printf("output: %s", res.Output.Buffer)
			}
		}
		if err := stream.Err(); err != nil {
			log.Panicf("failed to receive: %s", err)
		}
	},
}
