package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/s-vvardenfell/observer/util"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"github.com/s-vvardenfell/observer/tracer"

	storageservice "github.com/s-vvardenfell/observer/storageservice/service"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func main() {
	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	bgCtx := context.Background()

	tracer, err := tracer.InitGrpcTracer(bgCtx, "grpc-tracer", fmt.Sprintf("%s:%s",
		util.CheckEnv("JAEGER_GRPC_HOST", "127.0.0.1"),
		util.CheckEnv("JAEGER_GRPC_PORT", "4317")))
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	defer func() {
		if err := tracer.Shutdown(bgCtx); err != nil {
			logger.Fatal().Err(err).Msg("failed to shutting down tracer provider")
		}
	}()

	storSvc, err := storageservice.NewStorageService(storageservice.StorageServiceOpts{
		Tracer:     tracer,
		Logger:     &logger,
		SqlConnStr: util.CheckEnv("STORAGE_CONN_STR", "postgres://0.0.0.0:5432/defaultdb?sslmode=disable"),
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init storage service")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s",
		util.CheckEnv("STORAGE_SVC_HOST", "127.0.0.1"),
		util.CheckEnv("STORAGE_SVC_PORT", "9991")))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen for storage service")
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler(
			// otelgrpc.WithMessageEvents(),
			// otelgrpc.WithSpanOptions(),
			otelgrpc.WithTracerProvider(tracer),
		)),
	)

	storageservice.RegisterStorageServiceServer(grpcServer, storSvc)

	logger.Info().Msgf("storage server listening at: %s", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatal().Err(err).Msg("failed to serve storage service")
	}
}
