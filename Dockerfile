FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY app/go.mod app/go.sum* ./
RUN go mod download

COPY app/*.go ./
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server .

FROM alpine:3.19

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/server .

RUN mkdir -p /app/logs && chown -R appuser:appgroup /app

USER appuser

ENV MODE=stable \
    APP_VERSION=1.0.0 \
    APP_PORT=3000

EXPOSE 3000

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:3000/healthz || exit 1

CMD ["./server"]