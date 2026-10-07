//go:build integration

package testing

import (
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func skipIfSignalsUnavailable(t *testing.T, resp *mcp.CallToolResult) {
	t.Helper()
	if resp == nil || !resp.IsError {
		return
	}
	text := callToolResultText(resp)
	lower := strings.ToLower(text)
	if strings.Contains(lower, "403") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "not enabled") ||
		strings.Contains(lower, "flipper") ||
		strings.Contains(lower, "fi_signals") {
		t.Skipf("signals not enabled for this token: %s", text)
	}
}

func TestSignalsGetExportOptions(t *testing.T) {
	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "signals-get-export-options", Arguments: map[string]any{}},
	})
	require.NoError(t, err)
	skipIfSignalsUnavailable(t, resp)
	require.False(t, resp.IsError, callToolResultText(resp))
	require.NotEmpty(t, callToolResultText(resp))
}

func TestSignalsGetAgentConsent(t *testing.T) {
	agentID := os.Getenv("SIGNALS_AGENT_ID")
	if agentID == "" {
		t.Skip("SIGNALS_AGENT_ID not set")
	}
	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "signals-get-agent-consent",
			Arguments: map[string]any{"AgentID": agentID},
		},
	})
	require.NoError(t, err)
	skipIfSignalsUnavailable(t, resp)
	require.False(t, resp.IsError, callToolResultText(resp))
	text := callToolResultText(resp)
	require.Contains(t, text, agentID)
}
