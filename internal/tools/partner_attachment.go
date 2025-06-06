package tools

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type PartnerAttachmentTool struct {
	client *godo.Client
}

func NewPartnerAttachmentTool(client *godo.Client) *PartnerAttachmentTool {
	return &PartnerAttachmentTool{
		client: client,
	}
}

func (p *PartnerAttachmentTool) CreatePartnerAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}

	region, err := req.RequireString("Region")
	if err != nil {
		return nil, err
	}

	bandwidthFloat, err := req.RequireFloat("Bandwidth")
	if err != nil {
		return nil, err
	}
	bandwidth := int(bandwidthFloat)

	createRequest := &godo.PartnerAttachmentCreateRequest{
		Name:                      name,
		Region:                    region,
		ConnectionBandwidthInMbps: bandwidth,
	}

	attachment, _, err := p.client.PartnerAttachment.Create(ctx, createRequest)
	if err != nil {
		return nil, err
	}

	jsonAttachment, err := json.MarshalIndent(attachment, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAttachment)), nil
}

func (p *PartnerAttachmentTool) DeletePartnerAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}
	_, err = p.client.PartnerAttachment.Delete(ctx, id)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText("Partner attachment deleted successfully"), nil
}

func (p *PartnerAttachmentTool) GetServiceKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}
	serviceKey, _, err := p.client.PartnerAttachment.GetServiceKey(ctx, id)
	if err != nil {
		return nil, err
	}

	jsonServiceKey, err := json.MarshalIndent(serviceKey, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonServiceKey)), nil
}

func (p *PartnerAttachmentTool) GetBGPConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}
	bgpAuthKey, _, err := p.client.PartnerAttachment.GetBGPAuthKey(ctx, id)
	if err != nil {
		return nil, err
	}

	jsonBGPAuthKey, err := json.MarshalIndent(bgpAuthKey, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBGPAuthKey)), nil
}

func (p *PartnerAttachmentTool) UpdatePartnerAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}

	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}

	vpcIDs, err := req.RequireStringSlice("VPCIDs")
	if err != nil {
		return nil, err
	}

	updateRequest := &godo.PartnerAttachmentUpdateRequest{
		Name:   name,
		VPCIDs: vpcIDs,
	}

	attachment, _, err := p.client.PartnerAttachment.Update(ctx, id, updateRequest)
	if err != nil {
		return nil, err
	}

	jsonAttachment, err := json.MarshalIndent(attachment, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonAttachment)), nil
}

func (p *PartnerAttachmentTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: p.CreatePartnerAttachment,
			Tool: mcp.NewTool("digitalocean-partner-attachment-create",
				mcp.WithDescription("Create a new partner attachment"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the partner attachment")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region for the partner attachment")),
				mcp.WithNumber("Bandwidth", mcp.Required(), mcp.Description("Bandwidth in Mbps")),
			),
		},
		{
			Handler: p.DeletePartnerAttachment,
			Tool: mcp.NewTool("digitalocean-partner-attachment-delete",
				mcp.WithDescription("Delete a partner attachment"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the partner attachment to delete")),
			),
		},
		{
			Handler: p.GetServiceKey,
			Tool: mcp.NewTool("digitalocean-partner-attachment-get-service-key",
				mcp.WithDescription("Get the service key of a partner attachment"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the partner attachment")),
			),
		},
		{
			Handler: p.GetBGPConfig,
			Tool: mcp.NewTool("digitalocean-partner-attachment-get-bgp-config",
				mcp.WithDescription("Get the BGP configuration of a partner attachment"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the partner attachment")),
			),
		},
		{
			Handler: p.UpdatePartnerAttachment,
			Tool: mcp.NewTool("digitalocean-partner-attachment-update",
				mcp.WithDescription("Update a partner attachment"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the partner attachment to update")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("New name for the partner attachment")),
				mcp.WithArray("VPCIDs", mcp.Required(), mcp.Description("VPC ID to associate with the partner attachment"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "VPC ID to associate with Partner attachment",
				})),
			),
		},
	}
}
