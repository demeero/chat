package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"syscall"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/demeero/bricks/slogbrick"
	wotelfloss "github.com/dentech-floss/watermill-opentelemetry-go-extra/pkg/opentelemetry"
	wotel "github.com/voi-oss/watermill-opentelemetry/pkg/opentelemetry"
	"golang.org/x/net/websocket"
)

type Subscriber struct {
	Topic string
	Sub   message.Subscriber
}

func (s Subscriber) Subscribe(ctx context.Context, ws *websocket.Conn) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	msgs, err := s.Sub.Subscribe(ctx, s.Topic)
	if err != nil {
		return fmt.Errorf("failed subscribe %s: %w", s.Topic, err)
	}

	h := msgHandler(ws)

	lg := slogbrick.FromCtx(ws.Request().Context()).With(slog.String("topic", s.Topic))
	for msg := range msgs {
		lg.Debug("received redis evt",
			slog.String("payload", string(msg.Payload)),
			slog.Any("metadata", msg.Metadata))
		_, err := h(msg)
		if errors.Is(err, syscall.EPIPE) {
			break
		}
		if err != nil {
			lg.Debug("failed send message to ws", slog.Any("err", err))
			break
		}
		msg.Ack()
	}

	return nil
}

func msgHandler(ws *websocket.Conn) message.HandlerFunc {
	return wotelfloss.ExtractRemoteParentSpanContextHandler(wotel.TraceHandler(func(msg *message.Message) ([]*message.Message, error) {
		err := websocket.JSON.Send(ws, json.RawMessage(msg.Payload))
		if err != nil {
			return nil, err
		}
		return []*message.Message{msg}, nil
	}))
}
