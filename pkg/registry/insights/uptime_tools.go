package insights

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultChecksPageSize = 20
	defaultChecksPage     = 1
)

// UptimeTool provides UptimeCheck and Alert management tools
type UptimeTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewUptimeTool creates a new UptimeCheck tool
func NewUptimeTool(client func(ctx context.Context) (*godo.Client, error)) *UptimeTool {
	return &UptimeTool{
		client: client,
	}
}

// getUptimeCheck fetches UptimeCheck information by ID
func (c *UptimeTool) getUptimeCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("UptimeCheck ID is required"), nil
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uptimeCheck, _, err := client.UptimeChecks.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUptimeCheck, err := json.MarshalIndent(uptimeCheck, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUptimeCheck)), nil
}

// getUptimeCheck fetches UptimeCheck information by ID
func (c *UptimeTool) getUptimeCheckState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("UptimeCheck ID is required"), nil
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uptimeCheck, _, err := client.UptimeChecks.GetState(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUptimeCheck, err := json.MarshalIndent(uptimeCheck, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUptimeCheck)), nil
}

// listUptimeChecks lists UptimeChecks with pagination support
func (c *UptimeTool) listUptimeChecks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := defaultChecksPage
	perPage := defaultChecksPageSize
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

	uptimeChecks, _, err := client.UptimeChecks.List(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonUptimeChecks, err := json.MarshalIndent(uptimeChecks, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonUptimeChecks)), nil
}

// createUptimeCheck creates a new UptimeCheck
func (c *UptimeTool) createUptimeCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)
	checkType := req.GetArguments()["Type"].(string)
	target := req.GetArguments()["Target"].(string)

	rawRegions, _ := req.GetArguments()["Regions"]
	var regions []string
	if arr, ok := rawRegions.([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				regions = append(regions, s)
			}
		}
	}

	enabled := req.GetArguments()["Enabled"].(bool)

	createRequest := &godo.CreateUptimeCheckRequest{
		Name:    name,
		Type:    checkType,
		Target:  target,
		Regions: regions,
		Enabled: enabled,
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uptimeCheck, _, err := client.UptimeChecks.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUptimeCheck, err := json.MarshalIndent(uptimeCheck, "", "  ")

	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUptimeCheck)), nil
}

// updateUptimeCheck updates a existing UptimeCheck
func (c *UptimeTool) updateUptimeCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("UptimeCheck ID is required"), nil
	}

	name := req.GetArguments()["Name"].(string)
	checkType := req.GetArguments()["Type"].(string)
	target := req.GetArguments()["Target"].(string)
	enabled := req.GetArguments()["Enabled"].(bool)

	rawRegions, _ := req.GetArguments()["Regions"]
	var regions []string
	if arr, ok := rawRegions.([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				regions = append(regions, s)
			}
		}
	}

	updateRequest := &godo.UpdateUptimeCheckRequest{
		Name:    name,
		Type:    checkType,
		Target:  target,
		Regions: regions,
		Enabled: enabled,
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uptimeCheck, _, err := client.UptimeChecks.Update(ctx, id, updateRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUptimeCheck, err := json.MarshalIndent(uptimeCheck, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUptimeCheck)), nil
}

// deleteUptimeCheck deletes a UptimeCheck
func (c *UptimeTool) deleteUptimeCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("UptimeCheck ID is required"), nil
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.UptimeChecks.Delete(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("uptimeCheckID deleted successfully"), nil
}

// Tools returns a list of tool functions
func (c *UptimeTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: c.getUptimeCheck,
			Tool: mcp.NewTool("uptimecheck-get",
				mcp.WithDescription("Get UptimeCheck information by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the UptimeCheck")),
			),
		},
		{
			Handler: c.getUptimeCheckState,
			Tool: mcp.NewTool("uptimecheck-get-state",
				mcp.WithDescription("Get UptimeCheck information by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the UptimeCheck")),
			),
		},
		{
			Handler: c.listUptimeChecks,
			Tool: mcp.NewTool("uptimecheck-list",
				mcp.WithDescription("List UptimeChecks with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultChecksPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultChecksPageSize), mcp.Description("Items per page")),
			),
		},
		{
			Handler: c.createUptimeCheck,
			Tool: mcp.NewTool("uptimecheck-create",
				mcp.WithDescription("Create a new UptimeCheck"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the UptimeCheck")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of the UptimeCheck. value : HTTPS, HTTP or PING")),
				mcp.WithString("Target", mcp.Required(), mcp.Description("Endpoint to check for the UptimeCheck")),
				mcp.WithArray("Regions", mcp.Description("Regions where you'd like to perform these checks. values : \"us_east\", \"us_west\", \"eu_west\", \"se_asia\""),
					mcp.WithStringEnumItems([]string{"us_east", "us_west", "eu_west", "se_asia"})),
				mcp.WithBoolean("Enabled", mcp.Required(), mcp.Description("A boolean value indicating whether the check is enabled or disabled.")),
			),
		},
		{
			Handler: c.updateUptimeCheck,
			Tool: mcp.NewTool("uptimecheck-update",
				mcp.WithDescription("Update a UptimeCheck"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the UptimeCheck")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the UptimeCheck")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of the UptimeCheck. value : HTTPS, HTTP or PING")),
				mcp.WithString("Target", mcp.Required(), mcp.Description("Endpoint to check for the UptimeCheck")),
				mcp.WithArray("Regions", mcp.Description("Regions where you'd like to perform these checks. values : \"us_east\", \"us_west\", \"eu_west\", \"se_asia\""),
					mcp.WithStringEnumItems([]string{"us_east", "us_west", "eu_west", "se_asia"})),
				mcp.WithBoolean("Enabled", mcp.Required(), mcp.Description("A boolean value indicating whether the check is enabled or disabled.")),
			),
		},
		{
			Handler: c.deleteUptimeCheck,
			Tool: mcp.NewTool("uptimecheck-delete",
				mcp.WithDescription("Delete a uptimeCheck"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the uptimeCheck to delete")),
			),
		},
	}
}
