package tools

import (
	"context"
	"encoding/json"

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

// CreateVPC creates a new VPC
func (v *VPCTool) CreateVPC(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}

	region, err := req.RequireString("Region")
	if err != nil {
		return nil, err
	}

	createRequest := &godo.VPCCreateRequest{
		Name:       name,
		RegionSlug: region,
	}

	vpc, _, err := v.client.VPCs.Create(ctx, createRequest)
	if err != nil {
		return nil, err
	}

	jsonVPC, err := json.MarshalIndent(vpc, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonVPC)), nil
}

// ListVPCMembers lists members of a VPC
func (v *VPCTool) ListVPCMembers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vpcID, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}

	members, _, err := v.client.VPCs.ListMembers(ctx, vpcID, nil, nil)
	if err != nil {
		return nil, err
	}

	jsonMembers, err := json.MarshalIndent(members, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonMembers)), nil
}

// DeleteVPC deletes a VPC
func (v *VPCTool) DeleteVPC(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vpcID, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}

	_, err = v.client.VPCs.Delete(ctx, vpcID)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("VPC deleted successfully"), nil
}

// Tools returns a list of tool functions
func (v *VPCTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: v.CreateVPC,
			Tool: mcp.NewTool("digitalocean-vpc-create",
				mcp.WithDescription("Create a new VPC"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the VPC")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug (e.g., nyc3)")),
			),
		},
		{
			Handler: v.ListVPCMembers,
			Tool: mcp.NewTool("digitalocean-vpc-list-members",
				mcp.WithDescription("List members of a VPC"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC")),
			),
		},
		{
			Handler: v.DeleteVPC,
			Tool: mcp.NewTool("digitalocean-vpc-delete",
				mcp.WithDescription("Delete a VPC"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC to delete")),
			),
		},
	}
}
