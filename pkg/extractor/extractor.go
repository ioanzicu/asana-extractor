package extractor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ioanzicu/asana-extractor/pkg/asana"
	"github.com/ioanzicu/asana-extractor/pkg/storage"
)

// Stats holds extraction statistics
type Stats struct {
	UsersExtracted    int
	ProjectsExtracted int
	Errors            int
	Duration          time.Duration
}

// Extractor orchestrates the extraction process
type Extractor struct {
	asanaClient *asana.Client
	storage     storage.Storage
}

// New creates a new extractor
func New(asanaClient *asana.Client, storage storage.Storage) *Extractor {
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
