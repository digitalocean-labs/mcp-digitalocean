package droplet

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mcp-digitalocean/pkg/registry/common"
)

const (
	defaultSizesPageSize = 50
	defaultSizesPage     = 1
)

// SizesTool provides tool-based handlers for DigitalOcean droplet sizes.
type SizesTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewSizesTool creates a new SizesTool instance.
func NewSizesTool(client func(ctx context.Context) (*godo.Client, error)) *SizesTool {
	return &SizesTool{client: client}
}

// listSizes lists all available droplet sizes with pagination support.
func (s *SizesTool) listSizes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultSizesPage
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultSizesPageSize
	}

	opt := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	sizes, _, err := client.Sizes.List(ctx, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	summaries := make([]sizeSummary, len(sizes))
	for i, size := range sizes {
		summaries[i] = newSizeSummary(size)
	}

	return sizeSummariesOut.Result(summaries)
}

// Tools returns the list of server tools for droplet sizes.
func (s *SizesTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.listSizes,
			Tool: mcp.NewTool(
				"size-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				sizeSummariesOut.Schema(),
				mcp.WithDescription("List all available droplet sizes. Supports pagination."),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultSizesPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultSizesPageSize), mcp.Description("Items per page")),
			),
		},
	}
}
