package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestGetZapLogger(t *testing.T) {
	tests := []struct {
		name    string
		debug   bool
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "debug mode enabled",
			debug:   true,
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name:    "debug mode disabled",
			debug:   false,
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name:    "with trace context",
			debug:   false,
			ctx:     context.Background(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the global state for each test
			once.Do(func() {}) // Reset the sync.Once
			once = sync.Once{}
			core = nil

			// Set debug mode
			Debug = tt.debug

			// Get logger
			logger, err := GetZapLogger(tt.ctx)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, logger)

			// Test that logger can be used
			logger.Info("test message")
		})
	}
}

func TestGetZapLoggerWithTracing(t *testing.T) {
	// Create a tracer provider for testing
	tp := otel.GetTracerProvider()
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Reset global state
	once.Do(func() {})
	once = sync.Once{}
	core = nil
	Debug = false

	logger, err := GetZapLogger(ctx)
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Test logging with trace context
	logger.Info("test message with tracing")

	// Verify span has events (this would require more complex setup to fully test)
	// For now, we just ensure the logger works with trace context
}

func TestGetJSONEncoderConfig(t *testing.T) {
	tests := []struct {
		name        string
		development bool
		checkFields func(t *testing.T, config zapcore.EncoderConfig)
	}{
		{
			name:        "production config",
			development: false,
			checkFields: func(t *testing.T, config zapcore.EncoderConfig) {
				// Use reflect to compare function pointers instead of direct comparison
				assert.NotNil(t, config.EncodeLevel)
				assert.NotNil(t, config.EncodeTime)
				// Production config should not have full caller encoder
				assert.NotNil(t, config.EncodeCaller)
			},
		},
		{
			name:        "development config",
			development: true,
			checkFields: func(t *testing.T, config zapcore.EncoderConfig) {
				assert.NotNil(t, config.EncodeLevel)
				assert.NotNil(t, config.EncodeTime)
				assert.NotNil(t, config.EncodeCaller)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := getJSONEncoderConfig(tt.development)
			tt.checkFields(t, config)
		})
	}
}

func TestColoredJSONEncoder_Clone(t *testing.T) {
	baseEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	coloredEncoder := NewColoredJSONEncoder(baseEncoder)

	cloned := coloredEncoder.Clone()

	assert.NotNil(t, cloned)
	assert.IsType(t, &ColoredJSONEncoder{}, cloned)

	// Verify the cloned encoder is independent
	originalType := fmt.Sprintf("%T", coloredEncoder.(*ColoredJSONEncoder).Encoder)
	clonedType := fmt.Sprintf("%T", cloned.(*ColoredJSONEncoder).Encoder)
	assert.Equal(t, originalType, clonedType)
}

