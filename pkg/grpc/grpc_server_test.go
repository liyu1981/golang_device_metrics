package grpc

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/db"
	pb "liyu1981.xyz/iot-metrics-service/pkg/grpc/iot_metric_service"
	"liyu1981.xyz/iot-metrics-service/pkg/iot"
	_ "liyu1981.xyz/iot-metrics-service/pkg/testing"

	"liyu1981.xyz/iot-metrics-service/pkg/iot/mocks"
)

const bufSize = 1024 * 1024

var listener *bufconn.Listener

func dialer() func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}
}

func startTestServer(t *testing.T) pb.IOTServiceClient {
	listener = bufconn.Listen(bufSize)

	iotCore := iot.IOT{
		Db: *db.GetInstance(db.UseMemorySqliteDialector()),
	}
	iotCore.WithServices(iot.ServiceOpts{
		Metric: iotCore.GetIMetric(),
		Alert:  iotCore.GetIAlert(),
		Config: iotCore.GetIConfig(),
	})

	iotServer := IOTServer{Iot: &iotCore}
	interceptor := grpc.UnaryInterceptor(iotServer.CreateRateLimitInterceptor([]proto.Message{
		&pb.PostMetricsRequest{},
		&pb.UpdateConfigRequest{},
		&pb.DeviceRequest{},
	}))
	server := grpc.NewServer(interceptor)
	pb.RegisterIOTServiceServer(server, &iotServer)

	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(dialer()),
		grpc.WithInsecure(),
	)
	require.NoError(t, err)

	return pb.NewIOTServiceClient(conn)
}

func TestPostMetricsAndGetAlerts(t *testing.T) {
	common.SetTestLoggerNop()
	client := startTestServer(t)

	deviceID := uuid.NewString()

	_, err := client.UpdateConfig(context.Background(), &pb.UpdateConfigRequest{
		DeviceId: deviceID,
		Config: &pb.ConfigRequest{
			TemperatureThreshold: 30.0,
			BatteryThreshold:     20.0,
		},
	})
	require.NoError(t, err)

	_, err = client.PostMetrics(context.Background(), &pb.PostMetricsRequest{
		DeviceId: deviceID,
		Metric: &pb.MetricRequest{
			Timestamp:   timestamppb.New(time.Now()),
			Temperature: 35.0,
			Battery:     10.0,
		},
	})
	require.NoError(t, err)

	resp, err := client.GetAlerts(context.Background(), &pb.DeviceRequest{DeviceId: deviceID})
	require.NoError(t, err)
	require.True(t, resp.Status.Success)
	require.Len(t, resp.Alerts, 2)
}

func TestPostMetricsEdgeCases(t *testing.T) {
	common.SetTestLoggerNop()
	client := startTestServer(t)

	deviceID := uuid.NewString()

	_, err := client.PostMetrics(context.Background(), &pb.PostMetricsRequest{
		DeviceId: deviceID,
		Metric: &pb.MetricRequest{
			Timestamp:   timestamppb.New(time.Now()),
			Temperature: 35.0,
			Battery:     10.0,
		},
	})
	fmt.Println(err)
}

func startTestServerWithInterceptor(t *testing.T, limiterStore *iot.RateLimiterStore) pb.IOTServiceClient {
	listener := bufconn.Listen(bufSize)

	iotCore := iot.IOT{
		Db: *db.GetInstance(db.UseMemorySqliteDialector()),
	}
	iotCore.WithServices(iot.ServiceOpts{
		Metric: iotCore.GetIMetric(),
		Alert:  iotCore.GetIAlert(),
		Config: iotCore.GetIConfig(),
	})

	iotServer := IOTServer{Iot: &iotCore, RateLimiterStore: limiterStore}
	interceptor := iotServer.CreateRateLimitInterceptor([]proto.Message{
		&pb.PostMetricsRequest{},
		&pb.UpdateConfigRequest{},
		&pb.DeviceRequest{},
	})
	server := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	pb.RegisterIOTServiceServer(server, &iotServer)

	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithInsecure(),
	)
	require.NoError(t, err)

	return pb.NewIOTServiceClient(conn)
}

