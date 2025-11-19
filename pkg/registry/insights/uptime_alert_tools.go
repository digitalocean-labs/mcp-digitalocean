package insights

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultAlertsPageSize = 20
	defaultAlertsPage     = 1
)

// UptimeCheckAlertTool provides UptimeCheck and Alert management tools
type UptimeCheckAlertTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewUptimeCheckAlertTool creates a new UptimeCheck tool
func NewUptimeCheckAlertTool(client func(ctx context.Context) (*godo.Client, error)) *UptimeCheckAlertTool {
	return &UptimeCheckAlertTool{
		client: client,
	}
}

// getUptimeCheck fetches UptimeCheck information by ID
func (u *UptimeCheckAlertTool) getUptimeCheckAlert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	checkId, ok := req.GetArguments()["CheckID"].(string)
	if !ok || checkId == "" {
		return mcp.NewToolResultError("Uptime CheckID is required"), nil
	}

	alertId, ok := req.GetArguments()["AlertID"].(string)
	if !ok || alertId == "" {
		return mcp.NewToolResultError("UptimeCheck AlertID is required"), nil
	}

	client, err := u.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uptimeCheckAlert, _, err := client.UptimeChecks.GetAlert(ctx, checkId, alertId)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUptimeCheckAlert, err := json.MarshalIndent(uptimeCheckAlert, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUptimeCheckAlert)), nil
}

// listUptimeCheckAlerts lists UptimeChecks with pagination support
func (u *UptimeCheckAlertTool) listUptimeCheckAlerts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["CheckID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Uptime CheckID is required"), nil
	}

	page := defaultAlertsPage
	perPage := defaultAlertsPageSize
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := u.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uptimeCheckAlerts, _, err := client.UptimeChecks.ListAlerts(ctx, id, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonUptimeCheckAlerts, err := json.MarshalIndent(uptimeCheckAlerts, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonUptimeCheckAlerts)), nil
}

// createUptimeCheck creates a new UptimeCheck
func (u *UptimeCheckAlertTool) createUptimeCheckAlert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	checkID, ok := req.GetArguments()["CheckID"].(string)
	if !ok || checkID == "" {
		return mcp.NewToolResultError("Uptime CheckID is required"), nil
	}
	name := req.GetArguments()["Name"].(string)
	alertType := req.GetArguments()["Type"].(string)
	var threshold int
	if vArg, ok := req.GetArguments()["Threshold"].(float64); ok && int(vArg) > 0 {
		threshold = int(vArg)
	}
	period := req.GetArguments()["Period"].(string)
	comparison := req.GetArguments()["Comparison"].(string)
	emailsRaw, ok := req.GetArguments()["Emails"]
	var emails []string
	if ok && emailsRaw != nil {
		bytes, _ := json.Marshal(emailsRaw)
		_ = json.Unmarshal(bytes, &emails)
	}

	slackDetailsRaw, ok := req.GetArguments()["SlackDetails"]
	var slackDetails []godo.SlackDetails
	if ok && slackDetailsRaw != nil {
		// Marshal the interface{} to JSON
		slackDetailsBytes, err := json.Marshal(slackDetailsRaw)
		if err != nil {
			return mcp.NewToolResultError("Invalid SlackDetails format"), nil
		}
		// Unmarshal JSON to your struct
		if err := json.Unmarshal(slackDetailsBytes, &slackDetails); err != nil {
			return mcp.NewToolResultError("Failed to parse SlackDetails"), nil
		}
	}

	createRequest := &godo.CreateUptimeAlertRequest{
		Name:       name,
		Type:       alertType,
		Threshold:  threshold,
		Period:     period,
		Comparison: godo.UptimeAlertComp(comparison),
		Notifications: &godo.Notifications{
			Email: emails,
			Slack: slackDetails,
		},
	}

	client, err := u.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uptimeCheckAlert, _, err := client.UptimeChecks.CreateAlert(ctx, checkID, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUptimeCheckAlert, err := json.MarshalIndent(uptimeCheckAlert, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUptimeCheckAlert)), nil
}

// updateUptimeCheck updates a existing UptimeCheck
func (u *UptimeCheckAlertTool) updateUptimeCheckAlert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	checkID, ok := req.GetArguments()["CheckID"].(string)
	if !ok || checkID == "" {
		return mcp.NewToolResultError("Uptime CheckID is required"), nil
	}

	alertId, ok := req.GetArguments()["AlertID"].(string)
	if !ok || alertId == "" {
		return mcp.NewToolResultError("UptimeCheck AlertID is required"), nil
	}

	name := req.GetArguments()["Name"].(string)
	alertType := req.GetArguments()["Type"].(string)
	var threshold int
	if vArg, ok := req.GetArguments()["Threshold"].(float64); ok && int(vArg) > 0 {
		threshold = int(vArg)
	}
	period := req.GetArguments()["Period"].(string)
	comparison := req.GetArguments()["Comparison"].(string)
	emailsRaw, ok := req.GetArguments()["Emails"]
	var emails []string
	if ok && emailsRaw != nil {
		bytes, _ := json.Marshal(emailsRaw)
		_ = json.Unmarshal(bytes, &emails)
	}

	slackDetailsRaw, ok := req.GetArguments()["SlackDetails"]
	var slackDetails []godo.SlackDetails
	if ok && slackDetailsRaw != nil {
		// Marshal the interface{} to JSON
		slackDetailsBytes, err := json.Marshal(slackDetailsRaw)
		if err != nil {
			return mcp.NewToolResultError("Invalid SlackDetails format"), nil
		}
		// Unmarshal JSON to your struct
		if err := json.Unmarshal(slackDetailsBytes, &slackDetails); err != nil {
			return mcp.NewToolResultError("Failed to parse SlackDetails"), nil
		}
	}

	updateRequest := &godo.UpdateUptimeAlertRequest{
		Name:       name,
		Type:       alertType,
		Threshold:  threshold,
		Period:     period,
		Comparison: godo.UptimeAlertComp(comparison),
		Notifications: &godo.Notifications{
			Email: emails,
			Slack: slackDetails,
		},
	}

	client, err := u.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	uptimeCheck, _, err := client.UptimeChecks.UpdateAlert(ctx, checkID, alertId, updateRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUptimeCheck, err := json.MarshalIndent(uptimeCheck, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUptimeCheck)), nil
}

