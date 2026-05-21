package openapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestExecute_truncatesLargeJSONBody(t *testing.T) {
	huge := strings.Repeat("a", 2_200_000)
	payload := `{"wrap":"` + huge + `"}`
	client := newTestGodoClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	})

	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return client, nil },
		api:       NewOpenAPIClient(),
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"OperationID": "droplets_list",
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, txt, "[truncated:")
	got := requireExecuteStructured(t, res)
	require.True(t, got.Truncated)
	require.False(t, got.SelectNotApplied)
}

func TestExecute_selectPlainText_skipped(t *testing.T) {
	client := newTestGodoClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	tool := &OpenAPITool{
		getClient: func(context.Context) (*godo.Client, error) { return client, nil },
		api:       NewOpenAPIClient(),
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"OperationID": "droplets_list",
		"Select":      "oops",
	}}}
	res, err := tool.execute(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)
	txt := res.Content[0].(mcp.TextContent).Text
	require.Contains(t, txt, "select-not-applied")
	require.Contains(t, txt, "hello")
	got := requireExecuteStructured(t, res)
	require.True(t, got.SelectNotApplied)
	require.Equal(t, "hello", got.Body)
}
