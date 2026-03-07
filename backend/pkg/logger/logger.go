package logger

import (
	"context"
	"errors"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"octomanger/backend/config"
)

type traceIDKey struct{}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	trimmed := strings.TrimSpace(traceID)
	if trimmed == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDKey{}, trimmed)
}

func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value := ctx.Value(traceIDKey{}); value != nil {
		if traceID, ok := value.(string); ok {
			return strings.TrimSpace(traceID)
		}
	}
	return ""
}

func WithContext(ctx context.Context, log *zap.Logger) *zap.Logger {
	if log == nil {
		return nil
	}
	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		return log
	}
	return log.With(zap.String("trace_id", traceID))
}

func Init(cfg config.LoggingConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if err := level.Set(strings.ToLower(strings.TrimSpace(cfg.Level))); err != nil {
		return nil, err
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "ts"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	ws, err := buildWriteSyncer(cfg.File)
	if err != nil {
		return nil, err
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		ws,
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger, nil
}

func buildWriteSyncer(path string) (zapcore.WriteSyncer, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return zapcore.AddSync(os.Stdout), nil
	}

	file, err := os.OpenFile(trimmed, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil, errors.New("failed to open log file")
	}

	return zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),
		zapcore.AddSync(file),
	), nil
}
