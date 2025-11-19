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
	defaultActionsPageSize = 30
	defaultActionsPage     = 1
)

// ActionTools provides tool-based handlers for DigitalOcean Actions.
type ActionTools struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewActionTools creates a new ActionTools instance.
func NewActionTools(client func(ctx context.Context) (*godo.Client, error)) *ActionTools {
	return &ActionTools{client: client}
}

// getAction retrieves a specific action by its ID.
func (a *ActionTools) getAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(float64)
	if !ok {
		return mcp.NewToolResultError("Action ID is required"), nil
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	action, _, err := client.Actions.Get(ctx, int(id))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// listActions lists actions with pagination support.
func (a *ActionTools) listActions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultActionsPage
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultActionsPageSize
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	actions, _, err := client.Actions.List(ctx, &godo.ListOptions{Page: int(page), PerPage: int(perPage)})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonData, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for actions.
func (a *ActionTools) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: a.getAction,
			Tool: mcp.NewTool("action-get",
				mcp.WithDescription("Get a specific action by ID"),
				mcp.WithNumber("ID", mcp.Required(), mcp.Description("Action ID")),
			),
		},
		{
			Handler: a.listActions,
			Tool: mcp.NewTool("action-list",
				mcp.WithDescription("List actions with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultActionsPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultActionsPageSize), mcp.Description("Items per page")),
			),
		},
	}
}
