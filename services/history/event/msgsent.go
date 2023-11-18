package event

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/demeero/bricks/errbrick"
	"github.com/demeero/bricks/slogbrick"
	"github.com/demeero/chat/history/writer"
)

type msgEvtUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type msgSentEvt struct {
	ChatRoomID string     `json:"chat_room_id"`
	PendingID  string     `json:"pending_id"`
	Msg        string     `json:"msg"`
	User       msgEvtUser `json:"user"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (e *msgSentEvt) WriteParams() writer.CreateParams {
	return writer.CreateParams{
		RoomChatID: e.ChatRoomID,
		Msg:        e.Msg,
		CreatedAt:  e.CreatedAt,
		PendingID:  e.PendingID,
		User: writer.UserParams{
			ID:        e.User.ID,
			Email:     e.User.Email,
			FirstName: e.User.FirstName,
			LastName:  e.User.LastName,
		},
	}
}

type msgStoredEvt struct {
	MsgID      string `json:"msg_id"`
	msgSentEvt `json:",inline"`
}

func MsgSentEvtHandler(topic string, w *writer.Writer) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		subLogger := slogbrick.WithOTELTrace(msg.Context(), slog.With(slog.String("topic", topic)))
		ctx := slogbrick.ToCtx(msg.Context(), subLogger)
		msg.SetContext(ctx)

		evt := msgSentEvt{}
		err := json.Unmarshal(msg.Payload, &evt)
		if err != nil {
			subLogger.Error("failed decode msg - skip", slog.Any("err", err), slog.String("payload", string(msg.Payload)))
			msg.Ack()
			return nil, fmt.Errorf("failed decode msg: %w", err)
		}

		msgID, err := w.Create(ctx, evt.WriteParams())
		if errbrick.IsOneOf(err) {
			subLogger.Error("failed write history - skip", slog.Any("err", err))
			msg.Ack()
			return nil, fmt.Errorf("failed write history: %w", err)
		}
		if err != nil {
			subLogger.Error("failed write history due to unexpected error", slog.Any("err", err))
			return nil, fmt.Errorf("failed write history: %w", err)
		}

		storedEvt := msgStoredEvt{
			MsgID:      msgID,
			msgSentEvt: evt,
		}
		storedEvtBytes, err := json.Marshal(storedEvt)
		if err != nil {
			subLogger.Error("failed encode stored evt", slog.Any("err", err))
			msg.Ack()
			return nil, fmt.Errorf("failed encode stored evt: %w", err)
		}
		storedEvtMsg := message.NewMessage(watermill.NewUUID(), storedEvtBytes)
		storedEvtMsg.SetContext(ctx)
		return []*message.Message{storedEvtMsg}, nil
	}
}
