FROM golang:1.21rc3-bullseye AS builder

WORKDIR /build

COPY . .

ENV CGO_ENABLED=0

RUN go build -o observice .

FROM alpine:3.18.2

COPY --from=builder /build/observice .

ENV STORAGE_SVC_HOST=0.0.0.0 \
    STORAGE_SVC_PORT=9991 \
    JAEGER_HOST=0.0.0.0 \
    JAEGER_PORT=4318 \
    HTTP_SRV_HOST=0.0.0.0 \
    HTTP_SRV_PORT=1323 \
    STORAGE_CONN_STR="postgres://0.0.0.0:5432/defaultdb?sslmode=disable"

EXPOSE 1323 9991

ENTRYPOINT ["./observice"]