package docs

import (
	"context"
	_ "embed"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:embed data/regions_snapshot.json
var regionsSnapshotJSON []byte

// StaticRegionTools exposes token-free region metadata for the docs-only MCP profile.
type StaticRegionTools struct{}

// NewStaticRegionTools returns handlers backed by embedded snapshot data.
func NewStaticRegionTools() *StaticRegionTools {
	return &StaticRegionTools{}
}

func (s *StaticRegionTools) listStaticRegions(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(string(regionsSnapshotJSON)), nil
}

// Tools registers docs-list-regions (JSON array of region slugs, no API token).
func (s *StaticRegionTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.listStaticRegions,
			Tool: mcp.NewTool(
				"docs-list-regions",
				mcp.WithDescription("List DigitalOcean region slugs from a static snapshot (no API token). For live droplet sizes and features, use region-list on a service that includes API credentials."),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithIdempotentHintAnnotation(true),
			),
		},
	}
}
