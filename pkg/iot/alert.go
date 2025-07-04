package iot

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/models"
)

func (i *IOT) checkAndStoreAlerts(deviceID string, metric *models.Metric) error {
	db := i.Db

	var config models.Config
	if err := db.Conn.First(&config, "device_id = ?", deviceID).Error; err != nil {
		// no config, then no need to calcualte alerts
		return nil
	}

	logger := common.GetLoggerWith(
		common.LoggerNameIOTCore,
		zap.String(common.LoggerFieldIOTCategory, common.LoggerCategoryIOTAlert),
	)

	now := time.Now()

	if metric.Temperature > config.TemperatureThreshold {
		alert := models.Alert{
			DeviceID:  deviceID,
			Timestamp: now,
			Type:      models.AlertTypeTemperature,
			Message:   fmt.Sprintf("Temperature %.2f exceeded threshold %.2f", metric.Temperature, config.TemperatureThreshold),
		}

		logger.Info("Alert found", zap.Reflect("alert", alert))

		if err := db.Conn.Create(&alert).Error; err != nil {
			return err
		}

		logger.Info("Alert saved", zap.Reflect("alert", alert))
	}

	if metric.Battery < config.BatteryThreshold {
		alert := models.Alert{
			DeviceID:  deviceID,
			Timestamp: now,
			Type:      models.AlertTypeBattery,
			Message:   fmt.Sprintf("Battery %.2f below threshold %.2f", metric.Battery, config.BatteryThreshold),
		}

		logger.Info("Alert found", zap.Reflect("alert", alert))

		if err := db.Conn.Create(&alert).Error; err != nil {
			return err
		}

		logger.Info("Alert saved", zap.Reflect("alert", alert))
	}

	return nil
}

func (i *IOT) getDeviceAlerts(deviceID string) ([]models.Alert, error) {
	var alerts []models.Alert
	err := i.Db.Conn.
		Where("device_id = ?", deviceID).
		Order("timestamp desc").
		Find(&alerts).Error
	return alerts, err
}

type IAlertImpl struct {
	iot *IOT
}

func (ia *IAlertImpl) GetDeviceAlerts(deviceID string) ([]models.Alert, error) {
	return ia.iot.getDeviceAlerts(deviceID)
}

func (ia *IAlertImpl) CheckAndStoreAlerts(deviceID string, metric *models.Metric) error {
	return ia.iot.checkAndStoreAlerts(deviceID, metric)
}

func (i *IOT) GetIAlert() IAlert {
	return &IAlertImpl{iot: i}
}
