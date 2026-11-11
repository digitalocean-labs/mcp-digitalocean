package common

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultRegionsPageSize = 50
	defaultRegionsPage     = 1
)

// regionsOut is the output contract for region-list. The payload is a bare
// array, so it takes the plural envelope that keeps structuredContent an
// object.
var regionsOut = NewOutput[[]godo.Region]("regions")

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

	return regionsOut.Result(regions)
}

// Tools returns the list of server tools for regions.
func (r *RegionTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: r.listRegions,
			Tool: mcp.NewTool("region-list",
				WithHints(HintsRead),
				WithRisk(RiskLow),
				regionsOut.Schema(),
				mcp.WithDescription("List all available regions with features and droplet size availability. Supports pagination."),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultRegionsPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultRegionsPageSize), mcp.Description("Items per page")),
			),
		},
	}
}
