package grpc

import (
	"context"
	"fmt"

	z "github.com/Oudwins/zog"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/types/known/timestamppb"
	"liyu1981.xyz/iot-metrics-service/pkg/common"
	pb "liyu1981.xyz/iot-metrics-service/pkg/grpc/iot_metric_service"
	"liyu1981.xyz/iot-metrics-service/pkg/models"
)

func validateDeviceID(deviceID *string) z.ZogIssueList {
	var deviceIdValidator = z.String().Min(1).Required()
	return deviceIdValidator.Validate(deviceID)
}

func (s *IOTServer) PostMetrics(ctx context.Context, req *pb.PostMetricsRequest) (*pb.PostMetricsResponse, error) {
	if err := validateDeviceID(&req.DeviceId); err != nil {
		return &pb.PostMetricsResponse{Status: &pb.StatusResponse{Success: false, Message: fmt.Sprintf("validation error: %v", err)}}, nil
	}

	{
		var metricValidator = z.Struct(z.Shape{
			// Timestamp need to be validated separately
			"Temperature": z.Float64().Required(),
			"Battery":     z.Float64().Required(),
		})

		if err := metricValidator.Validate(req.Metric); err != nil {
			return &pb.PostMetricsResponse{Status: &pb.StatusResponse{Success: false, Message: fmt.Sprintf("validation error: %v", err)}}, nil
		}
	}

	{
		if req.Metric.Timestamp == nil {
			return &pb.PostMetricsResponse{Status: &pb.StatusResponse{Success: false, Message: "validation error: timestamp can not be empty"}}, nil
		}
		t := req.Metric.Timestamp.AsTime()
		if t.IsZero() {
			return &pb.PostMetricsResponse{Status: &pb.StatusResponse{Success: false, Message: "validation error: timestamp can not be parsed"}}, nil
		}
	}

	err := s.Iot.Metric.UpsertMetric(req.DeviceId, &models.Metric{
		Timestamp:   req.Metric.Timestamp.AsTime(),
		Temperature: req.Metric.Temperature,
		Battery:     req.Metric.Battery,
	})

	if err != nil {
		return &pb.PostMetricsResponse{
			Status: &pb.StatusResponse{Success: false, Message: err.Error()},
		}, nil
	}

	return &pb.PostMetricsResponse{
		Status: &pb.StatusResponse{Success: true, Message: "OK"},
	}, nil
}

func (s *IOTServer) UpdateConfig(ctx context.Context, req *pb.UpdateConfigRequest) (*pb.UpdateConfigResponse, error) {
	if err := validateDeviceID(&req.DeviceId); err != nil {
		return &pb.UpdateConfigResponse{Status: &pb.StatusResponse{Success: false, Message: fmt.Sprintf("validation error: %v", err)}}, nil
	}

	var updateConfigValidator = z.Struct(z.Shape{
		"TemperatureThreshold": z.Float64().Required(),
		"BatteryThreshold":     z.Float64().Required(),
	})

	if err := updateConfigValidator.Validate(req.Config); err != nil {
		return &pb.UpdateConfigResponse{Status: &pb.StatusResponse{Success: false, Message: fmt.Sprintf("validation error: %v", err)}}, nil
	}

	payload := models.Config{
		TemperatureThreshold: req.Config.TemperatureThreshold,
		BatteryThreshold:     req.Config.BatteryThreshold,
	}

	err := s.Iot.Config.UpsertConfig(req.DeviceId, &payload)

	if err != nil {
		return &pb.UpdateConfigResponse{Status: &pb.StatusResponse{Success: false, Message: err.Error()}}, nil
	}

	return &pb.UpdateConfigResponse{Status: &pb.StatusResponse{Success: true, Message: "OK"}}, nil
}

func (s *IOTServer) GetAlerts(ctx context.Context, req *pb.DeviceRequest) (*pb.GetAlertsResponse, error) {
	if err := validateDeviceID(&req.DeviceId); err != nil {
		return &pb.GetAlertsResponse{Status: &pb.StatusResponse{Success: false, Message: fmt.Sprintf("validation error: %v", err)}}, nil
	}

	alerts, err := s.Iot.Alert.GetDeviceAlerts(req.DeviceId)

	if err != nil {
		return &pb.GetAlertsResponse{
			Status: &pb.StatusResponse{
				Success: false,
				Message: err.Error(),
			},
			Alerts: nil,
		}, nil
	}

	return &pb.GetAlertsResponse{
		Status: &pb.StatusResponse{
			Success: true,
			Message: "OK",
		},
		Alerts: common.Mapper(alerts, func(a models.Alert) *pb.Alert {
			return &pb.Alert{
				Id:        uint64(a.ID),
				DeviceId:  a.DeviceID,
				Timestamp: timestamppb.New(a.Timestamp),
				Type:      string(a.Type),
				Message:   a.Message,
			}
		}),
	}, nil
}

func (s *IOTServer) PostLimiter(ctx context.Context, req *pb.PostLimiterRequest) (*pb.PostLimiterResponse, error) {
	if err := validateDeviceID(&req.DeviceId); err != nil {
		return &pb.PostLimiterResponse{Status: &pb.StatusResponse{Success: false, Message: fmt.Sprintf("validation error: %v", err)}}, nil
	}

	var rateValidator = z.Float64().Required()
	if err := rateValidator.Validate(&req.DeviceRate); err != nil {
		return &pb.PostLimiterResponse{Status: &pb.StatusResponse{Success: false, Message: fmt.Sprintf("validation error: %v", err)}}, nil
	}

	var burstValidator = z.Int32().Required()
	if err := burstValidator.Validate(&req.DeviceBurst); err != nil {
		return &pb.PostLimiterResponse{Status: &pb.StatusResponse{Success: false, Message: fmt.Sprintf("validation error: %v", err)}}, nil
	}

	if s.RateLimiterStore == nil {
		return &pb.PostLimiterResponse{
			Status: &pb.StatusResponse{
				Success: false,
				Message: "RateLimiterStore is not used. No effect.",
			},
		}, nil
	}

	s.RateLimiterStore.SetLimiter(req.DeviceId, rate.Limit(req.DeviceRate), int(req.DeviceBurst))
	return &pb.PostLimiterResponse{Status: &pb.StatusResponse{Success: true, Message: "OK"}}, nil
}
