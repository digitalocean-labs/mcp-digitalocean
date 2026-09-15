package account

import (
	"context"
	"fmt"
	"mcp-digitalocean/pkg/response"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mcp-digitalocean/pkg/registry/common"
)

const (
	defaultInvoicesPageSize = 30
	defaultInvoicesPage     = 1
)

// InvoiceTools provides tool-based handlers for DigitalOcean Invoices.
type InvoiceTools struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewInvoiceTools creates a new InvoiceTools instance.
func NewInvoiceTools(client func(ctx context.Context) (*godo.Client, error)) *InvoiceTools {
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

	client, err := i.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	invoices, _, err := client.Invoices.List(ctx, &godo.ListOptions{Page: int(page), PerPage: int(perPage)})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := response.CompactJSON(invoices)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonData), nil
}

// getInvoice retrieves a specific invoice by UUID.
func (i *InvoiceTools) getInvoice(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultInvoicesPage
	}

	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultInvoicesPageSize
	}

	invoiceUUID, ok := req.GetArguments()["InvoiceUUID"].(string)
	if !ok {
		return mcp.NewToolResultError("missing InvoiceUUID"), nil
	}

	client, err := i.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	invoice, _, err := client.Invoices.Get(ctx, invoiceUUID, &godo.ListOptions{Page: int(page), PerPage: int(perPage)})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := response.CompactJSON(invoice)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(jsonData), nil
}

// Tools returns the list of server tools for invoices.
func (i *InvoiceTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: i.listInvoices,
			Tool: mcp.NewTool("invoice-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List invoices with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultInvoicesPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultInvoicesPageSize), mcp.Description("Items per page")),
			),
		},
		{
			Handler: i.getInvoice,
			Tool: mcp.NewTool("get-invoice",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a specific invoice"),
				mcp.WithString("InvoiceUUID", mcp.Required(), mcp.Description("The UUID of the invoice")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultInvoicesPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultInvoicesPageSize), mcp.Description("Items per page")),
			),
		},
	}
}
