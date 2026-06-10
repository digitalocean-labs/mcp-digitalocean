package genaicustommodels

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestListModelsHasFilters(t *testing.T) {
	require.False(t, listModelsHasFilters(map[string]any{}))
	require.True(t, listModelsHasFilters(map[string]any{"status": "STATUS_READY"}))
	require.True(t, listModelsHasFilters(map[string]any{"page": float64(2)}))
	require.True(t, listModelsHasFilters(map[string]any{"per_page": float64(10)}))
}

func TestCustomModelsTool_listModels_unfilteredIncludesFailedUUID(t *testing.T) {
	tool := setupCustomModelsToolWithTestServer(t, []*CustomModel{
		{UUID: "ok-uuid", Name: "ready", Status: CustomModelStatusReady},
		{UUID: "bad-uuid", Name: "failed-one", Status: CustomModelStatusFailed, Architecture: "LlamaForCausalLM"},
	})

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	resp, err := tool.listModels(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)

	text := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, text, "| bad-uuid | failed-one | custom | STATUS_FAILED | LlamaForCausalLM |")
	require.NotContains(t, text, "## Model Catalog")
}

func TestCustomModelsTool_listModels_filteredReturnsTable(t *testing.T) {
	tool := setupCustomModelsToolWithTestServer(t, []*CustomModel{
		{
			UUID:         "uuid-1",
			Name:         "ready-model",
			Status:       CustomModelStatusReady,
			Architecture: "LlamaForCausalLM",
		},
	})

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"status": "STATUS_READY",
	}}}
	resp, err := tool.listModels(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)

	text := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, text, "## Custom Models (1 models)")
	require.Contains(t, text, "ready-model")
	require.Contains(t, text, "STATUS_READY")
	require.Contains(t, text, "LlamaForCausalLM")
}
