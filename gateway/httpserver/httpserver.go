package httpserver

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	storageservice "github.com/s-vvardenfell/observer/storageservice/service"
	"google.golang.org/grpc/metadata"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type MetricsStack struct {
	totalRequestsAcceptedCounter prometheus.Counter
	// totalRequestsFailedCounter        prometheus.Counter
	// totalRequestsHandBugFailedCounter prometheus.Counter
	dataTransferGauge prometheus.Gauge
	// averageFailedRequestsGauge    prometheus.Gauge
	// serverProcessingTimeHistogram prometheus.Histogram
	// contentTransferTimeSummary    prometheus.Summary
}

type HttpServer struct {
	logger        *zerolog.Logger
	tracer        *tracesdk.TracerProvider
	storageClient storageservice.StorageServiceClient
	MetricsStack
	mutex sync.RWMutex
}

func NewHttpServer(
	loggger *zerolog.Logger,
	storageClient storageservice.StorageServiceClient,
	tracer *tracesdk.TracerProvider) (*HttpServer, error) {
	return &HttpServer{
		logger:        loggger,
		tracer:        tracer,
		storageClient: storageClient,
		MetricsStack:  initMetrics(),
	}, nil
}

func (serv *HttpServer) GetValueById(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return ctx.JSON(http.StatusBadRequest, "empty id")
	}

	// ----------------------tracing----------------------
	// track name - usually package or component name
	spanCtx, span := serv.tracer.Tracer("http-tracer").Start(
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

	traceId := span.SpanContext().TraceID().String()
	distCtx := metadata.AppendToOutgoingContext(spanCtx, "x-trace-id", traceId)

	idNum, err := strconv.Atoi(id)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "wrong id format")
	}

	resp, err := serv.storageClient.GetBookById(distCtx, &storageservice.GetValueRequest{
		Id: int32(idNum),
	})

	if err != nil {
		serv.logger.Error().Err(err).Msg("got err from stoage via grpc")
		return ctx.JSON(http.StatusInternalServerError, "Server error")
	}

	serv.dataTransferGauge.Add(float64(len(resp.String()))) // for test purposes

	ctx.Response().Header().Add("Trace-Id", span.SpanContext().TraceID().String())

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
	spanCtx, span := serv.tracer.Tracer("http-tracer").Start(
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

	traceId := span.SpanContext().TraceID().String()
	distCtx := metadata.AppendToOutgoingContext(spanCtx, "x-trace-id", traceId)

	resp, err := serv.storageClient.AddBook(distCtx, &storageservice.SetValueRequest{
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

	ctx.Response().Header().Add("Trace-Id", span.SpanContext().TraceID().String())

	return ctx.JSON(http.StatusOK, resp.Id)
}

func (serv *HttpServer) CountTotalReqMetricMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := next(c); err != nil {
			c.Error(err)
		}

		serv.mutex.Lock()
		defer serv.mutex.Unlock()

		serv.totalRequestsAcceptedCounter.Inc()

		return nil
	}
}

func initMetrics() MetricsStack {
	totalRequestsAcceptedCounter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "total_req_accepted",
			Help: "total request count served by http server",
		})

	// totalRequestsFailedCounter := prometheus.NewCounter(
	// 	prometheus.CounterOpts{
	// 		Name: "total_req_failed",
	// 		Help: "total request count failed on server side",
	// 	})

	// totalRequestsHandBugFailedCounter := prometheus.NewCounter(
	// 	prometheus.CounterOpts{
	// 		Name: "total_req_hand_bug_failed",
	// 		Help: "total request count failed on server side",
	// 	})

	dataTransferGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "data_transf_gauge",
			Help: "approximate amount data transferred",
		})

	// totalRequestsSendGauge := prometheus.NewGauge(
	// 	prometheus.GaugeOpts{
	// 		Name: "total_requests_send",
	// 		Help: "some test help 2",
	// 	})

	// requestProcessingTimeHistogramMs := prometheus.NewHistogram(
	// 	prometheus.HistogramOpts{
	// 		Name:    "serv_processing_time_histogram",
	// 		Buckets: prometheus.LinearBuckets(0, 30, 50),
	// 		Help:    "some test help 2",
	// 	})

	// requestProcessingTimeSummaryMs := prometheus.NewSummary(
	// 	prometheus.SummaryOpts{
	// 		Name:       "content_transf_time_summary",
	// 		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	// 		Help:       "some test help 2",
	// 	})

	prometheus.MustRegister(
		totalRequestsAcceptedCounter,
		dataTransferGauge,
		// totalRequestsHandBugFailedCounter,
		// approximateReviewCountGauge,
	)

	return MetricsStack{
		totalRequestsAcceptedCounter: totalRequestsAcceptedCounter,
		dataTransferGauge:            dataTransferGauge,
		// totalRequestsHandBugFailedCounter: totalRequestsHandBugFailedCounter,
		// approximateReviewCountGauge:       approximateReviewCountGauge,
	}
}
