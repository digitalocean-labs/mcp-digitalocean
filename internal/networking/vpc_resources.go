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

const VPCURI = "vpcs://"

type VPCMCPResource struct {
	client *godo.Client
}

func NewVPCMCPResource(client *godo.Client) *VPCMCPResource {
	return &VPCMCPResource{
		client: client,
	}
}

func (v *VPCMCPResource) getVPCResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		VPCURI+"{id}",
		"VPC",
		mcp.WithTemplateDescription("Returns VPC information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (v *VPCMCPResource) handleGetVPCResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	vpcID, err := common.ExtractStringIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid VPC URI: %w", err)
	}

	vpc, _, err := v.client.VPCs.Get(ctx, vpcID)
	if err != nil {
		return nil, fmt.Errorf("error fetching VPC: %w", err)
	}

	jsonData, err := json.MarshalIndent(vpc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing VPC: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (v *VPCMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		v.getVPCResourceTemplate(): v.handleGetVPCResource,
	}
}
