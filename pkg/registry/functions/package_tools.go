package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type PackageTool struct {
	resolver *OWResolver
}

func NewPackageTool(resolver *OWResolver) *PackageTool {
	return &PackageTool{resolver: resolver}
}

func (t *PackageTool) listPackages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	q := url.Values{}
	if limit, ok := args["Limit"].(float64); ok {
		q.Set("limit", strconv.Itoa(int(limit)))
	}
	if skip, ok := args["Skip"].(float64); ok {
		q.Set("skip", strconv.Itoa(int(skip)))
	}
	if public, ok := args["Public"].(bool); ok && public {
		q.Set("public", "true")
	}

	path := fmt.Sprintf("/namespaces/%s/packages", nsName)
	data, err := ow.get(ctx, path, q)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("list packages", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *PackageTool) getPackage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	pkgName, ok := args["PackageName"].(string)
	if !ok {
		return mcp.NewToolResultError("PackageName is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	path := fmt.Sprintf("/namespaces/%s/packages/%s", nsName, pkgName)
	data, err := ow.get(ctx, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("get package", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *PackageTool) createOrUpdatePackage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	pkgName, ok := args["PackageName"].(string)
	if !ok {
		return mcp.NewToolResultError("PackageName is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	body := map[string]any{}

	if publish, ok := args["Publish"].(bool); ok {
		body["publish"] = publish
	}
	if annotations, ok := args["Annotations"].([]any); ok {
		body["annotations"] = annotations
	}
	if parameters, ok := args["Parameters"].([]any); ok {
		body["parameters"] = parameters
	}
	if binding, ok := args["Binding"].(map[string]any); ok {
		body["binding"] = binding
	}

	q := url.Values{"overwrite": {"true"}}
	path := fmt.Sprintf("/namespaces/%s/packages/%s", nsName, pkgName)

	data, err := ow.put(ctx, path, q, body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("create/update package", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *PackageTool) deletePackage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	pkgName, ok := args["PackageName"].(string)
	if !ok {
		return mcp.NewToolResultError("PackageName is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	q := url.Values{}
	if force, ok := args["Force"].(bool); ok && force {
		q.Set("force", "true")
	}

	path := fmt.Sprintf("/namespaces/%s/packages/%s", nsName, pkgName)
	_, err = ow.del(ctx, path, q)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("delete package", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Package %s deleted successfully", pkgName)), nil
}

func (t *PackageTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listPackages,
			Tool: mcp.NewTool("functions-list-packages",
				mcp.WithDescription("List all packages in a DigitalOcean Functions namespace. Packages group related actions together."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace (from functions-list-namespaces)")),
				mcp.WithNumber("Limit", mcp.Description("Number of packages to return (0-200, default 30). Use 0 for maximum.")),
				mcp.WithNumber("Skip", mcp.Description("Number of packages to skip for pagination")),
				mcp.WithBoolean("Public", mcp.Description("Include publicly shared packages in the result")),
			),
		},
		{
			Handler: t.getPackage,
			Tool: mcp.NewTool("functions-get-package",
				mcp.WithDescription("Get detailed information about a specific package in a DigitalOcean Functions namespace, including its actions, parameters, and annotations."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("PackageName", mcp.Required(), mcp.Description("The name of the package")),
			),
		},
		{
			Handler: t.createOrUpdatePackage,
			Tool: mcp.NewTool("functions-create-or-update-package",
				mcp.WithDescription("Create or update a package in a DigitalOcean Functions namespace. Packages are used to group related actions."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("PackageName", mcp.Required(), mcp.Description("The name of the package to create or update")),
				mcp.WithBoolean("Publish", mcp.Description("Whether to make the package publicly accessible")),
				mcp.WithArray("Annotations", mcp.Description("Key-value annotations for the package"), mcp.Items(map[string]any{"type": "object"})),
				mcp.WithArray("Parameters", mcp.Description("Default parameter bindings for actions in the package"), mcp.Items(map[string]any{"type": "object"})),
				mcp.WithObject("Binding", mcp.Description("Package binding with 'namespace' and 'name' fields to bind to another package")),
			),
		},
		{
			Handler: t.deletePackage,
			Tool: mcp.NewTool("functions-delete-package",
				mcp.WithDescription("Delete a package from a DigitalOcean Functions namespace."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("PackageName", mcp.Required(), mcp.Description("The name of the package to delete")),
				mcp.WithBoolean("Force", mcp.Description("Force delete the package even if it contains actions. Default is false.")),
				mcp.WithDestructiveHintAnnotation(true),
			),
		},
	}
}
