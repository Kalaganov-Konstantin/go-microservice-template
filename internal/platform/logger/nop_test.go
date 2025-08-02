package logger

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type NopLoggerTestSuite struct {
	suite.Suite
}

func (s *NopLoggerTestSuite) TestNewNop() {
	logger := NewNop()
	s.Assert().NotNil(logger)
	s.Assert().IsType(&nopLogger{}, logger)
}

func (s *NopLoggerTestSuite) TestNopLogger_Info() {
	logger := NewNop()

	s.Assert().NotPanics(func() {
		logger.Info("test message")
	})

	s.Assert().NotPanics(func() {
		logger.Info("test message", String("key", "value"))
	})

	s.Assert().NotPanics(func() {
		logger.Info("test message", String("key1", "value1"), Int("key2", 42))
	})
}

func (s *NopLoggerTestSuite) TestNopLogger_Error() {
	logger := NewNop()

	s.Assert().NotPanics(func() {
		logger.Error("test error")
	})

	s.Assert().NotPanics(func() {
		logger.Error("test error", Error(errors.New("sample error")))
	})

	s.Assert().NotPanics(func() {
		logger.Error("test error", String("context", "test"), Int("code", 500))
	})
}

func (s *NopLoggerTestSuite) TestNopLogger_Debug() {
	logger := NewNop()

	s.Assert().NotPanics(func() {
		logger.Debug("debug message")
	})

	s.Assert().NotPanics(func() {
		logger.Debug("debug message", String("component", "test"))
	})
}

func (s *NopLoggerTestSuite) TestNopLogger_Warn() {
	logger := NewNop()

	s.Assert().NotPanics(func() {
		logger.Warn("warning message")
	})

	s.Assert().NotPanics(func() {
		logger.Warn("warning message", String("reason", "deprecated"))
	})
}

func (s *NopLoggerTestSuite) TestNopLogger_With() {
	logger := NewNop()

	newLogger := logger.With(String("component", "test"))
	s.Assert().NotNil(newLogger)
	s.Assert().Equal(logger, newLogger)

	anotherLogger := newLogger.With(Int("version", 1), Error(errors.New("test")))
	s.Assert().Equal(logger, anotherLogger)
}

func (s *NopLoggerTestSuite) TestNopLogger_WithComplexFields() {
	logger := NewNop()

	s.Assert().NotPanics(func() {
		complexLogger := logger.With(
			String("service", "microservice"),
			Int("port", 8080),
			Error(errors.New("connection failed")),
			Field{Key: "custom", Value: map[string]interface{}{
				"nested": "value",
				"count":  100,
			}},
		)

		s.Assert().Equal(logger, complexLogger)

		complexLogger.Info("test info")
		complexLogger.Error("test error")
		complexLogger.Debug("test debug")
		complexLogger.Warn("test warn")
	})
}

func (s *NopLoggerTestSuite) TestNopLogger_AllMethodsWithMixedFields() {
	logger := NewNop()

	fields := []Field{
		String("service", "test"),
		Int("attempt", 3),
		Error(errors.New("sample error")),
		{Key: "metadata", Value: struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{
			Name: "test",
			Age:  25,
		}},
	}

	s.Assert().NotPanics(func() {
		logger.Info("info with mixed fields", fields...)
		logger.Error("error with mixed fields", fields...)
		logger.Debug("debug with mixed fields", fields...)
		logger.Warn("warn with mixed fields", fields...)
	})
}

func (s *NopLoggerTestSuite) TestNopLogger_EmptyFields() {
	logger := NewNop()

	s.Assert().NotPanics(func() {
		logger.Info("message", []Field{}...)
		logger.Error("message", []Field{}...)
		logger.Debug("message", []Field{}...)
		logger.Warn("message", []Field{}...)
	})
}

func (s *NopLoggerTestSuite) TestNopLogger_NilFields() {
	logger := NewNop()

	s.Assert().NotPanics(func() {
		logger.Info("message", Field{Key: "nil_value", Value: nil})
		logger.Error("message", Field{Key: "nil_value", Value: nil})
		logger.Debug("message", Field{Key: "nil_value", Value: nil})
		logger.Warn("message", Field{Key: "nil_value", Value: nil})
	})

	s.Assert().NotPanics(func() {
		newLogger := logger.With(Field{Key: "nil_field", Value: nil})
		s.Assert().Equal(logger, newLogger)
	})
}

func (s *NopLoggerTestSuite) TestNopLogger_Performance() {
	logger := NewNop()

	for i := 0; i < 1000; i++ {
		logger.Info("performance test", String("iteration", fmt.Sprintf("%d", i)))
		logger.Error("performance test", Int("iteration", i))
		logger.Debug("performance test", String("type", "debug"))
		logger.Warn("performance test", String("type", "warn"))
	}

	for i := 0; i < 100; i++ {
		newLogger := logger.With(String("test", "value"), Int("count", i))
		s.Assert().Equal(logger, newLogger)
	}
}

func (s *NopLoggerTestSuite) TestNopLogger_Chaining() {
	logger := NewNop()
	result := logger.
		With(String("component", "test")).
		With(Int("version", 1)).
		With(Error(errors.New("test error")))

	s.Assert().Equal(logger, result)
	s.Assert().NotPanics(func() {
		result.Info("chained logger test")
		result.Error("chained error test")
		result.Debug("chained debug test")
		result.Warn("chained warn test")
	})
}

func BenchmarkNewNop(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger := NewNop()
		_ = logger
	}
}

func BenchmarkNopLogger_Info(b *testing.B) {
	logger := NewNop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", String("key", "value"), Int("number", i))
	}
}

func BenchmarkNopLogger_Error(b *testing.B) {
	logger := NewNop()
	testErr := errors.New("benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error("benchmark error", Error(testErr), Int("iteration", i))
	}
}

func BenchmarkNopLogger_With(b *testing.B) {
	logger := NewNop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newLogger := logger.With(String("component", "test"), Int("version", i))
		_ = newLogger
	}
}

func BenchmarkNopLogger_AllMethods(b *testing.B) {
	logger := NewNop()
	fields := []Field{String("test", "value"), Int("count", 42)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("info", fields...)
		logger.Error("error", fields...)
		logger.Debug("debug", fields...)
		logger.Warn("warn", fields...)
	}
}

func TestNopLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(NopLoggerTestSuite))
}
