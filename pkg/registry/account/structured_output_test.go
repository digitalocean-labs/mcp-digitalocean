package account

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

	mockActions := NewMockActionsService(ctrl)
	// started_at/completed_at are left nil on purpose: godo.Timestamp has no
	// omitempty, so it serialises to null and the schema has to admit it.
	mockActions.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return([]godo.Action{
			{ID: 1, Status: "completed", Type: "create", ResourceID: 42, ResourceType: "droplet", RegionSlug: "blr1"},
		}, nil, nil).
		Times(1)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(setupActionToolsWithMock(mockActions).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"action-list","arguments":{}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool                       `json:"isError"`
			Content           []mcp.TextContent          `json:"content"`
			StructuredContent map[string]json.RawMessage `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// structuredContent carries the payload under the declared field.
	require.Contains(t, resp.Result.StructuredContent, actionsOut.Field())
	var actions []godo.Action
	require.NoError(t, json.Unmarshal(resp.Result.StructuredContent[actionsOut.Field()], &actions))
	require.Len(t, actions, 1)
	require.Equal(t, 1, actions[0].ID)

	// The text block is retained for backwards compatibility, and still holds
	// the bare payload rather than the envelope.
	require.Len(t, resp.Result.Content, 1)
	var fromText []godo.Action
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, actions, fromText)
}
