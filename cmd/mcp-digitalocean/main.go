package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"strings"

	registry "mcp-digitalocean/internal"
	"mcp-digitalocean/internal/webapp/oauth"

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

	var services []string
	if *serviceFlag != "" {
		services = strings.Split(*serviceFlag, ",")
	}

	var token string
	var err error
	// Attempt to retrieve the last used token's team UUID from the keyring.
	teamUUID, teamErr := oauth.GetLastUsedTeamUUID()
	if teamErr == nil {
		// If a team UUID was found, try to get the token for it.
		token, err = oauth.GetTokenFromKeyring(teamUUID)
	}
	// If no token was found (either because the team didn't exist, the token
	// didn't exist, or another keyring error occurred), start the authorization
	// flow.
	if token == "" || err != nil {
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			// Log if the error was something other than just "not found".
			logger.Warn("Could not retrieve token from keyring: " + err.Error())
		}
		// Open browser with localhost redirect to aquire OAuth token
		logger.Info("Starting authorization flow")
		token, err = oauth.LocalhostAuthorize()
		if err != nil {
			logger.Error("DigitalOcean OAuth authorization failed: " + err.Error())
			os.Exit(1)
		}
	}
	logger.Info("Successfully configured DigitalOcean API token.")

	client := godo.NewFromToken(token)
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
