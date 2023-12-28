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
	logger        *zerolog.Logger
	tracer        *tracesdk.TracerProvider
	storageClient storageservice.StorageServiceClient
}

func NewHttpServer(loggger *zerolog.Logger, storageClient storageservice.StorageServiceClient, tracer *tracesdk.TracerProvider) (*HttpServer, error) {
	return &HttpServer{
		logger:        loggger,
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
		ctx.Request().Context(),
		"GetValueById", // operation name - usually func name
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

	resp, err := serv.storageClient.GetBookById(spanCtx, &storageservice.GetValueRequest{
		Id: int32(idNum),
	})

	if err != nil {
		serv.logger.Error().Err(err).Msg("got err from stoage via grpc")
		return ctx.JSON(http.StatusInternalServerError, "Server error")
	}

	return ctx.JSON(http.StatusOK, Book{
		BookID: resp.Id,
		BookToAdd: BookToAdd{
			Title:       resp.Title,
			Author:      resp.Author,
			Price:       float64(resp.Price),
			Description: resp.Description,
			AuthorBio:   resp.AuthorBio,
		},
	})
}

func (serv *HttpServer) AddValue(ctx echo.Context) error {
	// ----------------------tracing----------------------
	spanCtx, span := serv.tracer.Tracer("httpserver").Start(
		// spanCtx,
		ctx.Request().Context(),
		"AddValue",
	)
	defer span.End()
	// ---------------------------------------------------

	var value BookToAdd

	err := ctx.Bind(&value)
	if err != nil {
		serv.logger.Error().Err(err).Msg("cannot bind request body with ValueToAdd struct")
		return ctx.JSON(http.StatusInternalServerError, "Server error")
	}

	resp, err := serv.storageClient.AddBook(spanCtx, &storageservice.SetValueRequest{
		Title:       value.Title,
		Author:      value.Author,
		Price:       float32(value.Price),
		Description: value.Description,
		AuthorBio:   value.AuthorBio,
	})

	if err != nil {
		serv.logger.Error().Err(err).Msg("got err from stoage via grpc")
		return ctx.JSON(http.StatusInternalServerError, "Server error")
	}

	return ctx.JSON(http.StatusOK, resp.Id)
}
