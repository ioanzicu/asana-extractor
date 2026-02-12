package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Helper to clear env variables that might interfere with tests
	clearEnv := func() {
		os.Unsetenv("ASANA_TOKEN")
		os.Unsetenv("ASANA_WORKSPACE")
		os.Unsetenv("SCHEDULE_CRON")
		os.Unsetenv("REQUESTS_PER_MINUTE")
	}

	t.Run("Success with valid environment", func(t *testing.T) {
		clearEnv()
		os.Setenv("ASANA_TOKEN", "test-token")
		os.Setenv("ASANA_WORKSPACE", "12345")
		os.Setenv("SCHEDULE_CRON", "0 * * * *")
		os.Setenv("REQUESTS_PER_MINUTE", "100")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if cfg.AsanaToken != "test-token" {
			t.Errorf("Expected token test-token, got %s", cfg.AsanaToken)
		}
		if cfg.ScheduleCron != "0 * * * *" {
			t.Errorf("Expected custom cron, got %s", cfg.ScheduleCron)
		}
		if cfg.RequestsPerMinute != 100 {
			t.Errorf("Expected RPM 100, got %d", cfg.RequestsPerMinute)
		}
	})

	t.Run("Failure when missing required ASANA_TOKEN", func(t *testing.T) {
		clearEnv()
		os.Setenv("ASANA_WORKSPACE", "12345")

		_, err := Load()
		if err == nil {
			t.Error("Expected error due to missing token, but got nil")
		}
	})

	t.Run("Default values work correctly", func(t *testing.T) {
		clearEnv()
		os.Setenv("ASANA_TOKEN", "any")
		os.Setenv("ASANA_WORKSPACE", "any")

		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}

		// Verify a few defaults
		if cfg.OutputDirectory != "./output" {
			t.Errorf("Expected default output dir, got %s", cfg.OutputDirectory)
		}
		if cfg.HTTPTimeout != 30*time.Second {
			t.Errorf("Expected default timeout 30s, got %v", cfg.HTTPTimeout)
		}
	})
}

func TestGetEnvHelpers(t *testing.T) {
	t.Run("getEnvInt returns default on invalid input", func(t *testing.T) {
		os.Setenv("INVALID_INT", "not-a-number")
		defer os.Unsetenv("INVALID_INT")

		val := getEnvInt("INVALID_INT", 42)
		if val != 42 {
			t.Errorf("Expected default 42, got %d", val)
		}
	})

	t.Run("getEnvDuration returns default on invalid input", func(t *testing.T) {
		os.Setenv("INVALID_DUR", "10 years") // Not a parsable Go duration
		defer os.Unsetenv("INVALID_DUR")

		val := getEnvDuration("INVALID_DUR", 5*time.Second)
		if val != 5*time.Second {
			t.Errorf("Expected default 5s, got %v", val)
		}
	})
}
