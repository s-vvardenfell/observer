package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"test_tracing/httpserver"
	"test_tracing/storageservice"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RunStorageService(logger *zerolog.Logger, tracer *tracesdk.TracerProvider) {
	storSvc, err := storageservice.NewStorageService(storageservice.StorageServiceOpts{
		Tracer:     tracer,
		Logger:     logger,
		SqlConnStr: CheckEnv("STORAGE_CONN_STR", "postgres://0.0.0.0:5432/defaultdb?sslmode=disable"),
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init storage service")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s",
		CheckEnv("STORAGE_SVC_HOST", "127.0.0.1"),
		CheckEnv("STORAGE_SVC_PORT", "9991")))
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

func main() {
	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	tracer, err := InitHttpTracer(context.Background(), "custom-http-tracer", fmt.Sprintf("%s:%s",
		CheckEnv("JAEGER_HOST", "127.0.0.1"),
		CheckEnv("JAEGER_PORT", "4318")))
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	go RunStorageService(&logger, tracer)

	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%s",
			CheckEnv("STORAGE_SVC_HOST", "127.0.0.1"),
			CheckEnv("STORAGE_SVC_PORT", "9991")),
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen grpc-server")
	}

	storageServiceClient := storageservice.NewStorageServiceClient(conn)

	httpServ, err := httpserver.NewHttpServer(&logger, storageServiceClient, tracer)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init http server")
	}

	echoInst := echo.New()
	echoInst.Use(otelecho.Middleware("custom-http-tracer", otelecho.WithTracerProvider(tracer)))
	echoInst.GET("/storage/:id", httpServ.GetValueById)
	echoInst.POST("/storage", httpServ.AddValue)

	echoInst.Logger.Fatal(echoInst.Start(fmt.Sprintf("%s:%s",
		CheckEnv("HTTP_SRV_HOST", "127.0.0.1"),
		CheckEnv("HTTP_SRV_PORT", "1323"))))
}

func CheckEnv(variableName, defaultValue string) string {
	if cn, ok := os.LookupEnv(variableName); ok {
		return cn
	}

	return defaultValue
}
