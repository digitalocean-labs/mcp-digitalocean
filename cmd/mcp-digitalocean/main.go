package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"strings"

	registry "mcp-digitalocean/internal"
	"mcp-digitalocean/internal/webapp"

	"github.com/digitalocean/godo"
	"github.com/zalando/go-keyring"

	"github.com/mark3labs/mcp-go/server"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "0.1.0"
)

func main() {
	logLevelFlag := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	serviceFlag := flag.String("services", "", "Comma-separated list of services to activate (e.g., apps,networking,droplets)")
	tokenFlag := flag.String("digitalocean-api-token", "", "DigitalOcean API token")
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

	var services []string
	if *serviceFlag != "" {
		services = strings.Split(*serviceFlag, ",")
	}

	var token string
	var err error

	// Attempt to retrieve the last used team's token from the keyring.
	teamUUID, teamErr := webapp.GetLastUsedTeamUUID()
	if teamErr == nil {
		// If a team UUID was found, try to get the token for it.
		token, err = webapp.GetTokenFromKeyring(teamUUID)
	}

	// If no token was found (either because the team didn't exist, the token didn't exist,
	// or another keyring error occurred), start the authorization flow.
	if token == "" || err != nil {
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			// Log if the error was something other than just "not found".
			slog.Warn("Could not retrieve token from keyring", "error", err)
		}

		slog.Info("Starting authorization flow.")
		token, err = webapp.LocalhostAuthorize()
		if err != nil {
			slog.Error("Authorization failed", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Successfully configured DigitalOcean token.")

	client := godo.NewFromToken(token)
	s := server.NewMCPServer(mcpName, mcpVersion)

	err := registry.Register(logger, s, client, services...)
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
