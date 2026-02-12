package extractor

import (
	"context"
	"fmt"
	"log"
	"sync"
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

	// results channel carries functions to update the stats struct safely
	results := make(chan func(*Stats), 100)
	// errChan captures fatal API errors
	errChan := make(chan error, 2)

	var wg sync.WaitGroup
	doneProcessing := make(chan struct{})

	// 1. THE ACTOR: Centralized Stats Collector
	// This is the only goroutine allowed to modify the 'stats' pointer
	go func() {
		for update := range results {
			update(stats)
		}
		close(doneProcessing)
	}()

	// 2. WORKER: User Extraction & Storage
	wg.Add(1)
	go func() {
		defer wg.Done()
		users, err := e.asanaClient.GetAllUsers(ctx)
		if err != nil {
			errChan <- fmt.Errorf("user API failure: %w", err)
			return
		}

		for _, user := range users {
			// THE WRITE HAPPENS HERE
			if err := e.storage.WriteUser(user); err != nil {
				log.Printf("Error writing user %s: %v", user.GID, err)
				results <- func(s *Stats) { s.Errors++ }
				continue
			}
			results <- func(s *Stats) { s.UsersExtracted++ }
		}
	}()

	// 3. WORKER: Project Extraction & Storage
	wg.Add(1)
	go func() {
		defer wg.Done()
		projects, err := e.asanaClient.GetAllProjects(ctx)
		if err != nil {
			errChan <- fmt.Errorf("project API failure: %w", err)
			return
		}

		for _, project := range projects {
			// THE WRITE HAPPENS HERE
			if err := e.storage.WriteProject(project); err != nil {
				log.Printf("Error writing project %s: %v", project.GID, err)
				results <- func(s *Stats) { s.Errors++ }
				continue
			}
			results <- func(s *Stats) { s.ProjectsExtracted++ }
		}
	}()

	// 4. COORDINATION
	// Wait for workers in the background so we can check errChan immediately
	go func() {
		wg.Wait()
		close(results)
		close(errChan)
	}()

	// Check if any worker sent a fatal API error
	for err := range errChan {
		if err != nil {
			stats.Duration = time.Since(startTime)
			return stats, err
		}
	}

	// Wait for the stats collector to finish processing the last updates
	<-doneProcessing

	stats.Duration = time.Since(startTime)
	return stats, nil
}
