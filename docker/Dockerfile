FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /build/server .
COPY api/config/ ./config/

EXPOSE 8080
ENTRYPOINT ["./server"]
