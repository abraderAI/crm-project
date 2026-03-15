# ---- Build stage ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.21

LABEL org.opencontainers.image.source="https://github.com/abraderAI/crm-project"
LABEL org.opencontainers.image.description="DEFT Evolution API Server"

RUN apk add --no-cache ca-certificates curl \
    && addgroup -S deft && adduser -S deft -G deft

WORKDIR /app
COPY --from=builder /build/server .
COPY api/config/ ./config/

RUN mkdir -p /data /app/uploads \
    && chown -R deft:deft /app /data

USER deft

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["./server"]
