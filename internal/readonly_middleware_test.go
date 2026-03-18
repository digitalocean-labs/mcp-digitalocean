package middleware

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestReadOnlyMiddleware_AllowsReadOnlyTool(t *testing.T) {
	called := false
	next := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("ok"), nil
	}

	m := ReadOnlyMiddleware{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	handler := m.ToolMiddleware(next)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "apps-list"},
	})

	require.NoError(t, err)
	require.True(t, called)
	require.NotNil(t, result)
	require.False(t, result.IsError)
}

func TestReadOnlyMiddleware_BlocksMutatingTool(t *testing.T) {
	next := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("should not be called"), nil
	}

	m := ReadOnlyMiddleware{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	handler := m.ToolMiddleware(next)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "apps-delete"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError)
}

func TestReadOnlyMiddleware_BlocksMutatingToolWithActionFirst(t *testing.T) {
	next := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("should not be called"), nil
	}

	m := ReadOnlyMiddleware{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Tools where the action token is not the last segment
	mutatingToolNames := []string{
		"power-on-droplet",
		"power-off-droplet",
		"power-cycle-droplet",
		"reset-droplet-password",
		"shutdown-droplets-tag",
		"snapshot-droplets-tag",
		"enable-backups-droplet",
		"disable-backups-droplet",
		"change-kernel-droplet",
	}

	handler := m.ToolMiddleware(next)
	for _, toolName := range mutatingToolNames {
		result, err := handler(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{Name: toolName},
		})
		require.NoError(t, err, "tool: %s", toolName)
		require.NotNil(t, result, "tool: %s", toolName)
		require.True(t, result.IsError, "tool %q should be blocked in read-only mode", toolName)
	}
}

func TestReadOnlyMiddleware_UsesCustomClassifier(t *testing.T) {
	called := false
	next := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("ok"), nil
	}

	m := ReadOnlyMiddleware{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		isMutating: func(string) bool {
			return false
		},
	}

	handler := m.ToolMiddleware(next)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "apps-delete"},
	})

	require.NoError(t, err)
	require.True(t, called)
	require.NotNil(t, result)
	require.False(t, result.IsError)
}
