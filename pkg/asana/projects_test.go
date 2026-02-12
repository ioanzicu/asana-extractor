package asana

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ioanzicu/asana-extractor/pkg/client"
	"github.com/ioanzicu/asana-extractor/pkg/ratelimit"
	"github.com/ioanzicu/asana-extractor/pkg/retry"
)

// setupMockClient ensures the rate limiter is never zero-initialized during tests
func setupMockClient() *client.Client {
	return client.New(client.Config{
		RateLimitConfig: ratelimit.Config{
			RequestsPerMinute:  600,
			MaxConcurrentRead:  10,
			MaxConcurrentWrite: 10,
		},
		RetryConfig: retry.Config{MaxRetries: 0},
	})
}

func TestGetProjects_Table(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		baseURL       string
		handler       http.HandlerFunc
		expectErr     bool
		expectedCount int
		errContains   string
	}{
		{
			name:      "Successful retrieval",
			workspace: "test-ws",
			handler: func(w http.ResponseWriter, r *http.Request) {
				resp := ProjectsResponse{
					Data: []Project{{GID: "p1", Name: "Project Alpha"}},
				}
				json.NewEncoder(w).Encode(resp)
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name:      "API 500 Error",
			workspace: "test-ws",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectErr:   true,
			errContains: "failed to get projects",
		},
		{
			name:      "Malformed JSON response",
			workspace: "test-ws",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{ "data": [ { "gid": `)) // Broken JSON
			},
			expectErr:   true,
			errContains: "failed to parse projects response",
		},
		{
			name:        "Invalid URL parsing",
			baseURL:     " http://bad-url", // Leading space
			expectErr:   true,
			errContains: "failed to parse URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var targetURL string
			if tt.handler != nil {
				server := httptest.NewServer(tt.handler)
				defer server.Close()
				targetURL = server.URL
			}
			if tt.baseURL != "" {
				targetURL = tt.baseURL
			}

			hc := setupMockClient()
			asanaClient := NewClient(hc, tt.workspace, targetURL, 100)

			projects, _, err := asanaClient.GetProjects(context.Background(), 100, "")

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if tt.expectErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			}

			if !tt.expectErr && len(projects) != tt.expectedCount {
				t.Errorf("expected %d projects, got %d", tt.expectedCount, len(projects))
			}
		})
	}
}

func TestGetAllProjects_Table(t *testing.T) {
	tests := []struct {
		name          string
		pages         []ProjectsResponse
		expectErr     bool
		expectedCount int
	}{
		{
			name: "Three-page pagination",
			pages: []ProjectsResponse{
				{Data: []Project{{GID: "1"}}, NextPage: &NextPage{Offset: "o1"}},
				{Data: []Project{{GID: "2"}}, NextPage: &NextPage{Offset: "o2"}},
				{Data: []Project{{GID: "3"}}, NextPage: nil},
			},
			expectErr:     false,
			expectedCount: 3,
		},
		{
			name: "Stops on empty data",
			pages: []ProjectsResponse{
				{Data: []Project{}},
			},
			expectErr:     false,
			expectedCount: 0,
		},
		{
			name: "Stops on empty offset string",
			pages: []ProjectsResponse{
				{Data: []Project{{GID: "1"}}, NextPage: &NextPage{Offset: ""}},
			},
			expectErr:     false,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if callCount < len(tt.pages) {
					json.NewEncoder(w).Encode(tt.pages[callCount])
					callCount++
				}
			}))
			defer server.Close()

			hc := setupMockClient()
			asanaClient := NewClient(hc, "ws", server.URL, 100)

			projects, err := asanaClient.GetAllProjects(context.Background())

			if (err != nil) != tt.expectErr {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.expectErr && len(projects) != tt.expectedCount {
				t.Errorf("expected %d projects, got %d", tt.expectedCount, len(projects))
			}
		})
	}
}
