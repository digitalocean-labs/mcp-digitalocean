package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type PartnerAttachmentTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewPartnerAttachmentTool(client func(ctx context.Context) (*godo.Client, error)) *PartnerAttachmentTool {
	return &PartnerAttachmentTool{
		client: client,
	}
}

func (p *PartnerAttachmentTool) createPartnerAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)
	region := req.GetArguments()["Region"].(string)
	bandwidth := int(req.GetArguments()["Bandwidth"].(float64))

	createRequest := &godo.PartnerAttachmentCreateRequest{
		Name:                      name,
		Region:                    region,
		ConnectionBandwidthInMbps: bandwidth,
	}

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	attachment, _, err := client.PartnerAttachment.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAttachment, err := json.MarshalIndent(attachment, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAttachment)), nil
}

// getPartnerAttachment fetches partner attachment information by ID
func (p *PartnerAttachmentTool) getPartnerAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Partner attachment ID is required"), nil
	}

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	attachment, _, err := client.PartnerAttachment.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonAttachment, err := json.MarshalIndent(attachment, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonAttachment)), nil
}

// listPartnerAttachments lists partner attachments with pagination support
func (p *PartnerAttachmentTool) listPartnerAttachments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := 1
	perPage := 20
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	attachments, _, err := client.PartnerAttachment.List(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonAttachments, err := json.MarshalIndent(attachments, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonAttachments)), nil
}

func (p *PartnerAttachmentTool) deletePartnerAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetArguments()["ID"].(string)

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.PartnerAttachment.Delete(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Partner attachment deleted successfully"), nil
}

func (p *PartnerAttachmentTool) getServiceKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetArguments()["ID"].(string)

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	serviceKey, _, err := client.PartnerAttachment.GetServiceKey(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonServiceKey, err := json.MarshalIndent(serviceKey, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonServiceKey)), nil
}

func (p *PartnerAttachmentTool) getBGPConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetArguments()["ID"].(string)

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	bgpAuthKey, _, err := client.PartnerAttachment.GetBGPAuthKey(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonBGPAuthKey, err := json.MarshalIndent(bgpAuthKey, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBGPAuthKey)), nil
}

func (p *PartnerAttachmentTool) updatePartnerAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetArguments()["ID"].(string)
	name := req.GetArguments()["Name"].(string)
	vpcIDs := req.GetArguments()["VPCIDs"].([]any)
	vpcIDsStr := make([]string, len(vpcIDs))
	for i, v := range vpcIDs {
		if vStr, ok := v.(string); ok {
			vpcIDsStr[i] = vStr
		}
	}

	updateRequest := &godo.PartnerAttachmentUpdateRequest{
		Name:   name,
		VPCIDs: vpcIDsStr,
	}

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	attachment, _, err := client.PartnerAttachment.Update(ctx, id, updateRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAttachment, err := json.MarshalIndent(attachment, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAttachment)), nil
}

func (p *PartnerAttachmentTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: p.getPartnerAttachment,
			Tool: mcp.NewTool("partner-attachment-get",
				mcp.WithDescription("Get partner attachment information by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the partner attachment")),
			),
		},
		{
			Handler: p.listPartnerAttachments,
			Tool: mcp.NewTool("partner-attachment-list",
				mcp.WithDescription("List partner attachments with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: p.createPartnerAttachment,
			Tool: mcp.NewTool("partner-attachment-create",
				mcp.WithDescription("Create a new partner attachment"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the partner attachment")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region for the partner attachment")),
				mcp.WithNumber("Bandwidth", mcp.Required(), mcp.Description("Bandwidth in Mbps")),
			),
		},
		{
			Handler: p.deletePartnerAttachment,
			Tool: mcp.NewTool("partner-attachment-delete",
				mcp.WithDescription("Delete a partner attachment"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the partner attachment to delete")),
			),
		},
		{
			Handler: p.getServiceKey,
			Tool: mcp.NewTool("partner-attachment-get-service-key",
				mcp.WithDescription("Get the service key of a partner attachment"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the partner attachment")),
			),
		},
		{
			Handler: p.getBGPConfig,
			Tool: mcp.NewTool("partner-attachment-get-bgp-config",
				mcp.WithDescription("Get the BGP configuration of a partner attachment"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the partner attachment")),
			),
		},
		{
			Handler: p.updatePartnerAttachment,
			Tool: mcp.NewTool("partner-attachment-update",
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
