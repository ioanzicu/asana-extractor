package extractor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ioanzicu/asana-extractor/pkg/asana"
)

// Stats holds extraction statistics
type Stats struct {
	UsersExtracted    int
	ProjectsExtracted int
	Errors            int
	Duration          time.Duration
}

// AsanaClient defines the subset of Asana operations the extractor needs.
type AsanaClient interface {
	GetAllUsers(ctx context.Context) ([]asana.User, error)
	GetAllProjects(ctx context.Context) ([]asana.Project, error)
}

// Storage defines the interface for storing extracted data
type Storage interface {
	WriteUser(user asana.User) error
	WriteProject(project asana.Project) error
}

// Extractor orchestrates the extraction process
type Extractor struct {
	asanaClient AsanaClient
	storage     Storage
}

// New creates a new extractor
func New(asanaClient AsanaClient, storage Storage) *Extractor {
	return &Extractor{
		asanaClient: asanaClient,
		storage:     storage,
	}
}

// Extract performs a full extraction of users and projects
func (e *Extractor) Extract(ctx context.Context) (*Stats, error) {
	startTime := time.Now()
	stats := &Stats{}

	log.Println("Starting extraction...")

	// Extract users
	log.Println("Extracting users...")
	users, err := e.asanaClient.GetAllUsers(ctx)
	if err != nil {
		return stats, fmt.Errorf("failed to extract users: %w", err)
	}

	log.Printf("Found %d users", len(users))

	// Write each user to storage
	for _, user := range users {
		if err := e.storage.WriteUser(user); err != nil {
			log.Printf("Error writing user %s: %v", user.GID, err)
			stats.Errors++
			continue
		}
		stats.UsersExtracted++
	}

	log.Printf("Successfully extracted %d users", stats.UsersExtracted)

	// Extract projects
	log.Println("Extracting projects...")
	projects, err := e.asanaClient.GetAllProjects(ctx)
	if err != nil {
		return stats, fmt.Errorf("failed to extract projects: %w", err)
	}

	log.Printf("Found %d projects", len(projects))

	// Write each project to storage
	for _, project := range projects {
		if err := e.storage.WriteProject(project); err != nil {
			log.Printf("Error writing project %s: %v", project.GID, err)
			stats.Errors++
			continue
		}
		stats.ProjectsExtracted++
	}

	log.Printf("Successfully extracted %d projects", stats.ProjectsExtracted)

	stats.Duration = time.Since(startTime)
	log.Printf("Extraction completed in %v", stats.Duration)

	return stats, nil
}