func TestRateLimitInterceptor_PostMetrics(t *testing.T) {
	common.SetTestLoggerNop()

	limiterStore := iot.NewRateLimiterStore(2, 2) // Allow 2 req/sec per device
	client := startTestServerWithInterceptor(t, limiterStore)

	ctx := context.Background()
	deviceID := uuid.NewString()

	var err error

	_, err = client.UpdateConfig(context.Background(), &pb.UpdateConfigRequest{
		DeviceId: deviceID,
		Config: &pb.ConfigRequest{
			TemperatureThreshold: 30.0,
			BatteryThreshold:     20.0,
		},
	})
	require.NoError(t, err)

	// Wait for 1 second to let token bucket refill
	time.Sleep(1 * time.Second)

	req := &pb.PostMetricsRequest{
		DeviceId: deviceID,
		Metric: &pb.MetricRequest{
			Timestamp:   timestamppb.New(time.Now()),
			Temperature: 35.0,
			Battery:     10.0,
		},
	}

	// First 2 requests should pass
	for i := range 2 {
		_, err := client.PostMetrics(ctx, req)
		require.NoError(t, err, "expected request %d to pass", i+1)
	}

	// 3rd request should fail immediately
	_, err = client.PostMetrics(ctx, req)
	require.Error(t, err, "expected third request to be rate limited")

	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error")
	require.Equal(t, codes.ResourceExhausted, st.Code(), "expected ResourceExhausted code")

	// increase rate limiter
	client.PostLimiter(ctx, &pb.PostLimiterRequest{
		DeviceId:    deviceID,
		DeviceRate:  3,
		DeviceBurst: 2,
	})

	// Should pass again
	_, err = client.PostMetrics(ctx, req)
	require.NoError(t, err, "expected request after sleep to pass")
}

func startTestServerWithMocks(t *testing.T, useMockIMetric, useMockIAlert, useMockIConfig bool) (
	*gomock.Controller,
	pb.IOTServiceClient,
	*mocks.MockIMetric,
	*mocks.MockIAlert,
	*mocks.MockIConfig,
) {
	ctrl := gomock.NewController(t)

	listener = bufconn.Listen(bufSize)

	iMockMetric := mocks.NewMockIMetric(ctrl)
	iMockAlert := mocks.NewMockIAlert(ctrl)
	iMockConfig := mocks.NewMockIConfig(ctrl)

	iotCore := iot.IOT{
		Db: *db.GetInstance(db.UseMemorySqliteDialector()),
	}
	iMetric := iotCore.GetIMetric()
	if useMockIMetric {
		iMetric = iMockMetric
	}
	iAlert := iotCore.GetIAlert()
	if useMockIAlert {
		iAlert = iMockAlert
	}
	iConfig := iotCore.GetIConfig()
	if useMockIConfig {
		iConfig = iMockConfig
	}
	iotCore.WithServices(iot.ServiceOpts{
		Metric: iMetric,
		Alert:  iAlert,
		Config: iConfig,
	})

	iotServer := IOTServer{Iot: &iotCore}
	interceptor := grpc.UnaryInterceptor(iotServer.CreateRateLimitInterceptor([]proto.Message{
		&pb.PostMetricsRequest{},
		&pb.UpdateConfigRequest{},
		&pb.DeviceRequest{},
	}))
	server := grpc.NewServer(interceptor)
	pb.RegisterIOTServiceServer(server, &iotServer)

	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(dialer()),
		grpc.WithInsecure(),
	)
	require.NoError(t, err)

	return ctrl, pb.NewIOTServiceClient(conn), iMockMetric, iMockAlert, iMockConfig
}

func TestPostConfig_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	{
		client := startTestServer(t)
		deviceID := uuid.NewString()

		{
			// empty DeviceId will fail validation
			r, err := client.UpdateConfig(context.Background(), &pb.UpdateConfigRequest{
				DeviceId: "",
				Config:   &pb.ConfigRequest{},
			})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected UpdateConfig to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected UpdateConfig to fail with validation error")
		}

		{
			// empty Config will fail validation
			r, err := client.UpdateConfig(context.Background(), &pb.UpdateConfigRequest{
				DeviceId: deviceID,
				Config:   &pb.ConfigRequest{},
			})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected UpdateConfig to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected UpdateConfig to fail with validation error")
		}
	}

	{
		ctrl, client, _, _, mockIConfig := startTestServerWithMocks(t, false, false, true)
		defer ctrl.Finish()

		deviceID := uuid.NewString()

		{
			// internal error should fail too
			mockIConfig.EXPECT().
				UpsertConfig(gomock.Eq(deviceID), gomock.Any()).
				Return(fmt.Errorf("test error")).
				Times(1)
			r, err := client.UpdateConfig(context.Background(), &pb.UpdateConfigRequest{
				DeviceId: deviceID,
				Config: &pb.ConfigRequest{
					TemperatureThreshold: 30.0,
					BatteryThreshold:     30.0,
				},
			})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected UpdateConfig to fail")
			assert.True(t, strings.Contains(r.Status.Message, "test error"), "expected UpdateConfig to fail with test error")
		}
	}
}

