//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"

	"mcp-digitalocean/pkg/registry/apps"
)

func TestCreateApp(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	// Generate a short unique ID for the app name
	shortID := fmt.Sprintf("%08x", uuid.New().ID())
	create := godo.AppCreateRequest{
		Spec: &godo.AppSpec{
			Name: fmt.Sprintf("mcp-%s", shortID),
			Services: []*godo.AppServiceSpec{
				{
					Name: "sample-golang",
					Git: &godo.GitSourceSpec{
						RepoCloneURL: "https://github.com/digitalocean/sample-golang.git",
						Branch:       "main",
					},
				},
			},
		},
	}

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "apps-create-app-from-spec",
			Arguments: create,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "apps-create-app-from-spec returned an error: %s", toolResultText(resp))

	var app godo.App
	appJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(appJSON), &app)
	require.NoError(t, err)
	require.NotEmpty(t, app.ID)

	t.Logf("Created app: %s\n", app.Spec.Name)

	// Get the app deployment status
	status, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "apps-get-deployment-status",
			Arguments: map[string]interface{}{
				"AppID": app.ID,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	require.False(t, status.IsError, "apps-get-deployment-status returned an error: %s", toolResultText(status))

	// unmarshall the response to an app deployment
	var deployment apps.DeploymentStatus
	deploymentJSON := status.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(deploymentJSON), &deployment)
	require.NoError(t, err)
	t.Logf("App deployment phase: %s", deployment.Deployment.Phase)

	// cleanup the app
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "apps-delete",
			Arguments: map[string]interface{}{
				"AppID": app.ID,
			},
		},
	})

	require.NoError(t, err)
	require.False(t, resp.IsError, "apps-delete returned an error: %s", toolResultText(resp))

	t.Logf("deleted app: %v", app.ID)
}

// toolResultText concatenates the text content of an MCP tool result, so a
// failing assertion on IsError can surface the actual tool/API error body.
func toolResultText(resp *mcp.CallToolResult) string {
	if resp == nil {
		return "<nil response>"
	}
	var b strings.Builder
	for _, content := range resp.Content {
		if tc, ok := content.(mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	if b.Len() == 0 {
		return "<no text content>"
	}
	return b.String()
}
