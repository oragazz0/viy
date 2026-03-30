package observability

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a structured JSON logger at the given level.
func NewLogger(level string) (*zap.Logger, error) {
	parsedLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(parsedLevel),
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			CallerKey:      "caller",
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
		},
	}

	return config.Build()
}
