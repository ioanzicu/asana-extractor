package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestLimiter_Acquire(t *testing.T) {
	cfg := Config{
		RequestsPerMinute:  60, // 1 per second
		MaxConcurrentRead:  2,
		MaxConcurrentWrite: 1,
	}

	limiter := NewLimiter(cfg)
	ctx := context.Background()

	// Test basic acquisition
	err := limiter.Acquire(ctx, RequestTypeRead)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	limiter.Release(RequestTypeRead)
}

func TestLimiter_ConcurrentReadLimit(t *testing.T) {
	cfg := Config{
		RequestsPerMinute:  600, // High enough to not be the bottleneck
		MaxConcurrentRead:  2,
		MaxConcurrentWrite: 1,
	}

	limiter := NewLimiter(cfg)
	ctx := context.Background()

	// Acquire 2 read slots (should succeed)
	err := limiter.Acquire(ctx, RequestTypeRead)
	if err != nil {
		t.Fatalf("First acquire failed: %v", err)
	}

	err = limiter.Acquire(ctx, RequestTypeRead)
	if err != nil {
		t.Fatalf("Second acquire failed: %v", err)
	}

	// Verify we have 2 concurrent reads
	reads, _ := limiter.Stats()
	if reads != 2 {
		t.Errorf("Expected 2 concurrent reads, got %d", reads)
	}

	// Try to acquire a third (should block, so we use a timeout)
	ctxTimeout, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	err = limiter.Acquire(ctxTimeout, RequestTypeRead)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Release one slot
	limiter.Release(RequestTypeRead)

	// Now we should be able to acquire
	err = limiter.Acquire(ctx, RequestTypeRead)
	if err != nil {
		t.Fatalf("Acquire after release failed: %v", err)
	}

	// Clean up
	limiter.Release(RequestTypeRead)
	limiter.Release(RequestTypeRead)
}

func TestLimiter_ConcurrentWriteLimit(t *testing.T) {
	cfg := Config{
		RequestsPerMinute:  600,
		MaxConcurrentRead:  10,
		MaxConcurrentWrite: 1,
	}

	limiter := NewLimiter(cfg)
	ctx := context.Background()

	// Acquire 1 write slot
	err := limiter.Acquire(ctx, RequestTypeWrite)
	if err != nil {
		t.Fatalf("First write acquire failed: %v", err)
	}

	// Verify we have 1 concurrent write
	_, writes := limiter.Stats()
	if writes != 1 {
		t.Errorf("Expected 1 concurrent write, got %d", writes)
	}

	// Try to acquire a second (should block)
	ctxTimeout, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	err = limiter.Acquire(ctxTimeout, RequestTypeWrite)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Clean up
	limiter.Release(RequestTypeWrite)
}

func TestLimiter_ThreadSafety(t *testing.T) {
	cfg := Config{
		RequestsPerMinute:  600,
		MaxConcurrentRead:  10,
		MaxConcurrentWrite: 5,
	}

	limiter := NewLimiter(cfg)
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 20

	// Launch multiple goroutines that acquire and release
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			reqType := RequestTypeRead
			if id%2 == 0 {
				reqType = RequestTypeWrite
			}

			err := limiter.Acquire(ctx, reqType)
			if err != nil {
				t.Errorf("Goroutine %d: acquire failed: %v", id, err)
				return
			}

			// Simulate some work
			time.Sleep(10 * time.Millisecond)

			limiter.Release(reqType)
		}(i)
	}

	wg.Wait()

	// Verify all slots are released
	reads, writes := limiter.Stats()
	if reads != 0 || writes != 0 {
		t.Errorf("Expected 0 concurrent requests, got reads=%d, writes=%d", reads, writes)
	}
}

func TestLimiter_RateLimit(t *testing.T) {
	cfg := Config{
		RequestsPerMinute:  60, // 1 per second
		MaxConcurrentRead:  100,
		MaxConcurrentWrite: 100,
	}

	limiter := NewLimiter(cfg)
	ctx := context.Background()

	start := time.Now()

	// Make 3 requests (should take ~2 seconds due to rate limiting)
	for i := 0; i < 3; i++ {
		err := limiter.Acquire(ctx, RequestTypeRead)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		limiter.Release(RequestTypeRead)
	}

	elapsed := time.Since(start)

	// Should take at least 2 seconds (3 requests at 1/sec = 2 seconds of waiting)
	if elapsed < 2*time.Second {
		t.Errorf("Rate limiting not working correctly, took %v (expected >= 2s)", elapsed)
	}
}
