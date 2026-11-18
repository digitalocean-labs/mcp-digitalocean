package harnessruntime

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	regionsPath = "v2/harness-runtime/regions"
)

// Region is one entry of the per-team region catalog.
type Region struct {
	Slug      string `json:"slug"`
	IsDefault bool   `json:"is_default"`
}

// ListRegionsResponse is the response of GET /v2/harness-runtime/regions.
type ListRegionsResponse struct {
	Regions []Region `json:"regions"`
}

// listRegions returns the regions the caller's team may create sessions and agent configs in.
func (t *Tool) listRegions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, regionsPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListRegionsResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list regions", err), nil
	}
	if out.Regions == nil {
		out.Regions = []Region{}
	}
	return regionListOut.Result(out)
}

// regionTools returns the region MCP tools to be merged into Tools().
func (t *Tool) regionTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listRegions,
			Tool: mcp.NewTool("harness-runtime-list-regions",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				regionListOut.Schema(),
				mcp.WithDescription("List the regions the team may create agent sessions and agent configs in, sorted by slug. Exactly one region has is_default=true; it is the region served by https://api.digitalocean.com. The region is chosen by the API host a request is sent to, not by a request field: slug 'ric1' means https://api-ric1.digitalocean.com. This MCP server sends every request to the one API host it is configured with, which is https://api.digitalocean.com unless configured otherwise. No pagination."),
			),
		},
	}
}
