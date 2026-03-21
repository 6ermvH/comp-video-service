# Root Dockerfile for Timeweb App Platform deployment
# Builds the backend service from the backend/ subdirectory

FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/seed ./cmd/seed

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app
COPY --from=builder /bin/server ./server
COPY --from=builder /bin/seed ./seed
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

CMD ["./server"]
