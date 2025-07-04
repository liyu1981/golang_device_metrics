package db

import (
	"sync"
	"testing"

	"liyu1981.xyz/iot-metrics-service/pkg/common"
	_ "liyu1981.xyz/iot-metrics-service/pkg/testing"

	"gorm.io/gorm"
)

func tableExists(db *gorm.DB, tableName string) bool {
	var count int64
	err := db.Raw(
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, tableName,
	).Scan(&count).Error
	return err == nil && count > 0
}

func TestWithMemorySqlite(t *testing.T) {
	common.SetTestLoggerNop()

	dialector := UseMemorySqliteDialector()

	instance := GetInstance(dialector)
	if instance == nil {
		t.Fatal("Expected non-nil DB instance")
	}

	var tables = []string{"metrics", "configs", "alerts"}
	for _, table := range tables {
		if !tableExists(instance.Conn, table) {
			t.Errorf("Expected table %q to exist after migration", table)
		}
	}
}

func TestSingletonConcurrency(t *testing.T) {
	common.SetTestLoggerNop()

	const goroutineCount = 20

	var wg sync.WaitGroup
	instances := make(chan *DB, goroutineCount)

	for range goroutineCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			instance := GetInstance(UseMemorySqliteDialector())
			instances <- instance
		}()
	}

	wg.Wait()
	close(instances)

	var first *DB
	for inst := range instances {
		if first == nil {
			first = inst
			continue
		}
		if inst != first {
			t.Error("Expected all instances to be the same (singleton), but found different ones")
		}
	}
}
