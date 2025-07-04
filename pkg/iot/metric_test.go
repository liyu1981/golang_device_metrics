package iot

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/models"
	_ "liyu1981.xyz/iot-metrics-service/pkg/testing"
)

func TestUpsertMetric(t *testing.T) {
	common.SetTestLoggerNop()

	ctrl, iotObj, _, mockIAlter, _ := GetMockIOTWithMemorySqliteDialector(t, false, true, false)
	defer ctrl.Finish()

	deviceID := uuid.NewString()

	var err error
	err = iotObj.Config.UpsertConfig(deviceID, &models.Config{
		DeviceID:             deviceID,
		TemperatureThreshold: 30.0,
		BatteryThreshold:     50.0,
	})
	assert.NoError(t, err)

	// Expect the alert checker to be called with correct args
	mockIAlter.
		EXPECT().
		CheckAndStoreAlerts(gomock.Eq(deviceID), gomock.Any()).
		Times(1)

	input := &models.Metric{
		Timestamp:   time.Now().Truncate(time.Second),
		Temperature: 30.2,
		Battery:     55.5,
	}
	err = iotObj.Metric.UpsertMetric(deviceID, input)
	assert.NoError(t, err)

	// Verify that the metric was inserted
	var saved models.Metric
	err = iotObj.Db.Conn.Where("device_id = ?", deviceID).First(&saved).Error
	assert.NoError(t, err)
	assert.Equal(t, input.Temperature, saved.Temperature)
}

func TestUpsertMetric_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	ctrl, iotObj, _, _, _ := GetMockIOTWithMemorySqliteDialector(t, false, false, false)
	defer ctrl.Finish()

	input := &models.Metric{
		Timestamp:   time.Now().Truncate(time.Second),
		Temperature: 30.2,
		Battery:     55.5,
	}

	deviceID := uuid.NewString()

	var err error
	err = iotObj.Metric.UpsertMetric(deviceID, input)
	require.Error(t, err, "FOREIGN KEY constraint failed")

	err = iotObj.Config.UpsertConfig(deviceID, &models.Config{
		DeviceID:             deviceID,
		TemperatureThreshold: 30.0,
		BatteryThreshold:     50.0,
	})
	assert.NoError(t, err)

	// force the alert service to be nil to cause alert not avaialable
	iotObj.Alert = nil

	err = iotObj.Metric.UpsertMetric(deviceID, input)
	require.Error(t, err, "alert service not available")
}
