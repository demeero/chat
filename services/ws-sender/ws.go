package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/demeero/chat/bricks/logger"
	"github.com/demeero/chat/bricks/session"
	"golang.org/x/net/websocket"
)

type wsMsgEvt struct {
	Msg string `json:"msg"`
}

type msgEvtUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type msgEvt struct {
	Msg       string     `json:"msg"`
	User      msgEvtUser `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
}

func newMsgEvt(msg string, sess session.Session) msgEvt {
	return msgEvt{
		Msg: msg,
		User: msgEvtUser{
			ID:        sess.Identity.ID,
			Email:     sess.Identity.Traits.Email,
			FirstName: sess.Identity.Traits.Name.First,
			LastName:  sess.Identity.Traits.Name.Last,
		},
		CreatedAt: time.Now().UTC(),
	}
}

type Sender struct {
	Topic string
	Sess  session.Session
	Pub   message.Publisher
}

func (s Sender) Execute(ws *websocket.Conn) {
	for {
		var wsEvt wsMsgEvt
		err := websocket.JSON.Receive(ws, &wsEvt)
		if errors.Is(err, io.EOF) {
			return
		}
		lg := logger.FromCtx(ws.Request().Context())
		if err != nil {
			lg.Debug("failed receive ws evt", slog.Any("err", err))
			return
		}
		lg.Debug("received ws evt", slog.Any("evt", wsEvt))
		if err := s.publish(ws.Request().Context(), wsEvt.Msg); err != nil {
			lg.Error("failed publish evt", slog.Any("err", err))
			return
		}
	}
}

func (s Sender) publish(ctx context.Context, msg string) error {
	b, err := json.Marshal(newMsgEvt(msg, s.Sess))
	if err != nil {
		return fmt.Errorf("failed encode evt: %w", err)
	}
	m := message.NewMessage(watermill.NewUUID(), b)
	m.SetContext(ctx)
	return s.Pub.Publish(s.Topic, m)
}