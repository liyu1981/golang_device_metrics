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

	logger := GetLogger()
	logger.Info("Test log message", zap.String("key", "value"))

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Test log message") {
		t.Errorf("expected log output to contain message, got: %s", logOutput)
	}
}
