package session

import "context"

type sessionCtxKey struct{}

var sessCtxKey = sessionCtxKey{}

func ToCtx(ctx context.Context, sess Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey{}, sess)
}

func FromCtx(ctx context.Context) Session {
	sess, _ := ctx.Value(sessCtxKey).(Session)
	return sess
}
