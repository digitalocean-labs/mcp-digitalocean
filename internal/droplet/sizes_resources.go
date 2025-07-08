package droplet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const SizesURI = "sizes://"

type SizesMCPResource struct {
	client *godo.Client
}

func NewSizesMCPResource(client *godo.Client) *SizesMCPResource {
	return &SizesMCPResource{
		client: client,
	}
}

// GetResource returns the resource for all droplet sizes
func (s *SizesMCPResource) getSizeResource() mcp.Resource {
	return mcp.NewResource(
		SizesURI+"all",
		"Droplet Sizes",
		mcp.WithResourceDescription("Returns all available droplet sizes"),
		mcp.WithMIMEType("application/json"),
	)
}

// HandleGetResource handles the Sizes MCP resource requests for all sizes
func (s *SizesMCPResource) handleGetSizeResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	sizes, _, err := s.client.Sizes.List(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error fetching droplet sizes: %w", err)
	}

	filteredSizes := make([]map[string]any, len(sizes))
	for i, size := range sizes {
		filteredSizes[i] = map[string]any{
			"slug":          size.Slug,
			"available":     size.Available,
			"price_monthly": size.PriceMonthly,
			"price_hourly":  size.PriceHourly,
		}
	}

	jsonData, err := json.MarshalIndent(filteredSizes, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing sizes: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// Resources returns the available resources for the Sizes MCP resource
func (s *SizesMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		s.getSizeResource(): s.handleGetSizeResource,
	}
}
