package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapAdapterTestSuite struct {
	suite.Suite
	buffer *bytes.Buffer
}

func (s *ZapAdapterTestSuite) SetupTest() {
	s.buffer = &bytes.Buffer{}
}

func (s *ZapAdapterTestSuite) TestNewZapLogger_Development() {
	config := Config{
		Environment: "development",
		Level:       LevelDebug,
		Format:      FormatJSON,
	}

	logger, err := NewZapLogger(config)
	s.Assert().NoError(err)
	s.Assert().NotNil(logger)
}

func (s *ZapAdapterTestSuite) TestNewZapLogger_Production() {
	config := Config{
		Environment: "production",
		Level:       LevelInfo,
		Format:      FormatJSON,
	}

	logger, err := NewZapLogger(config)
	s.Assert().NoError(err)
	s.Assert().NotNil(logger)
}

func (s *ZapAdapterTestSuite) TestNewZapLogger_Staging() {
	config := Config{
		Environment: "staging",
		Level:       LevelWarn,
		Format:      FormatText,
	}

	logger, err := NewZapLogger(config)
	s.Assert().NoError(err)
	s.Assert().NotNil(logger)
}

func (s *ZapAdapterTestSuite) TestNewZapLogger_Test() {
	config := Config{
		Environment: "test",
		Level:       LevelError,
		Format:      FormatJSON,
	}

	logger, err := NewZapLogger(config)
	s.Assert().NoError(err)
	s.Assert().NotNil(logger)
}

func (s *ZapAdapterTestSuite) TestNewZapLogger_Unknown() {
	config := Config{
		Environment: "unknown",
		Level:       LevelInfo,
		Format:      FormatJSON,
	}

	logger, err := NewZapLogger(config)
	s.Assert().NoError(err)
	s.Assert().NotNil(logger)
}

func (s *ZapAdapterTestSuite) TestNewZapLogger_AllLevels() {
	levels := []Level{LevelDebug, LevelInfo, LevelWarn, LevelError}

	for _, level := range levels {
		s.Run(string(level), func() {
			config := Config{
				Environment: "production",
				Level:       level,
				Format:      FormatJSON,
			}

			logger, err := NewZapLogger(config)
			s.Assert().NoError(err)
			s.Assert().NotNil(logger)
		})
	}
}

func (s *ZapAdapterTestSuite) TestNewZapLogger_AllFormats() {
	formats := []Format{FormatJSON, FormatText}

	for _, format := range formats {
		s.Run(string(format), func() {
			config := Config{
				Environment: "production",
				Level:       LevelInfo,
				Format:      format,
			}

			logger, err := NewZapLogger(config)
			s.Assert().NoError(err)
			s.Assert().NotNil(logger)
		})
	}
}

func (s *ZapAdapterTestSuite) TestNewZapLogger_UnknownFormat() {
	config := Config{
		Environment: "production",
		Level:       LevelInfo,
		Format:      Format("unknown"),
	}

	logger, err := NewZapLogger(config)
	s.Assert().NoError(err)
	s.Assert().NotNil(logger)
}

func (s *ZapAdapterTestSuite) TestZapLogger_LoggingMethods() {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(s.buffer),
		zapcore.DebugLevel,
	)
	zapLoggerInstance := zap.New(core)

	config := Config{
		Environment: "test",
		Level:       LevelDebug,
		Format:      FormatJSON,
	}
	logger, err := NewZapLogger(config)
	s.Require().NoError(err)

	zapAdapter := logger.(*zapLogger)
	zapAdapter.logger = zapLoggerInstance

	logger.Info("test info message", String("key", "value"))
	s.Assert().Contains(s.buffer.String(), "test info message")
	s.Assert().Contains(s.buffer.String(), "\"level\":\"info\"")
	s.buffer.Reset()

	logger.Error("test error message", String("key", "value"))
	s.Assert().Contains(s.buffer.String(), "test error message")
	s.Assert().Contains(s.buffer.String(), "\"level\":\"error\"")
	s.buffer.Reset()

	logger.Debug("test debug message", String("key", "value"))
	s.Assert().Contains(s.buffer.String(), "test debug message")
	s.Assert().Contains(s.buffer.String(), "\"level\":\"debug\"")
	s.buffer.Reset()

	logger.Warn("test warn message", String("key", "value"))
	s.Assert().Contains(s.buffer.String(), "test warn message")
	s.Assert().Contains(s.buffer.String(), "\"level\":\"warn\"")
	s.buffer.Reset()
}

