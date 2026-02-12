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
	// Create a context that is canceled when the OS sends an interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatalf("Application failed: %v", err)
	}

	log.Println("Extractor stopped gracefully")
}

// run handles initialization and execution. It is now exported/visible to tests.
func run(ctx context.Context) error {
	log.Println("Starting Asana Extractor...")

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log.Printf("Configuration loaded: workspace=%s, schedule=%s, output=%s",
		cfg.AsanaWorkspace, cfg.ScheduleCron, cfg.OutputDirectory)

	// 2. Build Dependencies
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
		BaseURL: cfg.BaseURL,
	})

	asanaClient := asana.NewClient(httpClient, cfg.AsanaWorkspace, cfg.BaseURL, cfg.UserPageSize)

	stor, err := storage.NewJSONStorage(cfg.OutputDirectory)
	if err != nil {
		return err
	}

	ext := extractor.New(asanaClient, stor)

	// 3. Define the Job
	extractionJob := func() {
		// Use a background context for the job itself, or pass ctx if you want
		// the job to be interrupted mid-flight during shutdown.
		stats, err := ext.Extract(context.Background())
		if err != nil {
			log.Printf("Extraction failed: %v", err)
			return
		}

		log.Printf("Extraction stats: users=%d, projects=%d, errors=%d, duration=%v",
			stats.UsersExtracted, stats.ProjectsExtracted, stats.Errors, stats.Duration)
	}

	// 4. Run initial extraction
	log.Println("Running initial extraction...")
	extractionJob()

	// 5. Start Scheduler
	sched := scheduler.NewCronScheduler(cfg.ScheduleCron)
	log.Println("Starting scheduler...")

	// This will block until the context is canceled (via SIGINT/SIGTERM)
	return sched.Start(ctx, extractionJob)
}
