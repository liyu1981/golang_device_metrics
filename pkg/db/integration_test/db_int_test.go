package test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	constant "liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/db"
)

func TestWithEnvPath(t *testing.T) {
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

	fmt.Println(os.Getenv(constant.EnvKeyIOTDbPath))

	instance := db.GetInstance(db.UseSqliteDialector())
	if instance == nil || instance.Conn == nil {
		t.Fatal("Expected non-nil DB connection")
	}

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Errorf("Expected database file to be created at %s", testPath)
	}
}
