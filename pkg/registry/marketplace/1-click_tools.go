package marketplace

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type OneClickTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewOneClickTool(client func(ctx context.Context) (*godo.Client, error)) *OneClickTool {
	return &OneClickTool{
		client: client,
	}
}

func (o *OneClickTool) listOneClickApps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Type parameter is optional, defaults to "droplet"
	oneClickType := "droplet"
	if typeArg, ok := args["Type"]; ok {
		if typeStr, ok := typeArg.(string); ok && typeStr != "" {
			oneClickType = typeStr
		}
	}

	client, err := o.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apps, _, err := client.OneClick.List(ctx, oneClickType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list 1-click apps: %v", err)), nil
	}

	result, err := json.Marshal(map[string]interface{}{
		"apps": apps,
		"type": oneClickType,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (o *OneClickTool) installKubernetesApps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Get cluster UUID
	clusterUUIDArg, ok := args["ClusterUUID"]
	if !ok {
		return mcp.NewToolResultError("ClusterUUID parameter is required"), nil
	}

	clusterUUID, ok := clusterUUIDArg.(string)
	if !ok {
		return mcp.NewToolResultError("ClusterUUID must be a string"), nil
	}

	if clusterUUID == "" {
		return mcp.NewToolResultError("ClusterUUID cannot be empty"), nil
	}

	// Get app slugs
	slugsArg, ok := args["AppSlugs"]
	if !ok {
		return mcp.NewToolResultError("AppSlugs parameter is required"), nil
	}

	slugsInterface, ok := slugsArg.([]interface{})
	if !ok {
		return mcp.NewToolResultError("AppSlugs must be an array"), nil
	}

	slugs := make([]string, len(slugsInterface))
	for i, slug := range slugsInterface {
		slugStr, ok := slug.(string)
		if !ok {
			return mcp.NewToolResultError("all AppSlugs must be strings"), nil
		}
		slugs[i] = slugStr
	}

	if len(slugs) == 0 {
		return mcp.NewToolResultError("AppSlugs cannot be empty"), nil
	}

	installRequest := &godo.InstallKubernetesAppsRequest{
		Slugs:       slugs,
		ClusterUUID: clusterUUID,
	}

	client, err := o.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	response, _, err := client.OneClick.InstallKubernetes(ctx, installRequest)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to install Kubernetes apps: %v", err)), nil
	}

	result, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (o *OneClickTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: o.listOneClickApps,
			Tool: mcp.NewTool("1-click-list",
				mcp.WithDescription("List available 1-click applications from the DigitalOcean marketplace"),
				mcp.WithString("Type", mcp.Description("Type of 1-click apps to list (e.g., 'droplet', 'kubernetes'). Defaults to 'droplet'")),
			),
		},
		{
			Handler: o.installKubernetesApps,
			Tool: mcp.NewTool("1-click-kubernetes-app-install",
				mcp.WithDescription("Install 1-click applications on a Kubernetes cluster"),
				mcp.WithString("ClusterUUID", mcp.Required(), mcp.Description("UUID of the Kubernetes cluster to install apps on")),
				mcp.WithArray("AppSlugs", mcp.Required(), mcp.Description("Array of app slugs to install")),
			),
		},
	}
}
