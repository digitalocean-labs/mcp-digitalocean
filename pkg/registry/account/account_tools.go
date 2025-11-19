package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

	jsonData, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling account: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (a *AccountTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: a.getAccountInformation,
			Tool: mcp.NewTool("account-get-information",
				mcp.WithDescription("Retrieves account information for the current user"),
			),
		},
	}
}
