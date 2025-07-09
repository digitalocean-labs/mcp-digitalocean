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
	client *godo.Client
}

// NewVPCPeeringTool creates a new VPCPeeringTool instance.
func NewVPCPeeringTool(client *godo.Client) *VPCPeeringTool {
	return &VPCPeeringTool{
		client: client,
	}
}

func (t *VPCPeeringTool) createPeering(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	peeringName := args["Name"].(string)
	vpc1 := args["Vpc1"].(string)
	vpc2 := args["Vpc2"].(string)

	// Create a new VPC peering connection
	peering, _, err := t.client.VPCs.CreateVPCPeering(ctx, &godo.VPCPeeringCreateRequest{
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

	// Delete the VPC peering connection
	_, err := t.client.VPCs.DeleteVPCPeering(ctx, peeringID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("VPC peering connection deleted"), nil
}

func (t *VPCPeeringTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.createPeering,
			Tool: mcp.NewTool("digitalocean-vpc-peering-create",
				mcp.WithDescription("Create a new VPC Peering connection between two VPCs"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the Peering connection")),
				mcp.WithString("Vpc1", mcp.Required(), mcp.Description("ID of the first VPC")),
				mcp.WithString("Vpc2", mcp.Required(), mcp.Description("ID of the second VPC")),
			),
		},
		{
			Handler: t.deletePeering,
			Tool: mcp.NewTool("digitalocean-vpc-peering-delete",
				mcp.WithDescription("Delete a VPC Peering connection"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the VPC Peering connection to delete")),
			),
		},
	}
}
