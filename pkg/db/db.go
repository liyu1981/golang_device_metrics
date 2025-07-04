package db

import (
	"log"
	"os"
	"sync"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	constant "liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/models"
)

type DB struct {
	Conn *gorm.DB
}

var (
	instance *DB
	once     sync.Once
)

func GetInstance(dialector gorm.Dialector) *DB {
	var logger = constant.GetLogger()
	once.Do(func() {
		conn, err := gorm.Open(dialector, &gorm.Config{})
		if err != nil {
			log.Fatal("Failed to connect to database:", err)
		}

		logger.Info("Connected to database with dialector:", zap.String("dialector", dialector.Name()))

		instance = &DB{Conn: conn}

		err = instance.Conn.AutoMigrate(&models.Config{}, &models.Metric{}, &models.Alert{})
		if err != nil {
			log.Fatal("Failed to migrate database:", err)
		}

		logger.Info("Database migration completed")

		if err := instance.Conn.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			log.Fatal("Failed to enable sqlite foreign key support", err)
		}

		if err := instance.Conn.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
			log.Fatal("Failed to set sqlite journal mode", err)
		}
	})
	return instance
}

func UseSqliteDialector() gorm.Dialector {
	var dbPath string
	var found bool
	if dbPath, found = os.LookupEnv(constant.EnvKeyIOTDbPath); !found {
		dbPath = "metrics.db"
	}
	return sqlite.Open(dbPath)
}

func UseMemorySqliteDialector() gorm.Dialector {
	return sqlite.Open("file::memory:?cache=shared")
}
