package ctxutil

import (
	"context"
)


const (
	RequestIDKey = "request_id"
	UserIDKey    = "user_id"
)

// WithRequestID stores a request ID in the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDKey, id)
}

// GetRequestID reads a request ID from the context.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(RequestIDKey).(string)
	return id
}

// WithUserID stores a user ID in the context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

// GetUserID reads a user ID from the context.
func GetUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(UserIDKey).(string)
	return id
}
