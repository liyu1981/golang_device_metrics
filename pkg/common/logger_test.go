package common

import (
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	_ "liyu1981.xyz/iot-metrics-service/pkg/testing"
)

func TestLoggingCapture(t *testing.T) {
	var buf bytes.Buffer
	SetTestCaptureLogger(&buf, zapcore.InfoLevel)

	{
		logger := GetLogger()
		logger.Info("Test log message", zap.String("key", "value"))

		logOutput := buf.String()
		if !strings.Contains(logOutput, "Test log message") {
			t.Errorf("expected log output to contain message, got: %s", logOutput)
		}
	}

	{
		logger := GetLoggerWith("test_logger", zap.String("category", "test_category"))
		logger.Info("Test log message", zap.String("key", "value"))

		logOutput := buf.String()
		if !strings.Contains(logOutput, "Test log message") || !strings.Contains(logOutput, "test_category") || !strings.Contains(logOutput, "test_logger") {
			t.Errorf("expected log output to contain message, got: %s", logOutput)
		}
	}
}

func TestLoggerOff(t *testing.T) {
	var buf bytes.Buffer
	SetTestCaptureLogger(&buf, zapcore.InfoLevel)

	SetTestLoggerNop()

	logger := GetLogger()
	logger.Info("Test log message", zap.String("key", "value"))

	logOutput := buf.String()
	if len(logOutput) > 0 {
		t.Errorf("expected log output to be empty, got: %s", logOutput)
	}
}
