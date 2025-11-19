package spaces

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CDNTool provides CDN management tools
type CDNTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewCDNTool creates a new CDN tool
func NewCDNTool(client func(ctx context.Context) (*godo.Client, error)) *CDNTool {
	return &CDNTool{
		client: client,
	}
}

// getCDN fetches CDN information by ID
func (c *CDNTool) getCDN(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("CDN ID is required"), nil
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	cdn, _, err := client.CDNs.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCDN, err := json.MarshalIndent(cdn, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonCDN)), nil
}

// listCDNs lists CDNs with pagination support
func (c *CDNTool) listCDNs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := 1
	perPage := 20
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	cdns, _, err := client.CDNs.List(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCDNs, err := json.MarshalIndent(cdns, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonCDNs)), nil
}

// createCDN creates a new CDN
func (c *CDNTool) createCDN(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	origin := req.GetArguments()["Origin"].(string)
	ttl := uint32(req.GetArguments()["TTL"].(float64))

	createRequest := &godo.CDNCreateRequest{
		Origin: origin,
		TTL:    ttl,
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	cdn, _, err := client.CDNs.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCDN, err := json.MarshalIndent(cdn, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonCDN)), nil
}

// deleteCDN deletes a CDN
func (c *CDNTool) deleteCDN(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cdnID := req.GetArguments()["ID"].(string)

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.CDNs.Delete(ctx, cdnID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("CDN deleted successfully"), nil
}

// flushCDNCache flushes the cache of a CDN
func (c *CDNTool) flushCDNCache(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cdnID := req.GetArguments()["ID"].(string)
	files := req.GetArguments()["Files"].([]any)

	filesStr := make([]string, len(files))
	for i, file := range files {
		if fileStr, ok := file.(string); ok {
			filesStr[i] = fileStr
		}
	}

	flushRequest := &godo.CDNFlushCacheRequest{
		Files: filesStr,
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.CDNs.FlushCache(ctx, cdnID, flushRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("CDN cache flushed successfully"), nil
}

// Tools returns a list of tool functions
func (c *CDNTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: c.getCDN,
			Tool: mcp.NewTool("spaces-cdn-get",
				mcp.WithDescription("Get CDN information by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the CDN")),
			),
		},
		{
			Handler: c.listCDNs,
			Tool: mcp.NewTool("spaces-cdn-list",
				mcp.WithDescription("List CDNs with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: c.createCDN,
			Tool: mcp.NewTool("spaces-cdn-create",
				mcp.WithDescription("Create a new CDN"),
				mcp.WithString("Origin", mcp.Required(), mcp.Description("Origin URL for the CDN")),
				mcp.WithNumber("TTL", mcp.Required(), mcp.Description("Time-to-live for the CDN cache")),
			),
		},
		{
			Handler: c.deleteCDN,
			Tool: mcp.NewTool("spaces-cdn-delete",
				mcp.WithDescription("Delete a CDN"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the CDN to delete")),
			),
		},

		{
			Handler: c.flushCDNCache,
			Tool: mcp.NewTool("spaces-cdn-flush-cache",
				mcp.WithDescription("Flush the cache of a CDN"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the CDN")),
				mcp.WithArray("Files", mcp.Required(), mcp.Description("file names to flush from the cache (max 50 per request)"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "name of file",
				})),
			),
		},
	}
}
