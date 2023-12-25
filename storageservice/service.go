package storageservice

import (
	"context"
	"test_tracing/storagedb"

	"github.com/pkg/errors"

	"github.com/rs/zerolog"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
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

func (serv *StorageService) GetValue(ctx context.Context, req *GetValueRequest) (*GetValueResponse, error) {
	data, err := serv.dbHandler.Queries.GetBookById(ctx, req.Key)
	if err != nil {
		return nil, errors.Wrap(err, "got err from sql db")
	}

	return &GetValueResponse{Val: data.Title}, nil
}

func (serv *StorageService) SetValue(ctx context.Context, req *SetValueRequest) (*SetValueResponse, error) {
	return &SetValueResponse{}, nil
}
