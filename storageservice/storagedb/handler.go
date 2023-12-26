package storagedb

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

type StorageDbHandler struct {
	Queries *Queries
	dbConn  *sql.DB
}

func NewStorageDbHandler(connStr string) (*StorageDbHandler, error) {
	dbConn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to db with conn string %s", connStr)
	}

	for i := 0; i < 5; i++ {
		err = dbConn.Ping()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
	}

	if err != nil {
		return nil, errors.Wrap(err, "ping failed")
	}

	queries := New(dbConn)

	return &StorageDbHandler{
		Queries: queries,
		dbConn:  dbConn,
	}, nil
}

func (hdl *StorageDbHandler) Close() error {
	return hdl.dbConn.Close()
}
