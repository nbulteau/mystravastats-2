package application

import (
	"context"
	"log/slog"
)

// Logger interface for structured logging
type Logger interface {
	InfoContext(ctx context.Context, msg string, args ...interface{})
	WarnContext(ctx context.Context, msg string, args ...interface{})
	ErrorContext(ctx context.Context, msg string, args ...interface{})
	DebugContext(ctx context.Context, msg string, args ...interface{})
}

// SimpleLogger implements Logger using standard slog
type SimpleLogger struct {
	logger *slog.Logger
}

// NewSimpleLogger creates a new SimpleLogger
func NewSimpleLogger(logger *slog.Logger) *SimpleLogger {
	return &SimpleLogger{logger: logger}
}

func (l *SimpleLogger) InfoContext(ctx context.Context, msg string, args ...interface{}) {
	l.logger.InfoContext(ctx, msg, args...)
}

func (l *SimpleLogger) WarnContext(ctx context.Context, msg string, args ...interface{}) {
	l.logger.WarnContext(ctx, msg, args...)
}

func (l *SimpleLogger) ErrorContext(ctx context.Context, msg string, args ...interface{}) {
	l.logger.ErrorContext(ctx, msg, args...)
}

func (l *SimpleLogger) DebugContext(ctx context.Context, msg string, args ...interface{}) {
	l.logger.DebugContext(ctx, msg, args...)
}

// Cache interface for caching
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int64) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// EventPublisher interface for publishing events
type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload interface{}) error
}

// MetricsCollector interface for collecting metrics
type MetricsCollector interface {
	RecordRequest(method, path string, statusCode int, durationMs int64)
	RecordError(errorType string)
	RecordCacheHit(cacheKey string)
	RecordCacheMiss(cacheKey string)
}
