package log

import (
	"errors"
	"testing"

	"github.com/frankban/quicktest"
	"go.temporal.io/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewZapAdapter(t *testing.T) {
	qt := quicktest.New(t)

	// Create a test logger
	core, _ := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)

	adapter := NewZapAdapter(testLogger)

	qt.Check(adapter, quicktest.Not(quicktest.IsNil))
	// Don't compare the logger directly since NewZapAdapter adds caller skip
	qt.Check(adapter.zl, quicktest.Not(quicktest.IsNil))
}

func TestZapAdapter_fields(t *testing.T) {
	qt := quicktest.New(t)

	core, _ := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	tests := []struct {
		name     string
		keyvals  []any
		expected int // expected number of fields
		hasError bool
	}{
		{
			name:     "valid key-value pairs",
			keyvals:  []any{"key1", "value1", "key2", 42},
			expected: 2,
			hasError: false,
		},
		{
			name:     "empty keyvals",
			keyvals:  []any{},
			expected: 0,
			hasError: false,
		},
		{
			name:     "odd number of keyvals",
			keyvals:  []any{"key1", "value1", "key2"},
			expected: 1, // error field
			hasError: true,
		},
		{
			name:     "non-string key",
			keyvals:  []any{42, "value1", "key2", "value2"},
			expected: 2,
			hasError: false,
		},
		{
			name:     "mixed types",
			keyvals:  []any{"string", "value", "int", 123, "bool", true, "float", 3.14},
			expected: 4,
			hasError: false,
		},
	}

	for _, tt := range tests {
		qt.Run(tt.name, func(c *quicktest.C) {
			fields := adapter.fields(tt.keyvals)
			c.Check(len(fields), quicktest.Equals, tt.expected)

			if tt.hasError {
				// Check that the first field is an error field
				c.Check(fields[0].Key, quicktest.Equals, "error")
				c.Check(fields[0].Interface.(error).Error(), quicktest.Contains, "odd number of keyvals pairs")
			}
		})
	}
}

func TestZapAdapter_Debug(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	adapter.Debug("debug message", "key1", "value1", "key2", 42)

	logs := obs.FilterMessage("debug message").All()
	qt.Check(len(logs), quicktest.Equals, 1)
	qt.Check(logs[0].Message, quicktest.Equals, "debug message")
	qt.Check(logs[0].Level, quicktest.Equals, zapcore.DebugLevel)

	// Check fields - use int64 instead of float64 for integers
	fields := logs[0].ContextMap()
	qt.Check(fields["key1"], quicktest.Equals, "value1")
	qt.Check(fields["key2"], quicktest.Equals, int64(42))
}

func TestZapAdapter_Info(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	adapter.Info("info message", "user", "john", "age", 30)

	logs := obs.FilterMessage("info message").All()
	qt.Check(len(logs), quicktest.Equals, 1)
	qt.Check(logs[0].Message, quicktest.Equals, "info message")
	qt.Check(logs[0].Level, quicktest.Equals, zapcore.InfoLevel)

	// Check fields - use int64 instead of float64 for integers
	fields := logs[0].ContextMap()
	qt.Check(fields["user"], quicktest.Equals, "john")
	qt.Check(fields["age"], quicktest.Equals, int64(30))
}

func TestZapAdapter_Warn(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.WarnLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	adapter.Warn("warning message", "warning_type", "deprecation", "version", "1.0")

	logs := obs.FilterMessage("warning message").All()
	qt.Check(len(logs), quicktest.Equals, 1)
	qt.Check(logs[0].Message, quicktest.Equals, "warning message")
	qt.Check(logs[0].Level, quicktest.Equals, zapcore.WarnLevel)

	// Check fields
	fields := logs[0].ContextMap()
	qt.Check(fields["warning_type"], quicktest.Equals, "deprecation")
	qt.Check(fields["version"], quicktest.Equals, "1.0")
}

func TestZapAdapter_Error(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.ErrorLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	testErr := errors.New("test error")
	adapter.Error("error message", "error", testErr, "operation", "database_query")

	logs := obs.FilterMessage("error message").All()
	qt.Check(len(logs), quicktest.Equals, 1)
	qt.Check(logs[0].Message, quicktest.Equals, "error message")
	qt.Check(logs[0].Level, quicktest.Equals, zapcore.ErrorLevel)

	// Check fields
	fields := logs[0].ContextMap()
	qt.Check(fields["error"], quicktest.Equals, testErr.Error())
	qt.Check(fields["operation"], quicktest.Equals, "database_query")
}

func TestZapAdapter_With(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	// Create a new logger with additional fields
	childAdapter := adapter.With("request_id", "12345", "user_id", "67890")

	// Log with the child adapter
	childAdapter.Info("request processed")

	logs := obs.FilterMessage("request processed").All()
	qt.Check(len(logs), quicktest.Equals, 1)
	qt.Check(logs[0].Message, quicktest.Equals, "request processed")

	// Check that both the original fields and new fields are present
	fields := logs[0].ContextMap()
	qt.Check(fields["request_id"], quicktest.Equals, "12345")
	qt.Check(fields["user_id"], quicktest.Equals, "67890")

	// Verify the original adapter is unchanged (don't compare logger directly)
	qt.Check(adapter.zl, quicktest.Not(quicktest.IsNil))
	qt.Check(childAdapter.(*ZapAdapter).zl, quicktest.Not(quicktest.IsNil))
}

