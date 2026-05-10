// Package observability provides structured logging and tracing utilities for EchoFS services.
package observability

import (
	"context"
	"log/slog"
	"os"
)

// NewLogger creates a structured JSON logger for production use.
func NewLogger(serviceName string, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler).With(
		slog.String("service", serviceName),
	)
}

// NewDevLogger creates a human-readable logger for development.
func NewDevLogger(serviceName string) *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return slog.New(handler).With(
		slog.String("service", serviceName),
	)
}

// RequestIDKey is the context key for request IDs.
type requestIDKey struct{}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// LoggerFromContext returns a logger enriched with context values (request ID, etc).
func LoggerFromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if reqID := GetRequestID(ctx); reqID != "" {
		return base.With(slog.String("request_id", reqID))
	}
	return base
}
