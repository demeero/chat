package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gocql/gocql"
)

type msgEvtUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type msgSentEvt struct {
	Msg       string     `json:"msg"`
	User      msgEvtUser `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
}

type msgStoredEvt struct {
	ChatRoomID string `json:"chat_room_id"`
	MsgID      string `json:"msg_id"`
	msgSentEvt `json:",inline"`
}

func msgEvtHandler(sess *gocql.Session) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		subLogger := slog.With(slog.String("topic", topic))
		subLogger.Debug("received redis evt", slog.String("payload", string(msg.Payload)))
		evt := msgSentEvt{}
		err := json.Unmarshal(msg.Payload, &evt)
		if err != nil {
			subLogger.Error("failed decode msg - skip", slog.Any("err", err))
			msg.Ack()
			return nil, fmt.Errorf("failed decode msg: %w", err)
		}
		msgID := gocql.TimeUUID()
		err = sess.Query("INSERT INTO chat.history (chat_room_id, msg_id, msg, user_id, user_email, user_first_name, user_last_name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			roomChatID, msgID, evt.Msg, evt.User.ID, evt.User.Email, evt.User.FirstName, evt.User.LastName, evt.CreatedAt).
			WithContext(msg.Context()).
			Exec()
		if err != nil {
			subLogger.Error("failed insert into history", slog.Any("err", err))
			return nil, fmt.Errorf("failed insert into history: %w", err)
		}
		storedEvt := msgStoredEvt{
			ChatRoomID: roomChatID.String(),
			MsgID:      msgID.String(),
			msgSentEvt: evt,
		}
		storedEvtBytes, err := json.Marshal(storedEvt)
		if err != nil {
			subLogger.Error("failed encode stored evt", slog.Any("err", err))
			msg.Ack()
			return nil, fmt.Errorf("failed encode stored evt: %w", err)
		}
		storedEvtMsg := message.NewMessage(watermill.NewUUID(), storedEvtBytes)
		storedEvtMsg.SetContext(msg.Context())
		return []*message.Message{storedEvtMsg}, nil
	}
}
