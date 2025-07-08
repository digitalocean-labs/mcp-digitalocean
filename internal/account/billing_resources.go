package account

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp-digitalocean/internal/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const BillingURI = "billing://"

type BillingMCPResource struct {
	client *godo.Client
}

func NewBillingMCPResource(client *godo.Client) *BillingMCPResource {
	return &BillingMCPResource{
		client: client,
	}
}

func (b *BillingMCPResource) getBillingResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		BillingURI+"{last}",
		"Billing History",
		mcp.WithTemplateDescription("Provide billing history for a user for last n months"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (b *BillingMCPResource) handleGetBillingResourceTemplate(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	perPage, err := common.ExtractNumericIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid billing URI: %s", request.Params.URI)
	}

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: int(perPage),
	}

	billingHistory, _, err := b.client.BillingHistory.List(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error fetching billing history: %w", err)
	}

	jsonData, err := json.MarshalIndent(billingHistory, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing billing history: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (b *BillingMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		b.getBillingResourceTemplate(): b.handleGetBillingResourceTemplate,
	}
}
