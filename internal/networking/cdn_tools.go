package networking

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

// createCDN creates a new CDN
func (c *CDNTool) createCDN(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	origin := req.GetArguments()["Origin"].(string)
	ttl := uint32(req.GetArguments()["TTL"].(float64))
	customDomain, _ := req.GetArguments()["CustomDomain"].(string)

	createRequest := &godo.CDNCreateRequest{
		Origin:       origin,
		TTL:          ttl,
		CustomDomain: customDomain,
	}

	cdn, _, err := c.client.CDNs.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCDN, err := json.MarshalIndent(cdn, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonCDN)), nil
}

// deleteCDN deletes a CDN
func (c *CDNTool) deleteCDN(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cdnID := req.GetArguments()["ID"].(string)
	_, err := c.client.CDNs.Delete(ctx, cdnID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("CDN deleted successfully"), nil
}

// flushCDNCache flushes the cache of a CDN
func (c *CDNTool) flushCDNCache(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cdnID := req.GetArguments()["ID"].(string)
	files := req.GetArguments()["Files"].([]string)

	flushRequest := &godo.CDNFlushCacheRequest{
		Files: files,
	}

	_, err := c.client.CDNs.FlushCache(ctx, cdnID, flushRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("CDN cache flushed successfully"), nil
}

// Tools returns a list of tool functions
func (c *CDNTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: c.createCDN,
			Tool: mcp.NewTool("digitalocean-cdn-create",
				mcp.WithDescription("Create a new CDN"),
				mcp.WithString("Origin", mcp.Required(), mcp.Description("Origin URL for the CDN")),
				mcp.WithNumber("TTL", mcp.Required(), mcp.Description("Time-to-live for the CDN cache")),
				mcp.WithString("CustomDomain", mcp.Description("Custom domain for the CDN")),
			),
		},
		{
			Handler: c.deleteCDN,
			Tool: mcp.NewTool("digitalocean-cdn-delete",
				mcp.WithDescription("Delete a CDN"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the CDN to delete")),
			),
		},
		{
			Handler: c.flushCDNCache,
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
