package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	pb "liyu1981.xyz/iot-metrics-service/pkg/grpc/iot_metric_service"
)

var maxDevices int = 2000
var httpHostPort string = "127.0.0.1:1080"
var grpcHostPort string = "127.0.0.1:10801"

var grpcClient pb.IOTServiceClient

var rnd *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
	deviceIDs := make([]string, maxDevices)
	for i := range maxDevices {
		deviceIDs[i] = uuid.NewString()
	}
	fmt.Printf("generated %v device IDs\n", maxDevices)

	resp, err := http.Get(fmt.Sprintf("http://%s/healthz", httpHostPort))
	if err != nil {
		log.Fatal("Failed to connect to HTTP server:", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatal("HTTP server not available")
	}

	fmt.Printf("http server verified\n")

	conn, err := grpc.Dial(grpcHostPort, grpc.WithInsecure())
	if err != nil {
		log.Fatal("Failed to connect to gRPC server:", err)
	}
	defer conn.Close()
	grpcClient = pb.NewIOTServiceClient(conn)

	fmt.Printf("gRPC server verified and connected\n")

	var startTime time.Time
	var usedTime time.Duration

	startTime = time.Now()
	wg := sync.WaitGroup{}
	for i := range maxDevices {
		wg.Add(1)
		go func() {
			insertConfig(deviceIDs[i])
			fmt.Printf("\rinserted config for device %v", i)
			wg.Done()
		}()
	}
	wg.Wait()
	usedTime = time.Since(startTime)

	fmt.Printf(
		"\rinserted config for %v devices: used time=%v seconds, throughput=%v action/second\n",
		maxDevices, usedTime.Seconds(), float64(maxDevices)/usedTime.Seconds(),
	)

	startTime = time.Now()
	wg = sync.WaitGroup{}
	for i := range maxDevices {
		wg.Add(1)
		go func() {
			doAction(deviceIDs[i])
			wg.Done()
		}()
	}
	wg.Wait()
	usedTime = time.Since(startTime)

	fmt.Printf(
		"\n\rdid actions for %v devices: used time=%v seconds, throughput=%v action/second\n",
		maxDevices, usedTime.Seconds(), float64(maxDevices*3)/usedTime.Seconds(),
	)
}

func flipCoin() bool {
	return rnd.Int31n(100000)%2 == 0
}

func rndFloat64(min, max float64, decimal int) float64 {
	val := min + rnd.Float64()*(max-min)
	multiplier := float64(math.Pow10(decimal))
	return float64(math.Round(float64(val)*float64(multiplier))) / multiplier
}

func insertConfig(deviceID string) {
	useHttp := flipCoin()

	t := rndFloat64(0.0, 100.0, 2)
	b := rndFloat64(0.0, 100.0, 2)
	payload := map[string]string{
		"temperature_threshold": fmt.Sprintf("%.2f", t),
		"battery_threshold":     fmt.Sprintf("%.2f", b),
	}

	if useHttp {
		jsonData, _ := json.Marshal(payload)
		resp, err := http.Post(fmt.Sprintf("http://%s/devices/%s/config", httpHostPort, deviceID), "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
	} else {
		resp, err := grpcClient.UpdateConfig(context.Background(), &pb.UpdateConfigRequest{
			DeviceId: deviceID,
			Config: &pb.ConfigRequest{
				TemperatureThreshold: t,
				BatteryThreshold:     b,
			},
		})
		if err != nil || !resp.Status.Success {
			panic(fmt.Sprintf("err: %v, resp: %v", err, resp))
		}
	}
}

func doAction(deviceID string) {
	actions := []func(){
		genUpsertConfigAction(deviceID),
		genGetAlertsAction(deviceID),
		genPostMetricsAction(deviceID),
	}
	actionNames := []string{
		"UpsertConfig",
		"GetAlerts",
		"PostMetrics",
	}
	rnd.Shuffle(len(actions), func(i, j int) {
		actions[i], actions[j] = actions[j], actions[i]
		actionNames[i], actionNames[j] = actionNames[j], actionNames[i]
	})
	for index, action := range actions {
		action()
		fmt.Printf("\rexecuted action %v for device %v", actionNames[index], deviceID)
		time.Sleep(time.Duration(100+rnd.Int31n(1000)) * time.Millisecond)
	}
}

func genUpsertConfigAction(deviceID string) func() {
	return func() {
		insertConfig(deviceID)
	}
}

func genPostMetricsAction(deviceID string) func() {
	return func() {
		useHttp := flipCoin()

		t := rndFloat64(0.0, 100.0, 2)
		b := rndFloat64(0.0, 100.0, 2)
		now := time.Now()
		payload := map[string]string{
			"timestamp":   now.Format(time.RFC3339),
			"temperature": fmt.Sprintf("%.2f", t),
			"battery":     fmt.Sprintf("%.2f", b),
		}

		if useHttp {
			jsonData, _ := json.Marshal(payload)
			resp, err := http.Post(fmt.Sprintf("http://%s/devices/%s/metrics", httpHostPort, deviceID), "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
			}
			defer resp.Body.Close()
		} else {
			resp, err := grpcClient.PostMetrics(context.Background(), &pb.PostMetricsRequest{
				DeviceId: deviceID,
				Metric: &pb.MetricRequest{
					Timestamp:   timestamppb.New(now),
					Temperature: t,
					Battery:     b,
				},
			})
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
			}
			if !resp.Status.Success {
				fmt.Printf("\nresponse success = false: %v\n", resp)
			}
		}
	}
}

func genGetAlertsAction(deviceID string) func() {
	return func() {
		useHttp := flipCoin()

		if useHttp {
			resp, err := http.Get(fmt.Sprintf("http://%s/devices/%s/alerts", httpHostPort, deviceID))
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("\nresponse status code != 200: %v\n", resp)
			}
		} else {
			resp, err := grpcClient.GetAlerts(context.Background(), &pb.DeviceRequest{DeviceId: deviceID})
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
			}
			if !resp.Status.Success {
				fmt.Printf("\nresponse success = false: %v\n", resp)
			}
		}
	}
}
