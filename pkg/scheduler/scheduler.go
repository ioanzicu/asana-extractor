package scheduler

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"
)

// Scheduler defines the interface for job scheduling
type Scheduler interface {
	Start(ctx context.Context, job func()) error
	Stop()
}

// CronScheduler implements Scheduler using cron expressions
type CronScheduler struct {
	cronExpr string
	cron     *cron.Cron
}

// NewCronScheduler creates a new cron-based scheduler
func NewCronScheduler(cronExpr string) *CronScheduler {
	return &CronScheduler{
		cronExpr: cronExpr,
		cron:     cron.New(cron.WithSeconds()),
	}
}

// Start starts the scheduler and runs the job according to the cron expression
func (s *CronScheduler) Start(ctx context.Context, job func()) error {
	// Add the job to the cron scheduler
	_, err := s.cron.AddFunc(s.cronExpr, func() {
		log.Printf("Running scheduled job...")
		job()
	})
	if err != nil {
		return err
	}

	// Start the cron scheduler
	s.cron.Start()
	log.Printf("Scheduler started with cron expression: %s", s.cronExpr)

	// Wait for context cancellation
	<-ctx.Done()
	s.Stop()

	return nil
}

// Stop stops the scheduler
func (s *CronScheduler) Stop() {
	if s.cron != nil {
		log.Println("Stopping scheduler...")
		s.cron.Stop()
	}
}
