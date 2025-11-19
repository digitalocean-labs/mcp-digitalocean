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
	jsonData, err := json.MarshalIndent(invoices, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
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

	jsonData, err := json.MarshalIndent(invoice, "", "  ")
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
			Tool: mcp.NewTool("invoice-list",
				mcp.WithDescription("List invoices with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultInvoicesPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultInvoicesPageSize), mcp.Description("Items per page")),
			),
		},
		{
			Handler: i.getInvoice,
			Tool: mcp.NewTool("get-invoice",
				mcp.WithDescription("Get a specific invoice"),
				mcp.WithString("InvoiceUUID", mcp.Required(), mcp.Description("The UUID of the invoice")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultInvoicesPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultInvoicesPageSize), mcp.Description("Items per page")),
			),
		},
	}
}
