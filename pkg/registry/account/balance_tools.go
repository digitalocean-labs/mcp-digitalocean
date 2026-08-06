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

// BalanceTools provides tool-based handlers for DigitalOcean account balance.
type BalanceTools struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewBalanceTools creates a new BalanceTools instance.
func NewBalanceTools(client func(ctx context.Context) (*godo.Client, error)) *BalanceTools {
	return &BalanceTools{client: client}
}

// getBalance retrieves the balance information for the user account.
func (b *BalanceTools) getBalance(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := b.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	balance, _, err := client.Balance.Get(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonData, err := response.CompactJSON(balance)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(jsonData), nil
}

// Tools returns the list of server tools for balance.
func (b *BalanceTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: b.getBalance,
			Tool: mcp.NewTool("balance-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get balance information for the user account"),
			),
		},
	}
}
