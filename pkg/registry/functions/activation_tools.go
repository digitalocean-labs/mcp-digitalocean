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

type ActivationTool struct {
	resolver *OWResolver
}

func NewActivationTool(resolver *OWResolver) *ActivationTool {
	return &ActivationTool{resolver: resolver}
}

func (t *ActivationTool) listActivations(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if name, ok := args["FunctionName"].(string); ok && name != "" {
		q.Set("name", name)
	}
	if limit, ok := args["Limit"].(float64); ok {
		q.Set("limit", strconv.Itoa(int(limit)))
	}
	if skip, ok := args["Skip"].(float64); ok {
		q.Set("skip", strconv.Itoa(int(skip)))
	}
	if since, ok := args["Since"].(float64); ok {
		q.Set("since", strconv.FormatInt(int64(since), 10))
	}
	if upto, ok := args["Upto"].(float64); ok {
		q.Set("upto", strconv.FormatInt(int64(upto), 10))
	}
	if docs, ok := args["IncludeDocs"].(bool); ok && docs {
		q.Set("docs", "true")
	}

	path := fmt.Sprintf("/namespaces/%s/activations", nsName)
	data, err := ow.get(ctx, path, q)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("list activations", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *ActivationTool) getActivation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	activationID, ok := args["ActivationID"].(string)
	if !ok {
		return mcp.NewToolResultError("ActivationID is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	path := fmt.Sprintf("/namespaces/%s/activations/%s", nsName, activationID)
	data, err := ow.get(ctx, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("get activation", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *ActivationTool) getActivationLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	activationID, ok := args["ActivationID"].(string)
	if !ok {
		return mcp.NewToolResultError("ActivationID is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	path := fmt.Sprintf("/namespaces/%s/activations/%s/logs", nsName, activationID)
	data, err := ow.get(ctx, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("get activation logs", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *ActivationTool) getActivationResult(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	activationID, ok := args["ActivationID"].(string)
	if !ok {
		return mcp.NewToolResultError("ActivationID is required and must be a string"), nil
	}

	ow, nsName, err := t.resolver.Resolve(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve namespace", err), nil
	}

	path := fmt.Sprintf("/namespaces/%s/activations/%s/result", nsName, activationID)
	data, err := ow.get(ctx, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("get activation result", err), nil
	}

	var result json.RawMessage = data
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json format", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *ActivationTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listActivations,
			Tool: mcp.NewTool("functions-list-activations",
				mcp.WithDescription("List activations (invocation records) for a DigitalOcean Functions namespace. Activations record every function invocation with timing, status, and optional response data."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace (from functions-list-namespaces)")),
				mcp.WithString("FunctionName", mcp.Description("Filter activations by function name")),
				mcp.WithNumber("Limit", mcp.Description("Number of activations to return (0-200, default 30). Use 0 for maximum.")),
				mcp.WithNumber("Skip", mcp.Description("Number of activations to skip for pagination")),
				mcp.WithNumber("Since", mcp.Description("Only include activations after this timestamp (milliseconds since epoch)")),
				mcp.WithNumber("Upto", mcp.Description("Only include activations before this timestamp (milliseconds since epoch)")),
				mcp.WithBoolean("IncludeDocs", mcp.Description("Include full activation details in the list response")),
			),
		},
		{
			Handler: t.getActivation,
			Tool: mcp.NewTool("functions-get-activation",
				mcp.WithDescription("Get the full activation record for a specific function invocation, including response, logs, timing, and status."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("ActivationID", mcp.Required(), mcp.Description("The activation ID")),
			),
		},
		{
			Handler: t.getActivationLogs,
			Tool: mcp.NewTool("functions-get-activation-logs",
				mcp.WithDescription("Get only the logs for a specific function activation. Useful for debugging function execution."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("ActivationID", mcp.Required(), mcp.Description("The activation ID")),
			),
		},
		{
			Handler: t.getActivationResult,
			Tool: mcp.NewTool("functions-get-activation-result",
				mcp.WithDescription("Get only the result of a specific function activation. Returns the function's return value and status."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("ActivationID", mcp.Required(), mcp.Description("The activation ID")),
			),
		},
	}
}
