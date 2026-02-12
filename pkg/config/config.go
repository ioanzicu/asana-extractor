package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	// Asana API configuration
	AsanaToken     string
	AsanaWorkspace string

	// Scheduling configuration
	ScheduleCron string

	// Output configuration
	OutputDirectory string

	// Rate limiting configuration
	RequestsPerMinute  int
	MaxConcurrentRead  int
	MaxConcurrentWrite int

	// HTTP client configuration
	HTTPTimeout time.Duration

	// Retry configuration
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		// We don't necessarily want to return an error here because
		// in production, variables are often set via the OS/Docker
		// without a .env file.
		log.Fatal("No .env file found, fetching from system environment")
	}

	cfg := &Config{
		// Defaults
		ScheduleCron:       getEnv("SCHEDULE_CRON", "*/5 * * * *"), // Every 5 minutes
		OutputDirectory:    getEnv("OUTPUT_DIR", "./output"),
		RequestsPerMinute:  getEnvInt("REQUESTS_PER_MINUTE", 150),
		MaxConcurrentRead:  getEnvInt("MAX_CONCURRENT_READ", 50),
		MaxConcurrentWrite: getEnvInt("MAX_CONCURRENT_WRITE", 15),
		HTTPTimeout:        getEnvDuration("HTTP_TIMEOUT", 30*time.Second),
		MaxRetries:         getEnvInt("MAX_RETRIES", 5),
		InitialBackoff:     getEnvDuration("INITIAL_BACKOFF", 1*time.Second),
		MaxBackoff:         getEnvDuration("MAX_BACKOFF", 60*time.Second),
	}

	// Required fields
	cfg.AsanaToken = os.Getenv("ASANA_TOKEN")
	if cfg.AsanaToken == "" {
		return nil, fmt.Errorf("ASANA_TOKEN environment variable is required")
	}

	cfg.AsanaWorkspace = os.Getenv("ASANA_WORKSPACE")
	if cfg.AsanaWorkspace == "" {
		return nil, fmt.Errorf("ASANA_WORKSPACE environment variable is required")
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration gets a duration environment variable or returns a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