func TestGetAlerts_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	{
		client := startTestServer(t)
		// deviceID := uuid.NewString()

		{
			// empty DeviceId will fail validation
			r, err := client.GetAlerts(context.Background(), &pb.DeviceRequest{DeviceId: ""})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected GetAlerts to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected GetAlerts to fail with validation error")
		}
	}

	{
		ctrl, client, _, mockIAlert, _ := startTestServerWithMocks(t, false, true, false)
		defer ctrl.Finish()

		deviceID := uuid.NewString()

		{
			// internal error should fail too
			mockIAlert.EXPECT().
				GetDeviceAlerts(gomock.Eq(deviceID)).
				Return(nil, fmt.Errorf("test error")).
				Times(1)
			r, err := client.GetAlerts(context.Background(), &pb.DeviceRequest{DeviceId: deviceID})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected GetAlerts to fail")
			assert.True(t, strings.Contains(r.Status.Message, "test error"), "expected GetAlerts to fail with test error")
		}
	}
}

func TestPostMetrics_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	{
		client := startTestServer(t)
		deviceID := uuid.NewString()

		{
			// empty DeviceId will fail validation
			r, err := client.PostMetrics(context.Background(), &pb.PostMetricsRequest{DeviceId: ""})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected PostMetrics to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected PostMetrics to fail with validation error")
		}

		{
			// empty temperature or battery will fail validation
			r, err := client.PostMetrics(context.Background(), &pb.PostMetricsRequest{
				DeviceId: deviceID,
				Metric:   &pb.MetricRequest{},
			})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected PostMetrics to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected PostMetrics to fail with validation error")
		}

		{
			// empty timestamp will fail validate too
			r, err := client.PostMetrics(context.Background(), &pb.PostMetricsRequest{
				DeviceId: deviceID,
				Metric: &pb.MetricRequest{
					Temperature: 30.0,
					Battery:     30.0,
				},
			})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected PostMetrics to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected PostMetrics to fail with validation error")
		}

		{
			// zero timestamp will fail validate too
			r, err := client.PostMetrics(context.Background(), &pb.PostMetricsRequest{
				DeviceId: deviceID,
				Metric: &pb.MetricRequest{
					Timestamp:   timestamppb.New(time.Time{}),
					Temperature: 30.0,
					Battery:     30.0,
				},
			})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected PostMetrics to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected PostMetrics to fail with validation error")
		}
	}
}

func TestPostLimiter_EdgeCases(t *testing.T) {
	common.SetTestLoggerNop()

	{
		client := startTestServer(t)
		deviceID := uuid.NewString()

		{
			// empty DeviceId will fail validation
			r, err := client.PostLimiter(context.Background(), &pb.PostLimiterRequest{DeviceId: ""})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected PostLimiter to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected PostLimiter to fail with validation error")
		}

		{
			// empty rate or burst will fail validation
			r, err := client.PostLimiter(context.Background(), &pb.PostLimiterRequest{
				DeviceId: deviceID,
			})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected PostLimiter to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected PostLimiter to fail with validation error")
		}

		{
			// empty rate or burst will fail validation
			r, err := client.PostLimiter(context.Background(), &pb.PostLimiterRequest{
				DeviceId:   deviceID,
				DeviceRate: 3.0,
			})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected PostLimiter to fail")
			assert.True(t, strings.Contains(r.Status.Message, "validation error"), "expected PostLimiter to fail with validation error")
		}
	}

	{
		client := startTestServer(t)
		deviceID := uuid.NewString()

		{
			// default there is no rate limitor so setting a rate will fail with no effect
			r, err := client.PostLimiter(context.Background(), &pb.PostLimiterRequest{
				DeviceId:    deviceID,
				DeviceRate:  3.0,
				DeviceBurst: 2,
			})
			assert.NoError(t, err)
			assert.False(t, r.Status.Success, "expected PostLimiter to fail")
			assert.True(t, strings.Contains(r.Status.Message, "No effect"), "expected PostLimiter to succeed with no effect")
		}
	}
}
