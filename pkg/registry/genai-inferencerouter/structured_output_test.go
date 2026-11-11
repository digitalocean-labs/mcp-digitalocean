package genaiinferencerouter

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

// TestStructuredOutputSatisfiesDeclaredSchema drives a migrated tool through a
// real MCPServer with WithOutputSchemaValidation enabled, so the server itself
// checks the emitted structuredContent against the outputSchema the tool
// advertises. Unit tests on the handler cannot catch a schema that the payload
// does not satisfy; only the server (or a spec-compliant client) validates.
//
// Validation is enabled only here, not in cmd/mcp-digitalocean: in production
// a schema mismatch should degrade to a slightly-wrong schema, not a failed
// tool call. This test is what keeps the mismatch from reaching production.
//
// genai-inference-router-get is the representative tool: it returns the
// "model_router" envelope shared by create and update, and its payload covers
// the awkward corners of the godo type — a *Timestamp and a json.RawMessage
// policies block.
func TestStructuredOutputSatisfiesDeclaredSchema(t *testing.T) {
	c := testGodoClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model_router":{"uuid":"11f117f7-d076-6272-b542-ca68c578b04b",` +
			`"name":"alex-test-2","created_at":"2024-01-02T03:04:05Z",` +
			`"config":{"fallback_models":["llama3.3-70b-instruct"],` +
			`"policies":[{"task_slug":"translation","models":["openai-gpt-5"]}]}}}`))
	}))

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(NewRouterTool(func(context.Context) (*godo.Client, error) { return c, nil }).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"genai-inference-router-get",`+
			`"arguments":{"UUID":"11f117f7-d076-6272-b542-ca68c578b04b"}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent json.RawMessage   `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is already an object, so it is published without an
	// envelope: structuredContent is the payload itself.
	var structured routerEnvelope
	require.NoError(t, json.Unmarshal(resp.Result.StructuredContent, &structured))
	require.NotNil(t, structured.ModelRouter)
	require.Equal(t, "11f117f7-d076-6272-b542-ca68c578b04b", structured.ModelRouter.UUID)
	require.Equal(t, "alex-test-2", structured.ModelRouter.Name)
	require.Equal(t, []string{"llama3.3-70b-instruct"}, structured.ModelRouter.Config.FallbackModels)

	// The text block is retained for backwards compatibility and carries the
	// same payload. Compared as JSON because the two differ in whitespace:
	// text is indented, while structuredContent is compacted on the wire.
	require.Len(t, resp.Result.Content, 1)
	require.JSONEq(t, resp.Result.Content[0].Text, string(resp.Result.StructuredContent))
}
