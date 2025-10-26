package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Init initializes the global logger with the given level and format
func Init(level, format string) error {
	var err error

	// Determine log level
	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn", "warning":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// Base logger configuration
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      zapLevel == zapcore.DebugLevel,
		Encoding:         strings.ToLower(format), // "json" or "console"
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	// Build logger instance
	log, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	zap.ReplaceGlobals(log)
	return nil
}

// Get returns the current logger instance
func Get() *zap.Logger {
	if log == nil {
		panic("logger not initialized — call logger.Init() first")
	}
	return log
}

// Sync flushes any buffered log entries
func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}
