package log

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var once sync.Once
var core zapcore.Core

// Debug determines the verbosity of the logger.
var Debug bool

// GetZapLogger returns an instance of zap logger
// It configures the logger based on the Debug value and sets up appropriate
// log levels and output destinations.
// The function also adds a hook to inject logs into OpenTelemetry traces.
func GetZapLogger(ctx context.Context) (*zap.Logger, error) {
	var err error
	once.Do(func() {
		// Enable debug and info level logs
		debugInfoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.DebugLevel || level == zapcore.InfoLevel
		})

		// Enable only info level logs
		infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.InfoLevel
		})

		// Enable warn, error, and fatal level logs
		warnErrorFatalLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.WarnLevel || level == zapcore.ErrorLevel || level == zapcore.FatalLevel
		})

		// Set up write syncers for stdout and stderr
		stdoutSyncer := zapcore.Lock(os.Stdout)
		stderrSyncer := zapcore.Lock(os.Stderr)

		// Configure the logger core based on debug mode
		if Debug {
			core = zapcore.NewTee(
				zapcore.NewCore(
					NewColoredJSONEncoder(zapcore.NewJSONEncoder(getJSONEncoderConfig(true))),
					stdoutSyncer,
					debugInfoLevel,
				),
				zapcore.NewCore(
					NewColoredJSONEncoder(zapcore.NewJSONEncoder(getJSONEncoderConfig(true))),
					stderrSyncer,
					warnErrorFatalLevel,
				),
			)
		} else {
			core = zapcore.NewTee(
				zapcore.NewCore(
					NewColoredJSONEncoder(zapcore.NewJSONEncoder(getJSONEncoderConfig(false))),
					stdoutSyncer,
					infoLevel,
				),
				zapcore.NewCore(
					NewColoredJSONEncoder(zapcore.NewJSONEncoder(getJSONEncoderConfig(false))),
					stderrSyncer,
					warnErrorFatalLevel,
				),
			)
		}
	})

	// Construct the logger with the configured core and add hooks
	logger := zap.New(core).WithOptions(
		zap.Hooks(func(entry zapcore.Entry) error {
			span := trace.SpanFromContext(ctx)
			if !span.IsRecording() {
				return nil
			}

			// Add log entry as an event to the current span
			span.AddEvent("log", trace.WithAttributes(
				attribute.KeyValue{
					Key:   "log.severity",
					Value: attribute.StringValue(entry.Level.String()),
				},
				attribute.KeyValue{
					Key:   "log.message",
					Value: attribute.StringValue(entry.Message),
				},
			))

			// Set span status based on log level
			if entry.Level >= zap.ErrorLevel {
				span.SetStatus(codes.Error, entry.Message)
			} else {
				span.SetStatus(codes.Ok, "")
			}

			return nil
		}),
		zap.AddCaller(),
		// Uncomment the following line to add stack traces for error logs
		// zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return logger, err
}

func getJSONEncoderConfig(development bool) zapcore.EncoderConfig {
	encoderConfig := zap.NewProductionEncoderConfig()
	if development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return encoderConfig
}

type ColoredJSONEncoder struct {
	zapcore.Encoder
}

func NewColoredJSONEncoder(encoder zapcore.Encoder) zapcore.Encoder {
	return &ColoredJSONEncoder{Encoder: encoder}
}

func (e *ColoredJSONEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	var logMap map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logMap)
	if err != nil {
		return buf, nil
	}

	colorCode := "\x1b[37m" // Default to white
	switch entry.Level {
	case zapcore.DebugLevel:
		colorCode = "\x1b[34m" // Blue
	case zapcore.InfoLevel:
		colorCode = "\x1b[32m" // Green
	case zapcore.WarnLevel:
		colorCode = "\x1b[33m" // Yellow
	case zapcore.ErrorLevel:
		colorCode = "\x1b[31m" // Red
	case zapcore.FatalLevel:
		colorCode = "\x1b[35m" // Magenta
	}
	coloredJSON := colorCode + buf.String() + "\x1b[0m"

	newBuf := buffer.NewPool().Get()
	newBuf.AppendString(coloredJSON)
	return newBuf, nil
}
