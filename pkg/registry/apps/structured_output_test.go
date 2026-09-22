package apps

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
//
// apps-list is the tool under test because it is the one whose payload is a
// hand-written projection (AppSummary) rather than a godo type, so its schema
// and its payload are the most likely pair to drift apart.
func TestStructuredOutputSatisfiesDeclaredSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApps := NewMockAppsService(ctrl)
	mockApps.EXPECT().
		List(gomock.Any(), &godo.ListOptions{Page: defaultPage, PerPage: defaultPageSize}).
		Return([]*godo.App{{
			ID:       "app-1",
			Spec:     &godo.AppSpec{Name: "my-app"},
			LiveURL:  "https://my-app.ondigitalocean.app",
			Region:   &godo.AppRegion{Slug: "nyc"},
			TierSlug: "basic",
		}}, nil, nil).
		Times(1)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Apps: mockApps}, nil
	}
	appTool, err := NewAppPlatformTool(client)
	require.NoError(t, err)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(appTool.Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"apps-list","arguments":{}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent struct {
				Apps []*AppSummary `json:"apps"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is a bare array, so it is published under the "apps"
	// envelope that keeps structuredContent an object.
	require.Len(t, resp.Result.StructuredContent.Apps, 1)
	require.Equal(t, "app-1", resp.Result.StructuredContent.Apps[0].ID)
	require.Equal(t, "my-app", resp.Result.StructuredContent.Apps[0].Name)

	// The text block is retained for backwards compatibility and carries the
	// unwrapped payload.
	require.Len(t, resp.Result.Content, 1)
	var fromText []*AppSummary
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, resp.Result.StructuredContent.Apps, fromText)
}
