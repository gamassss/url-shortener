package logger

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	loggerKey    contextKey = "logger"
)

func NewRequestID() string {
	return uuid.New().String()
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := Get().With(slog.String("request_id", requestID))
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	ctx = context.WithValue(ctx, loggerKey, logger)
	return ctx
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return Get()
}

func RequestIDFromContext(ctx context.Context) string {
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		return reqID
	}
	return ""
}
