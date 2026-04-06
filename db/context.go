package db

import "context"

type contextKey string

const (
	contextKeyUserID  contextKey = "user_id"
	contextKeyStorage contextKey = "storage"
)

func GetUserIDFromCtx(ctx context.Context) string {
	if v := ctx.Value(contextKeyUserID); v != nil {
		return v.(string)
	}
	return ""
}

func GetStorageFromCtx(ctx context.Context) Storage {
	if v := ctx.Value(contextKeyStorage); v != nil {
		return v.(Storage)
	}
	return nil
}

func WithUserStorage(ctx context.Context, userID string, storage Storage) context.Context {
	ctx = context.WithValue(ctx, contextKeyUserID, userID)
	ctx = context.WithValue(ctx, contextKeyStorage, storage)
	return ctx
}
