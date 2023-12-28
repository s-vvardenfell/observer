package storageservice

import (
	"context"
	"database/sql"

	"github.com/s-vvardenfell/observer/storageservice/storagedb"

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

func (serv *StorageService) GetBookById(ctx context.Context, req *GetValueRequest) (*GetValueResponse, error) {
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
