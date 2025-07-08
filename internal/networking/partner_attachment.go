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

const PartnerAttachmentURI = "partner_attachment://"

type PartnerAttachmentMCPResource struct {
	client *godo.Client
}

func NewPartnerAttachmentMCPResource(client *godo.Client) *PartnerAttachmentMCPResource {
	return &PartnerAttachmentMCPResource{
		client: client,
	}
}

func (p *PartnerAttachmentMCPResource) getPartnerAttachmentResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		PartnerAttachmentURI+"{id}",
		"Partner Attachment",
		mcp.WithTemplateDescription("Returns partner attachment information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (p *PartnerAttachmentMCPResource) handleGetPartnerAttachmentResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	attachmentID, err := common.ExtractStringIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid partner attachment URI: %w", err)
	}

	attachment, _, err := p.client.PartnerAttachment.Get(ctx, attachmentID)
	if err != nil {
		return nil, fmt.Errorf("error fetching partner attachment: %w", err)
	}

	jsonData, err := json.MarshalIndent(attachment, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing partner attachment: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (p *PartnerAttachmentMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		p.getPartnerAttachmentResourceTemplate(): p.handleGetPartnerAttachmentResource,
	}
}
