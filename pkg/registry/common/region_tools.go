package common

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultRegionsPageSize = 50
	defaultRegionsPage     = 1
)

// RegionTools provides tool-based handlers for DigitalOcean regions.
type RegionTools struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewRegionTools creates a new RegionTools instance.
func NewRegionTools(client func(ctx context.Context) (*godo.Client, error)) *RegionTools {
	return &RegionTools{client: client}
}

// listRegions lists all available regions with pagination support.
func (r *RegionTools) listRegions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultRegionsPage
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultRegionsPageSize
	}

	opt := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	regions, _, err := client.Regions.List(ctx, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := json.MarshalIndent(regions, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for regions.
func (r *RegionTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: r.listRegions,
			Tool: mcp.NewTool(
				"region-list",
				mcp.WithDescription("List all available regions with features and droplet size availability. Supports pagination."),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultRegionsPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultRegionsPageSize), mcp.Description("Items per page")),
			),
		},
	}
}
