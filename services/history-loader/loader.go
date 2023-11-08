package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

type MsgUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type Message struct {
	ID        string    `json:"id"`
	Msg       string    `json:"msg"`
	User      MsgUser   `json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

type cqlMsg struct {
	MsgID         string    `json:"msg_id"`
	Msg           string    `json:"msg"`
	UserID        string    `json:"user_id"`
	UserEmail     string    `json:"user_email"`
	UserFirstName string    `json:"user_first_name"`
	UserLastName  string    `json:"user_last_name"`
	CreatedAt     time.Time `json:"created_at"`
}

func (m cqlMsg) toMsg() Message {
	return Message{
		ID:  m.MsgID,
		Msg: m.Msg,
		User: MsgUser{
			ID:        m.UserID,
			Email:     m.UserEmail,
			FirstName: m.UserFirstName,
			LastName:  m.UserLastName,
		},
		CreatedAt: m.CreatedAt,
	}
}

type Loader struct {
	Sess *gocql.Session
}

func (l *Loader) Load(ctx context.Context) ([]Message, error) {
	iter := l.Sess.Query("SELECT * FROM chat.history WHERE chat_room_id = ? ORDER BY created_at ASC, msg_id ASC", roomChatID).
		WithContext(ctx).
		Iter()
	data, err := iter.SliceMap()
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed slice map: %w", err)
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed encode data: %w", err)
	}
	var cqlMsgs []cqlMsg
	if err := json.Unmarshal(b, &cqlMsgs); err != nil {
		return nil, fmt.Errorf("failed decode data: %w", err)
	}
	msgs := make([]Message, 0, len(cqlMsgs))
	for _, m := range cqlMsgs {
		msgs = append(msgs, m.toMsg())
	}
	return msgs, nil
}
