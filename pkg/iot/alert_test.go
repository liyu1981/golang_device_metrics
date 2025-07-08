package iot

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"

	"liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/iot/mocks"
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

func TestUpsertAlert(t *testing.T) {
	common.SetTestLoggerNop()

	ctrl, iotObj, _, _, _ := GetMockIOTWithMemorySqliteDialector(t, false, false, false)
	defer ctrl.Finish()

	deviceID := uuid.NewString()

	{
		err := iotObj.Config.UpsertConfig(deviceID, &models.Config{
			DeviceID:             deviceID,
			TemperatureThreshold: 30.0,
			BatteryThreshold:     20.0,
		})

		assert.NoError(t, err)
	}

	{
		err := iotObj.Alert.UpsertAlert(&models.Alert{
			DeviceID:  deviceID,
			Timestamp: time.Now(),
			Type:      models.AlertTypeTemperature,
			Message:   fmt.Sprintf("Temperature %.2f exceeded threshold %.2f", 35.0, 30.0),
		})

		assert.NoError(t, err)
	}
}

type IAlertFallbackMock struct {
	iotObj     *IOT
	mockIAlert *mocks.MockIAlert
}

func (ia *IAlertFallbackMock) CheckAndStoreAlerts(deviceID string, metric *models.Metric) error {
	return ia.iotObj.checkAlerts(deviceID, metric, func(alert *models.Alert) error {
		return ia.UpsertAlert(alert)
	})
}

func (ia *IAlertFallbackMock) UpsertAlert(data *models.Alert) error {
	return ia.mockIAlert.UpsertAlert(data)
}

func (ia *IAlertFallbackMock) GetDeviceAlerts(deviceID string) ([]models.Alert, error) {
	return ia.iotObj.Alert.GetDeviceAlerts(deviceID)
}

func TestCheckAndStoreAlerts_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	ctrl, iotObj, _, mockIAlert, _ := GetMockIOTWithMemorySqliteDialector(t, false, false, false)
	defer ctrl.Finish()

	// replace alert services with our partial mock
	iotObj.Alert = &IAlertFallbackMock{
		iotObj:     iotObj,
		mockIAlert: mockIAlert,
	}

	deviceID := uuid.NewString()

	{
		err := iotObj.Config.UpsertConfig(deviceID, &models.Config{
			DeviceID:             deviceID,
			TemperatureThreshold: 30.0,
			BatteryThreshold:     50.0,
		})
		assert.NoError(t, err)
	}

	{
		// Create a metric that triggers both alerts
		metric := &models.Metric{
			DeviceID:    deviceID,
			Timestamp:   time.Now(),
			Temperature: 35.0, // triggers temperature alert
			Battery:     15.0, // triggers battery alert
		}

		mockIAlert.
			EXPECT().
			UpsertAlert(gomock.Any()).
			DoAndReturn(func(data *models.Alert) error {
				if data.Type == models.AlertTypeTemperature {
					return fmt.Errorf("save temperature alert error")
				}
				return nil
			}).
			Times(1)

		err := iotObj.Alert.CheckAndStoreAlerts(deviceID, metric)
		require.Error(t, err, "save temperature alert error")
	}

	{
		// Create a metric that triggers both alerts
		metric := &models.Metric{
			DeviceID:    deviceID,
			Timestamp:   time.Now(),
			Temperature: 35.0, // triggers temperature alert
			Battery:     15.0, // triggers battery alert
		}

		mockIAlert.
			EXPECT().
			UpsertAlert(gomock.Any()).
			DoAndReturn(func(data *models.Alert) error {
				if data.Type == models.AlertTypeBattery {
					return fmt.Errorf("save battery alert error")
				}
				return nil
			}).
			Times(2)

		err := iotObj.Alert.CheckAndStoreAlerts(deviceID, metric)
		require.Error(t, err, "save battery alert error")
	}
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
