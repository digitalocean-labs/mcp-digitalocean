package functions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type NamespaceTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewNamespaceTool(client func(ctx context.Context) (*godo.Client, error)) *NamespaceTool {
	return &NamespaceTool{client: client}
}

func (t *NamespaceTool) listNamespaces(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	namespaces, _, err := client.Functions.ListNamespaces(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("list namespaces", err), nil
	}

	out, err := json.MarshalIndent(namespaces, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *NamespaceTool) getNamespace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nsID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	ns, _, err := client.Functions.GetNamespace(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("get namespace", err), nil
	}

	out, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *NamespaceTool) createNamespace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	label, ok := args["Label"].(string)
	if !ok {
		return mcp.NewToolResultError("Label is required and must be a string"), nil
	}
	region, ok := args["Region"].(string)
	if !ok {
		return mcp.NewToolResultError("Region is required and must be a string"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	ns, _, err := client.Functions.CreateNamespace(ctx, &godo.FunctionsNamespaceCreateRequest{
		Label:  label,
		Region: region,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("create namespace", err), nil
	}

	out, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *NamespaceTool) deleteNamespace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nsID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Functions.DeleteNamespace(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("delete namespace", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Namespace %s deleted successfully", nsID)), nil
}

func (t *NamespaceTool) listAccessKeys(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nsID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	keys, _, err := client.Functions.ListAccessKeys(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("list access keys", err), nil
	}

	out, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *NamespaceTool) createAccessKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	name, ok := args["Name"].(string)
	if !ok {
		return mcp.NewToolResultError("Name is required and must be a string"), nil
	}

	createReq := &godo.FunctionsAccessKeyCreateRequest{Name: name}
	if expiresIn, ok := args["ExpiresIn"].(string); ok && expiresIn != "" {
		createReq.ExpiresIn = expiresIn
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	key, _, err := client.Functions.CreateAccessKey(ctx, nsID, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("create access key", err), nil
	}

	out, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *NamespaceTool) deleteAccessKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	keyID, ok := args["KeyID"].(string)
	if !ok {
		return mcp.NewToolResultError("KeyID is required and must be a string"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Functions.DeleteAccessKey(ctx, nsID, keyID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("delete access key", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Access key %s deleted successfully", keyID)), nil
}

func (t *NamespaceTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listNamespaces,
			Tool: mcp.NewTool("functions-list-namespaces",
				mcp.WithDescription("List all DigitalOcean Functions namespaces. Returns namespace metadata including api_host, region, label, and UUID."),
			),
		},
		{
			Handler: t.getNamespace,
			Tool: mcp.NewTool("functions-get-namespace",
				mcp.WithDescription("Get a DigitalOcean Functions namespace by ID. Returns full namespace details including api_host and key for data plane access."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
			),
		},
		{
			Handler: t.createNamespace,
			Tool: mcp.NewTool("functions-create-namespace",
				mcp.WithDescription("Create a new DigitalOcean Functions namespace."),
				mcp.WithString("Label", mcp.Required(), mcp.Description("A human-readable label for the namespace")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("The region slug where the namespace will be created (e.g. nyc1, sfo1)")),
			),
		},
		{
			Handler: t.deleteNamespace,
			Tool: mcp.NewTool("functions-delete-namespace",
				mcp.WithDescription("Delete a DigitalOcean Functions namespace. This permanently removes the namespace and all its functions, packages, and triggers."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace to delete")),
				mcp.WithDestructiveHintAnnotation(true),
			),
		},
		{
			Handler: t.listAccessKeys,
			Tool: mcp.NewTool("functions-list-access-keys",
				mcp.WithDescription("List access keys for a DigitalOcean Functions namespace. There is a limit of 200 access keys per account."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
			),
		},
		{
			Handler: t.createAccessKey,
			Tool: mcp.NewTool("functions-create-access-key",
				mcp.WithDescription("Create an access key for a DigitalOcean Functions namespace. The secret is only returned once at creation time. WARNING: There is a limit of 200 access keys per account; use judiciously."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("A name for the access key")),
				mcp.WithString("ExpiresIn", mcp.Description("Optional expiration duration (e.g. '24h', '7d'). Minimum is 1h. Omit for non-expiring key.")),
			),
		},
		{
			Handler: t.deleteAccessKey,
			Tool: mcp.NewTool("functions-delete-access-key",
				mcp.WithDescription("Delete an access key for a DigitalOcean Functions namespace."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("KeyID", mcp.Required(), mcp.Description("The ID of the access key to delete")),
				mcp.WithDestructiveHintAnnotation(true),
			),
		},
	}
}
