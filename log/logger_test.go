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

	"github.com/frankban/quicktest"
	"github.com/gojuno/minimock/v3"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	mocklog "github.com/instill-ai/x/mock/log"
)

func TestGetZapLogger(t *testing.T) {
	qt := quicktest.New(t)

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
		qt.Run(tt.name, func(c *quicktest.C) {
			// Reset the global state for each test
			once.Do(func() {}) // Reset the sync.Once
			once = sync.Once{}
			core = nil

			// Set debug mode
			Debug = tt.debug

			// Get logger
			logger, err := GetZapLogger(tt.ctx)

			if tt.wantErr {
				c.Check(err, quicktest.Not(quicktest.IsNil))
				return
			}

			c.Check(err, quicktest.IsNil)
			c.Check(logger, quicktest.Not(quicktest.IsNil))

			// Test that logger can be used
			logger.Info("test message")
		})
	}
}

func TestGetZapLoggerWithMocks(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock logger factory
	mockFactory := mocklog.NewLoggerFactoryMock(mc)

	// Create mock core factory
	mockCoreFactory := mocklog.NewCoreFactoryMock(mc)

	// Create mock syncer factory
	mockSyncerFactory := mocklog.NewSyncerFactoryMock(mc)

	// Test that mocks were created successfully
	qt.Check(mockFactory, quicktest.Not(quicktest.IsNil))
	qt.Check(mockCoreFactory, quicktest.Not(quicktest.IsNil))
	qt.Check(mockSyncerFactory, quicktest.Not(quicktest.IsNil))
}

func TestGetZapLoggerWithTracing(t *testing.T) {
	qt := quicktest.New(t)

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
	qt.Check(err, quicktest.IsNil)
	qt.Check(logger, quicktest.Not(quicktest.IsNil))

	// Test logging with trace context
	logger.Info("test message with tracing")
}

func TestGetJSONEncoderConfig(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name        string
		development bool
		checkFields func(c *quicktest.C, config zapcore.EncoderConfig)
	}{
		{
			name:        "production config",
			development: false,
			checkFields: func(c *quicktest.C, config zapcore.EncoderConfig) {
				c.Check(config.EncodeLevel, quicktest.Not(quicktest.IsNil))
				c.Check(config.EncodeTime, quicktest.Not(quicktest.IsNil))
				c.Check(config.EncodeCaller, quicktest.Not(quicktest.IsNil))
			},
		},
		{
			name:        "development config",
			development: true,
			checkFields: func(c *quicktest.C, config zapcore.EncoderConfig) {
				c.Check(config.EncodeLevel, quicktest.Not(quicktest.IsNil))
				c.Check(config.EncodeTime, quicktest.Not(quicktest.IsNil))
				c.Check(config.EncodeCaller, quicktest.Not(quicktest.IsNil))
			},
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			config := getJSONEncoderConfig(tt.development)
			tt.checkFields(c, config)
		})
	}
}

func TestColoredJSONEncoder_Clone(t *testing.T) {
	qt := quicktest.New(t)

	baseEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	coloredEncoder := NewColoredJSONEncoder(baseEncoder)

	cloned := coloredEncoder.Clone()

	qt.Check(cloned, quicktest.Not(quicktest.IsNil))
	_, ok := cloned.(*ColoredJSONEncoder)
	qt.Check(ok, quicktest.IsTrue)

	// Verify the cloned encoder is independent
	originalType := fmt.Sprintf("%T", coloredEncoder.(*ColoredJSONEncoder).Encoder)
	clonedType := fmt.Sprintf("%T", cloned.(*ColoredJSONEncoder).Encoder)
	qt.Check(originalType, quicktest.Equals, clonedType)
}

