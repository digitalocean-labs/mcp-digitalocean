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
	client *godo.Client
}

// NewBalanceTools creates a new BalanceTools instance.
func NewBalanceTools(client *godo.Client) *BalanceTools {
	return &BalanceTools{client: client}
}

// getBalance retrieves the balance information for the user account.
func (b *BalanceTools) getBalance(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	balance, _, err := b.client.Balance.Get(ctx)
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
			Tool: mcp.NewTool("digitalocean-balance-get",
				mcp.WithDescription("Get balance information for the user account"),
			),
		},
	}
}
