package main

import (
	"context"
	"testing"
	"time"
)

func TestRun_Table(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		timeout     time.Duration
		expectError bool
	}{
		{
			name: "Missing Asana Token",
			envVars: map[string]string{
				"ASANA_TOKEN":     "",
				"ASANA_WORKSPACE": "123",
			},
			expectError: true,
		},
		{
			name: "Invalid Cron Expression",
			envVars: map[string]string{
				"ASANA_TOKEN":     "valid-token",
				"ASANA_WORKSPACE": "123",
				"SCHEDULE_CRON":   "invalid-cron",
			},
			expectError: true,
		},
		{
			name: "Successful Initialization and Shutdown",
			envVars: map[string]string{
				"ASANA_TOKEN":     "valid-token",
				"ASANA_WORKSPACE": "123",
				"SCHEDULE_CRON":   "0 0 0 1 1 *", // Jan 1st
				"OUTPUT_DIR":      t.TempDir(),   // Use temp dir for tests
			},
			timeout:     200 * time.Millisecond,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Set Environment for this sub-test
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			// 2. Setup Context
			var ctx context.Context
			var cancel context.CancelFunc

			if tc.timeout > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tc.timeout)
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()

			// 3. Execute run()
			err := run(ctx)

			// 4. Assertions
			if tc.expectError {
				if err == nil {
					t.Error("expected an error but got nil")
				}
			} else {
				// For the success path, we expect a timeout or cancel error
				// because run() blocks on the scheduler.
				if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
					t.Errorf("expected no error (or timeout), got: %v", err)
				}
			}
		})
	}
}
