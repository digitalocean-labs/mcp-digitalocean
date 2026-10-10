package microdroplet

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

// TestStructuredOutputSatisfiesDeclaredSchema drives list through a real
// MCPServer with WithOutputSchemaValidation so structuredContent is checked
// against the declared outputSchema.
func TestStructuredOutputSatisfiesDeclaredSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/microvms", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"microvms":[{"id":"aaa-111","name":"one","region":"nyc3","state":"running"}],
			"links":{},
			"meta":{"total":1}
		}`))
	}))
	t.Cleanup(srv.Close)

	client, err := godo.New(http.DefaultClient, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	mcpServer := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	mcpServer.AddTools(NewMicroDropletTool(func(ctx context.Context) (*godo.Client, error) {
		return client, nil
	}).Tools()...)

	ctx := context.Background()
	mcpServer.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(mcpServer.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"microvm-list","arguments":{"page":1,"per_page":10}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool                 `json:"isError"`
			Content           []mcp.TextContent    `json:"content"`
			StructuredContent microVMsListResponse `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.Nil(t, resp.Error, "server rejected the call: %s", raw)
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	require.Len(t, resp.Result.StructuredContent.MicroVMs, 1)
	require.Equal(t, "aaa-111", resp.Result.StructuredContent.MicroVMs[0].ID)
	require.NotNil(t, resp.Result.StructuredContent.Meta)
	require.Equal(t, 1, resp.Result.StructuredContent.Meta.Total)

	require.Len(t, resp.Result.Content, 1)
	var fromText microVMsListResponse
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, resp.Result.StructuredContent.MicroVMs, fromText.MicroVMs)
}

func TestCreateStructuredContentIsLossless(t *testing.T) {
	body := `{"microvm":{"id":"9f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f","name":"agent-sandbox-1","state":"creating","extra_field":"kept"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	client, err := godo.New(http.DefaultClient, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	mcpServer := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	mcpServer.AddTools(NewMicroDropletTool(func(ctx context.Context) (*godo.Client, error) {
		return client, nil
	}).Tools()...)

	ctx := context.Background()
	mcpServer.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	args := `{"name":"agent-sandbox-1","region":"nyc1","size":{"cpu":2,"memory":4096},"source":{"oci_ref":"docker.io/library/nginx:1.27"}}`
	raw, err := json.Marshal(mcpServer.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"microvm-create","arguments":`+args+`}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool            `json:"isError"`
			StructuredContent json.RawMessage `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.Nil(t, resp.Error, "server rejected the call: %s", raw)
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)
	require.JSONEq(t, body, string(resp.Result.StructuredContent))
}
