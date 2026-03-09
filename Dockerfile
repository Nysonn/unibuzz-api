# -----------------------------
# Migration stage
# -----------------------------
FROM golang:1.25-alpine AS migrate
WORKDIR /app

RUN apk add --no-cache netcat-openbsd && \
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

CMD ["sh", "-c", "migrate -path /app/migrations -database \"$DATABASE_URL\" up"]


# -----------------------------
# Development stage
# -----------------------------
FROM golang:1.25-alpine AS dev
WORKDIR /app

RUN go install github.com/cosmtrek/air@v1.49.0

COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]


# -----------------------------
# Builder stage
# -----------------------------
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org
RUN go mod download

COPY . .

ENV CGO_ENABLED=0

RUN go build -ldflags="-s -w" -o /app/unibuzz-api ./cmd/server


# -----------------------------
# Production image
# -----------------------------
FROM alpine:latest AS production
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/unibuzz-api /app/unibuzz-api

RUN addgroup -S app && adduser -S app -G app

RUN chown -R app:app /app

USER app

EXPOSE 8080

ENV GIN_MODE=release

ENTRYPOINT ["/app/unibuzz-api"]

# Worker stage
FROM golang:1.25-alpine AS worker
WORKDIR /app

RUN apk add --no-cache ffmpeg

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["go", "run", "cmd/worker/main.go"]