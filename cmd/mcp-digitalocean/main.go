package main

import (
	"flag"
	"log/slog"
	registry "mcp-digitalocean/internal"
	"net"
	"net/http"
	"os"

	"github.com/digitalocean/godo"

	"github.com/mark3labs/mcp-go/server"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "0.1.0"
)

func main() {
	// Command-line flags
	var (
		stdio    = flag.Bool("stdio", false, "Run server in stdio mode")
		httpAddr = flag.String("http", ":8080", "HTTP bind address (ignored if --unix is set)")
		unixSock = flag.String("unix", "", "Path to UNIX socket (if set, takes precedence over --http)")
		baseUrl  = flag.String("base-url", "http://localhost:8080", "Base URL for the server (optional)")
	)
	flag.Parse()

	// Default to stdio if no flags are given
	if len(os.Args) == 1 {
		*stdio = true
	}

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

	if *stdio {
		slog.Info("Starting MCP server in stdio mode")
		// Start the stdio server
		if err := server.ServeStdio(s); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
	} else {
		slog.Info("Starting MCP server in HTTP mode")
		sseServer := server.NewSSEServer(s, server.WithBaseURL(*baseUrl))

		// Start on UNIX socket if path to bind on given, otherwise start on address and port
		if *unixSock != "" {
			listener, err := net.Listen("unix", *unixSock)
			if err != nil {
				slog.Error("Failed to listen on UNIX socket", "error", err)
				os.Exit(1)
			}
			defer listener.Close()
			slog.Info("Listening on UNIX socket", "path", *unixSock)
			if err := http.Serve(listener, sseServer); err != nil {
				slog.Error("HTTP server error", "error", err)
				os.Exit(1)
			}
		} else {
			slog.Info("Listening on HTTP address", "address", *httpAddr)
			if err := http.ListenAndServe(*httpAddr, sseServer); err != nil {
				slog.Error("HTTP server error", "error", err)
				os.Exit(1)
			}
		}
	}

}