func (s *ZapAdapterTestSuite) TestZapLogger_With() {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(s.buffer),
		zapcore.DebugLevel,
	)
	zapLoggerInstance := zap.New(core)

	config := Config{
		Environment: "test",
		Level:       LevelDebug,
		Format:      FormatJSON,
	}
	logger, err := NewZapLogger(config)
	s.Require().NoError(err)

	zapAdapter := logger.(*zapLogger)
	zapAdapter.logger = zapLoggerInstance

	newLogger := logger.With(String("component", "test"), Int("version", 1))
	s.Assert().NotNil(newLogger)
	s.Assert().NotEqual(logger, newLogger)

	newLogger.Info("test message")
	output := s.buffer.String()
	s.Assert().Contains(output, "component")
	s.Assert().Contains(output, "test")
	s.Assert().Contains(output, "version")
	s.Assert().Contains(output, "1")
}

func (s *ZapAdapterTestSuite) TestParseZapLevel() {
	tests := []struct {
		input    Level
		expected zapcore.Level
	}{
		{LevelDebug, zapcore.DebugLevel},
		{LevelInfo, zapcore.InfoLevel},
		{LevelWarn, zapcore.WarnLevel},
		{LevelError, zapcore.ErrorLevel},
		{Level("unknown"), zapcore.InfoLevel},
	}

	for _, test := range tests {
		s.Run(string(test.input), func() {
			result := parseZapLevel(test.input)
			s.Assert().Equal(test.expected, result)
		})
	}
}

func (s *ZapAdapterTestSuite) TestFieldsToZapFields_StringField() {
	fields := []Field{String("key", "value")}
	zapFields := fieldsToZapFields(fields)

	s.Assert().Len(zapFields, 1)
	s.Assert().Equal("key", zapFields[0].Key)
	s.Assert().Equal("value", zapFields[0].String)
}

func (s *ZapAdapterTestSuite) TestFieldsToZapFields_IntField() {
	fields := []Field{Int("count", 42)}
	zapFields := fieldsToZapFields(fields)

	s.Assert().Len(zapFields, 1)
	s.Assert().Equal("count", zapFields[0].Key)
	s.Assert().Equal(int64(42), zapFields[0].Integer)
}

func (s *ZapAdapterTestSuite) TestFieldsToZapFields_ErrorField() {
	testErr := errors.New("test error")
	fields := []Field{Error(testErr)}
	zapFields := fieldsToZapFields(fields)

	s.Assert().Len(zapFields, 1)
	s.Assert().Equal("error", zapFields[0].Key)
}

func (s *ZapAdapterTestSuite) TestFieldsToZapFields_AnyField() {
	type customStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	custom := customStruct{Name: "test", Age: 25}
	fields := []Field{{Key: "custom", Value: custom}}
	zapFields := fieldsToZapFields(fields)

	s.Assert().Len(zapFields, 1)
	s.Assert().Equal("custom", zapFields[0].Key)
}

func (s *ZapAdapterTestSuite) TestFieldsToZapFields_MixedFields() {
	testErr := errors.New("test error")
	fields := []Field{
		String("message", "hello"),
		Int("count", 10),
		Error(testErr),
		{Key: "custom", Value: map[string]interface{}{"key": "value"}},
	}

	zapFields := fieldsToZapFields(fields)
	s.Assert().Len(zapFields, 4)
	s.Assert().Equal("message", zapFields[0].Key)
	s.Assert().Equal("count", zapFields[1].Key)
	s.Assert().Equal("error", zapFields[2].Key)
	s.Assert().Equal("custom", zapFields[3].Key)
}

