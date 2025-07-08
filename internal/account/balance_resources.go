package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const BalanceURI = "balance://"

type BalanceMCPResource struct {
	client *godo.Client
}

func NewBalanceMCPResource(client *godo.Client) *BalanceMCPResource {
	return &BalanceMCPResource{
		client: client,
	}
}

func (b *BalanceMCPResource) getBalanceResource() mcp.Resource {
	return mcp.NewResource(
		BalanceURI+"current",
		"Balance Information",
		mcp.WithResourceDescription("Provide balance information for the user account"),
		mcp.WithMIMEType("application/json"),
	)
}

func (b *BalanceMCPResource) handleGetBalanceResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	balance, _, err := b.client.Balance.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching balance: %w", err)
	}

	jsonData, err := json.MarshalIndent(balance, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing balance: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (b *BalanceMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		b.getBalanceResource(): b.handleGetBalanceResource,
	}
}
