FROM golang:1.25-alpine AS build

RUN apk update && \
    apk add --no-cache ca-certificates tzdata git build-base && \
    update-ca-certificates

RUN adduser -D -g '' appuser

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
ENV GOOS=linux

RUN go build -tags=jsoniter -ldflags="-w -s" -o uk-weather-overlays .

FROM alpine:latest AS runtime
ENV GIN_MODE=release
ENV TZ=UTC

RUN apk --no-cache add curl ca-certificates tzdata && \
    update-ca-certificates

RUN adduser -D -g '' appuser
WORKDIR /app

COPY --from=build /app/uk-weather-overlays .

USER appuser
EXPOSE 8080/tcp

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["./uk-weather-overlays"]
