package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the configuration for the MCP DigitalOcean server
type Config struct {
	// API Configuration
	APIToken    string
	APIEndpoint string
	
	// Server Configuration
	LogLevel string
	Services []string
	
	// Performance Configuration
	RequestTimeout    time.Duration
	MaxRetries       int
	RetryWaitMin     time.Duration
	RetryWaitMax     time.Duration
	
	// Cache Configuration
	CacheEnabled bool
	CacheTTL     time.Duration
	
	// Rate Limiting
	RateLimitEnabled bool
	RateLimitRPS     int
}

// LoadConfig loads configuration from environment variables and flags
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// Defaults
		APIEndpoint:      getEnvOrDefault("DIGITALOCEAN_API_ENDPOINT", "https://api.digitalocean.com"),
		LogLevel:         getEnvOrDefault("LOG_LEVEL", "info"),
		RequestTimeout:   getDurationEnvOrDefault("REQUEST_TIMEOUT", 30*time.Second),
		MaxRetries:       getIntEnvOrDefault("MAX_RETRIES", 4),
		RetryWaitMin:     getDurationEnvOrDefault("RETRY_WAIT_MIN", 1*time.Second),
		RetryWaitMax:     getDurationEnvOrDefault("RETRY_WAIT_MAX", 30*time.Second),
		CacheEnabled:     getBoolEnvOrDefault("CACHE_ENABLED", true),
		CacheTTL:         getDurationEnvOrDefault("CACHE_TTL", 5*time.Minute),
		RateLimitEnabled: getBoolEnvOrDefault("RATE_LIMIT_ENABLED", true),
		RateLimitRPS:     getIntEnvOrDefault("RATE_LIMIT_RPS", 100),
	}
	
	// Required fields
	cfg.APIToken = os.Getenv("DIGITALOCEAN_API_TOKEN")
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("DIGITALOCEAN_API_TOKEN environment variable is required")
	}
	
	// Parse services
	if services := os.Getenv("SERVICES"); services != "" {
		cfg.Services = strings.Split(services, ",")
		for i, service := range cfg.Services {
			cfg.Services[i] = strings.TrimSpace(service)
		}
	}
	
	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.APIToken == "" {
		return fmt.Errorf("API token is required")
	}
	
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}
	
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	
	if c.RateLimitRPS <= 0 && c.RateLimitEnabled {
		return fmt.Errorf("rate limit RPS must be positive when rate limiting is enabled")
	}
	
	return nil
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnvOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnvOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnvOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
