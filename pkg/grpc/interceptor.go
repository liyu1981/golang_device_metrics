package grpc

import (
	"context"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"liyu1981.xyz/iot-metrics-service/pkg/common"
)

func (i *IOTServer) CreateRateLimitInterceptor(targetReqTypes []proto.Message) grpc.UnaryServerInterceptor {
	targetTypeMap := common.Reducer(targetReqTypes,
		func(m map[reflect.Type]bool, t proto.Message) map[reflect.Type]bool {
			m[reflect.TypeOf(t)] = true
			return m
		},
		map[reflect.Type]bool{},
	)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if _, ok := targetTypeMap[reflect.TypeOf(req)]; ok {
			if r, ok := req.(interface{ GetDeviceId() string }); ok {
				deviceID := r.GetDeviceId()
				if !i.CheckDeviceLimiter(deviceID) {
					return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
				}
			}
		}

		return handler(ctx, req)
	}
}
