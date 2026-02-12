package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ioanzicu/asana-extractor/pkg/ratelimit"
	"github.com/ioanzicu/asana-extractor/pkg/retry"
)

func TestClient_Requests(t *testing.T) {
	type testCase struct {
		name           string
		token          string
		serverHandler  func(attempts *int) http.HandlerFunc
		retryCfg       retry.Config
		call           func(ctx context.Context, c *Client, url string) (interface{}, error)
		expectedResult string
		expectedCalls  int
		expectError    bool
	}

	testToken := "super-secret-token"

	tests := []testCase{
		{
			name:  "Successful GET with Headers",
			token: testToken,
			serverHandler: func(attempts *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attempts++
					if r.Header.Get("Authorization") != "Bearer "+testToken {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					if r.Header.Get("Accept") != "application/json" {
						w.WriteHeader(http.StatusNotAcceptable)
						return
					}
					w.WriteHeader(http.StatusOK)
				}
			},
			retryCfg: retry.DefaultConfig(),
			call: func(ctx context.Context, c *Client, url string) (interface{}, error) {
				return c.Get(ctx, url)
			},
			expectedCalls: 1,
			expectError:   false,
		},
		{
			name:  "GetBody returns error message on 400",
			token: "test",
			serverHandler: func(attempts *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attempts++
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("bad request details"))
				}
			},
			retryCfg: retry.Config{MaxRetries: 0},
			call: func(ctx context.Context, c *Client, url string) (interface{}, error) {
				return c.GetBody(ctx, url)
			},
			expectedResult: "unexpected status code 400: bad request details",
			expectedCalls:  1,
			expectError:    true,
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
				MaxBackoff:     5 * time.Millisecond,
			},
			call: func(ctx context.Context, c *Client, url string) (interface{}, error) {
				return c.Get(ctx, url)
			},
			expectedCalls: 3,
			expectError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attempts := 0
			server := httptest.NewServer(tc.serverHandler(&attempts))
			defer server.Close()

			cfg := Config{
				Token: tc.token,
				RateLimitConfig: ratelimit.Config{
					RequestsPerMinute: 600, MaxConcurrentRead: 10, MaxConcurrentWrite: 10,
				},
				RetryConfig: tc.retryCfg,
			}

			c := New(cfg)
			_, err := tc.call(context.Background(), c, server.URL)

			// Validate error status
			if (err != nil) != tc.expectError {
				t.Fatalf("expectError %v, but got error: %v", tc.expectError, err)
			}

			// Validate specific error message if expected
			if tc.expectError && tc.expectedResult != "" {
				if err.Error() != tc.expectedResult {
					t.Errorf("expected error %q, got %q", tc.expectedResult, err.Error())
				}
			}

			// Validate successful body content if applicable
			if !tc.expectError && tc.name == "GetBody returns error message on 400" {
				// (Optional) add body validation here
			}

			// Validate total calls made to server
			if attempts != tc.expectedCalls {
				t.Errorf("expected %d calls, got %d", tc.expectedCalls, attempts)
			}
		})
	}
}
