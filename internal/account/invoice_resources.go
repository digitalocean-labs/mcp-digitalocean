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

const InvoiceURI = "invoice://"

type InvoicesMCPResource struct {
	client *godo.Client
}

func NewInvoicesMCPResource(client *godo.Client) *InvoicesMCPResource {
	return &InvoicesMCPResource{
		client: client,
	}
}

func (i *InvoicesMCPResource) getInvoiceResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		InvoiceURI+"{last}",
		"Invoices History",
		mcp.WithTemplateDescription("Provides invoice history for last n months"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (i *InvoicesMCPResource) handleGetInvoiceResourceTemplate(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract the last parameter from the URI
	perPage, err := common.ExtractNumericIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid invoice URI: %s", request.Params.URI)
	}

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: int(perPage),
	}

	invoices, _, err := i.client.Invoices.List(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error fetching invoice history: %w", err)
	}

	jsonData, err := json.MarshalIndent(invoices, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing invoice history: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (i *InvoicesMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		i.getInvoiceResourceTemplate(): i.handleGetInvoiceResourceTemplate,
	}
}
