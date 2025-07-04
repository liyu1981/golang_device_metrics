package models

import "time"

type AlertType string

const (
	AlertTypeTemperature AlertType = "temperature"
	AlertTypeBattery     AlertType = "battery"
)

type Metric struct {
	ID          uint   `gorm:"primaryKey"`
	DeviceID    string `gorm:"index"`
	Timestamp   time.Time
	Temperature float64
	Battery     float64
}

type Config struct {
	DeviceID             string `gorm:"primaryKey"`
	TemperatureThreshold float64
	BatteryThreshold     float64

	Metrics []Metric `gorm:"foreignKey:DeviceID;references:DeviceID"`
	Alerts  []Alert  `gorm:"foreignKey:DeviceID;references:DeviceID"`
}

type Alert struct {
	ID        uint   `gorm:"primaryKey"`
	DeviceID  string `gorm:"index"`
	Timestamp time.Time
	Type      AlertType `gorm:"type:varchar(20);check:type IN ('temperature','battery')"`
	Message   string
}
