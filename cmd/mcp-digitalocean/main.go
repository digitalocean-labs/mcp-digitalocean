package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	registry "mcp-digitalocean/internal"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "0.1.0"
)

func main() {
	logLevelFlag := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	serviceFlag := flag.String("services", "", "Comma-separated list of services to activate (e.g., apps,networking,droplets)")
	tokenFlag := flag.String("digitalocean-api-token", "", "DigitalOcean API token")
	stdio := flag.Bool("stdio", false, "Run server in stdio mode")
	httpAddr := flag.String("http", "127.0.0.1:8080", "HTTP bind address (ignored if --unix is set)")
	unixSock := flag.String("unix", "", "Path to UNIX socket (if set, takes precedence over --http)")
	baseUrl := flag.String("base-url", "http://127.0.0.1:8080", "Base URL for the server (optional)")
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

	// Default to stdio if no flags are given
	if len(os.Args) == 1 {
		*stdio = true
	}

	if token == "" {
		token = os.Getenv("DIGITALOCEAN_API_TOKEN")
		if token == "" {
			logger.Error("DigitalOcean API token not provided. Use --digitalocean-api-token flag or set DIGITALOCEAN_API_TOKEN environment variable")
			os.Exit(1)
		}
	}

	var services []string
	if *serviceFlag != "" {
		services = strings.Split(*serviceFlag, ",")
	}

	client, err := newGodoClientWithToken(context.Background(), token)
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

	if *stdio {
		logger.Debug("starting MCP server", "mode", "stdio", "name", mcpName, "version", mcpVersion)
		// Start the stdio server
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
	} else {
		sseServer := server.NewSSEServer(s, server.WithBaseURL(*baseUrl))

		// Start on UNIX socket if path to bind on given, otherwise start on address and port
		if *unixSock != "" {
			logger.Debug("starting MCP server", "mode", "unix", "unix", *unixSock, "name", mcpName, "version", mcpVersion)
			listener, err := net.Listen("unix", *unixSock)
			if err != nil {
				slog.Error("Failed to listen on UNIX socket", "error", err)
				os.Exit(1)
			}
			defer listener.Close()
			err = http.Serve(listener, sseServer)
			// if context cancelled or sigterm then shutdown gracefully
			if errors.Is(err, context.Canceled) {
				logger.Info("Server shutdown gracefully")
				os.Exit(0)
			} else {
				logger.Error("Failed to serve MCP server: " + err.Error())
				os.Exit(1)
			}
		} else {
			logger.Debug("starting MCP server", "mode", "http", "http", *httpAddr, "name", mcpName, "version", mcpVersion)
			err := http.ListenAndServe(*httpAddr, sseServer)
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
}

// newGodoClientWithToken initializes a new godo client with a custom user agent.
func newGodoClientWithToken(ctx context.Context, token string) (*godo.Client, error) {
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
		godo.SetUserAgent(fmt.Sprintf("%s/%s", mcpName, mcpVersion)))
}
