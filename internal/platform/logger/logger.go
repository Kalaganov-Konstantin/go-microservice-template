package logger

import (
	"context"
	"fmt"
	"strings"
)

type Config struct {
	Environment string
	Level       Level
	Format      Format
}

type Logger interface {
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	Warn(msg string, fields ...Field)

	With(fields ...Field) Logger
}

type Field struct {
	Key   string
	Value interface{}
}

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

func (l *Level) Decode(value string) error {
	switch strings.ToLower(value) {
	case "debug":
		*l = LevelDebug
	case "info":
		*l = LevelInfo
	case "warn":
		*l = LevelWarn
	case "error":
		*l = LevelError
	default:
		return fmt.Errorf("invalid log level: %s", value)
	}
	return nil
}

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

func (f *Format) Decode(value string) error {
	switch strings.ToLower(value) {
	case "json":
		*f = FormatJSON
	case "text":
		*f = FormatText
	default:
		return fmt.Errorf("invalid log format: %s", value)
	}
	return nil
}

type loggerKey struct{}

func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return logger
	}
	return &nopLogger{}
}
