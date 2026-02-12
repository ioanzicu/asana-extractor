package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCronScheduler_StartStop(t *testing.T) {
	// Use a frequent cron (every second) for testing
	// Note: robfig/cron/v3 defaults to 5-field standard cron (minutes).
	// To use seconds, you'd need cron.WithSeconds(), but we'll stick to
	// standard and test the "Start/Done" lifecycle.
	cronExpr := "* * * * * *"
	s := NewCronScheduler(cronExpr)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// We use a channel or WaitGroup to see if the scheduler blocks until ctx.Done()
	errChan := make(chan error, 1)

	go func() {
		errChan <- s.Start(ctx, func() {
			// This might not even trigger given the 100ms timeout
			// and minute-level precision, which is fine for this lifecycle test.
		})
	}()

	// Wait for the Start function to return after context timeout
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Start() returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Start() did not return after context cancellation")
	}
}

func TestCronScheduler_InvalidExpression(t *testing.T) {
	s := NewCronScheduler("invalid-cron-expr")

	// Start should return an error immediately if the cron expression is bad
	err := s.Start(context.Background(), func() {})
	if err == nil {
		t.Error("Expected error for invalid cron expression, got nil")
	}
}

func TestCronScheduler_JobExecution(t *testing.T) {
	// 1. Every second (Requires WithSeconds() in constructor)
	s := NewCronScheduler("*/1 * * * * *")

	var wg sync.WaitGroup
	wg.Add(1)

	job := func() {
		wg.Done()
	}

	// 2. Use a context we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Start scheduler in background (because Start blocks)
	go func() {
		if err := s.Start(ctx, job); err != nil {
			t.Logf("Scheduler stopped: %v", err)
		}
	}()

	// 4. Wait for the signal
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success!
	case <-time.After(2500 * time.Millisecond): // Give it 2.5s for safety
		t.Fatal("Job was not called within 2.5 seconds")
	}
}
