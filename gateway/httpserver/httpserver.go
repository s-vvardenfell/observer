package httpserver

import (
	"net/http"
	"strconv"

	storageservice "github.com/s-vvardenfell/observer/storageservice/service"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type HttpServer struct {
	loggger       *zerolog.Logger
	tracer        *tracesdk.TracerProvider
	storageClient storageservice.StorageServiceClient
}

func NewHttpServer(loggger *zerolog.Logger, storageClient storageservice.StorageServiceClient, tracer *tracesdk.TracerProvider) (*HttpServer, error) {
	return &HttpServer{
		loggger:       loggger,
		tracer:        tracer,
		storageClient: storageClient,
	}, nil
}

func (serv *HttpServer) GetValueById(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return ctx.JSON(http.StatusBadRequest, "empty id")
	}

	// ----------------------tracing----------------------
	// track name - usually package or component name
	spanCtx, span := serv.tracer.Tracer("httpserver").Start(
		// spanCtx,
		ctx.Request().Context(),
		"AddValue", // operation name - usually func name
		trace.WithAttributes(
			attribute.KeyValue{
				Key:   attribute.Key("id"),
				Value: attribute.StringValue(id),
			},
		),
	)
	defer span.End()
	// ---------------------------------------------------

	idNum, err := strconv.Atoi(id)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "wrong id format")
	}

	resp, err := serv.storageClient.GetValue(spanCtx, &storageservice.GetValueRequest{
		Key: int32(idNum),
	})

	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, "server error")
	}

	if resp.Val == "" {
		return ctx.JSON(http.StatusNoContent, "")
	}

	return ctx.JSON(http.StatusOK, resp.Val)
}

func (serv *HttpServer) AddValue(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, "id")
}
