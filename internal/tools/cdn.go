package tools

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CDNTool provides CDN management tools
type CDNTool struct {
	client *godo.Client
}

// NewCDNTool creates a new CDN tool
func NewCDNTool(client *godo.Client) *CDNTool {
	return &CDNTool{
		client: client,
	}
}

// CreateCDN creates a new CDN
func (c *CDNTool) CreateCDN(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	origin, err := req.RequireString("Origin")
	if err != nil {
		return nil, err
	}

	ttlFloat, err := req.RequireFloat("TTL")
	if err != nil {
		return nil, err
	}
	ttl := uint32(ttlFloat)

	customDomain := req.GetString("CustomDomain", "")

	createRequest := &godo.CDNCreateRequest{
		Origin:       origin,
		TTL:          ttl,
		CustomDomain: customDomain,
	}

	cdn, _, err := c.client.CDNs.Create(ctx, createRequest)
	if err != nil {
		return nil, err
	}

	jsonCDN, err := json.MarshalIndent(cdn, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonCDN)), nil
}

// DeleteCDN deletes a CDN
func (c *CDNTool) DeleteCDN(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cdnID, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}
	_, err = c.client.CDNs.Delete(ctx, cdnID)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("CDN deleted successfully"), nil
}

// FlushCDNCache flushes the cache of a CDN
func (c *CDNTool) FlushCDNCache(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cdnID, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}

	filesInterface, err := req.RequireStringSlice("Files")
	if err != nil {
		return nil, err
	}

	flushRequest := &godo.CDNFlushCacheRequest{
		Files: filesInterface,
	}

	_, err = c.client.CDNs.FlushCache(ctx, cdnID, flushRequest)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("CDN cache flushed successfully"), nil
}

// Tools returns a list of tool functions
func (c *CDNTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: c.CreateCDN,
			Tool: mcp.NewTool("digitalocean-cdn-create",
				mcp.WithDescription("Create a new CDN"),
				mcp.WithString("Origin", mcp.Required(), mcp.Description("Origin URL for the CDN")),
				mcp.WithNumber("TTL", mcp.Required(), mcp.Description("Time-to-live for the CDN cache")),
				mcp.WithString("CustomDomain", mcp.Description("Custom domain for the CDN")),
			),
		},
		{
			Handler: c.DeleteCDN,
			Tool: mcp.NewTool("digitalocean-cdn-delete",
				mcp.WithDescription("Delete a CDN"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the CDN to delete")),
			),
		},
		{
			Handler: c.FlushCDNCache,
			Tool: mcp.NewTool("digitalocean-cdn-flush-cache",
				mcp.WithDescription("Flush the cache of a CDN"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the CDN")),
				mcp.WithArray("Files", mcp.Required(), mcp.Description("file names to flush from the cache"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "name of file",
				})),
			),
		},
	}
}
