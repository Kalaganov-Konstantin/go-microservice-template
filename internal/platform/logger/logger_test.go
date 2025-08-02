package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevel_Decode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Level
		expectError bool
	}{
		{
			name:     "debug level",
			input:    "debug",
			expected: LevelDebug,
		},
		{
			name:     "debug level uppercase",
			input:    "DEBUG",
			expected: LevelDebug,
		},
		{
			name:     "info level",
			input:    "info",
			expected: LevelInfo,
		},
		{
			name:     "info level mixed case",
			input:    "InFo",
			expected: LevelInfo,
		},
		{
			name:     "warn level",
			input:    "warn",
			expected: LevelWarn,
		},
		{
			name:     "error level",
			input:    "error",
			expected: LevelError,
		},
		{
			name:        "invalid level",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var level Level
			err := level.Decode(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid log level")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, level)
			}
		})
	}
}

func TestFormat_Decode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Format
		expectError bool
	}{
		{
			name:     "json format",
			input:    "json",
			expected: FormatJSON,
		},
		{
			name:     "json format uppercase",
			input:    "JSON",
			expected: FormatJSON,
		},
		{
			name:     "text format",
			input:    "text",
			expected: FormatText,
		},
		{
			name:     "text format mixed case",
			input:    "TeXt",
			expected: FormatText,
		},
		{
			name:        "invalid format",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var format Format
			err := format.Decode(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid log format")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, format)
			}
		})
	}
}

func TestWithLogger_FromContext(t *testing.T) {
	logger := NewNop()

	ctx := context.Background()
	ctxWithLogger := WithLogger(ctx, logger)

	retrievedLogger := FromContext(ctxWithLogger)
	assert.Equal(t, logger, retrievedLogger)
}

func TestFromContext_NoLogger(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)

	assert.NotNil(t, logger)

	logger.Info("test message")
	logger.Error("test error")
	logger.Debug("test debug")
	logger.Warn("test warn")

	withLogger := logger.With(String("key", "value"))
	assert.NotNil(t, withLogger)
}

func TestFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), loggerKey{}, "not a logger")
	logger := FromContext(ctx)

	assert.NotNil(t, logger)
	logger.Info("test message")
}

func TestNopLogger_Methods(t *testing.T) {
	logger := NewNop()

	logger.Info("test info")
	logger.Error("test error")
	logger.Debug("test debug")
	logger.Warn("test warn")

	withLogger := logger.With(String("key", "value"))
	assert.Equal(t, logger, withLogger, "With should return the same nop logger")

	chainedLogger := logger.With(String("key1", "value1")).With(Int("key2", 42))
	assert.Equal(t, logger, chainedLogger)
}

func TestLoggerKey(t *testing.T) {
	key1 := loggerKey{}
	key2 := loggerKey{}

	assert.Equal(t, key1, key2)

	ctx := context.WithValue(context.Background(), key1, "value1")
	ctx = context.WithValue(ctx, key2, "value2")

	value := ctx.Value(loggerKey{})
	assert.Equal(t, "value2", value)
}

func TestConcurrentContextAccess(t *testing.T) {
	logger := NewNop()
	ctx := WithLogger(context.Background(), logger)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			retrievedLogger := FromContext(ctx)
			retrievedLogger.Info("concurrent test")
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func BenchmarkWithLogger(b *testing.B) {
	logger := NewNop()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithLogger(ctx, logger)
	}
}

func BenchmarkFromContext(b *testing.B) {
	logger := NewNop()
	ctx := WithLogger(context.Background(), logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FromContext(ctx)
	}
}

func BenchmarkFromContext_NoLogger(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FromContext(ctx)
	}
}
