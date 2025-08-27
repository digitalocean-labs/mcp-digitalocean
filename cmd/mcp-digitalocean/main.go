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

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "1.0.10"

	defaultEndpoint = "https://api.digitalocean.com"
)

func main() {
	logLevelFlag := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	serviceFlag := flag.String("services", "", "Comma-separated list of services to activate (e.g., apps,networking,droplets)")
	tokenFlag := flag.String("digitalocean-api-token", "", "DigitalOcean API token")

	// optional
	endpointFlag := flag.String("digitalocean-api-endpoint", "", "DigitalOcean API endpoint")
	flag.Parse()

	var level slog.Level
	switch strings.ToLower(*logLevelFlag) {
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
	token := *tokenFlag
	if token == "" {
		token = os.Getenv("DIGITALOCEAN_API_TOKEN")
		if token == "" {
			logger.Error("DigitalOcean API token not provided. Use --digitalocean-api-token flag or set DIGITALOCEAN_API_TOKEN environment variable")
			os.Exit(1)
		}
	}

	endpoint := *endpointFlag
	if endpoint != "" {
		endpoint = os.Getenv("DIGITALOCEAN_API_ENDPOINT")
		if endpoint == "" {
			endpoint = defaultEndpoint
		}
	}

	var services []string
	if *serviceFlag != "" {
		services = strings.Split(*serviceFlag, ",")
	}

	client, err := newGodoClientWithTokenAndEndpoint(context.Background(), token, endpoint)
	if err != nil {
		logger.Error("Failed to create DigitalOcean client: " + err.Error())
		os.Exit(1)
	}

	s := server.NewMCPServer(mcpName, mcpVersion)
	err = registry.Register(logger, s, client, services...)
	if err != nil {
		logger.Error("Failed to register tools: " + err.Error())
		os.Exit(1)
	}

	logger.Debug("starting MCP server", "name", mcpName, "version", mcpVersion)
	err = server.ServeStdio(s)
	if err != nil {
		// if context cancelled or sigterm then shutdown gracefully
		if errors.Is(err, context.Canceled) {
			logger.Info("Server shutdown gracefully")
			os.Exit(0)
		} else {
			logger.Error("Failed to serve MCP server: " + err.Error())
			os.Exit(1)
		}
	}
}

// newGodoClientWithTokenAndEndpoint initializes a new godo client with a custom user agent and endpoint.
func newGodoClientWithTokenAndEndpoint(ctx context.Context, token string, endpoint string) (*godo.Client, error) {
	cleanToken := strings.Trim(strings.TrimSpace(token), "'")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cleanToken})
	oauthClient := oauth2.NewClient(ctx, ts)

	retry := godo.RetryConfig{
		RetryMax:     4,
		RetryWaitMin: godo.PtrTo(float64(1)),
		RetryWaitMax: godo.PtrTo(float64(30)),
	}

	return godo.New(oauthClient,
		godo.WithRetryAndBackoffs(retry),
		godo.SetBaseURL(endpoint),
		godo.SetUserAgent(fmt.Sprintf("%s/%s", mcpName, mcpVersion)))
}
