package health

import (
	"context"
	"time"

	"github.com/digitalocean/godo"
	"mcp-digitalocean/internal/cache"
	"mcp-digitalocean/internal/metrics"
	"mcp-digitalocean/internal/ratelimit"
)

// HealthStatus represents the health status of the service
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Name      string        `json:"name"`
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthChecker performs health checks on various components
type HealthChecker struct {
	client      *godo.Client
	cache       *cache.Cache
	metrics     *metrics.Metrics
	rateLimiter *ratelimit.RateLimiter
}

// New creates a new health checker
func New(client *godo.Client, cache *cache.Cache, metrics *metrics.Metrics, rateLimiter *ratelimit.RateLimiter) *HealthChecker {
	return &HealthChecker{
		client:      client,
		cache:       cache,
		metrics:     metrics,
		rateLimiter: rateLimiter,
	}
}

// CheckHealth performs all health checks and returns the overall status
func (h *HealthChecker) CheckHealth(ctx context.Context) (*HealthReport, error) {
	checks := []HealthCheck{
		h.checkAPI(ctx),
		h.checkCache(),
		h.checkRateLimit(),
		h.checkMetrics(),
	}
	
	report := &HealthReport{
		Status:    h.determineOverallStatus(checks),
		Checks:    checks,
		Timestamp: time.Now(),
	}
	
	return report, nil
}

// checkAPI checks if the DigitalOcean API is accessible
func (h *HealthChecker) checkAPI(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "digitalocean_api",
		Timestamp: start,
	}
	
	// Create a context with timeout for the API call
	apiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	// Try to get account information as a simple API test
	_, _, err := h.client.Account.Get(apiCtx)
	check.Duration = time.Since(start)
	
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = "Failed to connect to DigitalOcean API: " + err.Error()
	} else {
		check.Status = StatusHealthy
		check.Message = "API connection successful"
	}
	
	return check
}

// checkCache checks the cache status
func (h *HealthChecker) checkCache() HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "cache",
		Timestamp: start,
	}
	
	// Test cache functionality
	testKey := "health_check_test"
	testValue := "test_value"
	
	h.cache.Set(testKey, testValue)
	retrieved, found := h.cache.Get(testKey)
	h.cache.Delete(testKey)
	
	check.Duration = time.Since(start)
	
	if !found || retrieved != testValue {
		check.Status = StatusDegraded
		check.Message = "Cache not functioning properly"
	} else {
		check.Status = StatusHealthy
		check.Message = "Cache functioning normally"
	}
	
	return check
}

// checkRateLimit checks the rate limiter status
func (h *HealthChecker) checkRateLimit() HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "rate_limiter",
		Timestamp: start,
	}
	
	stats := h.rateLimiter.GetStats()
	check.Duration = time.Since(start)
	
	if !stats.Enabled {
		check.Status = StatusHealthy
		check.Message = "Rate limiting disabled"
	} else if stats.CurrentTokens > 0 {
		check.Status = StatusHealthy
		check.Message = "Rate limiter functioning normally"
	} else {
		check.Status = StatusDegraded
		check.Message = "Rate limiter has no available tokens"
	}
	
	return check
}

// checkMetrics checks the metrics collection status
func (h *HealthChecker) checkMetrics() HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "metrics",
		Timestamp: start,
	}
	
	snapshot := h.metrics.GetSnapshot()
	check.Duration = time.Since(start)
	
	// Check if metrics are being collected
	if snapshot.TotalRequests >= 0 {
		check.Status = StatusHealthy
		check.Message = "Metrics collection active"
	} else {
		check.Status = StatusDegraded
		check.Message = "Metrics collection may not be functioning"
	}
	
	return check
}

// determineOverallStatus determines the overall health status based on individual checks
func (h *HealthChecker) determineOverallStatus(checks []HealthCheck) HealthStatus {
	hasUnhealthy := false
	hasDegraded := false
	
	for _, check := range checks {
		switch check.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}
	
	if hasUnhealthy {
		return StatusUnhealthy
	} else if hasDegraded {
		return StatusDegraded
	}
	
	return StatusHealthy
}

// HealthReport represents the overall health report
type HealthReport struct {
	Status    HealthStatus  `json:"status"`
	Checks    []HealthCheck `json:"checks"`
	Timestamp time.Time     `json:"timestamp"`
}

// IsHealthy returns true if the overall status is healthy
func (r *HealthReport) IsHealthy() bool {
	return r.Status == StatusHealthy
}
