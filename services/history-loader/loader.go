package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/demeero/chat/bricks/apperr"
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

func newCQLMsgs(data []map[string]interface{}) ([]cqlMsg, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed encode data: %w", err)
	}
	var cqlMsgs []cqlMsg
	if err := json.Unmarshal(b, &cqlMsgs); err != nil {
		return nil, fmt.Errorf("failed decode data: %w", err)
	}
	return cqlMsgs, nil
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

// Pagination is the pagination parameters.
type Pagination struct {
	// PageToken is the token to get the next page.
	pageToken []byte
	// PageSize is the number of items per page.
	pageSize uint16
}

func NewPagination(pageToken string, pageSize uint16) (Pagination, error) {
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	var tokenBytes []byte
	if pageToken != "" {
		b, err := base64.StdEncoding.DecodeString(pageToken)
		if err != nil {
			return Pagination{}, fmt.Errorf("%w: failed to decode token from base64: %s", apperr.ErrInvalidData, err)
		}
		tokenBytes = b
	}
	return Pagination{
		pageSize:  pageSize,
		pageToken: tokenBytes,
	}, nil
}

type Loader struct {
	Sess *gocql.Session
}

func (l *Loader) Load(ctx context.Context, p Pagination) ([]Message, string, error) {
	q, err := l.buildQuery(ctx, int(p.pageSize), p.pageToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed build query: %w", err)
	}
	iter := q.Iter()
	data, err := iter.SliceMap()
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed slice map: %w", err)
	}
	cqlMsgs, err := newCQLMsgs(data)
	if err != nil {
		return nil, "", fmt.Errorf("failed create cql messages: %w", err)
	}
	pageState := iter.PageState()
	return l.convertFromCQLMsgs(cqlMsgs), base64.StdEncoding.EncodeToString(pageState), nil
}

func (l *Loader) convertFromCQLMsgs(cqlMsgs []cqlMsg) []Message {
	msgs := make([]Message, 0, len(cqlMsgs))
	for _, m := range cqlMsgs {
		msgs = append(msgs, m.toMsg())
	}
	return msgs
}

func (l *Loader) buildQuery(ctx context.Context, pSize int, pToken []byte) (*gocql.Query, error) {
	return l.Sess.Query("SELECT * FROM chat.history WHERE chat_room_id = ? ORDER BY created_at DESC, msg_id DESC", roomChatID).
		WithContext(ctx).
		PageState(pToken).
		PageSize(pSize), nil
}
