package storageservice

import (
	"context"
	"database/sql"

	"github.com/s-vvardenfell/observer/storageservice/storagedb"
	"google.golang.org/grpc/metadata"

	"github.com/pkg/errors"

	"github.com/rs/zerolog"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrNoSuchKey = errors.New("no value by given key stored")
)

type StorageServiceOpts struct {
	Tracer     *tracesdk.TracerProvider
	Logger     *zerolog.Logger
	SqlConnStr string
}

type StorageService struct {
	tracer    *tracesdk.TracerProvider
	logger    *zerolog.Logger
	dbHandler *storagedb.StorageDbHandler
	UnimplementedStorageServiceServer
}

func NewStorageService(opts StorageServiceOpts) (*StorageService, error) {
	dbHdl, err := storagedb.NewStorageDbHandler(opts.SqlConnStr)
	if err != nil {
		return nil, err
	}

	return &StorageService{
		tracer:    opts.Tracer,
		logger:    opts.Logger,
		dbHandler: dbHdl,
	}, nil
}

func (serv *StorageService) GetBookById(ctx context.Context, req *GetValueRequest) (*GetValueResponse, error) {
	// Extract TraceID from header
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		var span trace.Span

		traceIdString := md["x-trace-id"][0]
		// Convert string to byte array
		traceId, err := trace.TraceIDFromHex(traceIdString)
		if err != nil {
			return nil, err
		}
		// Creating a span context with a predefined trace-id
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: traceId,
		})
		// Embedding span config into the context
		ctx = trace.ContextWithSpanContext(ctx, spanContext)

		ctx, span = serv.tracer.Tracer("grpc-tracer").Start(ctx, "GetBookById")
		defer span.End()
	}

	//-----------------------------------------

	data, err := serv.dbHandler.Queries.GetBookById(ctx, req.Id)
	if err != nil {
		return nil, errors.Wrap(err, "got err from sql db")
	}

	return &GetValueResponse{
		Id:          data.BookID,
		Title:       data.Title,
		Author:      data.Author,
		Price:       float32(data.Price.Float64),
		Description: data.Description.String,
		AuthorBio:   data.AuthorBio.String,
	}, nil
}

func (serv *StorageService) AddBook(ctx context.Context, req *SetValueRequest) (*SetValueResponse, error) {
	// Extract TraceID from header
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		var span trace.Span

		traceIdString := md["x-trace-id"][0]
		// Convert string to byte array
		traceId, err := trace.TraceIDFromHex(traceIdString)
		if err != nil {
			return nil, err
		}
		// Creating a span context with a predefined trace-id
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: traceId,
		})
		// Embedding span config into the context
		ctx = trace.ContextWithSpanContext(ctx, spanContext)

		ctx, span = serv.tracer.Tracer("grpc-tracer").Start(ctx, "AddBook")
		defer span.End()
	}
	//-----------------------------------------

	id, err := serv.dbHandler.Queries.InsertBook(ctx, storagedb.InsertBookParams{
		Title:       req.Title,
		Author:      req.Author,
		Price:       sql.NullFloat64{Float64: float64(req.Price), Valid: true},
		Description: sql.NullString{String: req.Description, Valid: true},
		AuthorBio:   sql.NullString{String: req.AuthorBio, Valid: true},
	})
	if err != nil {
		return nil, errors.Wrap(err, "got err from sql db")
	}

	return &SetValueResponse{Id: id}, nil
}
