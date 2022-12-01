package logger

import (
	"io"

	"go.uber.org/zap/zapcore"
)

// Level is a log level.
type Level = zapcore.Level

const (
	// DebugLevel is a debug log level.
	DebugLevel = zapcore.DebugLevel
	// ErrorLevel is an error log level.
	ErrorLevel = zapcore.ErrorLevel
)

// Config is the configuration for the logger.
type Config struct {
	Output io.Writer
	Level  Level
	// StripTime disables time variance in logger.
	StripTime bool
}
