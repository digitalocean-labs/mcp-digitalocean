package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const AccountURI = "account://"

type AccountMCPResource struct {
	client *godo.Client
}

func NewAccountMCPResource(client *godo.Client) *AccountMCPResource {
	return &AccountMCPResource{
		client: client,
	}
}

func (a *AccountMCPResource) getAccountResource() mcp.Resource {
	return mcp.NewResource(
		AccountURI+"current",
		"Account Information",
		mcp.WithResourceDescription("Provides information about user account"),
		mcp.WithMIMEType("application/json"),
	)
}

func (a *AccountMCPResource) handleGetAccountResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	account, _, err := a.client.Account.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching account: %w", err)
	}

	jsonData, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing account: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (a *AccountMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		a.getAccountResource(): a.handleGetAccountResource,
	}
}
