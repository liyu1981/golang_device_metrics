package iot

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"

	"liyu1981.xyz/iot-metrics-service/pkg/common" // generated mock folder
	"liyu1981.xyz/iot-metrics-service/pkg/models"
	_ "liyu1981.xyz/iot-metrics-service/pkg/testing"
)

func TestUpsertConfig(t *testing.T) {
	common.SetTestLoggerNop()

	ctrl, iotObj, _, _, _ := GetMockIOTWithMemorySqliteDialector(t, false, false, false)
	defer ctrl.Finish()

	deviceID := uuid.NewString()

	input := &models.Config{
		TemperatureThreshold: 30.0,
		BatteryThreshold:     50.0,
	}

	// Call UpsertConfig and verify no error
	err := iotObj.Config.UpsertConfig(deviceID, input)
	assert.NoError(t, err)

	// Verify the configuration was inserted into the database
	var savedConfig models.Config
	err = iotObj.Db.Conn.Where("device_id = ?", deviceID).First(&savedConfig).Error
	assert.NoError(t, err)
	assert.Equal(t, input.TemperatureThreshold, savedConfig.TemperatureThreshold)
	assert.Equal(t, input.BatteryThreshold, savedConfig.BatteryThreshold)

	// Update the configuration with new values
	updatedInput := &models.Config{
		TemperatureThreshold: 35.0,
		BatteryThreshold:     60.0,
	}
	err = iotObj.Config.UpsertConfig(deviceID, updatedInput)
	assert.NoError(t, err)

	// Verify the updated configuration
	var updatedConfig models.Config
	err = iotObj.Db.Conn.Where("device_id = ?", deviceID).First(&updatedConfig).Error
	assert.NoError(t, err)
	assert.Equal(t, updatedInput.TemperatureThreshold, updatedConfig.TemperatureThreshold)
	assert.Equal(t, updatedInput.BatteryThreshold, updatedConfig.BatteryThreshold)
}

func TestUpsertConfig_WithLog(t *testing.T) {
	var buf = &bytes.Buffer{}
	common.SetTestCaptureLogger(buf, zapcore.InfoLevel)

	ctrl, iotObj, _, _, _ := GetMockIOTWithMemorySqliteDialector(t, false, false, false)
	defer ctrl.Finish()

	deviceID := uuid.NewString()

	{
		input := &models.Config{
			TemperatureThreshold: 30.0,
			BatteryThreshold:     50.0,
		}

		err := iotObj.Config.UpsertConfig(deviceID, input)
		assert.NoError(t, err)
	}

	logs := ParseLogs(buf)

	{
		found := false
		for _, log := range logs {
			lobj := log.(map[string]any)
			if lobj["category"] == "config" &&
				lobj["logger"] == "iot_core" &&
				lobj["msg"] == "Received config for device" &&
				lobj["config"].(map[string]any)["DeviceID"] == deviceID &&
				lobj["config"].(map[string]any)["TemperatureThreshold"] == 30.0 &&
				lobj["config"].(map[string]any)["BatteryThreshold"] == 50.0 {
				found = true
			}
		}
		assert.True(t, found, "log not found")
	}

	{
		found := false
		for _, log := range logs {
			lobj := log.(map[string]any)
			if lobj["category"] == "config" &&
				lobj["logger"] == "iot_core" &&
				lobj["msg"] == "Upserted config for device" &&
				lobj["config"].(map[string]any)["DeviceID"] == deviceID &&
				lobj["config"].(map[string]any)["TemperatureThreshold"] == 30.0 &&
				lobj["config"].(map[string]any)["BatteryThreshold"] == 50.0 {
				found = true
			}
		}
		assert.True(t, found, "log not found")
	}

}
