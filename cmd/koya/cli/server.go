package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	apiv1 "github.com/yutopp/koya/pkg/server"
)

var uid int
var gid int

var logger = zap.Must(zap.NewDevelopment())

func init() {
	serverCmd.Flags().IntVar(&uid, "uid", 0, "runner uid")
	serverCmd.Flags().IntVar(&gid, "gid", 0, "runner gid")

	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		port := 9000

		logger.Info("start server", zap.Int("port", port))
		run(port)
	},
}

func run(port int) {
	mux := http.NewServeMux()

	srv := apiv1.NewServer(&apiv1.Config{
		ProfilePath: profilePath,

		RunnerUID: uid,
		RunnerGID: gid,

		Logger: logger,
	})
	apiv1.Register(mux, srv)

	corsHandler := cors.New(cors.Options{
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
		},
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{
			"Accept-Encoding",
			"Content-Encoding",
			"Content-Type",
			"Connect-Protocol-Version",
			"Connect-Timeout-Ms",
			"Connect-Accept-Encoding",  // Unused in web browsers, but added for future-proofing
			"Connect-Content-Encoding", // Unused in web browsers, but added for future-proofing
			"Grpc-Timeout",             // Used for gRPC-web
			"X-Grpc-Web",               // Used for gRPC-web
			"X-User-Agent",             // Used for gRPC-web
		},
		ExposedHeaders: []string{
			"Content-Encoding",         // Unused in web browsers, but added for future-proofing
			"Connect-Content-Encoding", // Unused in web browsers, but added for future-proofing
			"Grpc-Status",              // Required for gRPC-web
			"Grpc-Message",             // Required for gRPC-web
		},
	})

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	httpSrv := &http.Server{
		Addr: addr,
		Handler: h2c.NewHandler(
			corsHandler.Handler(mux),
			&http2.Server{},
		),
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		MaxHeaderBytes:    8 * 1024, // 8KiB
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("HTTP listen and serve", zap.Error(err))
		}
	}()

	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Fatal("HTTP shutdown", zap.Error(err))
	}
}
