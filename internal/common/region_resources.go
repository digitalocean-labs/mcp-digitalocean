package common

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const RegionsURI = "regions://"

type RegionMCPResource struct {
	client *godo.Client
}

func NewRegionMCPResource(client *godo.Client) *RegionMCPResource {
	return &RegionMCPResource{
		client: client,
	}
}

func (r *RegionMCPResource) getRegionsResource() mcp.Resource {
	return mcp.NewResource(
		RegionsURI+"all",
		"Regions",
		mcp.WithResourceDescription("Returns all available regions with features and droplet size availability"),
		mcp.WithMIMEType("application/json"),
	)
}

func (r *RegionMCPResource) handleGetRegionsResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	regions, _, err := r.client.Regions.List(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error fetching regions: %w", err)
	}

	jsonData, err := json.MarshalIndent(regions, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing regions: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (r *RegionMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		r.getRegionsResource(): r.handleGetRegionsResource,
	}
}
