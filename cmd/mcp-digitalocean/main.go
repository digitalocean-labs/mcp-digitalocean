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
	"mcp-digitalocean/internal/wslogging"
	"mcp-digitalocean/pkg/registry"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

const (
	mcpName                 = "mcp-digitalocean"
	mcpVersion              = "1.0.22"
	wsLoggingContextTimeout = 15 * time.Second
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
	wsLoggingURL := flag.String("ws-logging-url", getEnv("WS_LOGGING_URL", ""), "WebSocket URL for WebSocket logging (optional)")
	wsLoggingToken := flag.String("ws-logging-token", getEnv("WS_LOGGING_TOKEN", ""), "Authentication token for WebSocket logging (optional)")
	enableToolErrorLogging := flag.Bool("enable-tool-error-logging", getEnv("ENABLE_TOOL_ERROR_LOGGING", "false") == "true", "Enable logging of tool errors")
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

	// setup signal context for graceful shutdown
	// This context is cancelled when the user presses Ctrl+C or the process receives SIGTERM/SIGINT.
	// It is used to signal all long-running goroutines (like WebSocket handlers) to stop their work.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// create WebSocket logging handler (drop-in replacement for slog.NewJSONHandler)
	wsLoggingHandler := wslogging.NewHandler(os.Stderr, &slog.HandlerOptions{Level: level})

	// configure WebSocket logging if URL is provided
	if *wsLoggingURL != "" {
		if err := wsLoggingHandler.ConfigureWebSocket(*wsLoggingURL, *wsLoggingToken); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to configure WebSocket logging: %v\n", err)
			os.Exit(1)
		}

		// start WebSocket logging with signal context for graceful shutdown
		// The context passed here controls when the background goroutines should stop.
		// When a signal is received, ctx is cancelled and the goroutines exit their loops.
		wsLoggingHandler.Start(ctx)

		defer func() {
			// give the handler time to flush remaining logs before shutdown
			// we are not reusing the signal context because:
			// 1. The signal context (ctx) is already cancelled at this point
			// 2. Close() needs time to flush buffered logs, so we give it a new wsLoggingContextTimeout (15-second)
			// which ensures logs are properly flushed even during shutdown
			closeCtx, cancel := context.WithTimeout(context.Background(), wsLoggingContextTimeout)
			defer cancel()
			if err := wsLoggingHandler.Close(closeCtx); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to close WebSocket handler: %v\n", err)
			}
		}()
	}

	var services []string
	if *serviceFlag != "" {
		services = strings.Split(*serviceFlag, ",")
	}

	// add enabled_services as persistent attribute for context/metrics
	// this helps with filtering and understanding server configuration
	if *serviceFlag != "" {
		wsLoggingHandler = wsLoggingHandler.WithAttrs([]slog.Attr{
			slog.String("enabled_services", *serviceFlag),
		}).(*wslogging.Handler)
	}

	// create logger after adding service attributes
	logger := slog.New(wsLoggingHandler)
	token := *tokenFlag
	if token == "" && *transport == "stdio" {
		logger.Error("DigitalOcean API token not provided. Use --digitalocean-api-token flag or set DIGITALOCEAN_API_TOKEN environment variable")
		os.Exit(1)
	}

	var opts []server.ServerOption
	if *enableToolErrorLogging {
		toolLoggingMiddleware := middleware.ToolLoggingMiddleware{Logger: logger}
		opts = append(opts, server.WithToolHandlerMiddleware(toolLoggingMiddleware.ToolMiddleware))
	}

	svr := server.NewMCPServer(mcpName, mcpVersion, opts...)

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