func (s *ZapAdapterTestSuite) TestFieldsToZapFields_EmptyFields() {
	var fields []Field
	zapFields := fieldsToZapFields(fields)

	s.Assert().Len(zapFields, 0)
	s.Assert().NotNil(zapFields)
}

func (s *ZapAdapterTestSuite) TestZapLogger_Integration() {
	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	zapConfig.Encoding = "json"

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapConfig.EncoderConfig),
		zapcore.AddSync(s.buffer),
		zapcore.DebugLevel,
	)
	zapLoggerInstance := zap.New(core, zap.AddCallerSkip(1))

	config := Config{
		Environment: "production",
		Level:       LevelDebug,
		Format:      FormatJSON,
	}
	logger, err := NewZapLogger(config)
	s.Require().NoError(err)

	zapAdapter := logger.(*zapLogger)
	zapAdapter.logger = zapLoggerInstance

	logger.Info("integration test",
		String("component", "test"),
		Int("attempt", 1),
		Error(errors.New("sample error")),
	)

	output := s.buffer.String()
	s.Assert().NotEmpty(output)

	var logEntry map[string]interface{}
	parseErr := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	s.Assert().NoError(parseErr)

	s.Assert().Equal("info", logEntry["level"])
	s.Assert().Equal("integration test", logEntry["msg"])
	s.Assert().Equal("test", logEntry["component"])
	s.Assert().Equal(float64(1), logEntry["attempt"])
	s.Assert().Contains(logEntry, "error")
}

func (s *ZapAdapterTestSuite) TestZapLogger_LevelFiltering() {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(s.buffer),
		zapcore.WarnLevel,
	)
	zapLoggerInstance := zap.New(core)

	config := Config{
		Environment: "production",
		Level:       LevelWarn,
		Format:      FormatJSON,
	}
	logger, err := NewZapLogger(config)
	s.Require().NoError(err)

	zapAdapter := logger.(*zapLogger)
	zapAdapter.logger = zapLoggerInstance

	logger.Debug("debug message")
	logger.Info("info message")
	s.Assert().Empty(s.buffer.String())

	logger.Warn("warn message")
	s.Assert().Contains(s.buffer.String(), "warn message")

	s.buffer.Reset()
	logger.Error("error message")
	s.Assert().Contains(s.buffer.String(), "error message")
}

func (s *ZapAdapterTestSuite) TestZapLogger_Performance() {
	config := Config{
		Environment: "production",
		Level:       LevelInfo,
		Format:      FormatJSON,
	}

	logger, err := NewZapLogger(config)
	s.Require().NoError(err)

	for i := 0; i < 100; i++ {
		logger.Info("performance test", String("iteration", fmt.Sprintf("%d", i)))
	}
}

func BenchmarkNewZapLogger(b *testing.B) {
	config := Config{
		Environment: "production",
		Level:       LevelInfo,
		Format:      FormatJSON,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger, err := NewZapLogger(config)
		if err != nil {
			b.Fatal(err)
		}
		_ = logger
	}
}

func BenchmarkFieldsToZapFields(b *testing.B) {
	fields := []Field{
		String("message", "test"),
		Int("count", 42),
		Error(errors.New("test error")),
		{Key: "custom", Value: map[string]string{"key": "value"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fieldsToZapFields(fields)
	}
}

func BenchmarkParseZapLevel(b *testing.B) {
	levels := []Level{LevelDebug, LevelInfo, LevelWarn, LevelError}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, level := range levels {
			_ = parseZapLevel(level)
		}
	}
}

func TestZapAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(ZapAdapterTestSuite))
}
