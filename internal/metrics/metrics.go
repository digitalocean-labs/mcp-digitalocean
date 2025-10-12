package metrics

import (
	"context"
	"sync"
	"time"
)

// Metrics holds various performance and usage metrics
type Metrics struct {
	mu sync.RWMutex
	
	// Request metrics
	TotalRequests     int64
	SuccessfulRequests int64
	FailedRequests    int64
	
	// Timing metrics
	AverageResponseTime time.Duration
	MaxResponseTime     time.Duration
	MinResponseTime     time.Duration
	
	// Service-specific metrics
	ServiceUsage map[string]int64
	
	// Cache metrics
	CacheHits   int64
	CacheMisses int64
	
	// Error metrics
	ErrorsByType map[string]int64
	
	// Rate limiting metrics
	RateLimitedRequests int64
	
	startTime time.Time
}

// New creates a new metrics instance
func New() *Metrics {
	return &Metrics{
		ServiceUsage: make(map[string]int64),
		ErrorsByType: make(map[string]int64),
		MinResponseTime: time.Duration(^uint64(0) >> 1), // Max duration
		startTime: time.Now(),
	}
}

// RecordRequest records a request with its duration and success status
func (m *Metrics) RecordRequest(duration time.Duration, success bool, service string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.TotalRequests++
	
	if success {
		m.SuccessfulRequests++
	} else {
		m.FailedRequests++
	}
	
	// Update timing metrics
	if duration > m.MaxResponseTime {
		m.MaxResponseTime = duration
	}
	
	if duration < m.MinResponseTime {
		m.MinResponseTime = duration
	}
	
	// Calculate average response time
	if m.TotalRequests > 0 {
		totalTime := time.Duration(m.TotalRequests-1) * m.AverageResponseTime + duration
		m.AverageResponseTime = totalTime / time.Duration(m.TotalRequests)
	}
	
	// Record service usage
	if service != "" {
		m.ServiceUsage[service]++
	}
}

// RecordError records an error by type
func (m *Metrics) RecordError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ErrorsByType[errorType]++
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.CacheHits++
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.CacheMisses++
}

// RecordRateLimited records a rate-limited request
func (m *Metrics) RecordRateLimited() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.RateLimitedRequests++
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Deep copy maps
	serviceUsage := make(map[string]int64)
	for k, v := range m.ServiceUsage {
		serviceUsage[k] = v
	}
	
	errorsByType := make(map[string]int64)
	for k, v := range m.ErrorsByType {
		errorsByType[k] = v
	}
	
	return MetricsSnapshot{
		TotalRequests:       m.TotalRequests,
		SuccessfulRequests:  m.SuccessfulRequests,
		FailedRequests:      m.FailedRequests,
		AverageResponseTime: m.AverageResponseTime,
		MaxResponseTime:     m.MaxResponseTime,
		MinResponseTime:     m.MinResponseTime,
		ServiceUsage:        serviceUsage,
		CacheHits:           m.CacheHits,
		CacheMisses:         m.CacheMisses,
		ErrorsByType:        errorsByType,
		RateLimitedRequests: m.RateLimitedRequests,
		Uptime:              time.Since(m.startTime),
	}
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	TotalRequests       int64
	SuccessfulRequests  int64
	FailedRequests      int64
	AverageResponseTime time.Duration
	MaxResponseTime     time.Duration
	MinResponseTime     time.Duration
	ServiceUsage        map[string]int64
	CacheHits           int64
	CacheMisses         int64
	ErrorsByType        map[string]int64
	RateLimitedRequests int64
	Uptime              time.Duration
}

// SuccessRate returns the success rate as a percentage
func (s MetricsSnapshot) SuccessRate() float64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.SuccessfulRequests) / float64(s.TotalRequests) * 100
}

// CacheHitRate returns the cache hit rate as a percentage
func (s MetricsSnapshot) CacheHitRate() float64 {
	total := s.CacheHits + s.CacheMisses
	if total == 0 {
		return 0
	}
	return float64(s.CacheHits) / float64(total) * 100
}

// RequestsPerSecond returns the average requests per second since startup
func (s MetricsSnapshot) RequestsPerSecond() float64 {
	if s.Uptime.Seconds() == 0 {
		return 0
	}
	return float64(s.TotalRequests) / s.Uptime.Seconds()
}

// Middleware wraps a function with metrics collection
func (m *Metrics) Middleware(service string, fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		start := time.Now()
		err := fn(ctx)
		duration := time.Since(start)
		
		success := err == nil
		m.RecordRequest(duration, success, service)
		
		if err != nil {
			m.RecordError(err.Error())
		}
		
		return err
	}
}
