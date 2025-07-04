package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"liyu1981.xyz/iot-metrics-service/pkg/common"
	"liyu1981.xyz/iot-metrics-service/pkg/db"
	iotGrpc "liyu1981.xyz/iot-metrics-service/pkg/grpc"
	pb "liyu1981.xyz/iot-metrics-service/pkg/grpc/iot_metric_service"
	iotHttp "liyu1981.xyz/iot-metrics-service/pkg/http"
	"liyu1981.xyz/iot-metrics-service/pkg/iot"
)

func main() {
	var err error

	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file, copy .env.example to .env first if in development")
	}

	var dbInstance *db.DB
	iotDbType := os.Getenv(common.EnvKeyIOTDBType)
	switch iotDbType {
	case "file":
		dbInstance = db.GetInstance(db.UseSqliteDialector())
	case "memory":
		dbInstance = db.GetInstance(db.UseMemorySqliteDialector())
	default:
		log.Fatal("Unknown IOT_DB_TYPE: " + iotDbType)
	}

	grpcHostPort := strings.TrimSpace(os.Getenv(common.EnvKeyIOTGrpcHostPort))
	httpHostPort := strings.TrimSpace(os.Getenv(common.EnvKeyIOTHttpHostPort))

	var defaultRate float64
	var defaultBurst int64

	if defaultRate, err = strconv.ParseFloat(os.Getenv(common.EnvKeyIOTDefaultRate), 64); err != nil {
		log.Fatal("Invalid IOT_DEFAULT_RATE, or not set in .env, should be a float64 value")
	}

	if defaultBurst, err = strconv.ParseInt(os.Getenv(common.EnvKeyIOTDefaultBurst), 10, 64); err != nil {
		log.Fatal("Invalid IOT_DEFAULT_BURST, or not set in .env, should be an int value")
	}

	logger := common.GetLogger()

	iotCore := iot.IOT{
		Db: *dbInstance,
	}
	iotCore.WithServices(iot.ServiceOpts{
		Metric: iotCore.GetIMetric(),
		Alert:  iotCore.GetIAlert(),
		Config: iotCore.GetIConfig(),
	})

	if grpcHostPort != "" {
		logger.Info("Starting gRPC server on port " + grpcHostPort)
		go func() {
			iotGrpcServer := iotGrpc.IOTServer{
				Iot:              &iotCore,
				RateLimiterStore: iot.NewRateLimiterStore(rate.Limit(defaultRate), int(defaultBurst)),
			}
			interceptor := iotGrpcServer.CreateRateLimitInterceptor([]proto.Message{
				&pb.PostMetricsRequest{},
				&pb.UpdateConfigRequest{},
				&pb.DeviceRequest{},
			})
			s := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
			pb.RegisterIOTServiceServer(s, &iotGrpcServer)
			logger.Info("gRPC server created with:",
				zap.String("default_limiter",
					fmt.Sprintf("{\"default_rate\": %v, \"default_burst\": %v}", defaultRate, defaultBurst)))

			listener, err := net.Listen("tcp", grpcHostPort)
			if err != nil {
				log.Fatalf("failed to listen: %v", err)
			}

			logger.Info("start gRPC server on " + grpcHostPort)
			if err := s.Serve(listener); err != nil {
				log.Fatalf("grpc server failed to serve: %v", err)
			}
		}()
	}

	if httpHostPort == "" {
		// fallback to default http port
		httpHostPort = ":1080"
	}

	logger.Info("Starting HTTP server on port " + httpHostPort)
	rs := &iotHttp.RestfulServer{
		Server:           gin.Default(),
		Iot:              &iotCore,
		RateLimiterStore: iot.NewRateLimiterStore(rate.Limit(defaultRate), int(defaultBurst)),
	}
	rs.Setup()

	logger.Info("http server created with:",
		zap.String("default_limiter",
			fmt.Sprintf("{\"default_rate\": %v, \"default_burst\": %v}", defaultRate, defaultBurst)))

	logger.Info("Starting HTTP server on: " + httpHostPort)
	if err := rs.Server.Run(httpHostPort); err != nil {
		log.Fatalf("http server failed to serve: %v", err)
	}
}
