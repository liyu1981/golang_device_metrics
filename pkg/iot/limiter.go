package iot

import (
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiterStore manages per-device rate limiters: device_id -> rate limiter
type RateLimiterStore struct {
	limiters     map[string]*rate.Limiter
	mu           sync.Mutex
	defaultRate  rate.Limit
	defaultBurst int
}

func NewRateLimiterStore(defaultRate rate.Limit, defaultBurst int) *RateLimiterStore {
	return &RateLimiterStore{
		limiters:     make(map[string]*rate.Limiter),
		defaultRate:  defaultRate,
		defaultBurst: defaultBurst,
	}
}

func (s *RateLimiterStore) GetLimiter(deviceID string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	limiter, exists := s.limiters[deviceID]
	if !exists {
		limiter = rate.NewLimiter(s.defaultRate, s.defaultBurst)
		s.limiters[deviceID] = limiter
	}
	return limiter
}

func (s *RateLimiterStore) SetLimiter(deviceID string, deviceRate rate.Limit, deviceBurst int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limiters[deviceID] = rate.NewLimiter(deviceRate, deviceBurst)
}
