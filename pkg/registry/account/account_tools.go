package account

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mcp-digitalocean/pkg/registry/common"
)

type AccountTools struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewAccountTools(client func(ctx context.Context) (*godo.Client, error)) *AccountTools {
	return &AccountTools{
		client: client,
	}
}

func (a *AccountTools) getAccountInformation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	account, _, err := client.Account.Get(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return accountOut.Result(account)
}

func (a *AccountTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: a.getAccountInformation,
			Tool: mcp.NewTool("account-get-information",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				accountOut.Schema(),
				mcp.WithDescription("Retrieves account information for the current user"),
			),
		},
	}
}
