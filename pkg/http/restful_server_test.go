package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"liyu1981.xyz/iot-metrics-service/pkg/iot/mocks"
	_ "liyu1981.xyz/iot-metrics-service/pkg/testing"

	"liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/db"
	"liyu1981.xyz/iot-metrics-service/pkg/iot"
	"liyu1981.xyz/iot-metrics-service/pkg/models"
)

func setupTestServer() *RestfulServer {
	iotObj := iot.IOT{
		Db: *db.GetInstance(db.UseMemorySqliteDialector()),
	}
	iotObj.WithServices(iot.ServiceOpts{
		Metric: iotObj.GetIMetric(),
		Alert:  iotObj.GetIAlert(),
		Config: iotObj.GetIConfig(),
	})

	rs := &RestfulServer{
		Server: gin.Default(),
		Iot:    &iotObj,
		// default we use no limiter, if need, later assign it rs.RateLimitStore = iot.NewRateLimiterStore(...)
	}

	rs.Setup()

	return rs
}

type RequestBodyData struct {
	Timestamp   string  `json:"timestamp"`
	Temperature float64 `json:"temperature"`
	Battery     float64 `json:"battery"`
}

func TestHealthCheck(t *testing.T) {
	rs := setupTestServer()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	rs.Server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

func TestPostMetricsAndGetAlerts(t *testing.T) {
	common.SetTestLoggerNop()

	rs := setupTestServer()

	deviceID := uuid.NewString()

	// Insert config first (required to trigger alerts)
	config := &models.Config{
		DeviceID:             deviceID,
		TemperatureThreshold: 30.0,
		BatteryThreshold:     50.0,
	}
	err := rs.Iot.Db.Conn.Create(config).Error
	assert.NoError(t, err)

	// Send a metric that triggers both alerts
	metricReq := MetricRequest{
		Timestamp:   time.Now(),
		Temperature: 45.5,
		Battery:     20.0,
	}
	body, _ := json.Marshal(metricReq)

	req := httptest.NewRequest("POST", "/devices/"+deviceID+"/metrics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rs.Server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	alertReq := httptest.NewRequest("GET", "/devices/"+deviceID+"/alerts", nil)
	alertW := httptest.NewRecorder()
	rs.Server.ServeHTTP(alertW, alertReq)

	assert.Equal(t, http.StatusOK, alertW.Code)

	var alerts []models.Alert
	err = json.Unmarshal(alertW.Body.Bytes(), &alerts)
	assert.NoError(t, err)
	assert.Len(t, alerts, 2)

	alertTypes := map[string]bool{}
	for _, alert := range alerts {
		alertTypes[string(alert.Type)] = true
	}
	assert.True(t, alertTypes[string(models.AlertTypeTemperature)])
	assert.True(t, alertTypes[string(models.AlertTypeBattery)])
}

func TestPostMetricsAndGetAlerts_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	{
		rs := setupTestServer()
		deviceID := uuid.NewString()
		// empty payload should be rejected
		payload := []byte("{}")
		req := httptest.NewRequest("POST", "/devices/"+deviceID+"/metrics", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		rs.Server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	}

	{
		rs := setupTestServer()
		deviceID := uuid.NewString()
		// Send a metric without valid deviceID should cause internal error
		metricReq := MetricRequest{
			Timestamp:   time.Now(),
			Temperature: 45.5,
			Battery:     20.0,
		}
		body, _ := json.Marshal(metricReq)

		req := httptest.NewRequest("POST", "/devices/"+deviceID+"/metrics", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		rs.Server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	}

	{
		rs := setupTestServer()
		deviceID := uuid.NewString()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockIAlert := mocks.NewMockIAlert(ctrl)
		rs.Iot.Alert = mockIAlert
		mockIAlert.EXPECT().
			GetDeviceAlerts(gomock.Eq(deviceID)).
			Return(nil, fmt.Errorf("just causing error")).
			Times(1)

		req := httptest.NewRequest("GET", "/devices/"+deviceID+"/alerts", nil)
		w := httptest.NewRecorder()
		rs.Server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	}
}

func TestUpdateConfig(t *testing.T) {
	common.SetTestLoggerNop()

	rs := setupTestServer()

	deviceID := uuid.NewString()

	configReq := ConfigRequest{
		TemperatureThreshold: 100.0,
		BatteryThreshold:     20.0,
	}
	body, _ := json.Marshal(configReq)
	req := httptest.NewRequest("POST", "/devices/"+deviceID+"/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rs.Server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify in DB
	var config models.Config
	err := rs.Iot.Db.Conn.
		Where("device_id = ?", deviceID).
		First(&config).Error
	assert.NoError(t, err)
	assert.Equal(t, 100.0, config.TemperatureThreshold)
}

func TestUpdateConfig_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	{
		rs := setupTestServer()
		deviceID := uuid.NewString()
		// empty payload should be rejected
		payload := []byte("{}")
		req := httptest.NewRequest("POST", "/devices/"+deviceID+"/config", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		rs.Server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	}

	{
		rs := setupTestServer()
		deviceID := uuid.NewString()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockIConfig := mocks.NewMockIConfig(ctrl)
		rs.Iot.Config = mockIConfig
		mockIConfig.EXPECT().
			UpsertConfig(gomock.Eq(deviceID), gomock.Any()).
			Return(fmt.Errorf("just causing error")).
			Times(1)

		// Send a config without valid deviceID should cause internal error
		configReq := ConfigRequest{
			TemperatureThreshold: 100.0,
			BatteryThreshold:     20.0,
		}
		body, _ := json.Marshal(configReq)
		req := httptest.NewRequest("POST", "/devices/"+deviceID+"/config", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		rs.Server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	}
}

func setupTestServerWithLimiter(limiter *iot.RateLimiterStore) *RestfulServer {
	iotObj := iot.IOT{
		Db: *db.GetInstance(db.UseMemorySqliteDialector()),
	}
	iotObj.WithServices(iot.ServiceOpts{
		Metric: iotObj.GetIMetric(),
		Alert:  iotObj.GetIAlert(),
		Config: iotObj.GetIConfig(),
	})

	rs := &RestfulServer{
		Server:           gin.Default(),
		Iot:              &iotObj,
		RateLimiterStore: limiter,
	}

	rs.Setup()

	return rs
}

func TestPostMetricsWithLimiter(t *testing.T) {
	common.SetTestLoggerNop()

	rs := setupTestServerWithLimiter(iot.NewRateLimiterStore(2, 2)) // 3 req/sec, burst 2

	deviceID := uuid.NewString()

	// Insert config first (required to trigger alerts)
	config := &models.Config{
		DeviceID:             deviceID,
		TemperatureThreshold: 30.0,
		BatteryThreshold:     50.0,
	}
	err := rs.Iot.Db.Conn.Create(config).Error
	assert.NoError(t, err)

	// Send a metric that triggers both alerts
	metricReq := MetricRequest{
		Timestamp:   time.Now(),
		Temperature: 45.5,
		Battery:     20.0,
	}
	metricReqBody, _ := json.Marshal(metricReq)

	// Simulate 3 requests in quick succession — only 2 should be allowed
	for i := range 3 {
		req := httptest.NewRequest(http.MethodPost, "/devices/"+deviceID+"/metrics", bytes.NewReader(metricReqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		rs.Server.ServeHTTP(w, req)

		if i < 2 {
			require.Equal(t, http.StatusOK, w.Code, "request %d should be allowed", i+1)
		} else {
			require.Equal(t, http.StatusTooManyRequests, w.Code, "request %d should be rate limited", i+1)
		}
	}

	limiterReq := LimiterRequest{
		Rate:  2,
		Burst: 2,
	}
	limiterReqBody, _ := json.Marshal(limiterReq)
	req := httptest.NewRequest(http.MethodPost, "/devices/"+deviceID+"/limiter", bytes.NewReader(limiterReqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rs.Server.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "limiter request should be allowed")

	req = httptest.NewRequest(http.MethodPost, "/devices/"+deviceID+"/metrics", bytes.NewReader(metricReqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	rs.Server.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "request after sleep should be allowed")
}

func TestPostLimiter_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	rs := setupTestServerWithLimiter(iot.NewRateLimiterStore(2, 2)) // 3 req/sec, burst 2

	deviceID := uuid.NewString()

	// empty payload should be rejected
	payload := []byte("{}")
	req := httptest.NewRequest("POST", "/devices/"+deviceID+"/limiter", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rs.Server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLimiter(t *testing.T) {
	common.SetTestLoggerNop()

	rs := setupTestServerWithLimiter(iot.NewRateLimiterStore(0, 0)) // 1 req/sec, burst 1

	deviceID := uuid.NewString()

	// nothing should pass below
	{
		configReq := ConfigRequest{
			TemperatureThreshold: 100.0,
			BatteryThreshold:     20.0,
		}
		body, _ := json.Marshal(configReq)
		req := httptest.NewRequest("POST", "/devices/"+deviceID+"/config", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		rs.Server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	}

	{
		req := httptest.NewRequest("GET", "/devices/"+deviceID+"/alerts", nil)
		w := httptest.NewRecorder()
		rs.Server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	}

	{
		metricReq := MetricRequest{
			Timestamp:   time.Now(),
			Temperature: 45.5,
			Battery:     20.0,
		}
		body, _ := json.Marshal(metricReq)
		req := httptest.NewRequest("POST", "/devices/"+deviceID+"/metrics", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		rs.Server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	}
}

func TestSetLimiter_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	rs := setupTestServer() // default without limiter store

	deviceID := uuid.NewString()

	{
		// without limiter store setup limiter should be allowed and just return ok (but no effect)
		limiterReq := LimiterRequest{
			Rate:  2,
			Burst: 2,
		}
		limiterReqBody, _ := json.Marshal(limiterReq)
		req := httptest.NewRequest(http.MethodPost, "/devices/"+deviceID+"/limiter", bytes.NewReader(limiterReqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		rs.Server.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "limiter request should be allowed")
	}

	{
		// and request to alert should return empty alerts instead of too many requests
		req := httptest.NewRequest("GET", "/devices/"+deviceID+"/alerts", nil)
		w := httptest.NewRecorder()
		rs.Server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}
