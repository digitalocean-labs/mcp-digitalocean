package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	middleware "mcp-digitalocean/internal"
	"mcp-digitalocean/pkg/edgelogging"
	"mcp-digitalocean/pkg/registry"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "1.0.17"
)

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is empty or not present, it returns the fallback value.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	logLevelFlag := flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level: debug, info, warn, error")
	serviceFlag := flag.String("services", getEnv("SERVICES", ""), "Comma-separated list of services to activate (e.g., apps,networking,droplets)")
	tokenFlag := flag.String("digitalocean-api-token", getEnv("DIGITALOCEAN_API_TOKEN", ""), "DigitalOcean API token")
	endpointFlag := flag.String("digitalocean-api-endpoint", getEnv("DIGITALOCEAN_API_ENDPOINT", "https://api.digitalocean.com"), "DigitalOcean API endpoint")
	transport := flag.String("transport", getEnv("TRANSPORT", "stdio"), "The transport protocol to use (http or stdio). Default is stdio.")
	bindAddr := flag.String("bind-addr", getEnv("BIND_ADDR", "127.0.0.1:8080"), "Bind address to bind to. Only used for http transport.")
	edgeLoggingURL := flag.String("edge-logging-url", getEnv("EDGE_LOGGING_URL", ""), "WebSocket URL for edge logging (optional)")
	edgeLoggingToken := flag.String("edge-logging-token", getEnv("EDGE_LOGGING_TOKEN", ""), "Authentication token for edge logging (optional)")
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

	// Create edge logging handler (drop-in replacement for slog.NewJSONHandler)
	handler := edgelogging.NewHandler(os.Stderr, &slog.HandlerOptions{Level: level})

	// Configure WebSocket logging if URL is provided
	if *edgeLoggingURL != "" {
		if err := handler.ConfigureWebSocket(*edgeLoggingURL, *edgeLoggingToken); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to configure edge logging: %v\n", err)
			os.Exit(1)
		}
		defer handler.Close()
	}

	logger := slog.New(handler)
	token := *tokenFlag
	if token == "" && *transport == "stdio" {
		logger.Error("DigitalOcean API token not provided. Use --digitalocean-api-token flag or set DIGITALOCEAN_API_TOKEN environment variable")
		os.Exit(1)
	}

	var services []string
	if *serviceFlag != "" {
		services = strings.Split(*serviceFlag, ",")
	}

	svr := server.NewMCPServer(mcpName, mcpVersion)

	// by default, we create a new client per request.
	getClientFn := func(ctx context.Context) (*godo.Client, error) {
		return clientFromContext(ctx, *endpointFlag)
	}

	// if using stdio, we can re-use the client.
	if *transport == "stdio" {
		godoClient, err := newGodoClientWithTokenAndEndpoint(context.Background(), token, *endpointFlag)
		if err != nil {
			logger.Error("Failed to create DigitalOcean client: " + err.Error())
			os.Exit(1)
		}
		getClientFn = func(ctx context.Context) (*godo.Client, error) {
			return godoClient, nil
		}
	}

	// register the tools.
	err := registry.Register(
		logger,
		svr,
		getClientFn,
		services...,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// start our server.
	err = runServer(ctx, svr, logger, *bindAddr, transport)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Info("shutting down mcp server")
			os.Exit(0)
		} else {
			logger.Error("Failed to serve MCP server: " + err.Error())
			os.Exit(1)
		}
	}
}

func clientFromContext(ctx context.Context, endpoint string) (*godo.Client, error) {
	auth, ok := ctx.Value(middleware.AuthKey{}).(string)
	if !ok || strings.TrimSpace(auth) == "" {
		return nil, errors.New("no auth header found")
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		return nil, errors.New("no bearer token found")
	}
	client, err := newGodoClientWithTokenAndEndpoint(ctx, token, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create godo client: %w", err)
	}

	return client, nil
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

func runServer(ctx context.Context, s *server.MCPServer, logger *slog.Logger, bindAddr string, transport *string) error {
	logger.Info("starting MCP server", "name", mcpName, "version", mcpVersion, "transport", *transport)
	switch *transport {
	case "stdio":
		logger.Info("stdio server started")
		err := server.ServeStdio(s)
		if err != nil {
			return fmt.Errorf("failed to start http server: %w", err)
		}
	// fallback to http
	default:
		errC := make(chan error, 1)
		logger.Info("http server started", "bind_addr", bindAddr)
		httpServer := server.NewStreamableHTTPServer(s,
			server.WithHTTPContextFunc(middleware.AuthFromRequest),
			server.WithStateLess(true),
		)
		go func() {
			errC <- httpServer.Start(bindAddr)
		}()

		select {
		case <-ctx.Done():

			// allow 15 seconds for graceful shutdown
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), time.Second*15)
			defer cancelFunc()

			logger.Info("received shutdown signal")
			err := httpServer.Shutdown(timeoutCtx)
			if err != nil {
				// this happens if the clients still hold connections after the timeout.
				if errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("timeout waiting for server to shutdown: %w", err)
				}

				return fmt.Errorf("failed to gracefully shutdown http server: %w", err)
			}

			return nil
		case err := <-errC:
			if err != nil {
				logger.Error("http server error", "error", err)
				return fmt.Errorf("http server error: %w", err)
			}
		}
	}

	return nil
}
