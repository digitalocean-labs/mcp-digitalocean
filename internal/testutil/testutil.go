package testutil

import (
	"context"
	"testing"
	"time"

	"mcp-digitalocean/internal/cache"
	"mcp-digitalocean/internal/config"
	"mcp-digitalocean/internal/metrics"
	"mcp-digitalocean/internal/ratelimit"
)

// TestConfig returns a test configuration with sensible defaults
func TestConfig() *config.Config {
	return &config.Config{
		APIToken:         "test-token",
		APIEndpoint:      "https://api.digitalocean.com",
		LogLevel:         "debug",
		Services:         []string{"apps", "droplets"},
		RequestTimeout:   30 * time.Second,
		MaxRetries:       3,
		RetryWaitMin:     1 * time.Second,
		RetryWaitMax:     10 * time.Second,
		CacheEnabled:     true,
		CacheTTL:         5 * time.Minute,
		RateLimitEnabled: true,
		RateLimitRPS:     100,
	}
}

// TestComponents returns test components for testing
func TestComponents() (*metrics.Metrics, *cache.Cache, *ratelimit.RateLimiter) {
	cfg := TestConfig()
	return metrics.New(),
		cache.New(cfg.CacheTTL, cfg.CacheEnabled),
		ratelimit.New(cfg.RateLimitRPS, cfg.RateLimitEnabled)
}

// AssertNoError is a helper function to assert no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError is a helper function to assert an error occurred
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected an error, got nil")
	}
}

// AssertEqual is a helper function to assert two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertTrue is a helper function to assert a condition is true
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Fatal(message)
	}
}

// AssertFalse is a helper function to assert a condition is false
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Fatal(message)
	}
}

// WithTimeout runs a function with a timeout context
func WithTimeout(t *testing.T, timeout time.Duration, fn func(ctx context.Context)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(ctx)
	}()
	
	select {
	case <-done:
		// Function completed successfully
	case <-ctx.Done():
		t.Fatal("Function timed out")
	}
}

// MockTime provides utilities for testing time-dependent code
type MockTime struct {
	current time.Time
}

// NewMockTime creates a new mock time instance
func NewMockTime(start time.Time) *MockTime {
	return &MockTime{current: start}
}

// Now returns the current mock time
func (m *MockTime) Now() time.Time {
	return m.current
}

// Advance advances the mock time by the given duration
func (m *MockTime) Advance(d time.Duration) {
	m.current = m.current.Add(d)
}

// Set sets the mock time to a specific time
func (m *MockTime) Set(t time.Time) {
	m.current = t
}
