package http

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"liyu1981.xyz/iot-metrics-service/pkg/iot"
)

type RestfulServer struct {
	Server           *gin.Engine
	Iot              *iot.IOT
	RateLimiterStore *iot.RateLimiterStore
}

func (rs *RestfulServer) GetLimiter(deviceID string) *rate.Limiter {
	if rs.RateLimiterStore == nil {
		return nil
	} else {
		return rs.RateLimiterStore.GetLimiter(deviceID)
	}
}

func (rs *RestfulServer) CheckDeviceLimiter(deviceID string) bool {
	limiter := rs.GetLimiter(deviceID)
	if limiter == nil {
		return true
	}
	return limiter.Allow()
}

func (rs *RestfulServer) SetLimiter(deviceID string, deviceRate float64, deviceBurst int) {
	if rs.RateLimiterStore == nil {
		return
	}
	rs.RateLimiterStore.SetLimiter(deviceID, rate.Limit(deviceRate), deviceBurst)
}

func (rs *RestfulServer) Setup() {
	rs.Server.GET("/healthz", rs.HealthCheck)

	devices := rs.Server.Group("/devices/:device_id")
	{
		devices.POST("/metrics", rs.PostMetrics)
		devices.POST("/config", rs.UpdateConfig)
		devices.GET("/alerts", rs.GetAlerts)
		devices.POST("/limiter", rs.PostLimiter)
	}
}
