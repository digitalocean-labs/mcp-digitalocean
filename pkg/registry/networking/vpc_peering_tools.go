package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// VPCPeeringTool represents a tool for managing VPC peering connections.
type VPCPeeringTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewVPCPeeringTool creates a new VPCPeeringTool instance.
func NewVPCPeeringTool(client func(ctx context.Context) (*godo.Client, error)) *VPCPeeringTool {
	return &VPCPeeringTool{
		client: client,
	}
}

func (t *VPCPeeringTool) getVPCPeering(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("VPC Peering ID is required"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	peering, _, err := client.VPCs.GetVPCPeering(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(peering, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

func (t *VPCPeeringTool) listVPCPeerings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := 1
	perPage := 20
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	peerings, _, err := client.VPCs.ListVPCPeerings(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonPeerings, err := json.MarshalIndent(peerings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonPeerings)), nil
}

func (t *VPCPeeringTool) createPeering(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	peeringName := args["Name"].(string)
	vpc1 := args["Vpc1"].(string)
	vpc2 := args["Vpc2"].(string)

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Create a new VPC peering connection
	peering, _, err := client.VPCs.CreateVPCPeering(ctx, &godo.VPCPeeringCreateRequest{
		Name:   peeringName,
		VPCIDs: []string{vpc1, vpc2},
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := json.MarshalIndent(peering, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (t *VPCPeeringTool) deletePeering(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	peeringID := args["ID"].(string)

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Delete the VPC peering connection
	_, err = client.VPCs.DeleteVPCPeering(ctx, peeringID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("VPC peering connection deleted"), nil
}

func (t *VPCPeeringTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.getVPCPeering,
			Tool: mcp.NewTool("vpc-peering-get",
				mcp.WithDescription("Get VPC Peering information by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC Peering connection")),
			),
		},
		{
			Handler: t.listVPCPeerings,
			Tool: mcp.NewTool("vpc-peering-list",
				mcp.WithDescription("List VPC Peering connections with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: t.createPeering,
			Tool: mcp.NewTool("vpc-peering-create",
				mcp.WithDescription("Create a new VPC Peering connection between two VPCs"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the Peering connection")),
				mcp.WithString("Vpc1", mcp.Required(), mcp.Description("ID of the first VPC")),
				mcp.WithString("Vpc2", mcp.Required(), mcp.Description("ID of the second VPC")),
			),
		},
		{
			Handler: t.deletePeering,
			Tool: mcp.NewTool("vpc-peering-delete",
				mcp.WithDescription("Delete a VPC Peering connection"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC Peering connection to delete")),
			),
		},
	}
}
