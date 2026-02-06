package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(levelStr, formatStr string) (*zap.Logger, error) {
	level := parseLevel(levelStr)
	format := strings.ToLower(strings.TrimSpace(formatStr))

	var cfg zap.Config
	switch format {
	case "json":
		cfg = zap.NewProductionConfig()
	default:
		cfg = zap.NewDevelopmentConfig()
	}

	cfg.Level = zap.NewAtomicLevelAt(level)

	return cfg.Build()
}

func parseLevel(levelStr string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