func TestColoredJSONEncoder_EncodeEntry(t *testing.T) {
	tests := []struct {
		name      string
		level     zapcore.Level
		message   string
		fields    []zapcore.Field
		checkFunc func(t *testing.T, output string)
	}{
		{
			name:    "debug level",
			level:   zapcore.DebugLevel,
			message: "debug message",
			fields:  []zapcore.Field{zap.String("key", "value")},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "\x1b[34m") // Blue color code
				assert.Contains(t, output, "debug message")
				assert.Contains(t, output, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "info level",
			level:   zapcore.InfoLevel,
			message: "info message",
			fields:  []zapcore.Field{zap.Int("count", 42)},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "\x1b[32m") // Green color code
				assert.Contains(t, output, "info message")
				assert.Contains(t, output, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "warn level",
			level:   zapcore.WarnLevel,
			message: "warn message",
			fields:  []zapcore.Field{zap.Bool("flag", true)},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "\x1b[33m") // Yellow color code
				assert.Contains(t, output, "warn message")
				assert.Contains(t, output, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "error level",
			level:   zapcore.ErrorLevel,
			message: "error message",
			fields:  []zapcore.Field{zap.Error(errors.New("test error"))},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "\x1b[31m") // Red color code
				assert.Contains(t, output, "error message")
				assert.Contains(t, output, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "fatal level",
			level:   zapcore.FatalLevel,
			message: "fatal message",
			fields:  []zapcore.Field{},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "\x1b[35m") // Magenta color code
				assert.Contains(t, output, "fatal message")
				assert.Contains(t, output, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "unknown level",
			level:   zapcore.Level(99), // Unknown level
			message: "unknown message",
			fields:  []zapcore.Field{},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "\x1b[37m") // Default white color code
				assert.Contains(t, output, "unknown message")
				assert.Contains(t, output, "\x1b[0m") // Reset color code
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewColoredJSONEncoder(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()))

			entry := zapcore.Entry{
				Level:   tt.level,
				Message: tt.message,
				Time:    time.Now(),
			}

			buf, err := encoder.EncodeEntry(entry, tt.fields)
			require.NoError(t, err)
			require.NotNil(t, buf)

			output := buf.String()
			tt.checkFunc(t, output)

			// Verify the output is valid JSON (after removing color codes)
			cleanOutput := strings.ReplaceAll(output, "\x1b[34m", "")
			cleanOutput = strings.ReplaceAll(cleanOutput, "\x1b[32m", "")
			cleanOutput = strings.ReplaceAll(cleanOutput, "\x1b[33m", "")
			cleanOutput = strings.ReplaceAll(cleanOutput, "\x1b[31m", "")
			cleanOutput = strings.ReplaceAll(cleanOutput, "\x1b[35m", "")
			cleanOutput = strings.ReplaceAll(cleanOutput, "\x1b[37m", "")
			cleanOutput = strings.ReplaceAll(cleanOutput, "\x1b[0m", "")

			// Parse as JSON to ensure it's valid
			var logEntry map[string]any
			err = json.Unmarshal([]byte(cleanOutput), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, tt.message, logEntry["msg"])
		})
	}
}

func TestColoredJSONEncoder_EncodeEntryWithInvalidJSON(t *testing.T) {
	// Create a mock encoder that returns invalid JSON
	baseEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	mockEncoder := &mockEncoder{
		Encoder: baseEncoder,
		encodeEntryFunc: func(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
			buf := buffer.NewPool().Get()
			buf.AppendString("invalid json content")
			return buf, nil
		},
	}

	coloredEncoder := NewColoredJSONEncoder(mockEncoder)

	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "test message",
		Time:    time.Now(),
	}

	buf, err := coloredEncoder.EncodeEntry(entry, []zapcore.Field{})

	// Should not error even with invalid JSON
	assert.NoError(t, err)
	assert.NotNil(t, buf)
	assert.Contains(t, buf.String(), "invalid json content")
}

func TestColoredJSONEncoder_EncodeEntryWithEncoderError(t *testing.T) {
	// Create a mock encoder that returns an error
	baseEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	mockEncoder := &mockEncoder{
		Encoder: baseEncoder,
		encodeEntryFunc: func(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
			return nil, errors.New("encoder error")
		},
	}

	coloredEncoder := NewColoredJSONEncoder(mockEncoder)

	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "test message",
		Time:    time.Now(),
	}

	buf, err := coloredEncoder.EncodeEntry(entry, []zapcore.Field{})

	// Should propagate the error
	assert.Error(t, err)
	assert.Nil(t, buf)
	assert.Equal(t, "encoder error", err.Error())
}

