package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

	jsonData, err := json.MarshalIndent(balance, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for balance.
func (b *BalanceTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: b.getBalance,
			Tool: mcp.NewTool("balance-get",
				mcp.WithDescription("Get balance information for the user account"),
			),
		},
	}
}
