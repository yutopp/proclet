package cli

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	apiv1 "github.com/yutopp/koya/pkg/server"
)

var uid int
var gid int

func init() {
	serverCmd.Flags().IntVar(&uid, "uid", 0, "runner uid")
	serverCmd.Flags().IntVar(&gid, "gid", 0, "runner gid")

	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		port := 9000

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

	mux := http.NewServeMux()

	srv := apiv1.NewServer(&apiv1.Config{
		RunnerUID: uid,
		RunnerGID: gid,
	})
	apiv1.Register(mux, srv)

	return http.Serve(lis, cors.AllowAll().Handler(h2c.NewHandler(mux, &http2.Server{})))
}
