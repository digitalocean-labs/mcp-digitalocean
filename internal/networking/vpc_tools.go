package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// VPCTool provides VPC management tools
type VPCTool struct {
	client *godo.Client
}

// NewVPCTool creates a new VPC tool
func NewVPCTool(client *godo.Client) *VPCTool {
	return &VPCTool{
		client: client,
	}
}

// createVPC creates a new VPC
func (v *VPCTool) createVPC(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)
	region := req.GetArguments()["Region"].(string)

	createRequest := &godo.VPCCreateRequest{
		Name:       name,
		RegionSlug: region,
	}

	vpc, _, err := v.client.VPCs.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonVPC, err := json.MarshalIndent(vpc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonVPC)), nil
}

// listVPCMembers lists members of a VPC
func (v *VPCTool) listVPCMembers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vpcID := req.GetArguments()["ID"].(string)

	members, _, err := v.client.VPCs.ListMembers(ctx, vpcID, nil, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonMembers, err := json.MarshalIndent(members, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonMembers)), nil
}

// deleteVPC deletes a VPC
func (v *VPCTool) deleteVPC(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vpcID := req.GetArguments()["ID"].(string)

	_, err := v.client.VPCs.Delete(ctx, vpcID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("VPC deleted successfully"), nil
}

// Tools returns a list of tool functions
func (v *VPCTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: v.createVPC,
			Tool: mcp.NewTool("digitalocean-vpc-create",
				mcp.WithDescription("Create a new VPC"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the VPC")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug (e.g., nyc3)")),
			),
		},
		{
			Handler: v.listVPCMembers,
			Tool: mcp.NewTool("digitalocean-vpc-list-members",
				mcp.WithDescription("List members of a VPC"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC")),
			),
		},
		{
			Handler: v.deleteVPC,
			Tool: mcp.NewTool("digitalocean-vpc-delete",
				mcp.WithDescription("Delete a VPC"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC to delete")),
			),
		},
	}
}
