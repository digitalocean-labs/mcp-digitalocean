//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
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
					GitHub: &godo.GitHubSourceSpec{
						Repo:   "digitalocean/sample-golang",
						Branch: "main",
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
	require.False(t, resp.IsError)

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
	require.False(t, status.IsError)

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
	require.False(t, resp.IsError)

	t.Logf("deleted app: %v", app.ID)
}
