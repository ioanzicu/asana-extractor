package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ioanzicu/asana-extractor/pkg/ratelimit"
	"github.com/ioanzicu/asana-extractor/pkg/retry"
)

func TestClient_Table(t *testing.T) {
	testToken := "super-secret-token"

	tests := []struct {
		name          string
		method        string
		token         string
		serverHandler func(attempts *int) http.HandlerFunc
		retryCfg      retry.Config
		rlCfg         ratelimit.Config
		ctx           func() (context.Context, context.CancelFunc)
		call          func(ctx context.Context, c *Client, url string) (interface{}, error)
		expectedCalls int
		expectError   bool
		errorContains string
	}{
		{
			name:  "Successful GET with Headers and Body",
			token: testToken,
			serverHandler: func(attempts *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attempts++
					if r.Header.Get("Authorization") != "Bearer "+testToken {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("success-body"))
				}
			},
			retryCfg: retry.Config{MaxRetries: 0},
			call: func(ctx context.Context, c *Client, url string) (interface{}, error) {
				return c.GetBody(ctx, url)
			},
			expectedCalls: 1,
			expectError:   false,
		},
		{
			name:  "Write Request (POST) Detection",
			token: testToken,
			serverHandler: func(attempts *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attempts++
					if r.Method != http.MethodPost {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
					w.WriteHeader(http.StatusCreated)
				}
			},
			rlCfg: ratelimit.Config{RequestsPerMinute: 60, MaxConcurrentRead: 1, MaxConcurrentWrite: 1},
			call: func(ctx context.Context, c *Client, url string) (interface{}, error) {
				req, _ := http.NewRequest(http.MethodPost, url, nil)
				return c.Do(ctx, req)
			},
			expectedCalls: 1,
			expectError:   false,
		},
		{
			name:  "Rate Limiter Context Cancellation",
			token: "test",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			rlCfg: ratelimit.Config{RequestsPerMinute: 60, MaxConcurrentRead: 1, MaxConcurrentWrite: 1},
			call: func(ctx context.Context, c *Client, url string) (interface{}, error) {
				return c.Get(ctx, url)
			},
			expectedCalls: 0, // Should never hit server
			expectError:   true,
			errorContains: "rate limiter error",
		},
		{
			name:  "Retry integration succeeds after failures",
			token: "test",
			serverHandler: func(attempts *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attempts++
					if *attempts < 3 {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					w.WriteHeader(http.StatusOK)
				}
			},
			retryCfg: retry.Config{
				MaxRetries:     3,
				InitialBackoff: 1 * time.Millisecond,
				MaxBackoff:     2 * time.Millisecond,
			},
			call: func(ctx context.Context, c *Client, url string) (interface{}, error) {
				return c.Get(ctx, url)
			},
			expectedCalls: 3,
			expectError:   false,
		},
		{
			name:  "Invalid URL error in Get",
			token: "test",
			call: func(ctx context.Context, c *Client, url string) (interface{}, error) {
				return c.Get(ctx, " http://invalid-leading-space")
			},
			expectError:   true,
			errorContains: "failed to create request",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attempts := 0
			var url string
			if tc.serverHandler != nil {
				server := httptest.NewServer(tc.serverHandler(&attempts))
				defer server.Close()
				url = server.URL
			}

			// Default RL config if not provided
			rl := tc.rlCfg
			if rl.RequestsPerMinute == 0 {
				rl = ratelimit.Config{RequestsPerMinute: 600, MaxConcurrentRead: 10, MaxConcurrentWrite: 10}
			}

			c := New(Config{
				Token:           tc.token,
				RateLimitConfig: rl,
				RetryConfig:     tc.retryCfg,
				Timeout:         time.Second,
			})

			ctx := context.Background()
			if tc.ctx != nil {
				var cancel context.CancelFunc
				ctx, cancel = tc.ctx()
				defer cancel()
			}

			res, err := tc.call(ctx, c, url)

			// 1. Check Error State
			if (err != nil) != tc.expectError {
				t.Fatalf("expectError %v, but got error: %v", tc.expectError, err)
			}

			// 2. Check Error Message
			if tc.expectError && tc.errorContains != "" {
				if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("expected error containing %q, got %q", tc.errorContains, err.Error())
				}
			}

			// 3. Check Success Content
			if !tc.expectError && tc.name == "Successful GET with Headers and Body" {
				body := res.([]byte)
				if string(body) != "success-body" {
					t.Errorf("expected body 'success-body', got %q", string(body))
				}
			}

			// 4. Check Server Interaction
			if attempts != tc.expectedCalls {
				t.Errorf("expected %d calls, got %d", tc.expectedCalls, attempts)
			}
		})
	}
}
