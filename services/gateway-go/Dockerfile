FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /gateway ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /gateway-worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /notifier-consumer ./cmd/notifier-consumer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /notifier-dispatcher ./cmd/notifier-dispatcher

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /gateway /usr/local/bin/gateway
COPY --from=builder /gateway-worker /usr/local/bin/gateway-worker
COPY --from=builder /notifier-consumer /usr/local/bin/notifier-consumer
COPY --from=builder /notifier-dispatcher /usr/local/bin/notifier-dispatcher
EXPOSE 8080
CMD ["/usr/local/bin/gateway"]
