package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	logger *zap.Logger
}

func NewZapLogger(config Config) (Logger, error) {
	var zapConfig zap.Config
	switch config.Environment {
	case "development":
		zapConfig = zap.NewDevelopmentConfig()
	case "production":
		zapConfig = zap.NewProductionConfig()
	case "staging":
		zapConfig = zap.NewProductionConfig()
	case "test":
		zapConfig = zap.NewProductionConfig()
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	default:
		zapConfig = zap.NewProductionConfig()
	}

	zapConfig.Level = zap.NewAtomicLevelAt(parseZapLevel(config.Level))

	switch config.Format {
	case FormatJSON:
		zapConfig.Encoding = "json"
	case FormatText:
		zapConfig.Encoding = "console"
	default:
		zapConfig.Encoding = "json"
	}

	logger, err := zapConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	return &zapLogger{
		logger: logger,
	}, nil
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fieldsToZapFields(fields)...)
}

func (l *zapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fieldsToZapFields(fields)...)
}

func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fieldsToZapFields(fields)...)
}

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fieldsToZapFields(fields)...)
}

func (l *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{
		logger: l.logger.With(fieldsToZapFields(fields)...),
	}
}

func parseZapLevel(level Level) zapcore.Level {
	switch level {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func fieldsToZapFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		switch v := field.Value.(type) {
		case string:
			zapFields = append(zapFields, zap.String(field.Key, v))
		case int:
			zapFields = append(zapFields, zap.Int(field.Key, v))
		case error:
			zapFields = append(zapFields, zap.Error(v))
		default:
			zapFields = append(zapFields, zap.Any(field.Key, v))
		}
	}
	return zapFields
}