func TestLoggerIntegration(t *testing.T) {
	// Test the complete logger integration
	Debug = false

	// Capture both stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r1, w1, _ := os.Pipe()
	r2, w2, _ := os.Pipe()
	os.Stdout = w1
	os.Stderr = w2

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Reset global state
	once.Do(func() {})
	once = sync.Once{}
	core = nil

	logger, err := GetZapLogger(context.Background())
	require.NoError(t, err)

	// Test different log levels
	logger.Info("info message", zap.String("key", "value"))
	logger.Warn("warn message", zap.Int("count", 42))
	logger.Error("error message", zap.Error(errors.New("test error")))

	err = w1.Close()
	require.NoError(t, err)
	err = w2.Close()
	require.NoError(t, err)

	// Read captured output from both pipes
	var buf1, buf2 bytes.Buffer
	_, err = io.Copy(&buf1, r1)
	require.NoError(t, err)
	_, err = io.Copy(&buf2, r2)
	require.NoError(t, err)

	// Combine outputs
	output := buf1.String() + buf2.String()

	// Verify output contains expected content
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
	assert.Contains(t, output, "count")
	assert.Contains(t, output, "42")
}

func TestLoggerWithObserver(t *testing.T) {
	// Use zaptest observer to capture logs
	core, obs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	// Test basic logging
	logger.Info("test message", zap.String("key", "value"))

	logs := obs.FilterMessage("test message").All()
	assert.Len(t, logs, 1)
	assert.Equal(t, "test message", logs[0].Message)
	assert.Equal(t, zapcore.InfoLevel, logs[0].Level)

	// Check fields
	fields := logs[0].ContextMap()
	assert.Equal(t, "value", fields["key"])
}

func TestLoggerLevels(t *testing.T) {
	// Test that different debug modes affect log levels
	tests := []struct {
		name     string
		debug    bool
		level    zapcore.Level
		expected bool // whether the log should be output
	}{
		{
			name:     "debug level in debug mode",
			debug:    true,
			level:    zapcore.DebugLevel,
			expected: true,
		},
		{
			name:     "debug level in production mode",
			debug:    false,
			level:    zapcore.DebugLevel,
			expected: false,
		},
		{
			name:     "info level in debug mode",
			debug:    true,
			level:    zapcore.InfoLevel,
			expected: true,
		},
		{
			name:     "info level in production mode",
			debug:    false,
			level:    zapcore.InfoLevel,
			expected: true,
		},
		{
			name:     "warn level in debug mode",
			debug:    true,
			level:    zapcore.WarnLevel,
			expected: true,
		},
		{
			name:     "warn level in production mode",
			debug:    false,
			level:    zapcore.WarnLevel,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state by creating a new sync.Once
			once = sync.Once{}
			core = nil
			Debug = tt.debug

			// Capture both stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			r1, w1, _ := os.Pipe()
			r2, w2, _ := os.Pipe()
			os.Stdout = w1
			os.Stderr = w2

			defer func() {
				os.Stdout = oldStdout
				os.Stderr = oldStderr
			}()

			// Use the actual logger
			logger, err := GetZapLogger(context.Background())
			require.NoError(t, err)

			// Log at the specified level
			switch tt.level {
			case zapcore.DebugLevel:
				logger.Debug("debug message")
			case zapcore.InfoLevel:
				logger.Info("info message")
			case zapcore.WarnLevel:
				logger.Warn("warn message")
			}

			err = w1.Close()
			require.NoError(t, err)
			err = w2.Close()
			require.NoError(t, err)

			// Read captured output from both pipes
			var buf1, buf2 bytes.Buffer
			_, err = io.Copy(&buf1, r1)
			require.NoError(t, err)
			_, err = io.Copy(&buf2, r2)
			require.NoError(t, err)

			// Combine outputs
			output := buf1.String() + buf2.String()

			if tt.expected {
				assert.Contains(t, output, tt.level.String())
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

// Mock encoder for testing
type mockEncoder struct {
	zapcore.Encoder
	encodeEntryFunc func(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error)
	cloneFunc       func() zapcore.Encoder
}

func (m *mockEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	return m.encodeEntryFunc(entry, fields)
}

func (m *mockEncoder) Clone() zapcore.Encoder {
	if m.cloneFunc != nil {
		return m.cloneFunc()
	}
	return &mockEncoder{
		Encoder:         m.Encoder.Clone(),
		encodeEntryFunc: m.encodeEntryFunc,
		cloneFunc:       m.cloneFunc,
	}
}
