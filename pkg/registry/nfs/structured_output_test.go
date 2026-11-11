package nfs

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

	mockNfs := NewMockNfsService(ctrl)
	mockNfs.EXPECT().
		List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}, "nyc1").
		Return([]*godo.Nfs{{
			ID:              "nfs-1",
			Name:            "share-a",
			SizeGib:         50,
			Region:          "nyc1",
			Status:          godo.NfsShareActive,
			CreatedAt:       "2024-01-01T00:00:00Z",
			VpcIDs:          []string{"vpc-1"},
			Host:            "10.0.0.1",
			MountPath:       "/mnt/share-a",
			PerformanceTier: "standard",
		}}, nil, nil).
		Times(1)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(setupNfsToolWithMocks(mockNfs).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"nfs-file-share-list","arguments":{"Region":"nyc1"}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent struct {
				Shares []fileShareSummary `json:"shares"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is an array, which MCP cannot accept at the root, so it is
	// published under the "shares" envelope.
	require.Len(t, resp.Result.StructuredContent.Shares, 1)
	require.Equal(t, "nfs-1", resp.Result.StructuredContent.Shares[0].ID)
	require.Equal(t, "/mnt/share-a", resp.Result.StructuredContent.Shares[0].MountPath)

	// The text block is retained for backwards compatibility and carries the
	// bare payload, without the envelope.
	require.Len(t, resp.Result.Content, 1)
	var fromText []fileShareSummary
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, resp.Result.StructuredContent.Shares, fromText)
}
