package http

import (
	"net/http"
	"time"

	"liyu1981.xyz/iot-metrics-service/pkg/models"

	"github.com/gin-gonic/gin"

	z "github.com/Oudwins/zog"
	"github.com/Oudwins/zog/zhttp"
)

type MetricRequest struct {
	Timestamp   time.Time `json:"timestamp"`
	Temperature float64   `json:"temperature"`
	Battery     float64   `json:"battery"`
}

var metricRequestSchema = z.Struct(z.Shape{
	"Timestamp":   z.Time().Required(),
	"Temperature": z.Float64().Required(),
	"Battery":     z.Float64().Required(),
})

func (rs *RestfulServer) PostMetrics(c *gin.Context) {
	deviceID := c.Param("device_id")

	if !rs.CheckDeviceLimiter(deviceID) {
		c.Status(http.StatusTooManyRequests)
		return
	}

	var req MetricRequest

	if err := metricRequestSchema.Parse(zhttp.Request(c.Request), &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	if err := rs.Iot.Metric.UpsertMetric(deviceID, &models.Metric{
		Timestamp:   req.Timestamp,
		Temperature: req.Temperature,
		Battery:     req.Battery,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

type ConfigRequest struct {
	TemperatureThreshold float64 `json:"temperature_threshold"`
	BatteryThreshold     float64 `json:"battery_threshold"`
}

var configRequestSchema = z.Struct(z.Shape{
	"TemperatureThreshold": z.Float64().Required(),
	"BatteryThreshold":     z.Float64().Required(),
})

func (rs *RestfulServer) UpdateConfig(c *gin.Context) {
	deviceID := c.Param("device_id")

	if !rs.CheckDeviceLimiter(deviceID) {
		c.Status(http.StatusTooManyRequests)
		return
	}

	var req ConfigRequest
	if err := configRequestSchema.Parse(zhttp.Request(c.Request), &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	config := models.Config{
		DeviceID:             deviceID,
		TemperatureThreshold: req.TemperatureThreshold,
		BatteryThreshold:     req.BatteryThreshold,
	}

	if err := rs.Iot.Config.UpsertConfig(deviceID, &config); err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

func (rs *RestfulServer) GetAlerts(c *gin.Context) {
	deviceID := c.Param("device_id")

	if !rs.CheckDeviceLimiter(deviceID) {
		c.Status(http.StatusTooManyRequests)
		return
	}

	var alerts []models.Alert
	var err error
	if alerts, err = rs.Iot.Alert.GetDeviceAlerts(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, alerts)
}

type LimiterRequest struct {
	Rate  float64 `json:"rate"`
	Burst int     `json:"burst"`
}

var limiterRequestSchema = z.Struct(z.Shape{
	"rate":  z.Float64().Required(),
	"burst": z.Int().Required(),
})

func (rs *RestfulServer) PostLimiter(c *gin.Context) {
	deviceID := c.Param("device_id")

	var req LimiterRequest
	if err := limiterRequestSchema.Parse(zhttp.Request(c.Request), &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	rs.SetLimiter(deviceID, req.Rate, req.Burst)

	c.Status(http.StatusOK)
}

func (rs *RestfulServer) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
