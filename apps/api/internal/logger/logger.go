package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
)

var defaultLogger *slog.Logger

func Init(levelStr, formatStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(formatStr) == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
	return defaultLogger
}

func Get() *slog.Logger {
	if defaultLogger == nil {
		return Init("info", "json")
	}
	return defaultLogger
}

func FromContext(ctx context.Context) *slog.Logger {
	l := Get()
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		return l.With(slog.String("request_id", reqID))
	}
	return l
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
