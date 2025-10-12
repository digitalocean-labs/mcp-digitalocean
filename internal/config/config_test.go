package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment
	originalToken := os.Getenv("DIGITALOCEAN_API_TOKEN")
	originalServices := os.Getenv("SERVICES")
	
	// Clean up after test
	defer func() {
		os.Setenv("DIGITALOCEAN_API_TOKEN", originalToken)
		os.Setenv("SERVICES", originalServices)
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"DIGITALOCEAN_API_TOKEN": "test-token",
				"SERVICES":               "apps,droplets",
				"LOG_LEVEL":              "debug",
			},
			expectError: false,
		},
		{
			name: "missing API token",
			envVars: map[string]string{
				"SERVICES": "apps",
			},
			expectError: true,
		},
		{
			name: "custom timeout",
			envVars: map[string]string{
				"DIGITALOCEAN_API_TOKEN": "test-token",
				"REQUEST_TIMEOUT":        "60s",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := LoadConfig()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Validate configuration
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Configuration validation failed: %v", err)
			}

			// Check specific values
			if tt.envVars["DIGITALOCEAN_API_TOKEN"] != "" {
				if cfg.APIToken != tt.envVars["DIGITALOCEAN_API_TOKEN"] {
					t.Errorf("Expected API token %s, got %s", tt.envVars["DIGITALOCEAN_API_TOKEN"], cfg.APIToken)
				}
			}

			if tt.envVars["REQUEST_TIMEOUT"] == "60s" {
				if cfg.RequestTimeout != 60*time.Second {
					t.Errorf("Expected timeout 60s, got %v", cfg.RequestTimeout)
				}
			}

			// Clean up environment variables
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				APIToken:       "test-token",
				RequestTimeout: 30 * time.Second,
				MaxRetries:     3,
				RateLimitRPS:   100,
				RateLimitEnabled: true,
			},
			expectError: false,
		},
		{
			name: "missing API token",
			config: &Config{
				RequestTimeout: 30 * time.Second,
			},
			expectError: true,
		},
		{
			name: "invalid timeout",
			config: &Config{
				APIToken:       "test-token",
				RequestTimeout: -1 * time.Second,
			},
			expectError: true,
		},
		{
			name: "invalid rate limit",
			config: &Config{
				APIToken:         "test-token",
				RequestTimeout:   30 * time.Second,
				RateLimitEnabled: true,
				RateLimitRPS:     -1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
