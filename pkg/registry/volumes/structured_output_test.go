package volumes

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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
// volume-list is the tool exercised here because it is the one with a
// hand-written projection: its payload is the volumeSummary shape rather than a
// godo resource, so the schema and the projection could drift independently.
//
// Validation is enabled only here, not in cmd/mcp-digitalocean: in production
// a schema mismatch should degrade to a slightly-wrong schema, not a failed
// tool call. This test is what keeps the mismatch from reaching production.
func TestStructuredOutputSatisfiesDeclaredSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := NewMockStorageService(ctrl)
	mockStorage.EXPECT().
		ListVolumes(gomock.Any(), gomock.Any()).
		Return([]godo.Volume{{
			ID:            "vol-1",
			Name:          "test-volume",
			SizeGigaBytes: 10,
			Region:        &godo.Region{Slug: "nyc3", Name: "New York 3"},
			CreatedAt:     time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			DropletIDs:    []int{123},
			Tags:          []string{"prod"},
		}}, nil, nil).
		Times(1)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(setupVolumeToolWithMocks(mockStorage).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"volume-list","arguments":{}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent struct {
				Volumes []volumeSummary `json:"volumes"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is an array, which MCP cannot accept at the root, so it is
	// published under the "volumes" envelope.
	require.Len(t, resp.Result.StructuredContent.Volumes, 1)
	require.Equal(t, "vol-1", resp.Result.StructuredContent.Volumes[0].ID)
	require.Equal(t, int64(10), resp.Result.StructuredContent.Volumes[0].SizeGigaBytes)

	// The text block is retained for backwards compatibility and carries the
	// payload bare, without the envelope.
	require.Len(t, resp.Result.Content, 1)
	var fromText []volumeSummary
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, resp.Result.StructuredContent.Volumes, fromText)
}
