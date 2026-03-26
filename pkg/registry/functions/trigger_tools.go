package functions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type TriggerTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewTriggerTool(client func(ctx context.Context) (*godo.Client, error)) *TriggerTool {
	return &TriggerTool{client: client}
}

func (t *TriggerTool) listTriggers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nsID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	triggers, _, err := client.Functions.ListTriggers(ctx, nsID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("list triggers", err), nil
	}

	out, err := json.MarshalIndent(triggers, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *TriggerTool) getTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	triggerName, ok := args["TriggerName"].(string)
	if !ok {
		return mcp.NewToolResultError("TriggerName is required and must be a string"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	trigger, _, err := client.Functions.GetTrigger(ctx, nsID, triggerName)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("get trigger", err), nil
	}

	out, err := json.MarshalIndent(trigger, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *TriggerTool) createTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	name, ok := args["Name"].(string)
	if !ok {
		return mcp.NewToolResultError("Name is required and must be a string"), nil
	}
	function, ok := args["Function"].(string)
	if !ok {
		return mcp.NewToolResultError("Function is required and must be a string"), nil
	}
	cron, ok := args["Cron"].(string)
	if !ok {
		return mcp.NewToolResultError("Cron is required and must be a string"), nil
	}

	isEnabled := true
	if v, ok := args["IsEnabled"].(bool); ok {
		isEnabled = v
	}

	createReq := &godo.FunctionsTriggerCreateRequest{
		Name:      name,
		Type:      "SCHEDULED",
		Function:  function,
		IsEnabled: isEnabled,
		ScheduledDetails: &godo.TriggerScheduledDetails{
			Cron: cron,
		},
	}

	if bodyArg, ok := args["Body"].(map[string]any); ok {
		createReq.ScheduledDetails.Body = bodyArg
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	trigger, _, err := client.Functions.CreateTrigger(ctx, nsID, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("create trigger", err), nil
	}

	out, err := json.MarshalIndent(trigger, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *TriggerTool) updateTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	triggerName, ok := args["TriggerName"].(string)
	if !ok {
		return mcp.NewToolResultError("TriggerName is required and must be a string"), nil
	}

	updateReq := &godo.FunctionsTriggerUpdateRequest{}

	if v, ok := args["IsEnabled"].(bool); ok {
		updateReq.IsEnabled = &v
	}

	if cron, ok := args["Cron"].(string); ok && cron != "" {
		if updateReq.ScheduledDetails == nil {
			updateReq.ScheduledDetails = &godo.TriggerScheduledDetails{}
		}
		updateReq.ScheduledDetails.Cron = cron
	}

	if bodyArg, ok := args["Body"].(map[string]any); ok {
		if updateReq.ScheduledDetails == nil {
			updateReq.ScheduledDetails = &godo.TriggerScheduledDetails{}
		}
		updateReq.ScheduledDetails.Body = bodyArg
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	trigger, _, err := client.Functions.UpdateTrigger(ctx, nsID, triggerName, updateReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("update trigger", err), nil
	}

	out, err := json.MarshalIndent(trigger, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("json marshal", err), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func (t *TriggerTool) deleteTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	nsID, ok := args["NamespaceID"].(string)
	if !ok {
		return mcp.NewToolResultError("NamespaceID is required and must be a string"), nil
	}
	triggerName, ok := args["TriggerName"].(string)
	if !ok {
		return mcp.NewToolResultError("TriggerName is required and must be a string"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Functions.DeleteTrigger(ctx, nsID, triggerName)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("delete trigger", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Trigger %s deleted successfully", triggerName)), nil
}

func (t *TriggerTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listTriggers,
			Tool: mcp.NewTool("functions-list-triggers",
				mcp.WithDescription("List all triggers for a DigitalOcean Functions namespace."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
			),
		},
		{
			Handler: t.getTrigger,
			Tool: mcp.NewTool("functions-get-trigger",
				mcp.WithDescription("Get a specific trigger in a DigitalOcean Functions namespace."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("TriggerName", mcp.Required(), mcp.Description("The name of the trigger")),
			),
		},
		{
			Handler: t.createTrigger,
			Tool: mcp.NewTool("functions-create-trigger",
				mcp.WithDescription("Create a scheduled trigger for a function in a DigitalOcean Functions namespace. Currently only SCHEDULED type triggers are supported."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("A name for the trigger")),
				mcp.WithString("Function", mcp.Required(), mcp.Description("The name of the function to invoke")),
				mcp.WithString("Cron", mcp.Required(), mcp.Description("A cron expression defining the schedule (e.g. '*/5 * * * *' for every 5 minutes)")),
				mcp.WithBoolean("IsEnabled", mcp.Description("Whether the trigger is enabled. Defaults to true.")),
				mcp.WithObject("Body", mcp.Description("Optional JSON payload to pass to the function on each invocation")),
			),
		},
		{
			Handler: t.updateTrigger,
			Tool: mcp.NewTool("functions-update-trigger",
				mcp.WithDescription("Update a trigger in a DigitalOcean Functions namespace. You can enable/disable the trigger or change the cron schedule."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("TriggerName", mcp.Required(), mcp.Description("The name of the trigger to update")),
				mcp.WithBoolean("IsEnabled", mcp.Description("Whether the trigger should be enabled or disabled")),
				mcp.WithString("Cron", mcp.Description("Updated cron expression for the schedule")),
				mcp.WithObject("Body", mcp.Description("Updated JSON payload to pass to the function")),
			),
		},
		{
			Handler: t.deleteTrigger,
			Tool: mcp.NewTool("functions-delete-trigger",
				mcp.WithDescription("Delete a trigger from a DigitalOcean Functions namespace."),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The UUID of the namespace")),
				mcp.WithString("TriggerName", mcp.Required(), mcp.Description("The name of the trigger to delete")),
				mcp.WithDestructiveHintAnnotation(true),
			),
		},
	}
}
