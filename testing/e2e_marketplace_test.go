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
)

// TestListOneClickApps tests listing 1-click applications from the marketplace
func TestListOneClickApps(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	t.Run("list droplet 1-click apps", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "1-click-list",
				Arguments: map[string]interface{}{},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		var result map[string]interface{}
		resultJSON := resp.Content[0].(mcp.TextContent).Text
		err = json.Unmarshal([]byte(resultJSON), &result)
		require.NoError(t, err)

		require.Contains(t, result, "apps")
		require.Contains(t, result, "type")
		require.Equal(t, "droplet", result["type"])

		apps, ok := result["apps"].([]interface{})
		require.True(t, ok, "apps should be an array")
		require.NotEmpty(t, apps, "should have at least one droplet 1-click app")

		t.Logf("Found %d droplet 1-click apps", len(apps))
	})

	t.Run("list kubernetes 1-click apps", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "1-click-list",
				Arguments: map[string]interface{}{
					"Type": "kubernetes",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		var result map[string]interface{}
		resultJSON := resp.Content[0].(mcp.TextContent).Text
		err = json.Unmarshal([]byte(resultJSON), &result)
		require.NoError(t, err)

		require.Contains(t, result, "apps")
		require.Contains(t, result, "type")
		require.Equal(t, "kubernetes", result["type"])

		apps, ok := result["apps"].([]interface{})
		require.True(t, ok, "apps should be an array")
		require.NotEmpty(t, apps, "should have at least one kubernetes 1-click app")

		t.Logf("Found %d kubernetes 1-click apps", len(apps))
	})
}

// TestInstallKubernetesApps tests installing single and multiple 1-click apps on a Kubernetes cluster
func TestInstallKubernetesApps(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	shortID := fmt.Sprintf("%08x", uuid.New().ID())
	clusterName := fmt.Sprintf("mcp-marketplace-test-%s", shortID)

	t.Logf("Creating test Kubernetes cluster: %s", clusterName)

	createClusterReq := godo.KubernetesClusterCreateRequest{
		Name:        clusterName,
		RegionSlug:  "sfo3",
		VersionSlug: "latest",
		NodePools: []*godo.KubernetesNodePoolCreateRequest{
			{
				Name:     "mcp-node-pool",
				Size:     "s-1vcpu-2gb",
				Count:    1,
				MinNodes: 1,
				MaxNodes: 1,
			},
		},
	}

	createResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "doks-create-cluster",
			Arguments: createClusterReq,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, createResp)
	require.False(t, createResp.IsError, "Failed to create cluster: %v", createResp.Content)

	var cluster godo.KubernetesCluster
	clusterJSON := createResp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(clusterJSON), &cluster)
	require.NoError(t, err)
	require.NotEmpty(t, cluster.ID)

	t.Logf("Created Kubernetes cluster: %s (ID: %s)", cluster.Name, cluster.ID)

	defer func() {
		t.Logf("Cleaning up Kubernetes cluster: %s", cluster.ID)
		deleteResp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "doks-delete-cluster",
				Arguments: map[string]interface{}{
					"ClusterID": cluster.ID,
				},
			},
		})

		if err != nil || (deleteResp != nil && deleteResp.IsError) {
			t.Logf("Warning: Failed to delete cluster %s: %v", cluster.ID, err)
		} else {
			t.Logf("Successfully deleted cluster: %s", cluster.ID)
		}
	}()

	listResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "1-click-list",
			Arguments: map[string]interface{}{
				"Type": "kubernetes",
			},
		},
	})

	require.NoError(t, err)
	require.False(t, listResp.IsError)

	var listResult map[string]interface{}
	listJSON := listResp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(listJSON), &listResult)
	require.NoError(t, err)

	apps, ok := listResult["apps"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, apps, "No Kubernetes 1-click apps available")

	t.Run("install single app", func(t *testing.T) {
		firstApp := apps[0].(map[string]interface{})
		appSlug := firstApp["slug"].(string)
		require.NotEmpty(t, appSlug)

		t.Logf("Installing 1-click app: %s on cluster %s", appSlug, cluster.ID)

		installResp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "1-click-kubernetes-app-install",
				Arguments: map[string]interface{}{
					"ClusterUUID": cluster.ID,
					"AppSlugs":    []string{appSlug},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, installResp)
		require.False(t, installResp.IsError, "Failed to install app: %v", installResp.Content)

		var installResult map[string]interface{}
		installJSON := installResp.Content[0].(mcp.TextContent).Text
		err = json.Unmarshal([]byte(installJSON), &installResult)
		require.NoError(t, err)

		t.Logf("Successfully installed app %s: %v", appSlug, installResult)
	})

	t.Run("install multiple apps", func(t *testing.T) {
		appSlugs := make([]string, 0, 2)
		for i := 0; i < len(apps) && i < 2; i++ {
			app := apps[i].(map[string]interface{})
			slug := app["slug"].(string)
			if slug != "" {
				appSlugs = append(appSlugs, slug)
			}
		}

		require.NotEmpty(t, appSlugs, "No valid app slugs found")

		t.Logf("Installing %d 1-click apps: %v on cluster %s", len(appSlugs), appSlugs, cluster.ID)

		installResp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "1-click-kubernetes-app-install",
				Arguments: map[string]interface{}{
					"ClusterUUID": cluster.ID,
					"AppSlugs":    appSlugs,
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, installResp)
		require.False(t, installResp.IsError, "Failed to install apps: %v", installResp.Content)

		var installResult map[string]interface{}
		installJSON := installResp.Content[0].(mcp.TextContent).Text
		err = json.Unmarshal([]byte(installJSON), &installResult)
		require.NoError(t, err)

		t.Logf("Successfully installed %d apps: %v", len(appSlugs), installResult)
	})
}

// TestInstallKubernetesApps_InvalidInputs tests error handling for invalid inputs
func TestInstallKubernetesApps_InvalidInputs(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	t.Run("missing cluster UUID", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "1-click-kubernetes-app-install",
				Arguments: map[string]interface{}{
					"AppSlugs": []string{"monitoring"},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for missing ClusterUUID")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "ClusterUUID parameter is required")
		t.Logf("Got expected error: %s", errorText)
	})

	t.Run("missing app slugs", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "1-click-kubernetes-app-install",
				Arguments: map[string]interface{}{
					"ClusterUUID": "test-cluster-uuid",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for missing AppSlugs")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "AppSlugs parameter is required")
		t.Logf("Got expected error: %s", errorText)
	})

	t.Run("empty cluster UUID", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "1-click-kubernetes-app-install",
				Arguments: map[string]interface{}{
					"ClusterUUID": "",
					"AppSlugs":    []string{"monitoring"},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for empty ClusterUUID")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "ClusterUUID cannot be empty")
		t.Logf("Got expected error: %s", errorText)
	})

	t.Run("empty app slugs array", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "1-click-kubernetes-app-install",
				Arguments: map[string]interface{}{
					"ClusterUUID": "test-cluster-uuid",
					"AppSlugs":    []string{},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for empty AppSlugs")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "AppSlugs cannot be empty")
		t.Logf("Got expected error: %s", errorText)
	})

	t.Run("invalid cluster UUID", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "1-click-kubernetes-app-install",
				Arguments: map[string]interface{}{
					"ClusterUUID": "invalid-cluster-uuid-that-does-not-exist",
					"AppSlugs":    []string{"monitoring"},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for invalid ClusterUUID")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "Failed to install Kubernetes apps")
		t.Logf("Got expected error: %s", errorText)
	})
}
