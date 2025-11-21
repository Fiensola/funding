package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(level, encoding string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		"stdout",
		"funding.log",
	}

	return config.Build()
}
