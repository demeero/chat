package writer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/demeero/bricks/errbrick"
	"github.com/gocql/gocql"
)

type CreateParams struct {
	RoomChatID string
	Msg        string
	CreatedAt  time.Time
	PendingID  string
	User       UserParams
}

func (p CreateParams) validate() error {
	if p.RoomChatID == "" {
		return errors.New("room chat id is empty")
	}
	if p.Msg == "" {
		return errors.New("msg is empty")
	}
	if p.CreatedAt.IsZero() {
		return errors.New("created at is zero")
	}
	if p.User.ID == "" {
		return errors.New("user id is empty")
	}
	if p.User.Email == "" {
		return errors.New("user email is empty")
	}
	return nil
}

type UserParams struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
}

type Writer struct {
	sess *gocql.Session
}

func New(sess *gocql.Session) *Writer {
	return &Writer{sess: sess}
}

func (w *Writer) Create(ctx context.Context, params CreateParams) (string, error) {
	if err := params.validate(); err != nil {
		return "", fmt.Errorf("%w: %s", errbrick.ErrInvalidData, err)
	}
	msgID := gocql.TimeUUID()
	stmt := `INSERT INTO chat.history (chat_room_id, msg_id, msg, user_id, user_email, user_first_name, user_last_name, created_at, pending_id) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	err := w.sess.Query(stmt,
		params.RoomChatID, msgID, params.Msg, params.User.ID, params.User.Email, params.User.FirstName,
		params.User.LastName, params.CreatedAt, params.PendingID).
		WithContext(ctx).
		Exec()
	if err != nil {
		return "", fmt.Errorf("failed insert into history: %w", err)
	}
	return msgID.String(), nil
}
