FROM golang:1.21.3-alpine3.18

RUN apk update && apk add --no-cache musl-dev gcc git build-base openssh

RUN go install github.com/cosmtrek/air@v1.49.0

WORKDIR /app

CMD ["air", "-c", ".air.toml"]
