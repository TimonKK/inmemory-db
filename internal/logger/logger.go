package logger

import (
	"errors"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultLoggerLevel      = zapcore.InfoLevel
	defaultLoggerOutputPath = "log.log"
)

var (
	ErrNilConfig          = errors.New("config is empty")
	ErrInvalidLoggerLevel = errors.New("invalid log level")
)

var supportedLoggingLevels = map[string]zapcore.Level{
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
}

func NewLogger(config *config.LoggingConfig) (*zap.Logger, error) {
	loggerLevel := defaultLoggerLevel
	loggerOutputPath := defaultLoggerOutputPath

	if config == nil {
		return nil, ErrNilConfig
	}

	if config.Level != "" {
		level, ok := supportedLoggingLevels[config.Level]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrInvalidLoggerLevel, config.Level)
		}

		loggerLevel = level
	}

	if config.Output != "" {
		loggerOutputPath = config.Output
	}

	loggerConfig := zap.Config{
		Encoding:    "json",
		Level:       zap.NewAtomicLevelAt(loggerLevel),
		OutputPaths: []string{loggerOutputPath},
	}

	return loggerConfig.Build()
}
