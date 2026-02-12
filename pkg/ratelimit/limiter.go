package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RequestType represents the type of HTTP request
type RequestType int

const (
	// RequestTypeRead represents GET requests
	RequestTypeRead RequestType = iota
	// RequestTypeWrite represents POST, PUT, PATCH, DELETE requests
	RequestTypeWrite
)

// Limiter manages rate limiting for Asana API requests
type Limiter struct {
	// Token bucket for overall request rate (e.g., 150 requests/minute)
	rateLimiter *rate.Limiter

	// Concurrent request tracking
	mu                 sync.Mutex
	currentReads       int
	currentWrites      int
	maxConcurrentRead  int
	maxConcurrentWrite int
}

// Config holds configuration for the rate limiter
type Config struct {
	RequestsPerMinute  int
	MaxConcurrentRead  int
	MaxConcurrentWrite int
}

// NewLimiter creates a new rate limiter with the specified configuration
func NewLimiter(cfg Config) *Limiter {
	// Convert requests per minute to requests per second for token bucket
	requestsPerSecond := float64(cfg.RequestsPerMinute) / 60.0

	return &Limiter{
		rateLimiter:        rate.NewLimiter(rate.Limit(requestsPerSecond), cfg.RequestsPerMinute),
		maxConcurrentRead:  cfg.MaxConcurrentRead,
		maxConcurrentWrite: cfg.MaxConcurrentWrite,
	}
}

// Acquire blocks until a request can be made according to rate limits
// Returns an error if context is cancelled
func (l *Limiter) Acquire(ctx context.Context, reqType RequestType) error {
	// First, wait for token bucket
	if err := l.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	// Then, wait for concurrent request slot
	for {
		l.mu.Lock()
		canProceed := false

		switch reqType {
		case RequestTypeRead:
			if l.currentReads < l.maxConcurrentRead {
				l.currentReads++
				canProceed = true
			}
		case RequestTypeWrite:
			if l.currentWrites < l.maxConcurrentWrite {
				l.currentWrites++
				canProceed = true
			}
		}

		l.mu.Unlock()

		if canProceed {
			return nil
		}

		// Wait a bit before trying again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue loop
		}
	}
}

// Release releases a concurrent request slot
func (l *Limiter) Release(reqType RequestType) {
	l.mu.Lock()
	defer l.mu.Unlock()

	switch reqType {
	case RequestTypeRead:
		if l.currentReads > 0 {
			l.currentReads--
		}
	case RequestTypeWrite:
		if l.currentWrites > 0 {
			l.currentWrites--
		}
	}
}

// Stats returns current rate limiter statistics
func (l *Limiter) Stats() (currentReads, currentWrites int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.currentReads, l.currentWrites
}
