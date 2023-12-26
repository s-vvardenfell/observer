package main

import (
	"context"
	"fmt"
	"os"

	"github.com/labstack/echo/v4/middleware"
	"github.com/s-vvardenfell/observer/tracer"
	"github.com/s-vvardenfell/observer/util"

	storageservice "github.com/s-vvardenfell/observer/storageservice/service"

	"github.com/s-vvardenfell/observer/gateway/httpserver"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	tracer, err := tracer.InitHttpTracer(context.Background(), "custom-http-tracer", fmt.Sprintf("%s:%s",
		util.CheckEnv("JAEGER_HTTP_HOST", "127.0.0.1"),
		util.CheckEnv("JAEGER_HTTP_PORT", "4318")))
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%s",
			util.CheckEnv("STORAGE_SVC_HOST", "127.0.0.1"),
			util.CheckEnv("STORAGE_SVC_PORT", "9991")),
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
	echoInst.Use(middleware.Logger())
	echoInst.Use(middleware.Recover())
	echoInst.GET("/storage/:id", httpServ.GetValueById)
	echoInst.POST("/storage", httpServ.AddValue)

	echoInst.Logger.Fatal(echoInst.Start(fmt.Sprintf("%s:%s",
		util.CheckEnv("HTTP_SRV_HOST", "127.0.0.1"),
		util.CheckEnv("HTTP_SRV_PORT", "1323"))))
}
