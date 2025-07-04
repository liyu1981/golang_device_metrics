package iot

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"

	"liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/models"
	_ "liyu1981.xyz/iot-metrics-service/pkg/testing"
)

func TestCheckAndStoreAlerts(t *testing.T) {
	common.SetTestLoggerNop()

	ctrl, iotObj, _, _, _ := GetMockIOTWithMemorySqliteDialector(t, false, false, false)
	defer ctrl.Finish()

	deviceID := uuid.NewString()

	// Seed config
	config := models.Config{
		DeviceID:             deviceID,
		TemperatureThreshold: 30.0,
		BatteryThreshold:     20.0,
	}
	err := iotObj.Db.Conn.Create(&config).Error
	assert.NoError(t, err)

	// Create a metric that triggers both alerts
	metric := &models.Metric{
		DeviceID:    deviceID,
		Timestamp:   time.Now(),
		Temperature: 35.0, // triggers temperature alert
		Battery:     15.0, // triggers battery alert
	}

	iotObj.Alert.CheckAndStoreAlerts(deviceID, metric)

	// Check that 2 alerts were stored
	alerts, err := iotObj.Alert.GetDeviceAlerts(deviceID)
	assert.NoError(t, err)
	assert.Len(t, alerts, 2)

	// Assert alert types
	alertTypes := map[models.AlertType]bool{}
	for _, alert := range alerts {
		alertTypes[alert.Type] = true
	}

	assert.True(t, alertTypes[models.AlertTypeTemperature])
	assert.True(t, alertTypes[models.AlertTypeBattery])
}

func TestCheckAndStoreAlertsNoConfig(t *testing.T) {
	common.SetTestLoggerNop()

	ctrl, iotObj, _, _, _ := GetMockIOTWithMemorySqliteDialector(t, false, false, false)
	defer ctrl.Finish()

	deviceID := uuid.NewString()

	metric := &models.Metric{
		DeviceID:    deviceID,
		Timestamp:   time.Now(),
		Temperature: 100,
		Battery:     0,
	}

	// No config exists, so alerts shouldn't be stored
	iotObj.Alert.CheckAndStoreAlerts(deviceID, metric)

	alerts, err := iotObj.Alert.GetDeviceAlerts(deviceID)
	assert.NoError(t, err)
	assert.Len(t, alerts, 0)
}

func TestCheckAndStoreAlerts_WithLog(t *testing.T) {
	var buf = &bytes.Buffer{}
	common.SetTestCaptureLogger(buf, zapcore.InfoLevel)

	ctrl, iotObj, _, _, _ := GetMockIOTWithMemorySqliteDialector(t, false, false, false)
	defer ctrl.Finish()

	deviceID := uuid.NewString()

	// Seed config
	config := models.Config{
		DeviceID:             deviceID,
		TemperatureThreshold: 30.0,
		BatteryThreshold:     20.0,
	}
	err := iotObj.Db.Conn.Create(&config).Error
	assert.NoError(t, err)

	// Create a metric that triggers both alerts
	metric := &models.Metric{
		DeviceID:    deviceID,
		Timestamp:   time.Now(),
		Temperature: 35.0, // triggers temperature alert
		Battery:     15.0, // triggers battery alert
	}

	iotObj.Alert.CheckAndStoreAlerts(deviceID, metric)

	// Check that 2 alerts were stored
	alerts, err := iotObj.Alert.GetDeviceAlerts(deviceID)
	assert.NoError(t, err)
	assert.Len(t, alerts, 2)

	// Assert alert types
	alertTypes := map[models.AlertType]bool{}
	for _, alert := range alerts {
		alertTypes[alert.Type] = true
	}

	assert.True(t, alertTypes[models.AlertTypeTemperature])
	assert.True(t, alertTypes[models.AlertTypeBattery])

	logs := ParseLogs(buf)

	{
		found := false
		for _, log := range logs {
			lobj := log.(map[string]any)
			if lobj["category"] == "alert" &&
				lobj["logger"] == "iot_core" &&
				lobj["msg"] == "Alert found" &&
				lobj["alert"].(map[string]any)["DeviceID"] == deviceID &&
				lobj["alert"].(map[string]any)["Type"] == "battery" &&
				lobj["alert"].(map[string]any)["Message"] == "Battery 15.00 below threshold 20.00" {
				found = true
			}
		}
		assert.True(t, found)
	}

	{
		found := false
		for _, log := range logs {
			lobj := log.(map[string]any)
			if lobj["category"] == "alert" &&
				lobj["logger"] == "iot_core" &&
				lobj["msg"] == "Alert found" &&
				lobj["alert"].(map[string]any)["DeviceID"] == deviceID &&
				lobj["alert"].(map[string]any)["Type"] == "temperature" &&
				lobj["alert"].(map[string]any)["Message"] == "Temperature 35.00 exceeded threshold 30.00" {
				found = true
			}
		}
		assert.True(t, found)
	}

	{
		found := false
		for _, log := range logs {
			lobj := log.(map[string]any)
			if lobj["category"] == "alert" &&
				lobj["logger"] == "iot_core" &&
				lobj["msg"] == "Alert saved" &&
				lobj["alert"].(map[string]any)["DeviceID"] == deviceID &&
				lobj["alert"].(map[string]any)["Type"] == "temperature" &&
				lobj["alert"].(map[string]any)["Message"] == "Temperature 35.00 exceeded threshold 30.00" {
				found = true
			}
		}
		assert.True(t, found)
	}

	{
		found := false
		for _, log := range logs {
			lobj := log.(map[string]any)
			if lobj["category"] == "alert" &&
				lobj["logger"] == "iot_core" &&
				lobj["msg"] == "Alert saved" &&
				lobj["alert"].(map[string]any)["DeviceID"] == deviceID &&
				lobj["alert"].(map[string]any)["Type"] == "battery" &&
				lobj["alert"].(map[string]any)["Message"] == "Battery 15.00 below threshold 20.00" {
				found = true
			}
		}
		assert.True(t, found)
	}
}
