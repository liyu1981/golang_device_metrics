package iot

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRateLimiterStore_Basic(t *testing.T) {
	store := NewRateLimiterStore(1, 2)

	limiter := store.GetLimiter("device1")
	if limiter == nil {
		t.Fatal("expected limiter, got nil")
	}
	if limiter.Limit() != 1 {
		t.Errorf("expected limit 1, got %v", limiter.Limit())
	}
}

func TestRateLimiterStore_CustomLimit(t *testing.T) {
	store := NewRateLimiterStore(1, 2)

	store.SetLimiter("device2", 5, 10)
	limiter := store.GetLimiter("device2")

	if limiter.Limit() != 5 {
		t.Errorf("expected limit 5, got %v", limiter.Limit())
	}
	if limiter.Burst() != 10 {
		t.Errorf("expected burst 10, got %v", limiter.Burst())
	}
}

func TestRateLimiterStore_Concurrency(t *testing.T) {
	store := NewRateLimiterStore(10, 5)
	deviceID := uuid.NewString()

	var wg sync.WaitGroup

	// Launch 100 goroutines that access GetLimiter concurrently
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			limiter := store.GetLimiter(deviceID)
			if limiter == nil {
				t.Error("expected limiter, got nil")
			}
		}()
	}

	wg.Wait()

	// Not directly accessing internal map, but functional behavior is implied
	limiter := store.GetLimiter(deviceID)
	if limiter == nil {
		t.Error("expected limiter to exist after concurrent access")
	}
}

func TestRateLimiter_Enforcement(t *testing.T) {
	store := NewRateLimiterStore(2, 2) // 2 events/sec

	deviceID := uuid.NewString()
	limiter := store.GetLimiter(deviceID)

	// Consume two tokens
	firstTry := limiter.Allow()
	secondTry := limiter.Allow()
	if !firstTry || !secondTry {
		t.Fatal("expected first two calls to be allowed")
	}

	// This call should fail immediately
	if limiter.Allow() {
		t.Error("expected third call to be rate limited")
	}

	// Wait for refill
	time.Sleep(600 * time.Millisecond)
	if !limiter.Allow() {
		t.Error("expected one token to be available after refill")
	}
}
