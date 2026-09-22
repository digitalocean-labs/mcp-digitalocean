package functions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
func TestStructuredOutputSatisfiesDeclaredSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockFunctionsService(ctrl)
	mock.EXPECT().
		ListNamespaces(gomock.Any()).
		Return([]godo.FunctionsNamespace{
			{Namespace: "ns-1", Label: "test-1", Region: "nyc1", ApiHost: "https://faas.example.com"},
		}, nil, nil).
		Times(1)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(setupNamespaceTool(mock).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"functions-list-namespaces","arguments":{}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent struct {
				Namespaces []godo.FunctionsNamespace `json:"namespaces"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is an array, so it is published under an envelope to keep
	// structuredContent a JSON object.
	require.Len(t, resp.Result.StructuredContent.Namespaces, 1)
	require.Equal(t, "ns-1", resp.Result.StructuredContent.Namespaces[0].Namespace)

	// The text block is retained for backwards compatibility and carries the
	// bare payload, without the envelope.
	require.Len(t, resp.Result.Content, 1)
	var fromText []godo.FunctionsNamespace
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, resp.Result.StructuredContent.Namespaces, fromText)
}
