package grpc

import (
	"golang.org/x/time/rate"
	pb "liyu1981.xyz/iot-metrics-service/pkg/grpc/iot_metric_service"

	"liyu1981.xyz/iot-metrics-service/pkg/iot"
)

type IOTServer struct {
	Iot              *iot.IOT
	RateLimiterStore *iot.RateLimiterStore
	pb.UnimplementedIOTServiceServer
}

func (i *IOTServer) GetLimiter(deviceID string) *rate.Limiter {
	if i.RateLimiterStore == nil {
		return nil
	} else {
		return i.RateLimiterStore.GetLimiter(deviceID)
	}
}

func (i *IOTServer) CheckDeviceLimiter(deviceID string) bool {
	limiter := i.GetLimiter(deviceID)
	if limiter == nil {
		return true
	}
	return limiter.Allow()
}
