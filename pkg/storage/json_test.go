package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ioanzicu/asana-extractor/pkg/asana"
)

func TestNewJSONStorage(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	storage, err := NewJSONStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONStorage() error = %v", err)
	}

	// Verify directories were created
	usersDir := filepath.Join(tmpDir, "users")
	projectsDir := filepath.Join(tmpDir, "projects")

	if _, err := os.Stat(usersDir); os.IsNotExist(err) {
		t.Error("Users directory was not created")
	}

	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		t.Error("Projects directory was not created")
	}

	if storage.baseDir != tmpDir {
		t.Errorf("Expected baseDir=%s, got %s", tmpDir, storage.baseDir)
	}
}

func TestWriteUser(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewJSONStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONStorage() error = %v", err)
	}

	// Create test user
	user := asana.User{
		GID:   "123456",
		Name:  "Test User",
		Email: "test@example.com",
	}

	// Write user
	err = storage.WriteUser(user)
	if err != nil {
		t.Fatalf("WriteUser() error = %v", err)
	}

	// Verify file was created
	filename := filepath.Join(tmpDir, "users", "123456.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("User file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read user file: %v", err)
	}

	var readUser asana.User
	if err := json.Unmarshal(data, &readUser); err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	if readUser.GID != user.GID {
		t.Errorf("Expected GID=%s, got %s", user.GID, readUser.GID)
	}

	if readUser.Name != user.Name {
		t.Errorf("Expected Name=%s, got %s", user.Name, readUser.Name)
	}

	if readUser.Email != user.Email {
		t.Errorf("Expected Email=%s, got %s", user.Email, readUser.Email)
	}
}

func TestWriteProject(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewJSONStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONStorage() error = %v", err)
	}

	// Create test project
	project := asana.Project{
		GID:      "proj123",
		Name:     "Test Project",
		Archived: false,
		Public:   true,
	}

	// Write project
	err = storage.WriteProject(project)
	if err != nil {
		t.Fatalf("WriteProject() error = %v", err)
	}

	// Verify file was created
	filename := filepath.Join(tmpDir, "projects", "proj123.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("Project file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read project file: %v", err)
	}

	var readProject asana.Project
	if err := json.Unmarshal(data, &readProject); err != nil {
		t.Fatalf("Failed to unmarshal project: %v", err)
	}

	if readProject.GID != project.GID {
		t.Errorf("Expected GID=%s, got %s", project.GID, readProject.GID)
	}

	if readProject.Name != project.Name {
		t.Errorf("Expected Name=%s, got %s", project.Name, readProject.Name)
	}

	if readProject.Archived != project.Archived {
		t.Errorf("Expected Archived=%v, got %v", project.Archived, readProject.Archived)
	}
}

func TestWriteUser_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewJSONStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONStorage() error = %v", err)
	}

	// Write first version
	user1 := asana.User{
		GID:   "123",
		Name:  "Original Name",
		Email: "original@example.com",
	}

	err = storage.WriteUser(user1)
	if err != nil {
		t.Fatalf("First WriteUser() error = %v", err)
	}

	// Write updated version
	user2 := asana.User{
		GID:   "123",
		Name:  "Updated Name",
		Email: "updated@example.com",
	}

	err = storage.WriteUser(user2)
	if err != nil {
		t.Fatalf("Second WriteUser() error = %v", err)
	}

	// Verify file contains updated data
	filename := filepath.Join(tmpDir, "users", "123.json")
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read user file: %v", err)
	}

	var readUser asana.User
	if err := json.Unmarshal(data, &readUser); err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	if readUser.Name != "Updated Name" {
		t.Errorf("Expected updated name, got %s", readUser.Name)
	}

	if readUser.Email != "updated@example.com" {
		t.Errorf("Expected updated email, got %s", readUser.Email)
	}
}
