package log

import (
	"fmt"

	"go.temporal.io/sdk/log"
	"go.uber.org/zap"
)

// ZapAdapter is a wrapper around a zap logger that implements the log.Logger interface.
type ZapAdapter struct {
	zl *zap.Logger
}

// NewZapAdapter creates a new ZapAdapter with the given zap logger.
func NewZapAdapter(zapLogger *zap.Logger) *ZapAdapter {
	return &ZapAdapter{
		// Skip one call frame to exclude zap_adapter itself.
		// Or it can be configured when logger is created (not always possible).
		zl: zapLogger.WithOptions(zap.AddCallerSkip(1)),
	}
}

func (log *ZapAdapter) fields(keyvals []any) []zap.Field {
	if len(keyvals)%2 != 0 {
		return []zap.Field{zap.Error(fmt.Errorf("odd number of keyvals pairs: %v", keyvals))}
	}

	var fields []zap.Field
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", keyvals[i])
		}
		fields = append(fields, zap.Any(key, keyvals[i+1]))
	}

	return fields
}

// Debug logs a debug message with the given keyvals.
func (log *ZapAdapter) Debug(msg string, keyvals ...any) {
	log.zl.Debug(msg, log.fields(keyvals)...)
}

// Info logs an info message with the given keyvals.
func (log *ZapAdapter) Info(msg string, keyvals ...any) {
	log.zl.Info(msg, log.fields(keyvals)...)
}

// Warn logs a warning message with the given keyvals.
func (log *ZapAdapter) Warn(msg string, keyvals ...any) {
	log.zl.Warn(msg, log.fields(keyvals)...)
}

// Error logs an error message with the given keyvals.
func (log *ZapAdapter) Error(msg string, keyvals ...any) {
	log.zl.Error(msg, log.fields(keyvals)...)
}

// With returns a copy of the logger with the given keyvals added.
func (log *ZapAdapter) With(keyvals ...any) log.Logger {
	return &ZapAdapter{zl: log.zl.With(log.fields(keyvals)...)}
}