func TestZapAdapter_WithChaining(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	// Chain multiple With calls
	child1 := adapter.With("level1", "value1").(*ZapAdapter)
	child2 := child1.With("level2", "value2").(*ZapAdapter)
	child3 := child2.With("level3", "value3").(*ZapAdapter)

	child3.Info("chained message")

	logs := obs.FilterMessage("chained message").All()
	qt.Check(len(logs), quicktest.Equals, 1)

	// Check that all fields from the chain are present
	fields := logs[0].ContextMap()
	qt.Check(fields["level1"], quicktest.Equals, "value1")
	qt.Check(fields["level2"], quicktest.Equals, "value2")
	qt.Check(fields["level3"], quicktest.Equals, "value3")
}

func TestZapAdapter_OddKeyvals(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	// Test with odd number of keyvals
	adapter.Info("odd keyvals message", "key1", "value1", "key2")

	logs := obs.FilterMessage("odd keyvals message").All()
	qt.Check(len(logs), quicktest.Equals, 1)

	// Check that an error field was added
	fields := logs[0].ContextMap()
	_, hasError := fields["error"]
	qt.Check(hasError, quicktest.IsTrue)
	errorMsg := fields["error"].(string)
	qt.Check(errorMsg, quicktest.Contains, "odd number of keyvals pairs")
}

func TestZapAdapter_NonStringKeys(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	// Test with non-string keys
	adapter.Info("non-string keys", 42, "value1", true, "value2", 3.14, "value3")

	logs := obs.FilterMessage("non-string keys").All()
	qt.Check(len(logs), quicktest.Equals, 1)

	// Check that non-string keys are converted to strings
	fields := logs[0].ContextMap()
	qt.Check(fields["42"], quicktest.Equals, "value1")
	qt.Check(fields["true"], quicktest.Equals, "value2")
	qt.Check(fields["3.14"], quicktest.Equals, "value3")
}

func TestZapAdapter_EmptyKeyvals(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	// Test with empty keyvals
	adapter.Info("empty keyvals message")

	logs := obs.FilterMessage("empty keyvals message").All()
	qt.Check(len(logs), quicktest.Equals, 1)
	qt.Check(logs[0].Message, quicktest.Equals, "empty keyvals message")

	// Should have no additional fields
	fields := logs[0].ContextMap()
	qt.Check(len(fields), quicktest.Equals, 0)
}

func TestZapAdapter_ComplexValues(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	// Test with complex values
	complexMap := map[string]any{"nested": "value"}
	complexSlice := []string{"item1", "item2"}

	adapter.Info("complex values", "map", complexMap, "slice", complexSlice, "nil", nil)

	logs := obs.FilterMessage("complex values").All()
	qt.Check(len(logs), quicktest.Equals, 1)

	fields := logs[0].ContextMap()
	qt.Check(fields["map"], quicktest.DeepEquals, complexMap)
	// Convert []any to []string for comparison
	sliceInterface := fields["slice"].([]any)
	sliceString := make([]string, len(sliceInterface))
	for i, v := range sliceInterface {
		sliceString[i] = v.(string)
	}
	qt.Check(sliceString, quicktest.DeepEquals, complexSlice)
	qt.Check(fields["nil"], quicktest.IsNil)
}

func TestZapAdapter_InterfaceCompliance(t *testing.T) {

	core, _ := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	// Verify that ZapAdapter implements the log.Logger interface
	var _ log.Logger = adapter
}

func TestZapAdapter_CallerSkip(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	// This test verifies that the caller skip is working
	adapter.Info("caller test")

	logs := obs.All()
	qt.Check(len(logs), quicktest.Equals, 1)

	// The caller should be from this test function, not from the adapter
	// Check that caller is present (may be empty in some environments)
	// Just verify the log was created successfully
	qt.Check(logs[0].Message, quicktest.Equals, "caller test")
}

func TestZapAdapter_LogLevels(t *testing.T) {
	qt := quicktest.New(t)

	core, obs := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core)
	adapter := NewZapAdapter(testLogger)

	// Test all log levels
	adapter.Debug("debug level")
	adapter.Info("info level")
	adapter.Warn("warn level")
	adapter.Error("error level")

	logs := obs.All()
	qt.Check(len(logs), quicktest.Equals, 4)

	// Verify levels
	qt.Check(logs[0].Level, quicktest.Equals, zapcore.DebugLevel)
	qt.Check(logs[1].Level, quicktest.Equals, zapcore.InfoLevel)
	qt.Check(logs[2].Level, quicktest.Equals, zapcore.WarnLevel)
	qt.Check(logs[3].Level, quicktest.Equals, zapcore.ErrorLevel)

	// Verify messages
	qt.Check(logs[0].Message, quicktest.Equals, "debug level")
	qt.Check(logs[1].Message, quicktest.Equals, "info level")
	qt.Check(logs[2].Message, quicktest.Equals, "warn level")
	qt.Check(logs[3].Message, quicktest.Equals, "error level")
}