// deleteUptimeCheck deletes a UptimeCheck
func (u *UptimeCheckAlertTool) deleteUptimeCheckAlert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uptimeCheckID, ok := req.GetArguments()["CheckID"].(string)

	if !ok || uptimeCheckID == "" {
		return mcp.NewToolResultError("Uptime CheckID is required"), nil
	}
	alertId, ok := req.GetArguments()["AlertID"].(string)
	if !ok || alertId == "" {
		return mcp.NewToolResultError("UptimeCheck AlertID is required"), nil
	}

	client, err := u.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.UptimeChecks.DeleteAlert(ctx, uptimeCheckID, alertId)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("uptimeCheck alert deleted successfully"), nil
}

// Tools returns a list of tool functions
func (c *UptimeCheckAlertTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: c.getUptimeCheckAlert,
			Tool: mcp.NewTool("uptimecheck-alert-get",
				mcp.WithDescription("Get UptimeCheck Alert information by CheckID and AlertID"),
				mcp.WithString("CheckID", mcp.Required(), mcp.Description("A unique identifier for a check")),
				mcp.WithString("AlertID", mcp.Required(), mcp.Description("A unique identifier for a alert")),
			),
		},
		{
			Handler: c.listUptimeCheckAlerts,
			Tool: mcp.NewTool("uptimecheck-alert-list",
				mcp.WithDescription("List UptimeChecks Alerts with pagination"),
				mcp.WithString("CheckID", mcp.Required(), mcp.Description("A unique identifier for a check")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultAlertsPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultAlertsPageSize), mcp.Description("Items per page")),
			),
		},
		{
			Handler: c.createUptimeCheckAlert,
			Tool: mcp.NewTool("uptimecheck-alert-create",
				mcp.WithDescription("Create a new UptimeCheck"),
				mcp.WithString("CheckID", mcp.Required(), mcp.Description("A unique identifier for a check")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the UptimeCheck Alert")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("latency, down, down_global or ssl_expiry. type of the UptimeCheck Alert")),
				mcp.WithNumber("Threshold", mcp.Description("The threshold at which the alert will enter a trigger state. The specific threshold is dependent on the alert type")),
				mcp.WithString("Comparison", mcp.Description("The comparison operator used against the alert's threshold. values : greater_than or less_than")),
				mcp.WithString("Period", mcp.Required(), mcp.WithStringEnumItems([]string{"2m", "3m", "5m", "10m", "15m", "30m", "1h"}), mcp.Description("Period of time the threshold must be exceeded to trigger the alert")),
				mcp.WithArray("Emails", mcp.Required(), mcp.Description("email addresses to notify"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "email address to notify",
				})),
				mcp.WithArray(
					"SlackDetails",
					mcp.Required(),
					mcp.Items(map[string]any{
						"type": "object",
						"properties": map[string]any{
							"channel": map[string]any{"type": "string"},
							"url":     map[string]any{"type": "string"},
						},
					}),
					mcp.Description("Array of Slack details for the alert"),
				),
			),
		},
		{
			Handler: c.updateUptimeCheckAlert,
			Tool: mcp.NewTool("uptimecheck-alert-update",
				mcp.WithDescription("Update a UptimeCheck"),
				mcp.WithString("CheckID", mcp.Required(), mcp.Description("A unique identifier for a check")),
				mcp.WithString("AlertID", mcp.Required(), mcp.Description("A unique identifier for a check alert")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the UptimeCheck Alert")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of the UptimeCheck Alert. value: latency, down, down_global or ssl_expiry")),
				mcp.WithNumber("Threshold", mcp.Description("The threshold at which the alert will enter a trigger state. The specific threshold is dependent on the alert type")),
				mcp.WithString("Comparison", mcp.Description("The comparison operator used against the alert's threshold. value : greater_than or less_than")),
				mcp.WithString("Period", mcp.Required(), mcp.WithStringEnumItems([]string{"2m", "3m", "5m", "10m", "15m", "30m", "1h"}), mcp.Description("Period of time the threshold must be exceeded to trigger the alert")),
				mcp.WithArray("Emails", mcp.Required(), mcp.Description("Email addresses to notify"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "email address to notify",
				})),
				mcp.WithArray(
					"SlackDetails",
					mcp.Required(),
					mcp.Items(map[string]any{
						"type": "object",
						"properties": map[string]any{
							"channel": map[string]any{"type": "string"},
							"url":     map[string]any{"type": "string"},
						},
					}),
					mcp.Description("Array of Slack details for the alert"),
				),
			),
		},
		{
			Handler: c.deleteUptimeCheckAlert,
			Tool: mcp.NewTool("uptimecheck-alert-delete",
				mcp.WithDescription("Delete a uptimeCheck"),
				mcp.WithString("CheckID", mcp.Required(), mcp.Description("A unique identifier for a check")),
				mcp.WithString("AlertID", mcp.Required(), mcp.Description("A unique identifier for a alert")),
			),
		},
	}
}
