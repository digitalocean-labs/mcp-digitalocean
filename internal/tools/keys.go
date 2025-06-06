package tools

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// KeysTool provides SSH key management tools
type KeysTool struct {
	client *godo.Client
}

// NewKeysTool creates a new KeysTool
func NewKeysTool(client *godo.Client) *KeysTool {
	return &KeysTool{
		client: client,
	}
}

// CreateKey creates a new SSH key
func (k *KeysTool) CreateKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}
	
	publicKey, err := req.RequireString("PublicKey")
	if err != nil {
		return nil, err
	}

	key, _, err := k.client.Keys.Create(ctx, &godo.KeyCreateRequest{
		Name:      name,
		PublicKey: publicKey,
	})
	if err != nil {
		return nil, err
	}

	jsonKey, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonKey)), nil
}

// DeleteKey deletes an SSH key by ID
func (k *KeysTool) DeleteKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyIDFloat, err := req.RequireFloat("ID")
	if err != nil {
		return nil, err
	}
	keyID := int(keyIDFloat)

	_, err = k.client.Keys.DeleteByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("SSH key deleted successfully"), nil
}

// Tools returns a list of tool functions
func (k *KeysTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: k.CreateKey,
			Tool: mcp.NewTool("digitalocean-key-create",
				mcp.WithDescription("Create a new SSH key"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the SSH key")),
				mcp.WithString("PublicKey", mcp.Required(), mcp.Description("Public key content")),
			),
		},
		{
			Handler: k.DeleteKey,
			Tool: mcp.NewTool("digitalocean-key-delete",
				mcp.WithDescription("Delete an SSH key"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the SSH key to delete")),
			),
		},
	}
}
