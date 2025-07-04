package common

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
	once   sync.Once
)

func getLogger() *zap.Logger {
	if logger == nil {
		initLogger()
	}
	return logger
}

func GetLogger() *zap.Logger {
	logger = getLogger()
	return logger.Named("default")
}

func GetLoggerWith(name string, fields ...zap.Field) *zap.Logger {
	logger = getLogger()
	return logger.Named(name).With(fields...)
}

func initLogger() {
	once.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Error getting current directory: %v", err)
		}

		logsDir := fmt.Sprintf("%s/logs", dir)
		logsFile := fmt.Sprintf("%s/app.log", logsDir)

		if err := os.MkdirAll(logsDir, os.ModePerm); err != nil {
			log.Fatalf("Error find/create logs directory: %v", err)
		}

		// TODO: set this in config
		logFile := &lumberjack.Logger{
			Filename:   logsFile,
			MaxSize:    10, // megabytes
			MaxBackups: 5,
			MaxAge:     28,   // days
			Compress:   true, // gzip
		}

		encoderCfg := zap.NewProductionEncoderConfig()
		encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(logFile),
			zap.InfoLevel,
		)

		if IsProduction() {
			logger = zap.New(fileCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		} else {
			consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
			consoleCore := zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zap.DebugLevel)

			combinedCore := zapcore.NewTee(fileCore, consoleCore)
			logger = zap.New(combinedCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		}
	})
}

func SetTestCaptureLogger(buf *bytes.Buffer, level zapcore.Level) {
	_ = GetLogger()

	writer := zapcore.AddSync(buf)
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderCfg)

	core := zapcore.NewCore(encoder, writer, level)
	logger = zap.New(core)
}

func SetTestLoggerNop() {
	_ = GetLogger()

	logger = zap.NewNop()
}
