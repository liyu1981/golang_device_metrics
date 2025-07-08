package iot

import (
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
	"liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/models"
)

func (i *IOT) upsertConfig(deviceID string, input *models.Config) error {
	logger := common.GetLoggerWith(
		common.LoggerNameIOTCore,
		zap.String(common.LoggerFieldIOTCategory, common.LoggerCategoryIOTConfig),
	)

	config := models.Config{
		DeviceID:             deviceID,
		TemperatureThreshold: input.TemperatureThreshold,
		BatteryThreshold:     input.BatteryThreshold,
	}

	logger.Info("Received config for device", zap.Reflect("config", config))

	err := i.Db.Conn.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}},
		UpdateAll: true,
	}).Create(&config).Error

	if err == nil {
		logger.Info("Upserted config for device", zap.Reflect("config", config))
	}

	return err
}

func (i *IOT) getDeviceConfig(deviceID string) (*models.Config, error) {
	var config models.Config
	err := i.Db.Conn.First(&config, "device_id = ?", deviceID).Error
	return &config, err
}

type IConfigImpl struct {
	iot *IOT
}

func (ic *IConfigImpl) UpsertConfig(deviceID string, input *models.Config) error {
	return ic.iot.upsertConfig(deviceID, input)
}

func (ic *IConfigImpl) GetDeviceConfig(deviceID string) (*models.Config, error) {
	return ic.iot.getDeviceConfig(deviceID)
}

func (i *IOT) GetIConfig() IConfig {
	return &IConfigImpl{iot: i}
}
