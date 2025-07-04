package iot

import (
	"liyu1981.xyz/iot-metrics-service/pkg/db"
	"liyu1981.xyz/iot-metrics-service/pkg/models"
)

type IMetric interface {
	UpsertMetric(deviceID string, input *models.Metric) error
}

type IAlert interface {
	CheckAndStoreAlerts(deviceID string, metric *models.Metric) error
	GetDeviceAlerts(deviceID string) ([]models.Alert, error)
}

type IConfig interface {
	UpsertConfig(deviceID string, input *models.Config) error
}

type IOT struct {
	Db     db.DB
	Metric IMetric
	Alert  IAlert
	Config IConfig
}

type ServiceOpts struct {
	Metric IMetric
	Alert  IAlert
	Config IConfig
}

func (i *IOT) WithServices(opts ServiceOpts) *IOT {
	if opts.Metric != nil {
		i.Metric = opts.Metric
	}
	if opts.Alert != nil {
		i.Alert = opts.Alert
	}
	if opts.Config != nil {
		i.Config = opts.Config
	}
	return i
}
