package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ioanzicu/asana-extractor/pkg/asana"
)

func TestNewJSONStorage(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		baseDir string
		wantErr bool
	}{
		{
			name:    "Successful creation",
			baseDir: tmpDir,
			wantErr: false,
		},
		{
			name:    "Nested directory creation",
			baseDir: filepath.Join(tmpDir, "level1", "level2"),
			wantErr: false,
		},
		// Note: testing a failure usually requires a read-only path
		// which is OS-dependent, but MkdirAll rarely fails on TempDir.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewJSONStorage(tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewJSONStorage() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify structure
				for _, sub := range []string{"users", "projects"} {
					path := filepath.Join(tt.baseDir, sub)
					if _, err := os.Stat(path); os.IsNotExist(err) {
						t.Errorf("directory %s was not created", sub)
					}
				}
			}
		})
	}
}

func TestWriteOperations(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewJSONStorage(tmpDir)

	t.Run("WriteUser_Table", func(t *testing.T) {
		tests := []struct {
			name    string
			user    asana.User
			wantErr bool
		}{
			{
				name: "Standard user",
				user: asana.User{GID: "123", Name: "John Doe", Email: "john@example.com"},
			},
			{
				name: "User with special characters in GID",
				user: asana.User{GID: "user-!@#", Name: "Special"},
			},
			{
				name: "Overwrite existing user",
				user: asana.User{GID: "123", Name: "John Updated"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := storage.WriteUser(tt.user)
				if (err != nil) != tt.wantErr {
					t.Errorf("WriteUser() error = %v, wantErr %v", err, tt.wantErr)
				}

				// Verify file content
				path := filepath.Join(tmpDir, "users", tt.user.GID+".json")
				data, _ := os.ReadFile(path)
				var saved asana.User
				json.Unmarshal(data, &saved)
				if saved.Name != tt.user.Name {
					t.Errorf("Expected name %s, got %s", tt.user.Name, saved.Name)
				}
			})
		}
	})

	t.Run("WriteProject_Table", func(t *testing.T) {
		tests := []struct {
			name    string
			project asana.Project
		}{
			{
				name:    "Standard project",
				project: asana.Project{GID: "p1", Name: "Alpha"},
			},
			{
				name:    "Archived project",
				project: asana.Project{GID: "p2", Name: "Beta", Archived: true},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := storage.WriteProject(tt.project); err != nil {
					t.Errorf("WriteProject() failed: %v", err)
				}
			})
		}
	})
}

func TestWriteJSON_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	s := &JSONStorage{baseDir: tmpDir}

	tests := []struct {
		name     string
		filename string
		data     interface{}
	}{
		{
			name:     "Marshaling error",
			filename: filepath.Join(tmpDir, "error.json"),
			data:     make(chan int), // Channels cannot be marshaled to JSON
		},
		{
			name:     "Invalid path error",
			filename: filepath.Join("/nonexistent", "error.json"),
			data:     map[string]string{"foo": "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.writeJSON(tt.filename, tt.data)
			if err == nil {
				t.Error("expected error but got nil")
			}
		})
	}
}
