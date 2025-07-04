package iot

import (
	"bufio"
	"encoding/json"
	"io"
	"testing"

	"go.uber.org/mock/gomock"
	"liyu1981.xyz/iot-metrics-service/pkg/db"
	"liyu1981.xyz/iot-metrics-service/pkg/iot/mocks"
)

func GetMockIOTWithMemorySqliteDialector(t *testing.T, useMockIMetric, useMockIAlert, useMockIConfig bool) (
	*gomock.Controller,
	*IOT,
	*mocks.MockIMetric,
	*mocks.MockIAlert,
	*mocks.MockIConfig,
) {
	ctrl := gomock.NewController(t)

	mockIMetric := mocks.NewMockIMetric(ctrl)
	mockIAlter := mocks.NewMockIAlert(ctrl)
	mockIConfig := mocks.NewMockIConfig(ctrl)
	dialector := db.UseMemorySqliteDialector()
	dbInstance := db.GetInstance(dialector) // ensure migrations
	iotInstance := (&IOT{Db: *dbInstance})

	metricService := iotInstance.GetIMetric()
	if useMockIMetric {
		metricService = mockIMetric
	}

	configService := iotInstance.GetIConfig()
	if useMockIConfig {
		configService = mockIConfig
	}

	alertService := iotInstance.GetIAlert()
	if useMockIAlert {
		alertService = mockIAlter
	}

	iotInstance.WithServices(ServiceOpts{
		Metric: metricService,
		Alert:  alertService,
		Config: configService,
	})

	return ctrl, iotInstance, mockIMetric, mockIAlter, mockIConfig
}

func ParseLogs(r io.Reader) []any {
	scanner := bufio.NewScanner(r)
	var logs []any

	for scanner.Scan() {
		line := scanner.Text()
		var j any
		if err := json.Unmarshal([]byte(line), &j); err == nil {
			logs = append(logs, j)
		}
	}
	return logs
}
