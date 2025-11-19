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
	defaultBillingPageSize = 30
	defaultBillingPage     = 1
)

// BillingTools provides tool-based handlers for DigitalOcean Billing History.
type BillingTools struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewBillingTools creates a new BillingTools instance.
func NewBillingTools(client func(ctx context.Context) (*godo.Client, error)) *BillingTools {
	return &BillingTools{client: client}
}

// listBillingHistory lists billing history with pagination support.
func (b *BillingTools) listBillingHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultBillingPage
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultBillingPageSize
	}
	opt := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	client, err := b.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	billingHistory, _, err := client.BillingHistory.List(ctx, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := json.MarshalIndent(billingHistory, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for billing history.
func (b *BillingTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: b.listBillingHistory,
			Tool: mcp.NewTool("billing-history-list",
				mcp.WithDescription("List billing history with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultBillingPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultBillingPageSize), mcp.Description("Items per page")),
			),
		},
	}
}
