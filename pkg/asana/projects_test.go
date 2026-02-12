package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ioanzicu/asana-extractor/pkg/client"
	"github.com/ioanzicu/asana-extractor/pkg/ratelimit"
	"github.com/ioanzicu/asana-extractor/pkg/retry"
)

func TestGetProjects(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/api/1.0/workspaces/test-workspace/projects" {
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
		resp := ProjectsResponse{
			Data: []Project{
				{
					GID:        "proj1",
					Name:       "Project 1",
					Archived:   false,
					CreatedAt:  time.Now(),
					ModifiedAt: time.Now(),
				},
				{
					GID:        "proj2",
					Name:       "Project 2",
					Archived:   true,
					CreatedAt:  time.Now(),
					ModifiedAt: time.Now(),
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

	asanaClient := NewClient(httpClient, "test-workspace", server.URL+"/api/1.0", 10)

	// Test GetProjects
	projects, _, err := asanaClient.GetProjects(context.Background(), 10, "0")
	if err != nil {
		t.Fatalf("GetProjects() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	if projects[0].GID != "proj1" {
		t.Errorf("Expected GID=proj1, got %s", projects[0].GID)
	}

	if projects[1].Name != "Project 2" {
		t.Errorf("Expected name='Project 2', got %s", projects[1].Name)
	}

	if !projects[1].Archived {
		t.Error("Expected project 2 to be archived")
	}
}

func TestGetAllProjects_Pagination(t *testing.T) {
	callCount := 0

	// Create mock server that returns different pages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		offset := r.URL.Query().Get("offset")

		var resp ProjectsResponse

		switch offset {
		case "":
			// First page (full page of 100)
			projects := make([]Project, 100)
			for i := 0; i < 100; i++ {
				projects[i] = Project{
					GID:  fmt.Sprintf("%d", i),
					Name: fmt.Sprintf("Project %d", i),
				}
			}
			resp = ProjectsResponse{Data: projects, NextPage: &NextPage{Offset: "token_for_page_2"}}
		case "token_for_page_2":
			// Second page (partial page)
			resp = ProjectsResponse{
				Data: []Project{
					{GID: "101", Name: "Project 101"},
					{GID: "102", Name: "Project 102"},
				},
				NextPage: nil,
			}
		default:
			// Empty page
			resp = ProjectsResponse{Data: []Project{}}
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

	asanaClient := NewClient(httpClient, "test-workspace", server.URL+"/api/1.0", 100)

	// Test GetAllProjects
	projects, err := asanaClient.GetAllProjects(context.Background())
	if err != nil {
		t.Fatalf("GetAllProjects() error = %v", err)
	}

	if len(projects) != 102 {
		t.Errorf("Expected 102 projects, got %d", len(projects))
	}

	if callCount != 2 {
		t.Errorf("Expected 2 API calls, got %d", callCount)
	}
}
