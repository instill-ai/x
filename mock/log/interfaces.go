package log

import (
	"context"

	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// Logger interface for logging operations
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
}

// OTELLogger interface for OpenTelemetry logging
type OTELLogger interface {
	Emit(ctx context.Context, record log.Record)
}

// Encoder interface for zapcore.Encoder operations
type Encoder interface {
	zapcore.Encoder
	EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error)
	Clone() zapcore.Encoder
}

// LoggerFactory interface for creating loggers
type LoggerFactory interface {
	GetZapLogger(ctx context.Context) (*zap.Logger, error)
}

// CoreFactory interface for creating zap cores
type CoreFactory interface {
	NewCore(encoder zapcore.Encoder, syncer zapcore.WriteSyncer, level zapcore.LevelEnabler) zapcore.Core
}

// SyncerFactory interface for creating write syncers
type SyncerFactory interface {
	NewStdoutSyncer() zapcore.WriteSyncer
	NewStderrSyncer() zapcore.WriteSyncer
}
