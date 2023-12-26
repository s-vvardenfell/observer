package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/s-vvardenfell/observer/util"

	"github.com/s-vvardenfell/observer/tracer"

	storageservice "github.com/s-vvardenfell/observer/storageservice/service"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func main() {
	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	tracer, err := tracer.InitGrpcTracer(context.Background(), "custom-grpc-tracer", fmt.Sprintf("%s:%s",
		util.CheckEnv("JAEGER_HOST", "127.0.0.1"),
		util.CheckEnv("JAEGER_PORT", "4317")))
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

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

	grpcServer := grpc.NewServer()

	storageservice.RegisterStorageServiceServer(grpcServer, storSvc)

	logger.Info().Msgf("storage server listening at: %s", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatal().Err(err).Msg("failed to serve storage service")
	}
}
