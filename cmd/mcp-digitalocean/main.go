package main

import (
	"log/slog"
	registry "mcp-digitalocean/internal"
	"os"

	"github.com/digitalocean/godo"

	"github.com/mark3labs/mcp-go/server"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "0.1.0"
)

func main() {
	// Read OAUTH token from environment
	token := os.Getenv("DO_TOKEN")
	if token == "" {
		slog.Error("DO_TOKEN environment variable is not set")
		os.Exit(1)
	}

	client := godo.NewFromToken(token)
	s := server.NewMCPServer(mcpName, mcpVersion)

	// Register the tools and resources
	registry.RegisterTools(s, client)
	registry.RegisterResources(s, client)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

}
