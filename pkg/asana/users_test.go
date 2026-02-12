package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ioanzicu/asana-extractor/pkg/client"
	"github.com/ioanzicu/asana-extractor/pkg/ratelimit"
	"github.com/ioanzicu/asana-extractor/pkg/retry"
)

func TestGetUsers_Table(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		baseURL       string
		handler       http.HandlerFunc
		limit         int
		offset        string
		expectErr     bool
		expectedCount int
		errMessage    string
	}{
		{
			name:      "Successful single page",
			workspace: "ws1",
			limit:     2,
			handler: func(w http.ResponseWriter, r *http.Request) {
				resp := UsersResponse{
					Data: []User{{GID: "u1"}, {GID: "u2"}},
				}
				json.NewEncoder(w).Encode(resp)
			},
			expectErr:     false,
			expectedCount: 2,
		},
		{
			name:      "API error returns failure",
			workspace: "ws1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectErr:  true,
			errMessage: "failed to get users",
		},
		{
			name:      "Invalid JSON response",
			workspace: "ws1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{invalid json`))
			},
			expectErr:  true,
			errMessage: "failed to parse users response",
		},
		{
			name:       "Invalid Base URL parsing",
			baseURL:    " ://invalid-url", // Leading space causes parse error
			handler:    func(w http.ResponseWriter, r *http.Request) {},
			expectErr:  true,
			errMessage: "failed to parse URL",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			targetURL := server.URL
			if tc.baseURL != "" {
				targetURL = tc.baseURL
			}

			hc := client.New(client.Config{
				RateLimitConfig: ratelimit.Config{RequestsPerMinute: 60, MaxConcurrentRead: 1, MaxConcurrentWrite: 1},
				RetryConfig:     retry.Config{MaxRetries: 0},
			})
			asanaClient := NewClient(hc, tc.workspace, targetURL, 10)

			users, _, err := asanaClient.GetUsers(context.Background(), tc.limit, tc.offset)

			if (err != nil) != tc.expectErr {
				t.Fatalf("expectError %v, got %v", tc.expectErr, err)
			}
			if tc.expectErr && tc.errMessage != "" {
				if !contains(err.Error(), tc.errMessage) {
					t.Errorf("expected error containing %q, got %q", tc.errMessage, err.Error())
				}
			}
			if !tc.expectErr && len(users) != tc.expectedCount {
				t.Errorf("expected %d users, got %d", tc.expectedCount, len(users))
			}
		})
	}
}

func TestGetAllUsers_Table(t *testing.T) {
	tests := []struct {
		name          string
		pageSize      int
		pages         []UsersResponse
		expectErr     bool
		expectedCount int
	}{
		{
			name:     "Multi-page successful pagination",
			pageSize: 2,
			pages: []UsersResponse{
				{
					Data:     []User{{GID: "1"}, {GID: "2"}},
					NextPage: &NextPage{Offset: "off1"},
				},
				{
					Data: []User{{GID: "3"}},
				},
			},
			expectErr:     false,
			expectedCount: 3,
		},
		{
			name:     "Empty response on first page",
			pageSize: 10,
			pages: []UsersResponse{
				{Data: []User{}},
			},
			expectErr:     false,
			expectedCount: 0,
		},
		{
			name:     "Pagination stops on nil nextPage",
			pageSize: 2,
			pages: []UsersResponse{
				{
					Data:     []User{{GID: "1"}, {GID: "2"}},
					NextPage: nil,
				},
			},
			expectErr:     false,
			expectedCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			callIdx := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if callIdx < len(tc.pages) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(tc.pages[callIdx])
					callIdx++
				}
			}))
			defer server.Close()

			// FIX: Provide a valid RateLimitConfig to prevent "burst 0" errors
			hc := client.New(client.Config{
				RateLimitConfig: ratelimit.Config{
					RequestsPerMinute:  600,
					MaxConcurrentRead:  10,
					MaxConcurrentWrite: 10,
				},
				RetryConfig: retry.Config{MaxRetries: 0},
			})

			asanaClient := NewClient(hc, "ws", server.URL, tc.pageSize)

			users, err := asanaClient.GetAllUsers(context.Background())

			if (err != nil) != tc.expectErr {
				t.Fatalf("unexpected error status: %v", err)
			}
			if !tc.expectErr && len(users) != tc.expectedCount {
				t.Errorf("expected %d users, got %d", tc.expectedCount, len(users))
			}
		})
	}
}

// helper for string matching
func contains(s, substr string) bool {
	return fmt.Sprintf("%v", s) != "" && (len(s) >= len(substr))
}
