package extractor

import (
	"context"
	"fmt"
	"testing"

	"github.com/ioanzicu/asana-extractor/pkg/asana"
)

type mockAsanaClient struct {
	usersFunc    func(ctx context.Context) ([]asana.User, error)
	projectsFunc func(ctx context.Context) ([]asana.Project, error)
}

func (m *mockAsanaClient) GetAllUsers(ctx context.Context) ([]asana.User, error) {
	return m.usersFunc(ctx)
}

func (m *mockAsanaClient) GetAllProjects(ctx context.Context) ([]asana.Project, error) {
	return m.projectsFunc(ctx)
}

type mockStorage struct {
	writeUserFunc    func(user asana.User) error
	writeProjectFunc func(project asana.Project) error
}

func (m *mockStorage) WriteUser(u asana.User) error       { return m.writeUserFunc(u) }
func (m *mockStorage) WriteProject(p asana.Project) error { return m.writeProjectFunc(p) }

func TestExtractor_Extract_WithInterface(t *testing.T) {
	tests := []struct {
		name             string
		mockUsers        []asana.User
		userErr          error
		mockProjects     []asana.Project
		projErr          error
		storageUserErr   error // Independent user error
		storageProjErr   error // Independent project error
		expectedUsers    int
		expectedProjects int
		expectedErrors   int
		expectErr        bool
	}{
		{
			name:             "Success path",
			mockUsers:        []asana.User{{GID: "u1"}},
			mockProjects:     []asana.Project{{GID: "p1"}},
			expectedUsers:    1,
			expectedProjects: 1,
			expectedErrors:   0,
		},
		{
			name:      "API failure on Users",
			userErr:   fmt.Errorf("asana down"),
			expectErr: true,
		},
		{
			name:             "Storage failure on Projects only",
			mockUsers:        []asana.User{{GID: "u1"}},
			mockProjects:     []asana.Project{{GID: "p1"}, {GID: "p2"}},
			storageUserErr:   nil,              // Users should succeed
			storageProjErr:   fmt.Errorf("db"), // Projects should fail
			expectedUsers:    1,
			expectedProjects: 0,
			expectedErrors:   2,
			expectErr:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mAsana := &mockAsanaClient{
				usersFunc:    func(ctx context.Context) ([]asana.User, error) { return tc.mockUsers, tc.userErr },
				projectsFunc: func(ctx context.Context) ([]asana.Project, error) { return tc.mockProjects, tc.projErr },
			}

			// Map the specific errors to the mock functions
			mStorage := &mockStorage{
				writeUserFunc:    func(u asana.User) error { return tc.storageUserErr },
				writeProjectFunc: func(p asana.Project) error { return tc.storageProjErr },
			}

			extractor := New(mAsana, mStorage)
			stats, err := extractor.Extract(context.Background())

			if (err != nil) != tc.expectErr {
				t.Fatalf("expectErr %v, got %v", tc.expectErr, err)
			}

			if !tc.expectErr {
				if stats.UsersExtracted != tc.expectedUsers {
					t.Errorf("expected users %d, got %d", tc.expectedUsers, stats.UsersExtracted)
				}
				if stats.ProjectsExtracted != tc.expectedProjects {
					t.Errorf("expected projects %d, got %d", tc.expectedProjects, stats.ProjectsExtracted)
				}
				if stats.Errors != tc.expectedErrors {
					t.Errorf("expected errors %d, got %d", tc.expectedErrors, stats.Errors)
				}
			}
		})
	}
}
