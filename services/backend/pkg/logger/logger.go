package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger() {
	logLevel := zap.NewAtomicLevelAt(zap.InfoLevel)
	envLevel := os.Getenv("LOG_LEVEL")
	if envLevel != "" {
		if level, err := zapcore.ParseLevel(envLevel); err == nil {
			logLevel.SetLevel(level)
		}
	}

	config := zap.NewProductionConfig()
	config.Level = logLevel
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = true

	if os.Getenv("APP_ENV") == "development" {
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var err error
	Log, err = config.Build()
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(Log)
}

func GetLogger() *zap.Logger {
	if Log == nil {
		InitLogger()
	}
	return Log
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// NewBgContext returns a background context for goroutines not tied to an HTTP request.
func NewBgContext() context.Context {
	return context.Background()
}
