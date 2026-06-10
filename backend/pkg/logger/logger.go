// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package logger

import (
	"errors"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 包装 zap.Logger，提供结构化日志能力。
// 嵌入的 *zap.Logger 可直接使用 zap 的全部方法：
//   log.Info("msg", zap.String("key", "val"))
//   log.Infow("msg", "key", "val")
//   log.Error("msg", zap.Error(err))
type Logger struct {
	*zap.Logger
}

func New(level string) (*Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		return nil, errors.New("failed to build logger: " + err.Error())
	}

	return &Logger{logger}, nil
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}