package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const VPCPeeringURI = "vpc_peering://"

type VPCPeeringMCPResource struct {
	client *godo.Client
}

func NewVPCPeeringMCPResource(client *godo.Client) *VPCPeeringMCPResource {
	return &VPCPeeringMCPResource{
		client: client,
	}
}

func (v *VPCPeeringMCPResource) getVPCPeeringResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		VPCPeeringURI+"{id}",
		"VPC Peering",
		mcp.WithTemplateDescription("Returns vpc peering information about a given peering id"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (v *VPCPeeringMCPResource) handleGetVPCPeering(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	peeringID, err := common.ExtractStringIDFromURI(req.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid vpc peering URI: %w", err)
	}

	vpcPeering, _, err := v.client.VPCs.GetVPCPeering(ctx, peeringID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vpc peering: %w", err)
	}

	jsonData, err := json.MarshalIndent(vpcPeering, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vpc peering: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (v *VPCPeeringMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		v.getVPCPeeringResourceTemplate(): v.handleGetVPCPeering,
	}
}
