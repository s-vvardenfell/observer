package main

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrNoSuchVal = errors.New("no value with such key")
)

type DbMock struct {
	storage map[string]string
}

func NewDbMock() (*DbMock, error) {
	return &DbMock{
		storage: map[string]string{
			"1": "ONE",
			"2": "TWO",
			"3": "THREE",
		},
	}, nil
}

func (mock *DbMock) GetValById(spanCtx context.Context, tracer *tracesdk.TracerProvider, id string) (string, error) {
	_, span := tracer.Tracer("http_req_GetDataById").Start(
		spanCtx,
		"GetValById",
		trace.WithAttributes(
			attribute.KeyValue{
				Key:   attribute.Key("key3"),
				Value: attribute.StringValue("value3"),
			},
			attribute.KeyValue{
				Key:   attribute.Key("key4"),
				Value: attribute.StringValue("value4"),
			},
		),
	)
	defer span.End()

	val, ok := mock.storage[id]
	if !ok {
		return "", ErrNoSuchVal
	}

	return val, nil
}
