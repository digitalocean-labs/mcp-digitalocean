package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultInvoicesPageSize = 30
	defaultInvoicesPage     = 1
)

// InvoiceTools provides tool-based handlers for DigitalOcean Invoices.
type InvoiceTools struct {
	client *godo.Client
}

// NewInvoiceTools creates a new InvoiceTools instance.
func NewInvoiceTools(client *godo.Client) *InvoiceTools {
	return &InvoiceTools{client: client}
}

// listInvoices lists invoices with pagination support.
func (i *InvoiceTools) listInvoices(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultInvoicesPage
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultInvoicesPageSize
	}
	invoices, _, err := i.client.Invoices.List(ctx, &godo.ListOptions{Page: int(page), PerPage: int(perPage)})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(invoices, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for invoices.
func (i *InvoiceTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: i.listInvoices,
			Tool: mcp.NewTool("digitalocean-invoice-list",
				mcp.WithDescription("List invoices with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultInvoicesPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultInvoicesPageSize), mcp.Description("Items per page")),
			),
		},
	}
}
