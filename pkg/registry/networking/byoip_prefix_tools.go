package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type BYOIPPrefixTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewBYOIPPrefixTool creates a new BYOIPPrefixTool
func NewBYOIPPrefixTool(client func(ctx context.Context) (*godo.Client, error)) *BYOIPPrefixTool {
	return &BYOIPPrefixTool{
		client: client,
	}
}

// getBYOIPPrefix fetches BYOIP prefix information by prefix UUID
func (t *BYOIPPrefixTool) getBYOIPPrefix(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prefixUUID, ok := req.GetArguments()["UUID"].(string)
	if !ok || prefixUUID == "" {
		return mcp.NewToolResultError("UUID is required"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	byoipPrefix, _, err := client.BYOIPPrefixes.Get(ctx, prefixUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(byoipPrefix, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// listBYOIPPrefix fetches BYOIP prefixes for a user
func (t *BYOIPPrefixTool) listBYOIPPrefix(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := 1
	perPage := 20
	if v, ok := req.GetArguments()["Page"].(float64); ok && v > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && v > 0 {
		perPage = int(v)
	}

	opts := &godo.ListOptions{Page: page, PerPage: perPage}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	byoipPrefixes, _, err := client.BYOIPPrefixes.List(ctx, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(byoipPrefixes, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// createBYOIPPrefix creates a new BYOIP prefix for a user
func (t *BYOIPPrefixTool) createBYOIPPrefix(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prefix, ok := req.GetArguments()["prefix"].(string)
	if !ok || prefix == "" {
		return mcp.NewToolResultError("prefix is required"), nil
	}

	signature, ok := req.GetArguments()["signature"].(string)
	if !ok || signature == "" {
		return mcp.NewToolResultError("signature is required"), nil
	}

	region, ok := req.GetArguments()["region"].(string)
	if !ok || region == "" {
		return mcp.NewToolResultError("region is required"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	byoipPrefixCreated, _, err := client.BYOIPPrefixes.Create(ctx, &godo.BYOIPPrefixCreateReq{
		Prefix:    prefix,
		Signature: signature,
		Region:    region,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := json.MarshalIndent(byoipPrefixCreated, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// getByOIPPrefixResources fetches resources for a BYOIP prefix
func (t *BYOIPPrefixTool) getByOIPPrefixResources(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	prefiUUID, ok := req.GetArguments()["UUID"].(string)
	if !ok || prefiUUID == "" {
		return mcp.NewToolResultError("UUID is required"), nil
	}

	page := 1
	perPage := 20

	if v, ok := req.GetArguments()["Page"].(float64); ok && v > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && v > 0 {
		perPage = int(v)
	}

	opts := &godo.ListOptions{Page: page, PerPage: perPage}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	byoipPrefixResources, _, err := client.BYOIPPrefixes.GetResources(ctx, prefiUUID, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(byoipPrefixResources, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// deleteBYOIPPrefix deletes BYOIP prefix by UUID
func (t *BYOIPPrefixTool) deleteBYOIPPrefix(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prefiUUID, ok := req.GetArguments()["UUID"].(string)
	if !ok || prefiUUID == "" {
		return mcp.NewToolResultError("UUID is required"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.BYOIPPrefixes.Delete(ctx, prefiUUID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("BYOIP Prefix deleted"), nil
}

// Tools returns a list of tools for managing byoip prexies
func (t *BYOIPPrefixTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.getBYOIPPrefix,
			Tool: mcp.NewTool("byoip-prefix-get",
				mcp.WithDescription("Get BYOIP prefix information by UUID"),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("The UUID of the BYOIP prefix")),
			),
		},
		{
			Handler: t.listBYOIPPrefix,
			Tool: mcp.NewTool("byoip-prefix-list",
				mcp.WithDescription("List BYOIP prefixes"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Number of items per page")),
			),
		},
		{
			Handler: t.getByOIPPrefixResources,
			Tool: mcp.NewTool("byoip-prefix-resources-get",
				mcp.WithDescription("Get all resources for a BYOIP prefix"),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("The UUID of the BYOIP prefix")),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Number of items per page")),
			),
		},
		{
			Handler: t.createBYOIPPrefix,
			Tool: mcp.NewTool("byoip-prefix-create",
				mcp.WithDescription("Create a new BYOIP prefix"),
				mcp.WithString("Prefix", mcp.Required(), mcp.Description("The CIDR of the BYOIP prefix")),
				mcp.WithString("Signature", mcp.Required(), mcp.Description("The signature for the prefix")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("The region for the prefix")),
			),
		},
		{
			Handler: t.deleteBYOIPPrefix,
			Tool: mcp.NewTool("byoip-prefix-delete",
				mcp.WithDescription("Delete a BYOIP prefix"),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("The UUID of the BYOIP prefix")),
			),
		},
	}
}
