//go:build integration

package testing

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// skipIfHarnessRuntimeUnavailable skips when the Harness Runtime API is not
// reachable or not enabled for the test token.
func skipIfHarnessRuntimeUnavailable(t *testing.T, resp *mcp.CallToolResult) {
	t.Helper()
	if resp == nil || !resp.IsError {
		return
	}
	text := callToolResultText(resp)
	lower := strings.ToLower(text)
	if strings.Contains(lower, "403") ||
		strings.Contains(lower, "404") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "not enabled") {
		t.Skipf("harness runtime not available for this token: %s", text)
	}
}

func TestHarnessRuntimeAgentConfigs(t *testing.T) {
	ctx, c := getTestClient(t)

	listResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "harness-runtime-list-agent-configs",
			Arguments: map[string]any{"PageSize": 5},
		},
	})
	require.NoError(t, err)
	skipIfHarnessRuntimeUnavailable(t, listResp)
	require.False(t, listResp.IsError, callToolResultText(listResp))

	var list struct {
		Configs []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"configs"`
	}
	require.NoError(t, json.Unmarshal([]byte(callToolResultText(listResp)), &list))
	if len(list.Configs) == 0 {
		t.Skip("no agent configs in this team; skipping get")
	}

	first := list.Configs[0]
	getResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "harness-runtime-get-agent-config",
			Arguments: map[string]any{"ConfigID": first.ID},
		},
	})
	require.NoError(t, err)
	skipIfHarnessRuntimeUnavailable(t, getResp)
	require.False(t, getResp.IsError, callToolResultText(getResp))
	require.Contains(t, callToolResultText(getResp), first.ID)
}

func TestHarnessRuntimeGetAgentConfigInvalidID(t *testing.T) {
	ctx, c := getTestClient(t)

	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "harness-runtime-get-agent-config",
			Arguments: map[string]any{"ConfigID": "not-a-uuid"},
		},
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
}
