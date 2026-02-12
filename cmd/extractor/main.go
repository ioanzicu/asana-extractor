package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ioanzicu/asana-extractor/pkg/asana"
	"github.com/ioanzicu/asana-extractor/pkg/client"
	"github.com/ioanzicu/asana-extractor/pkg/config"
	"github.com/ioanzicu/asana-extractor/pkg/extractor"
	"github.com/ioanzicu/asana-extractor/pkg/ratelimit"
	"github.com/ioanzicu/asana-extractor/pkg/retry"
	"github.com/ioanzicu/asana-extractor/pkg/scheduler"
	"github.com/ioanzicu/asana-extractor/pkg/storage"
)

func main() {
	log.Println("Starting Asana Extractor...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded: workspace=%s, schedule=%s, output=%s",
		cfg.AsanaWorkspace, cfg.ScheduleCron, cfg.OutputDirectory)

	// Create HTTP client with rate limiting and retry logic
	httpClient := client.New(client.Config{
		Token: cfg.AsanaToken,
		RateLimitConfig: ratelimit.Config{
			RequestsPerMinute:  cfg.RequestsPerMinute,
			MaxConcurrentRead:  cfg.MaxConcurrentRead,
			MaxConcurrentWrite: cfg.MaxConcurrentWrite,
		},
		RetryConfig: retry.Config{
			MaxRetries:     cfg.MaxRetries,
			InitialBackoff: cfg.InitialBackoff,
			MaxBackoff:     cfg.MaxBackoff,
		},
		Timeout: cfg.HTTPTimeout,
	})

	// Create Asana API client
	asanaClient := asana.NewClient(httpClient, cfg.AsanaWorkspace)

	// Create storage
	stor, err := storage.NewJSONStorage(cfg.OutputDirectory)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	// Create extractor
	ext := extractor.New(asanaClient, stor)

	// Create extraction job
	extractionJob := func() {
		ctx := context.Background()
		stats, err := ext.Extract(ctx)
		if err != nil {
			log.Printf("Extraction failed: %v", err)
			return
		}

		log.Printf("Extraction stats: users=%d, projects=%d, errors=%d, duration=%v",
			stats.UsersExtracted, stats.ProjectsExtracted, stats.Errors, stats.Duration)
	}

	// Run initial extraction
	log.Println("Running initial extraction...")
	extractionJob()

	// Create scheduler
	sched := scheduler.NewCronScheduler(cfg.ScheduleCron)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal...")
		cancel()
	}()

	// Start scheduler
	log.Println("Starting scheduler...")
	if err := sched.Start(ctx, extractionJob); err != nil {
		log.Fatalf("Scheduler error: %v", err)
	}

	log.Println("Extractor stopped gracefully")
}
