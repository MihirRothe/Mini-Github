# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download

COPY apps/api/ ./apps/api/
COPY migrations/ ./migrations/

WORKDIR /build/apps/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/forgehub-api cmd/server/main.go

# Production runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates git tzdata

# Create unprivileged user
RUN addgroup -S forgehub && adduser -S forgehub -G forgehub

WORKDIR /app

COPY --from=builder /build/forgehub-api /app/forgehub-api
COPY migrations /app/migrations

RUN mkdir -p /data/git && chown -R forgehub:forgehub /data /app

USER forgehub

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/app/forgehub-api"]
