package asana

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ioanzicu/asana-extractor/pkg/client"
	"github.com/ioanzicu/asana-extractor/pkg/ratelimit"
	"github.com/ioanzicu/asana-extractor/pkg/retry"
)

func TestGetUsers(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/api/1.0/workspaces/test-workspace/users" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		// Check query parameters
		limit := r.URL.Query().Get("limit")
		offset := r.URL.Query().Get("offset")

		if limit != "10" {
			t.Errorf("Expected limit=10, got %s", limit)
		}
		if offset != "0" {
			t.Errorf("Expected offset=0, got %s", offset)
		}

		// Return mock response
		resp := UsersResponse{
			Data: []User{
				{
					GID:   "123",
					Name:  "Test User 1",
					Email: "test1@example.com",
				},
				{
					GID:   "456",
					Name:  "Test User 2",
					Email: "test2@example.com",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	httpClient := client.New(client.Config{
		Token: "test-token",
		RateLimitConfig: ratelimit.Config{
			RequestsPerMinute:  600,
			MaxConcurrentRead:  10,
			MaxConcurrentWrite: 5,
		},
		RetryConfig: retry.DefaultConfig(),
	})

	// Override base URL for testing
	originalBaseURL := baseURL
	defer func() { baseURL = originalBaseURL }()
	baseURL = server.URL + "/api/1.0"

	asanaClient := NewClient(httpClient, "test-workspace")

	// Test GetUsers
	users, err := asanaClient.GetUsers(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	if users[0].GID != "123" {
		t.Errorf("Expected GID=123, got %s", users[0].GID)
	}

	if users[1].Name != "Test User 2" {
		t.Errorf("Expected name='Test User 2', got %s", users[1].Name)
	}
}

func TestGetAllUsers_Pagination(t *testing.T) {
	callCount := 0

	// Create mock server that returns different pages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		offset := r.URL.Query().Get("offset")

		var resp UsersResponse

		switch offset {
		case "0":
			// First page
			resp = UsersResponse{
				Data: []User{
					{GID: "1", Name: "User 1"},
					{GID: "2", Name: "User 2"},
				},
			}
		case "2":
			// Second page
			resp = UsersResponse{
				Data: []User{
					{GID: "3", Name: "User 3"},
				},
			}
		default:
			// Empty page
			resp = UsersResponse{
				Data: []User{},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	httpClient := client.New(client.Config{
		Token: "test-token",
		RateLimitConfig: ratelimit.Config{
			RequestsPerMinute:  600,
			MaxConcurrentRead:  10,
			MaxConcurrentWrite: 5,
		},
		RetryConfig: retry.DefaultConfig(),
	})

	// Override base URL
	originalBaseURL := baseURL
	defer func() { baseURL = originalBaseURL }()
	baseURL = server.URL + "/api/1.0"

	asanaClient := NewClient(httpClient, "test-workspace")

	// Test GetAllUsers
	users, err := asanaClient.GetAllUsers(context.Background())
	if err != nil {
		t.Fatalf("GetAllUsers() error = %v", err)
	}

	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}

	if callCount != 2 {
		t.Errorf("Expected 2 API calls, got %d", callCount)
	}

	// Verify all users were retrieved
	expectedGIDs := []string{"1", "2", "3"}
	for i, user := range users {
		if user.GID != expectedGIDs[i] {
			t.Errorf("User %d: expected GID=%s, got %s", i, expectedGIDs[i], user.GID)
		}
	}
}
