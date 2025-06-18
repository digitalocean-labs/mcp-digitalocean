// database.go
// Tools for wrapping DigitalOcean Database APIs.
// Add functions here to interact with DigitalOcean's managed database services.

package tools

import (
	"context"
	"encoding/json"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DatabaseTool wraps DigitalOcean database API operations.
type DatabaseTool struct {
	client *godo.Client
}

// NewDatabaseTool creates a new DatabaseTool with the given godo client.
func NewDatabaseTool(client *godo.Client) *DatabaseTool {
	return &DatabaseTool{client: client}
}

// GetDatabase gets a database cluster by UUID.
func (dt *DatabaseTool) GetDatabase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.Params.Arguments["UUID"].(string)
	db, _, err := dt.client.Databases.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	jsonDb, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(jsonDb)), nil
}

// Tools registers all tool handlers and their schemas.
func (dt *DatabaseTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: dt.GetDatabase,
			Tool: mcp.NewTool("digitalocean-database-get",
				mcp.WithDescription("Get a database cluster by UUID"),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("UUID of the database cluster")),
			),
		},
	}
}
