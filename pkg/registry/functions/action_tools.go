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

type ActionTool struct {
	resolver *OWResolver
}

func NewActionTool(resolver *OWResolver) *ActionTool {
	return &ActionTool{resolver: resolver}
}

// actionPath builds the OW API path for an action, handling the optional
// package prefix. If PackageName is non-empty the path is
// /namespaces/{ns}/actions/{pkg}/{action}, otherwise /namespaces/{ns}/actions/{action}.
func actionPath(ns, pkg, action string) string {
	if pkg != "" {
		return fmt.Sprintf("/namespaces/%s/actions/%s/%s", ns, pkg, action)
	}
	return fmt.Sprintf("/namespaces/%s/actions/%s", ns, action)
}

func (t *ActionTool) listActions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	path := fmt.Sprintf("/namespaces/%s/actions", nsName)
	data, err := ow.get(ctx, path, q)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("list actions", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *ActionTool) getAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	actionName, ok := args["ActionName"].(string)
	if !ok {
		return mcp.NewToolResultError("ActionName is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	q := url.Values{}
	if code, ok := args["IncludeCode"].(bool); ok && code {
		q.Set("code", "true")
	}

	pkgName, _ := args["PackageName"].(string)
	path := actionPath(nsName, pkgName, actionName)

	data, err := ow.get(ctx, path, q)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("get action", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *ActionTool) createOrUpdateAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	actionName, ok := args["ActionName"].(string)
	if !ok {
		return mcp.NewToolResultError("ActionName is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	body := map[string]any{}

	// Build exec object
	exec := map[string]any{}
	if kind, ok := args["Kind"].(string); ok {
		exec["kind"] = kind
	}
	if code, ok := args["Code"].(string); ok {
		exec["code"] = code
	}
	if image, ok := args["Image"].(string); ok {
		exec["image"] = image
	}
	if main, ok := args["Main"].(string); ok {
		exec["main"] = main
	}
	if components, ok := args["Components"].([]any); ok {
		exec["components"] = components
	}
	if len(exec) > 0 {
		body["exec"] = exec
	}

	// Build limits object
	limits := map[string]any{}
	if timeout, ok := args["Timeout"].(float64); ok {
		limits["timeout"] = int(timeout)
	}
	if memory, ok := args["Memory"].(float64); ok {
		limits["memory"] = int(memory)
	}
	if logs, ok := args["Logs"].(float64); ok {
		limits["logs"] = int(logs)
	}
	if len(limits) > 0 {
		body["limits"] = limits
	}

	if annotations, ok := args["Annotations"].([]any); ok {
		body["annotations"] = annotations
	}
	if parameters, ok := args["Parameters"].([]any); ok {
		body["parameters"] = parameters
	}

	q := url.Values{"overwrite": {"true"}}

	pkgName, _ := args["PackageName"].(string)
	path := actionPath(nsName, pkgName, actionName)

	data, err := ow.put(ctx, path, q, body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("create/update action", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *ActionTool) deleteAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	actionName, ok := args["ActionName"].(string)
	if !ok {
		return mcp.NewToolResultError("ActionName is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	pkgName, _ := args["PackageName"].(string)
	path := actionPath(nsName, pkgName, actionName)

	_, err = ow.del(ctx, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("delete action", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Action %s deleted successfully", actionName)), nil
}

func (t *ActionTool) invokeAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	actionName, ok := args["ActionName"].(string)
	if !ok {
		return mcp.NewToolResultError("ActionName is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	q := url.Values{}

	blocking := true
	if v, ok := args["Blocking"].(bool); ok {
		blocking = v
	}
	q.Set("blocking", strconv.FormatBool(blocking))

	resultOnly := false
	if v, ok := args["Result"].(bool); ok {
		resultOnly = v
	}
	q.Set("result", strconv.FormatBool(resultOnly))

	if timeout, ok := args["InvokeTimeout"].(float64); ok {
		q.Set("timeout", strconv.Itoa(int(timeout)))
	}

	var payload any
	if p, ok := args["Payload"].(map[string]any); ok {
		payload = p
	}

	pkgName, _ := args["PackageName"].(string)
	path := actionPath(nsName, pkgName, actionName)

	data, err := ow.post(ctx, path, q, payload)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invoke action", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *ActionTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listActions,
			Tool: mcp.NewTool("functions-list-actions",
				mcp.WithDescription("List all actions in a DigitalOcean Functions namespace. Returns action metadata including name, namespace, version, and limits."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace (from functions-list-namespaces)")),
				mcp.WithNumber("Limit", mcp.Description("Number of actions to return (0-200, default 30). Use 0 for maximum.")),
				mcp.WithNumber("Skip", mcp.Description("Number of actions to skip for pagination")),
			),
		},
		{
			Handler: t.getAction,
			Tool: mcp.NewTool("functions-get-action",
				mcp.WithDescription("Get detailed information about a specific action in a DigitalOcean Functions namespace, including its configuration and optionally its source code."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("ActionName", mcp.Required(), mcp.Description("The name of the action")),
				mcp.WithString("PackageName", mcp.Description("The package containing the action, if applicable")),
				mcp.WithBoolean("IncludeCode", mcp.Description("Whether to include the action's source code in the response. Default is false.")),
			),
		},
		{
			Handler: t.createOrUpdateAction,
			Tool: mcp.NewTool("functions-create-or-update-action",
				mcp.WithDescription("Create or update an action in a DigitalOcean Functions namespace. If the action already exists it will be overwritten."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("ActionName", mcp.Required(), mcp.Description("The name of the action to create or update")),
				mcp.WithString("PackageName", mcp.Description("The package to place the action in, if applicable")),
				mcp.WithString("Kind", mcp.Description("Runtime kind (e.g. nodejs:20, python:3.11, go:default, php:default, blackbox, sequence)")),
				mcp.WithString("Code", mcp.Description("The source code for the action (when kind is not blackbox)")),
				mcp.WithString("Image", mcp.Description("Container image name (when kind is blackbox)")),
				mcp.WithString("Main", mcp.Description("Main entrypoint of the action code")),
				mcp.WithArray("Components", mcp.Description("For sequence actions, the list of action names in order"), mcp.Items(map[string]any{"type": "string"})),
				mcp.WithNumber("Timeout", mcp.Description("Action timeout in milliseconds (default 60000)")),
				mcp.WithNumber("Memory", mcp.Description("Action memory in megabytes (default 256)")),
				mcp.WithNumber("Logs", mcp.Description("Max log size in megabytes (default 10)")),
				mcp.WithArray("Annotations", mcp.Description("Key-value annotations for the action"), mcp.Items(map[string]any{"type": "object"})),
				mcp.WithArray("Parameters", mcp.Description("Default parameter bindings for the action"), mcp.Items(map[string]any{"type": "object"})),
			),
		},
		{
			Handler: t.deleteAction,
			Tool: mcp.NewTool("functions-delete-action",
				mcp.WithDescription("Delete an action from a DigitalOcean Functions namespace."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("ActionName", mcp.Required(), mcp.Description("The name of the action to delete")),
				mcp.WithString("PackageName", mcp.Description("The package containing the action, if applicable")),
				mcp.WithDestructiveHintAnnotation(true),
			),
		},
		{
			Handler: t.invokeAction,
			Tool: mcp.NewTool("functions-invoke-action",
				mcp.WithDescription("Invoke a function action in a DigitalOcean Functions namespace. By default this is a blocking invocation that waits for the result."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("ActionName", mcp.Required(), mcp.Description("The name of the action to invoke")),
				mcp.WithString("PackageName", mcp.Description("The package containing the action, if applicable")),
				mcp.WithBoolean("Blocking", mcp.Description("Whether to wait for the invocation to complete. Default is true.")),
				mcp.WithBoolean("Result", mcp.Description("Return only the result of a blocking activation. Default is false.")),
				mcp.WithNumber("InvokeTimeout", mcp.Description("Max wait time in milliseconds for a blocking response (default/max 60000)")),
				mcp.WithObject("Payload", mcp.Description("JSON payload to pass as parameters to the action")),
			),
		},
	}
}
