package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ioanzicu/asana-extractor/pkg/asana"
)

// JSONStorage implements Storage by writing individual JSON files
type JSONStorage struct {
	baseDir string
}

// NewJSONStorage creates a new JSON storage instance
func NewJSONStorage(baseDir string) (*JSONStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create subdirectories
	usersDir := filepath.Join(baseDir, "users")
	projectsDir := filepath.Join(baseDir, "projects")

	if err := os.MkdirAll(usersDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create users directory: %w", err)
	}

	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create projects directory: %w", err)
	}

	return &JSONStorage{
		baseDir: baseDir,
	}, nil
}

// WriteUser writes a user to a JSON file
func (s *JSONStorage) WriteUser(user asana.User) error {
	filename := filepath.Join(s.baseDir, "users", fmt.Sprintf("%s.json", user.GID))
	return s.writeJSON(filename, user)
}

// WriteProject writes a project to a JSON file
func (s *JSONStorage) WriteProject(project asana.Project) error {
	filename := filepath.Join(s.baseDir, "projects", fmt.Sprintf("%s.json", project.GID))
	return s.writeJSON(filename, project)
}

// writeJSON writes data to a JSON file atomically
func (s *JSONStorage) writeJSON(filename string, data interface{}) error {
	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to temporary file first
	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename to final filename (atomic operation)
	if err := os.Rename(tempFile, filename); err != nil {
		os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
