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
	defaultKeysPageSize = 30
	defaultKeysPage     = 1
)

// KeysTool provides SSH key management tools
type KeysTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewKeysTool creates a new KeysTool
func NewKeysTool(client func(ctx context.Context) (*godo.Client, error)) *KeysTool {
	return &KeysTool{
		client: client,
	}
}

func (k *KeysTool) createKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	name := args["Name"].(string)
	publicKey := args["PublicKey"].(string)

	client, err := k.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	key, _, err := client.Keys.Create(ctx, &godo.KeyCreateRequest{
		Name:      name,
		PublicKey: publicKey,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonKey, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonKey)), nil
}

func (k *KeysTool) deleteKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyID := int(req.GetArguments()["ID"].(float64))

	client, err := k.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Keys.DeleteByID(ctx, keyID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("SSH key deleted successfully"), nil
}

// getKey retrieves a specific SSH key by its ID.
func (k *KeysTool) getKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(float64)
	if !ok {
		return mcp.NewToolResultError("Key ID is required"), nil
	}

	client, err := k.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	key, _, err := client.Keys.GetByID(ctx, int(id))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// listKeys lists SSH keys with pagination support.
func (k *KeysTool) listKeys(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultKeysPage
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultKeysPageSize
	}

	client, err := k.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	keys, _, err := client.Keys.List(ctx, &godo.ListOptions{Page: int(page), PerPage: int(perPage)})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns a list of tool functions
func (k *KeysTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: k.createKey,
			Tool: mcp.NewTool("key-create",
				mcp.WithDescription("Create a new SSH key"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the SSH key")),
				mcp.WithString("PublicKey", mcp.Required(), mcp.Description("Public key content")),
			),
		},
		{
			Handler: k.deleteKey,
			Tool: mcp.NewTool("key-delete",
				mcp.WithDescription("Delete an SSH key"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the SSH key to delete")),
			),
		},
		{
			Handler: k.getKey,
			Tool: mcp.NewTool("key-get",
				mcp.WithDescription("Get a specific SSH key by ID"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the SSH key")),
			),
		},
		{
			Handler: k.listKeys,
			Tool: mcp.NewTool("key-list",
				mcp.WithDescription("List SSH keys with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultKeysPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultKeysPageSize), mcp.Description("Items per page")),
			),
		},
	}
}
