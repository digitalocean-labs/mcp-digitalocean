package account

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

func (k *KeysTool) createKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	name := args["Name"].(string)
	publicKey := args["PublicKey"].(string)

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

func (k *KeysTool) deleteKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyID := int(req.GetArguments()["ID"].(float64))

	_, err := k.client.Keys.DeleteByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("SSH key deleted successfully"), nil
}

// Tools returns a list of tool functions
func (k *KeysTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: k.createKey,
			Tool: mcp.NewTool("digitalocean-key-create",
				mcp.WithDescription("Create a new SSH key"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the SSH key")),
				mcp.WithString("PublicKey", mcp.Required(), mcp.Description("Public key content")),
			),
		},
		{
			Handler: k.deleteKey,
			Tool: mcp.NewTool("digitalocean-key-delete",
				mcp.WithDescription("Delete an SSH key"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("ID of the SSH key to delete")),
			),
		},
	}
}
