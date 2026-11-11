package vectordb

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

	mockSvc := NewMockVectorDBsService(ctrl)
	mockSvc.EXPECT().
		List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
		Return([]godo.VectorDB{{
			ID:     "vdb-1",
			Name:   "db-a",
			Region: "tor1",
			Size:   "small",
			Status: "active",
			Endpoints: &godo.VectorDBEndpoints{
				HTTP: "https://vdb-1.weaviate.tor1.do.com",
				GRPC: "grpc://vdb-1.weaviate.tor1.do.com:50051",
			},
		}}, nil, nil).
		Times(1)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(setupVectorDBToolWithMocks(mockSvc).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"vector-db-list","arguments":{}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent struct {
				VectorDBs []vectorDBSummary `json:"vector_dbs"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is an array, so it is published under the "vector_dbs"
	// envelope that keeps structuredContent a JSON object.
	require.Len(t, resp.Result.StructuredContent.VectorDBs, 1)
	require.Equal(t, "vdb-1", resp.Result.StructuredContent.VectorDBs[0].ID)
	require.Equal(t, "https://vdb-1.weaviate.tor1.do.com", resp.Result.StructuredContent.VectorDBs[0].EndpointHTTP)

	// The text block is retained for backwards compatibility and carries the
	// same payload, unwrapped.
	require.Len(t, resp.Result.Content, 1)
	var fromText []vectorDBSummary
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, resp.Result.StructuredContent.VectorDBs, fromText)
}
