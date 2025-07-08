package db

import (
	"os"
	"path/filepath"
	"testing"

	"liyu1981.xyz/iot-metrics-service/pkg/common"
	constant "liyu1981.xyz/iot-metrics-service/pkg/common"
)

func TestWithEnvPath(t *testing.T) {
	common.SetTestLoggerNop()

	if os.Getenv(constant.EnvKeyRunIntegrationTests) != "true" {
		t.Skip("Skipping integration test: RUN_INTEGRATION_TESTS environment variable not set")
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	testPath := filepath.Join(wd, "test.db")

	originalDBPath, hadOriginal := os.LookupEnv(constant.EnvKeyIOTDbPath)

	if err := os.Setenv(constant.EnvKeyIOTDbPath, testPath); err != nil {
		t.Fatalf("Failed to set DB_IOT_PATH: %v", err)
	}

	defer func() {
		if hadOriginal {
			_ = os.Setenv(constant.EnvKeyIOTDbPath, originalDBPath)
		} else {
			_ = os.Unsetenv(constant.EnvKeyIOTDbPath)
		}
		_ = os.Remove(testPath)
	}()

	instance := GetInstance(UseSqliteDialector())
	if instance == nil || instance.Conn == nil {
		t.Fatal("Expected non-nil DB connection")
	}

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Errorf("Expected database file to be created at %s", testPath)
	}
}