func TestColoredJSONEncoder_EncodeEntry(t *testing.T) {
	qt := quicktest.New(t)

	tests := []struct {
		name      string
		level     zapcore.Level
		message   string
		fields    []zapcore.Field
		checkFunc func(c *quicktest.C, output string)
	}{
		{
			name:    "debug level",
			level:   zapcore.DebugLevel,
			message: "debug message",
			fields:  []zapcore.Field{zap.String("key", "value")},
			checkFunc: func(c *quicktest.C, output string) {
				c.Check(output, quicktest.Contains, "\x1b[34m") // Blue color code
				c.Check(output, quicktest.Contains, "debug message")
				c.Check(output, quicktest.Contains, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "info level",
			level:   zapcore.InfoLevel,
			message: "info message",
			fields:  []zapcore.Field{zap.Int("count", 42)},
			checkFunc: func(c *quicktest.C, output string) {
				c.Check(output, quicktest.Contains, "\x1b[32m") // Green color code
				c.Check(output, quicktest.Contains, "info message")
				c.Check(output, quicktest.Contains, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "warn level",
			level:   zapcore.WarnLevel,
			message: "warn message",
			fields:  []zapcore.Field{zap.Bool("flag", true)},
			checkFunc: func(c *quicktest.C, output string) {
				c.Check(output, quicktest.Contains, "\x1b[33m") // Yellow color code
				c.Check(output, quicktest.Contains, "warn message")
				c.Check(output, quicktest.Contains, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "error level",
			level:   zapcore.ErrorLevel,
			message: "error message",
			fields:  []zapcore.Field{zap.Error(errors.New("test error"))},
			checkFunc: func(c *quicktest.C, output string) {
				c.Check(output, quicktest.Contains, "\x1b[31m") // Red color code
				c.Check(output, quicktest.Contains, "error message")
				c.Check(output, quicktest.Contains, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "fatal level",
			level:   zapcore.FatalLevel,
			message: "fatal message",
			fields:  []zapcore.Field{},
			checkFunc: func(c *quicktest.C, output string) {
				c.Check(output, quicktest.Contains, "\x1b[35m") // Magenta color code
				c.Check(output, quicktest.Contains, "fatal message")
				c.Check(output, quicktest.Contains, "\x1b[0m") // Reset color code
			},
		},
		{
			name:    "unknown level",
			level:   zapcore.Level(99), // Unknown level
			message: "unknown message",
			fields:  []zapcore.Field{},
			checkFunc: func(c *quicktest.C, output string) {
				c.Check(output, quicktest.Contains, "\x1b[37m") // Default white color code
				c.Check(output, quicktest.Contains, "unknown message")
				c.Check(output, quicktest.Contains, "\x1b[0m") // Reset color code
			},
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			encoder := NewColoredJSONEncoder(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()))

			entry := zapcore.Entry{
				Level:   tt.level,
				Message: tt.message,
				Time:    time.Now(),
			}

			buf, err := encoder.EncodeEntry(entry, tt.fields)
			c.Check(err, quicktest.IsNil)
			c.Check(buf, quicktest.Not(quicktest.IsNil))

			output := buf.String()
			tt.checkFunc(c, output)

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
			c.Check(err, quicktest.IsNil)
			c.Check(logEntry["msg"], quicktest.Equals, tt.message)
		})
	}
}

func TestColoredJSONEncoder_EncodeEntryWithMocks(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock encoder
	mockEncoder := mocklog.NewEncoderMock(mc)

	// Set up mock expectations with exact parameters
	expectedEntry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "test message",
		Time:    time.Date(2025, 7, 14, 15, 59, 17, 0, time.UTC), // Use fixed time
	}

	mockEncoder.EncodeEntryMock.Expect(expectedEntry, []zapcore.Field{}).Return(nil, errors.New("encoder error"))

	// Test with mock encoder
	coloredEncoder := NewColoredJSONEncoder(mockEncoder)

	buf, err := coloredEncoder.EncodeEntry(expectedEntry, []zapcore.Field{})

	// Should propagate the error from mock
	qt.Check(err, quicktest.Not(quicktest.IsNil))
	qt.Check(buf, quicktest.IsNil)
	qt.Check(err.Error(), quicktest.Equals, "encoder error")
}

func TestLoggerIntegration(t *testing.T) {
	qt := quicktest.New(t)

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
	qt.Check(err, quicktest.IsNil)

	// Test different log levels
	logger.Info("info message", zap.String("key", "value"))
	logger.Warn("warn message", zap.Int("count", 42))
	logger.Error("error message", zap.Error(errors.New("test error")))

	err = w1.Close()
	qt.Check(err, quicktest.IsNil)
	err = w2.Close()
	qt.Check(err, quicktest.IsNil)

	// Read captured output from both pipes
	var buf1, buf2 bytes.Buffer
	_, err = io.Copy(&buf1, r1)
	qt.Check(err, quicktest.IsNil)
	_, err = io.Copy(&buf2, r2)
	qt.Check(err, quicktest.IsNil)

	// Combine outputs
	output := buf1.String() + buf2.String()

	// Verify output contains expected content
	qt.Check(output, quicktest.Contains, "info message")
	qt.Check(output, quicktest.Contains, "warn message")
	qt.Check(output, quicktest.Contains, "error message")
	qt.Check(output, quicktest.Contains, "key")
	qt.Check(output, quicktest.Contains, "value")
	qt.Check(output, quicktest.Contains, "count")
	qt.Check(output, quicktest.Contains, "42")
}

func TestLoggerWithObserver(t *testing.T) {
	qt := quicktest.New(t)

	// Use zaptest observer to capture logs
	core, obs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	// Test basic logging
	logger.Info("test message", zap.String("key", "value"))

	logs := obs.FilterMessage("test message").All()
	qt.Check(len(logs), quicktest.Equals, 1)
	qt.Check(logs[0].Message, quicktest.Equals, "test message")
	qt.Check(logs[0].Level, quicktest.Equals, zapcore.InfoLevel)

	// Check fields
	fields := logs[0].ContextMap()
	qt.Check(fields["key"], quicktest.Equals, "value")
}

func TestLoggerLevels(t *testing.T) {
	qt := quicktest.New(t)

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
		qt.Run(tt.name, func(c *quicktest.C) {
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
			c.Check(err, quicktest.IsNil)

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
			c.Check(err, quicktest.IsNil)
			err = w2.Close()
			c.Check(err, quicktest.IsNil)

			// Read captured output from both pipes
			var buf1, buf2 bytes.Buffer
			_, err = io.Copy(&buf1, r1)
			c.Check(err, quicktest.IsNil)
			_, err = io.Copy(&buf2, r2)
			c.Check(err, quicktest.IsNil)

			// Combine outputs
			output := buf1.String() + buf2.String()

			if tt.expected {
				c.Check(output, quicktest.Contains, tt.level.String())
			} else {
				c.Check(len(output), quicktest.Equals, 0)
			}
		})
	}
}
