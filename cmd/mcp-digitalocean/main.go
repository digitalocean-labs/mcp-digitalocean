package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	registry "mcp-digitalocean/internal"
	"mcp-digitalocean/internal/cache"
	"mcp-digitalocean/internal/config"
	"mcp-digitalocean/internal/health"
	"mcp-digitalocean/internal/metrics"
	"mcp-digitalocean/internal/ratelimit"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "1.0.12"
)

func main() {
	// Parse command line flags for backward compatibility
	logLevelFlag := flag.String("log-level", "", "Log level: debug, info, warn, error")
	serviceFlag := flag.String("services", "", "Comma-separated list of services to activate")
	tokenFlag := flag.String("digitalocean-api-token", "", "DigitalOcean API token")
	endpointFlag := flag.String("digitalocean-api-endpoint", "", "DigitalOcean API endpoint")
	flag.Parse()

	// Override environment variables with command line flags if provided
	if *logLevelFlag != "" {
		os.Setenv("LOG_LEVEL", *logLevelFlag)
	}
	if *serviceFlag != "" {
		os.Setenv("SERVICES", *serviceFlag)
	}
	if *tokenFlag != "" {
		os.Setenv("DIGITALOCEAN_API_TOKEN", *tokenFlag)
	}
	if *endpointFlag != "" {
		os.Setenv("DIGITALOCEAN_API_ENDPOINT", *endpointFlag)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	// Initialize components
	metricsCollector := metrics.New()
	cacheInstance := cache.New(cfg.CacheTTL, cfg.CacheEnabled)
	rateLimiter := ratelimit.New(cfg.RateLimitRPS, cfg.RateLimitEnabled)

	// Create DigitalOcean client with enhanced configuration
	client, err := newGodoClientWithConfig(context.Background(), cfg)
	if err != nil {
		logger.Error("Failed to create DigitalOcean client", "error", err)
		os.Exit(1)
	}

	// Initialize health checker
	healthChecker := health.New(client, cacheInstance, metricsCollector, rateLimiter)

	// Perform initial health check
	ctx := context.Background()
	healthReport, err := healthChecker.CheckHealth(ctx)
	if err != nil {
		logger.Warn("Initial health check failed", "error", err)
	} else {
		logger.Info("Initial health check completed", 
			"status", healthReport.Status,
			"healthy_checks", countHealthyChecks(healthReport.Checks))
	}

	// Create MCP server
	s := server.NewMCPServer(mcpName, mcpVersion)
	
	// Register tools with enhanced registry
	err = registry.RegisterWithComponents(logger, s, client, metricsCollector, cacheInstance, rateLimiter, cfg.Services...)
	if err != nil {
		logger.Error("Failed to register tools", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting MCP server", 
		"name", mcpName, 
		"version", mcpVersion,
		"services", cfg.Services,
		"cache_enabled", cfg.CacheEnabled,
		"rate_limit_enabled", cfg.RateLimitEnabled)

	// Start server
	err = server.ServeStdio(s)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info("Server shutdown gracefully")
			
			// Log final metrics
			finalMetrics := metricsCollector.GetSnapshot()
			logger.Info("Final metrics",
				"total_requests", finalMetrics.TotalRequests,
				"success_rate", fmt.Sprintf("%.2f%%", finalMetrics.SuccessRate()),
				"cache_hit_rate", fmt.Sprintf("%.2f%%", finalMetrics.CacheHitRate()),
				"uptime", finalMetrics.Uptime)
			
			os.Exit(0)
		} else {
			logger.Error("Failed to serve MCP server", "error", err)
			os.Exit(1)
		}
	}
}

// newGodoClientWithConfig initializes a new godo client with enhanced configuration.
func newGodoClientWithConfig(ctx context.Context, cfg *config.Config) (*godo.Client, error) {
	cleanToken := strings.Trim(strings.TrimSpace(cfg.APIToken), "'")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cleanToken})
	oauthClient := oauth2.NewClient(ctx, ts)

	retry := godo.RetryConfig{
		RetryMax:     cfg.MaxRetries,
		RetryWaitMin: godo.PtrTo(cfg.RetryWaitMin.Seconds()),
		RetryWaitMax: godo.PtrTo(cfg.RetryWaitMax.Seconds()),
	}

	return godo.New(oauthClient,
		godo.WithRetryAndBackoffs(retry),
		godo.SetBaseURL(cfg.APIEndpoint),
		godo.SetUserAgent(fmt.Sprintf("%s/%s", mcpName, mcpVersion)))
}

// countHealthyChecks counts the number of healthy checks in a health report
func countHealthyChecks(checks []health.HealthCheck) int {
	count := 0
	for _, check := range checks {
		if check.Status == health.StatusHealthy {
			count++
		}
	}
	return count
}
