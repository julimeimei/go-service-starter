# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26.6
ARG ALPINE_VERSION=3.22

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -buildid=" \
    -o /out/go-service-starter \
    ./cmd/api

FROM alpine:${ALPINE_VERSION} AS runtime

RUN apk add --no-cache ca-certificates && \
    addgroup -S app && \
    adduser -S -D -H -G app app

WORKDIR /app

COPY --from=build /out/go-service-starter /app/go-service-starter

ENV APP_NAME=go-service-starter \
    APP_ENV=development \
    HTTP_HOST=0.0.0.0 \
    HTTP_PORT=8080 \
    LOG_LEVEL=info

EXPOSE 8080

USER app:app

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

ENTRYPOINT ["/app/go-service-starter"]
