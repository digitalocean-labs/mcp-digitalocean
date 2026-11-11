package dedicatedinference

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

	mock := NewMockDedicatedInferenceService(ctrl)
	mock.EXPECT().
		Get(gomock.Any(), "di-uuid-1234").
		Return(testDI, &godo.Response{}, nil).
		Times(1)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(setupToolWithMock(mock).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"dedicated-inference-get",`+
			`"arguments":{"DedicatedInferenceID":"di-uuid-1234"}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent struct {
				DedicatedInference *godo.DedicatedInference `json:"dedicated_inference"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is a bare resource, so it is published under the
	// "dedicated_inference" envelope that keeps structuredContent an object.
	require.NotNil(t, resp.Result.StructuredContent.DedicatedInference)
	require.Equal(t, testDI.ID, resp.Result.StructuredContent.DedicatedInference.ID)
	require.Equal(t, testDI.Name, resp.Result.StructuredContent.DedicatedInference.Name)

	// The text block is retained for backwards compatibility and carries the
	// unwrapped payload.
	require.Len(t, resp.Result.Content, 1)
	var fromText godo.DedicatedInference
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, *resp.Result.StructuredContent.DedicatedInference, fromText)
}
