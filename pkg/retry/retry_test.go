package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		err      error
		expected bool
	}{
		{
			name:     "network error",
			resp:     nil,
			err:      errors.New("network error"),
			expected: true,
		},
		{
			name: "429 rate limit",
			resp: &http.Response{
				StatusCode: http.StatusTooManyRequests,
			},
			err:      nil,
			expected: true,
		},
		{
			name: "500 server error",
			resp: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
			err:      nil,
			expected: true,
		},
		{
			name: "503 service unavailable",
			resp: &http.Response{
				StatusCode: http.StatusServiceUnavailable,
			},
			err:      nil,
			expected: true,
		},
		{
			name: "200 success",
			resp: &http.Response{
				StatusCode: http.StatusOK,
			},
			err:      nil,
			expected: false,
		},
		{
			name: "404 not found",
			resp: &http.Response{
				StatusCode: http.StatusNotFound,
			},
			err:      nil,
			expected: false,
		},
		{
			name: "400 bad request",
			resp: &http.Response{
				StatusCode: http.StatusBadRequest,
			},
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRetry(tt.resp, tt.err)
			if result != tt.expected {
				t.Errorf("ShouldRetry() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected time.Duration
	}{
		{
			name:     "30 seconds",
			header:   "30",
			expected: 30 * time.Second,
		},
		{
			name:     "60 seconds",
			header:   "60",
			expected: 60 * time.Second,
		},
		{
			name:     "empty header",
			header:   "",
			expected: 0,
		},
		{
			name:     "invalid header",
			header:   "invalid",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{},
			}
			if tt.header != "" {
				resp.Header.Set("Retry-After", tt.header)
			}

			result := GetRetryAfter(resp)
			if result != tt.expected {
				t.Errorf("GetRetryAfter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	cfg := Config{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     60 * time.Second,
	}

	tests := []struct {
		name        string
		attempt     int
		retryAfter  time.Duration
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "first attempt",
			attempt:     0,
			retryAfter:  0,
			minExpected: 750 * time.Millisecond,  // 1s - 25% jitter
			maxExpected: 1250 * time.Millisecond, // 1s + 25% jitter
		},
		{
			name:        "second attempt",
			attempt:     1,
			retryAfter:  0,
			minExpected: 1500 * time.Millisecond, // 2s - 25% jitter
			maxExpected: 2500 * time.Millisecond, // 2s + 25% jitter
		},
		{
			name:        "with retry-after",
			attempt:     0,
			retryAfter:  30 * time.Second,
			minExpected: 30 * time.Second,
			maxExpected: 30 * time.Second,
		},
		{
			name:        "capped at max",
			attempt:     10,
			retryAfter:  0,
			minExpected: 60 * time.Second,
			maxExpected: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBackoff(tt.attempt, cfg, tt.retryAfter)

			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("CalculateBackoff() = %v, want between %v and %v",
					result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestDo_Success(t *testing.T) {
	cfg := DefaultConfig()
	ctx := context.Background()

	callCount := 0
	fn := func() (*http.Response, error) {
		callCount++
		return &http.Response{
			StatusCode: http.StatusOK,
		}, nil
	}

	resp, err := Do(ctx, cfg, fn)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if callCount != 1 {
		t.Errorf("Function called %d times, want 1", callCount)
	}
}

func TestDo_RetryOn429(t *testing.T) {
	cfg := Config{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}
	ctx := context.Background()

	callCount := 0
	fn := func() (*http.Response, error) {
		callCount++
		if callCount < 3 {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header: http.Header{
					"Retry-After": []string{"0"},
				},
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
		}, nil
	}

	resp, err := Do(ctx, cfg, fn)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if callCount != 3 {
		t.Errorf("Function called %d times, want 3", callCount)
	}
}

func TestDo_MaxRetriesExceeded(t *testing.T) {
	cfg := Config{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}
	ctx := context.Background()

	callCount := 0
	fn := func() (*http.Response, error) {
		callCount++
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Retry-After": []string{"0"},
			},
		}, nil
	}

	_, err := Do(ctx, cfg, fn)
	if err == nil {
		t.Fatal("Do() error = nil, want error")
	}

	// Should be called MaxRetries + 1 times (initial + retries)
	expectedCalls := cfg.MaxRetries + 1
	if callCount != expectedCalls {
		t.Errorf("Function called %d times, want %d", callCount, expectedCalls)
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	cfg := Config{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	fn := func() (*http.Response, error) {
		callCount++
		if callCount == 2 {
			cancel() // Cancel after first retry
		}
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
		}, nil
	}

	_, err := Do(ctx, cfg, fn)
	if err == nil {
		t.Fatal("Do() error = nil, want context error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Do() error = %v, want context.Canceled", err)
	}
}
