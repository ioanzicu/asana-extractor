package extractor

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ioanzicu/asana-extractor/pkg/asana"
)

type mockAsanaClient struct {
	users    []asana.User
	projects []asana.Project
	err      error
}

func (m *mockAsanaClient) GetAllUsers(ctx context.Context) ([]asana.User, error) {
	return m.users, m.err
}
func (m *mockAsanaClient) GetAllProjects(ctx context.Context) ([]asana.Project, error) {
	return m.projects, m.err
}

type mockStorage struct {
	mu        sync.Mutex
	users     []asana.User
	projects  []asana.Project
	failWrite bool
}

func (m *mockStorage) WriteUser(u asana.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failWrite {
		return fmt.Errorf("disk error")
	}
	m.users = append(m.users, u)
	return nil
}

func (m *mockStorage) WriteProject(p asana.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failWrite {
		return fmt.Errorf("disk error")
	}
	m.projects = append(m.projects, p)
	return nil
}
func TestExtractor_Extract(t *testing.T) {
	tests := []struct {
		name             string
		mockUsers        []asana.User
		mockProjects     []asana.Project
		apiError         error
		storageFail      bool
		expectErr        bool
		expectedUsers    int
		expectedProjects int
		expectedErrors   int
	}{
		{
			name:             "Successful full extraction",
			mockUsers:        []asana.User{{GID: "u1"}, {GID: "u2"}},
			mockProjects:     []asana.Project{{GID: "p1"}},
			expectErr:        false,
			expectedUsers:    2,
			expectedProjects: 1,
		},
		{
			name:      "API failure returns error immediately",
			apiError:  fmt.Errorf("unauthorized"),
			expectErr: true,
		},
		{
			name:             "Storage failures tracked but don't stop extraction",
			mockUsers:        []asana.User{{GID: "u1"}},
			mockProjects:     []asana.Project{{GID: "p1"}},
			storageFail:      true,
			expectErr:        false,
			expectedUsers:    0,
			expectedProjects: 0,
			expectedErrors:   2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockAsanaClient{
				users:    tc.mockUsers,
				projects: tc.mockProjects,
				err:      tc.apiError,
			}
			mockStore := &mockStorage{failWrite: tc.storageFail}

			e := New(mockClient, mockStore)
			stats, err := e.Extract(context.Background())

			if (err != nil) != tc.expectErr {
				t.Fatalf("expected error: %v, got: %v", tc.expectErr, err)
			}

			if err == nil {
				if stats.UsersExtracted != tc.expectedUsers {
					t.Errorf("expected %d users, got %d", tc.expectedUsers, stats.UsersExtracted)
				}
				if stats.ProjectsExtracted != tc.expectedProjects {
					t.Errorf("expected %d projects, got %d", tc.expectedProjects, stats.ProjectsExtracted)
				}
				if stats.Errors != tc.expectedErrors {
					t.Errorf("expected %d errors, got %d", tc.expectedErrors, stats.Errors)
				}
				if stats.Duration <= 0 {
					t.Error("duration should be positive")
				}
			}
		})
	}
}
