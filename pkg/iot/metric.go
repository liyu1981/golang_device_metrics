package iot

import (
	"fmt"

	"go.uber.org/zap"
	"liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/models"
)

func (i *IOT) upsertMetric(deviceID string, input *models.Metric) error {
	logger := common.GetLoggerWith(
		common.LoggerNameIOTCore,
		zap.String(common.LoggerFieldIOTCategory, common.LoggerCategoryIOTMetric),
	)

	metric := models.Metric{
		DeviceID:    deviceID,
		Timestamp:   input.Timestamp,
		Temperature: input.Temperature,
		Battery:     input.Battery,
	}

	logger.Info("Received metric for device", zap.Reflect("metric", metric))

	if err := i.Db.Conn.Create(&metric).Error; err != nil {
		return err
	}

	logger.Info("Upserted metric for device,", zap.Reflect("metric", metric))

	if i.Alert == nil {
		return fmt.Errorf("alert service not available")
	}

	i.Alert.CheckAndStoreAlerts(deviceID, &metric)
	return nil
}

type IMetricImpl struct {
	iot *IOT
}

func (im *IMetricImpl) UpsertMetric(deviceID string, input *models.Metric) error {
	return im.iot.upsertMetric(deviceID, input)
}

func (i *IOT) GetIMetric() IMetric {
	return &IMetricImpl{iot: i}
}
