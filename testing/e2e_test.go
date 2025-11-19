//go:build integration

package testing

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	mcpPort      string
	apiToken     string
	mcpServerURL string
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	apiToken = os.Getenv("DIGITALOCEAN_API_TOKEN")
	mcpServerURL = os.Getenv("MCP_SERVER_URL")

	// If the user doesn't provide an MCP Server URL, start a new mcp server from a container.
	if mcpServerURL == "" {
		cfg := McpServerConfig{
			BindAddr:             "0.0.0.0:8080",
			DigitalOceanAPIToken: apiToken,
			LogLevel:             "debug",
			Transport:            "http",
		}
		container, err := startMcpServer(ctx, cfg)
		if err != nil {
			fmt.Printf("Could not start MCP server container: %s\n", err)
		}
		defer container.Terminate(ctx)
		port, err := container.MappedPort(ctx, "8080/tcp")
		if err != nil {
			log.Fatalf("Could not get mapped port: %v", err)
		}
		mcpPort = port.Port()
		mcpServerURL = fmt.Sprintf("http://127.0.0.1:%s/mcp", mcpPort)
	} else {
		fmt.Println("Using existing MCP server at:", mcpServerURL)
	}

	code := m.Run()
	os.Exit(code)
}

// McpServerConfig holds configuration for starting an MCP server container
type McpServerConfig struct {
	BindAddr             string
	DigitalOceanAPIToken string
	LogLevel             string
	Transport            string
	Services             string // optional: comma-separated list of services
	WSLoggingURL         string // optional
	WSLoggingToken       string // optional
}

// ToMap converts the config to a map of environment variables
func (cfg McpServerConfig) ToMap() map[string]string {
	env := map[string]string{
		"BIND_ADDR":                 cfg.BindAddr,
		"DIGITALOCEAN_API_TOKEN":    cfg.DigitalOceanAPIToken,
		"LOG_LEVEL":                 cfg.LogLevel,
		"TRANSPORT":                 cfg.Transport,
		"ENABLE_TOOL_ERROR_LOGGING": "true",
	}

	// add optional services configuration if provided
	if cfg.Services != "" {
		env["SERVICES"] = cfg.Services
	}

	// add optional WebSocket logging configuration if provided
	if cfg.WSLoggingURL != "" {
		env["WS_LOGGING_URL"] = cfg.WSLoggingURL
	}
	if cfg.WSLoggingToken != "" {
		env["WS_LOGGING_TOKEN"] = cfg.WSLoggingToken
	}

	return env
}

func startMcpServer(ctx context.Context, cfg McpServerConfig) (container testcontainers.Container, err error) {
	dockerfilePath := filepath.Join("..", "Dockerfile")
	buildCtx := filepath.Dir(dockerfilePath)

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    buildCtx,
			Dockerfile: "Dockerfile",
			KeepImage:  true, // Keep the image after container stops to enable Docker layer caching
			Repo:       "mcp-digitalocean",
			Tag:        "latest",
		},
		ExposedPorts: []string{"8080/tcp"},
		Env:          cfg.ToMap(),
		WaitingFor:   wait.ForListeningPort("8080/tcp").WithStartupTimeout(60 * time.Second), // 60s
		// Add host.docker.internal mapping for containers to reach the host
		// This is needed for WebSocket logging tests where the fake WS server runs on the host
		// Using "host-gateway" which Docker automatically resolves to the host's gateway IP
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func TestListTools(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	tools, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.NotEmpty(t, tools)
}

// initializeClient initializes and returns a new MCP client for testing.
func initializeClient(ctx context.Context, t *testing.T) *client.Client {
	c, err := newClient(
		mcpServerURL,
		transport.WithHTTPHeaders(map[string]string{"Authorization": fmt.Sprintf("Bearer %s", apiToken)}),
	)

	require.NoError(t, err)
	err = c.Start(ctx)
	require.NoError(t, err)

	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "test-client2",
				Version: "1.0.0",
			},
		},
	}

	_, err = c.Initialize(ctx, initRequest)
	require.NoError(t, err)

	return c
}

func newClient(baseURL string, options ...transport.StreamableHTTPCOption) (*client.Client, error) {
	trans, err := transport.NewStreamableHTTP(baseURL, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamable HTTP transport: %w", err)
	}
	clientOptions := make([]client.ClientOption, 0)

	return client.NewClient(trans, clientOptions...), nil
}
