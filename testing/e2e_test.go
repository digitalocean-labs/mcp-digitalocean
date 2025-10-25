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
		container, err := startMcpServer(ctx)
		if err != nil {
			fmt.Printf("Could not start MCP server container: %s\n", err)
		}
		defer container.Terminate(ctx)
		port, err := container.MappedPort(ctx, "8080/tcp")
		if err != nil {
			log.Fatalf("Could not get mapped port: %v", err)
		}
		mcpPort = port.Port()
		mcpServerURL = fmt.Sprintf("http://localhost:%s/mcp", mcpPort)
	} else {
		fmt.Println("Using existing MCP server at:", mcpServerURL)
	}

	code := m.Run()
	os.Exit(code)
}

func startMcpServer(ctx context.Context) (container testcontainers.Container, err error) {
	dockerfilePath := filepath.Join("..", "Dockerfile")
	buildCtx := filepath.Dir(dockerfilePath)

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    buildCtx,
			Dockerfile: "Dockerfile",
		},
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"BIND_ADDR":              "0.0.0.0:8080",
			"DIGITALOCEAN_API_TOKEN": apiToken,
			"LOG_LEVEL":              "debug",
			"TRANSPORT":              "http",
		},
		WaitingFor: wait.ForListeningPort("8080/tcp").WithStartupTimeout(60 * time.Second), // 60s
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
