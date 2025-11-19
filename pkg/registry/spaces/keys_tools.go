package spaces

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type KeysTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewSpacesKeysTool(client func(ctx context.Context) (*godo.Client, error)) *KeysTool {
	return &KeysTool{
		client: client,
	}
}

func (s *KeysTool) createSpacesKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	nameArg, ok := args["Name"]
	if !ok {
		return mcp.NewToolResultError("Name parameter is required"), nil
	}

	name, ok := nameArg.(string)
	if !ok {
		return mcp.NewToolResultError("Name must be a string"), nil
	}

	if name == "" {
		return mcp.NewToolResultError("Name cannot be empty"), nil
	}

	createRequest := &godo.SpacesKeyCreateRequest{
		Name: name,
		Grants: []*godo.Grant{
			{
				Bucket:     "",
				Permission: godo.SpacesKeyFullAccess,
			},
		},
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	key, _, err := client.SpacesKeys.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonKey, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonKey)), nil
}

func (s *KeysTool) updateSpacesKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	accessKeyArg, ok := args["AccessKey"]
	if !ok {
		return mcp.NewToolResultError("AccessKey parameter is required"), nil
	}

	accessKey, ok := accessKeyArg.(string)
	if !ok {
		return mcp.NewToolResultError("AccessKey must be a string"), nil
	}

	if accessKey == "" {
		return mcp.NewToolResultError("AccessKey cannot be empty"), nil
	}

	nameArg, ok := args["Name"]
	if !ok {
		return mcp.NewToolResultError("Name parameter is required"), nil
	}

	name, ok := nameArg.(string)
	if !ok {
		return mcp.NewToolResultError("Name must be a string"), nil
	}

	if name == "" {
		return mcp.NewToolResultError("Name cannot be empty"), nil
	}

	updateRequest := &godo.SpacesKeyUpdateRequest{
		Name: name,
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	key, _, err := client.SpacesKeys.Update(ctx, accessKey, updateRequest)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonKey, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonKey)), nil
}

func (s *KeysTool) deleteSpacesKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	accessKeyArg, ok := args["AccessKey"]
	if !ok {
		return mcp.NewToolResultError("AccessKey parameter is required"), nil
	}

	accessKey, ok := accessKeyArg.(string)
	if !ok {
		return mcp.NewToolResultError("AccessKey must be a string"), nil
	}

	if accessKey == "" {
		return mcp.NewToolResultError("AccessKey cannot be empty"), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.SpacesKeys.Delete(ctx, accessKey)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText("Spaces key deleted successfully"), nil
}

func (s *KeysTool) listSpacesKeys(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Set up pagination options with defaults
	listOpts := &godo.ListOptions{
		Page:    1,  // Default to page 1
		PerPage: 10, // Default to 10 per page
	}

	// Handle Page parameter
	if pageRaw, ok := args["Page"]; ok {
		if pageFloat, ok := pageRaw.(float64); ok {
			listOpts.Page = int(pageFloat)
		} else {
			return mcp.NewToolResultError("Page must be a number"), nil
		}
	}

	// Handle PerPage parameter
	if perPageRaw, ok := args["PerPage"]; ok {
		if perPageFloat, ok := perPageRaw.(float64); ok {
			listOpts.PerPage = int(perPageFloat)
		} else {
			return mcp.NewToolResultError("PerPage must be a number"), nil
		}
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	keys, resp, err := client.SpacesKeys.List(ctx, listOpts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Create response with pagination info
	result := struct {
		Keys []*godo.SpacesKey `json:"keys"`
		Meta *godo.Meta        `json:"meta,omitempty"`
	}{
		Keys: keys,
		Meta: resp.Meta,
	}

	jsonKeys, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonKeys)), nil
}

func (s *KeysTool) getSpacesKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	accessKeyArg, ok := args["AccessKey"]
	if !ok {
		return mcp.NewToolResultError("AccessKey parameter is required"), nil
	}

	accessKey, ok := accessKeyArg.(string)
	if !ok {
		return mcp.NewToolResultError("AccessKey must be a string"), nil
	}

	if accessKey == "" {
		return mcp.NewToolResultError("AccessKey cannot be empty"), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	key, _, err := client.SpacesKeys.Get(ctx, accessKey)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonKey, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonKey)), nil
}

func (s *KeysTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.listSpacesKeys,
			Tool: mcp.NewTool("spaces-key-list",
				mcp.WithDescription("List all Spaces keys"),
				mcp.WithNumber("Page", mcp.Required(), mcp.DefaultNumber(1), mcp.Description("Page number for pagination")),
				mcp.WithNumber("PerPage", mcp.Required(), mcp.DefaultNumber(10), mcp.Description("Number of items per page"), mcp.Max(100)),
			),
		},
		{
			Handler: s.getSpacesKey,
			Tool: mcp.NewTool("spaces-key-get",
				mcp.WithDescription("Get a specific Spaces key"),
				mcp.WithString("AccessKey", mcp.Required(), mcp.Description("Access Key of the Spaces key to retrieve")),
			),
		},
		{
			Handler: s.createSpacesKey,
			Tool: mcp.NewTool("spaces-key-create",
				mcp.WithDescription("Create a new Spaces key. SECURITY WARNING: The returned secret key should NEVER be added to files or committed to source control. Always store the secret key in environment variables (e.g., DO_SPACES_SECRET_KEY) and access it securely at runtime. The secret key should be treated as highly sensitive credential information and should not be displayed in logs or output when possible."),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the Spaces key")),
			),
		},
		{
			Handler: s.updateSpacesKey,
			Tool: mcp.NewTool("spaces-key-update",
				mcp.WithDescription("Update an existing Spaces key"),
				mcp.WithString("AccessKey", mcp.Required(), mcp.Description("Access Key of the Spaces key to update")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("New name for the Spaces key")),
			),
		},
		{
			Handler: s.deleteSpacesKey,
			Tool: mcp.NewTool("spaces-key-delete",
				mcp.WithDescription("Delete a Spaces key"),
				mcp.WithString("AccessKey", mcp.Required(), mcp.Description("Access Key of the Spaces key to delete")),
			),
		},
	}
}
