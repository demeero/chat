FROM golang:1.21-alpine AS builder

WORKDIR /usr/local/src/ws-sender
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /usr/local/bin/ ./...

FROM alpine:3 AS runner
COPY --from=builder /usr/local/bin/wssender /usr/local/bin/wssender
ENTRYPOINT ["/usr/local/bin/wssender"]
