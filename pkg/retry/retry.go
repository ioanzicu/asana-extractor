package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// Config holds retry configuration
type Config struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultConfig returns sensible default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     60 * time.Second,
	}
}

// ShouldRetry determines if a response should trigger a retry
func ShouldRetry(resp *http.Response, err error) bool {
	// Network errors should be retried
	if err != nil {
		return true
	}

	// Retry on rate limit (429) and server errors (5xx)
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}

	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		return true
	}

	return false
}

// GetRetryAfter extracts the Retry-After header value in seconds
// Returns 0 if header is not present or invalid
func GetRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if t, err := http.ParseTime(retryAfter); err == nil {
		duration := time.Until(t)
		if duration > 0 {
			return duration
		}
	}

	return 0
}

// CalculateBackoff calculates the backoff duration with exponential backoff and jitter
func CalculateBackoff(attempt int, cfg Config, retryAfter time.Duration) time.Duration {
	// If Retry-After is specified, use it
	if retryAfter > 0 {
		return retryAfter
	}

	// Exponential backoff: initialBackoff * 2^attempt
	backoff := float64(cfg.InitialBackoff) * math.Pow(2, float64(attempt))

	// Add jitter (±25%)
	jitter := backoff * 0.25 * (rand.Float64()*2 - 1)
	backoff += jitter

	// Cap at max backoff
	if backoff > float64(cfg.MaxBackoff) {
		backoff = float64(cfg.MaxBackoff)
	}

	return time.Duration(backoff)
}

// Do executes a function with retry logic
// The function should return the HTTP response and any error
func Do(ctx context.Context, cfg Config, fn func() (*http.Response, error)) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Execute the function
		resp, err = fn()

		// Check if we should retry
		if !ShouldRetry(resp, err) {
			// Success or non-retryable error
			return resp, err
		}

		// Don't retry if we've exhausted attempts
		if attempt == cfg.MaxRetries {
			if err != nil {
				return nil, fmt.Errorf("max retries exceeded: %w", err)
			}
			return resp, fmt.Errorf("max retries exceeded, last status: %d", resp.StatusCode)
		}

		// Calculate backoff
		retryAfter := GetRetryAfter(resp)
		backoff := CalculateBackoff(attempt, cfg, retryAfter)

		// Close the response body if present
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return resp, err
}
