package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/s-vvardenfell/observer/tracer"
	"github.com/s-vvardenfell/observer/util"

	storageservice "github.com/s-vvardenfell/observer/storageservice/service"

	"github.com/s-vvardenfell/observer/gateway/httpserver"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var logger zerolog.Logger

	logFile, err := os.OpenFile("gateway.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		logger = zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	} else {
		logger = zerolog.New(logFile).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	}

	defer logFile.Close()

	bgCtx := context.Background()

	tracer, err := tracer.InitHttpTracer(
		bgCtx, "http-tracer",
		fmt.Sprintf("%s:%s",
			util.CheckEnv("JAEGER_HTTP_HOST", "127.0.0.1"),
			util.CheckEnv("JAEGER_HTTP_PORT", "4318")))
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	defer func() {
		if err := tracer.Shutdown(bgCtx); err != nil {
			logger.Fatal().Err(err).Msg("failed to shutting down tracer provider")
		}
	}()

	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%s",
			util.CheckEnv("STORAGE_SVC_HOST", "127.0.0.1"),
			util.CheckEnv("STORAGE_SVC_PORT", "9991")),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(
			// otelgrpc.WithMessageEvents(),
			// otelgrpc.WithSpanOptions(),
			otelgrpc.WithTracerProvider(tracer),
		)),
	)

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen grpc-server")
	}

	go func() { // metrics server
		http.Handle("/metrics", promhttp.Handler())

		if err := http.ListenAndServe(
			fmt.Sprintf("%s:%s",
				util.CheckEnv("PROM_HOST", "127.0.0.1"),
				util.CheckEnv("PROM_PORT", "9101")), nil); err != nil {
			logger.Error().Err(err).Msg("failed run metrics exporter endpoint")
		}
	}()

	storageServiceClient := storageservice.NewStorageServiceClient(conn)

	httpServ, err := httpserver.NewHttpServer(&logger, storageServiceClient, tracer)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init http server")
	}

	echoInst := echo.New()
	echoInst.Use(otelecho.Middleware("http-tracer", otelecho.WithTracerProvider(tracer)))
	echoInst.Use(middleware.Logger())
	echoInst.Use(middleware.Recover())
	echoInst.Use(httpServ.CountTotalReqMetricMiddleware)
	echoInst.GET("/storage/:id", httpServ.GetValueById)
	echoInst.POST("/storage", httpServ.AddValue)

	echoInst.Logger.Fatal(echoInst.Start(fmt.Sprintf("%s:%s",
		util.CheckEnv("HTTP_SRV_HOST", "127.0.0.1"),
		util.CheckEnv("HTTP_SRV_PORT", "1323"))))
}
