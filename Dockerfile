# ---- Stage 1: Build ----
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/fiapx-video-service ./cmd/server

# ---- Stage 2: Runtime ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /app/fiapx-video-service .
COPY migrations/ ./migrations/

USER appuser

EXPOSE 8082 8083

CMD ["/app/fiapx-video-service"]
