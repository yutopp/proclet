package server

import (
	"context"

	"connectrpc.com/connect"
	"go.uber.org/zap"
)

type loggingInterceptor struct {
	logger *zap.Logger
}

var _ connect.Interceptor = (*loggingInterceptor)(nil)

func NewLoggingInterceptor(logger *zap.Logger) *loggingInterceptor {
	return &loggingInterceptor{
		logger: logger,
	}
}

func (i *loggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return connect.UnaryFunc(func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		i.logger.Info(
			"Request",
			zap.String("Procedure", req.Spec().Procedure),
			zap.String("Protocol", req.Peer().Protocol),
			zap.String("Addr", req.Peer().Addr),
		)
		return next(ctx, req)
	})
}

func (i *loggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		i.logger.Info(
			"Request",
			zap.String("StreamType", spec.StreamType.String()),
			zap.Any("Schema", spec.Schema),
			zap.String("Procedure", spec.Procedure),
		)
		return next(ctx, spec)
	})
}

func (i *loggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(
		ctx context.Context,
		conn connect.StreamingHandlerConn,
	) error {
		spec := conn.Spec()
		i.logger.Info("Request", zap.String("procedure", spec.Procedure))
		err := next(ctx, conn)
		i.logger.Info("Error", zap.Error(err))
		return err
	})
}
