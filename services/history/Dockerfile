FROM golang:1.21-alpine AS builder

WORKDIR /usr/local/src/history
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /usr/local/bin/historyapi ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /usr/local/bin/historysub ./cmd/sub/main.go

FROM alpine:3 AS runner
COPY --from=builder /usr/local/bin/historyapi /usr/local/bin/historyapi
COPY --from=builder /usr/local/bin/historysub /usr/local/bin/historysub
ENTRYPOINT ["/usr/local/bin/historyloader"]
