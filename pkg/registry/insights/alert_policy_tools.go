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
	defaultAlertPoliciesPageSize = 20
	defaultAlertPoliciesPage     = 1
)

// AlertPolicyTool provides alert policy management tools
type AlertPolicyTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewAlertPolicyTool creates a new alert policy tool
func NewAlertPolicyTool(client func(ctx context.Context) (*godo.Client, error)) *AlertPolicyTool {
	return &AlertPolicyTool{
		client: client,
	}
}

// getAlertPolicy fetches alert policy information by UUID
func (a *AlertPolicyTool) getAlertPolicy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid, ok := req.GetArguments()["UUID"].(string)
	if !ok || uuid == "" {
		return mcp.NewToolResultError("Alert Policy UUID is required"), nil
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	alertPolicy, _, err := client.Monitoring.GetAlertPolicy(ctx, uuid)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAlertPolicy, err := json.MarshalIndent(alertPolicy, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAlertPolicy)), nil
}

// listAlertPolicies lists alert policies with pagination support
func (a *AlertPolicyTool) listAlertPolicies(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := defaultAlertPoliciesPage
	perPage := defaultAlertPoliciesPageSize
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	alertPolicies, _, err := client.Monitoring.ListAlertPolicies(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAlertPolicies, err := json.MarshalIndent(alertPolicies, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAlertPolicies)), nil
}

// createAlertPolicy creates a new alert policy
func (a *AlertPolicyTool) createAlertPolicy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alertType := req.GetArguments()["Type"].(string)
	description := req.GetArguments()["Description"].(string)
	compare := godo.AlertPolicyComp(req.GetArguments()["Compare"].(string))
	value := float32(req.GetArguments()["Value"].(float64))
	window := req.GetArguments()["Window"].(string)

	// Parse entities array
	rawEntities, _ := req.GetArguments()["Entities"]
	var entities []string
	if arr, ok := rawEntities.([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				entities = append(entities, s)
			}
		}
	}

	// Parse tags array
	rawTags, _ := req.GetArguments()["Tags"]
	var tags []string
	if arr, ok := rawTags.([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	// Parse alerts
	rawAlerts, _ := req.GetArguments()["Alerts"]
	var alerts godo.Alerts
	if alertsMap, ok := rawAlerts.(map[string]interface{}); ok {
		// Parse email alerts
		if rawEmails, ok := alertsMap["Email"].([]interface{}); ok {
			for _, v := range rawEmails {
				if email, ok := v.(string); ok {
					alerts.Email = append(alerts.Email, email)
				}
			}
		}

		// Parse Slack alerts
		if rawSlack, ok := alertsMap["Slack"].([]interface{}); ok {
			for _, v := range rawSlack {
				if slackMap, ok := v.(map[string]interface{}); ok {
					slackDetails := godo.SlackDetails{
						URL:     slackMap["URL"].(string),
						Channel: slackMap["Channel"].(string),
					}
					alerts.Slack = append(alerts.Slack, slackDetails)
				}
			}
		}
	}

	enabled := true
	if v, ok := req.GetArguments()["Enabled"].(bool); ok {
		enabled = v
	}

	createRequest := &godo.AlertPolicyCreateRequest{
		Type:        alertType,
		Description: description,
		Compare:     compare,
		Value:       value,
		Window:      window,
		Entities:    entities,
		Tags:        tags,
		Alerts:      alerts,
		Enabled:     &enabled,
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	alertPolicy, _, err := client.Monitoring.CreateAlertPolicy(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAlertPolicy, err := json.MarshalIndent(alertPolicy, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAlertPolicy)), nil
}

// updateAlertPolicy updates an existing alert policy
func (a *AlertPolicyTool) updateAlertPolicy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid, ok := req.GetArguments()["UUID"].(string)
	if !ok || uuid == "" {
		return mcp.NewToolResultError("Alert Policy UUID is required"), nil
	}

	alertType := req.GetArguments()["Type"].(string)
	description := req.GetArguments()["Description"].(string)
	compare := godo.AlertPolicyComp(req.GetArguments()["Compare"].(string))
	value := float32(req.GetArguments()["Value"].(float64))
	window := req.GetArguments()["Window"].(string)

	// Parse entities array
	rawEntities, _ := req.GetArguments()["Entities"]
	var entities []string
	if arr, ok := rawEntities.([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				entities = append(entities, s)
			}
		}
	}

	// Parse tags array
	rawTags, _ := req.GetArguments()["Tags"]
	var tags []string
	if arr, ok := rawTags.([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	// Parse alerts
	rawAlerts, _ := req.GetArguments()["Alerts"]
	var alerts godo.Alerts
	if alertsMap, ok := rawAlerts.(map[string]interface{}); ok {
		// Parse email alerts
		if rawEmails, ok := alertsMap["Email"].([]interface{}); ok {
			for _, v := range rawEmails {
				if email, ok := v.(string); ok {
					alerts.Email = append(alerts.Email, email)
				}
			}
		}

		// Parse Slack alerts
		if rawSlack, ok := alertsMap["Slack"].([]interface{}); ok {
			for _, v := range rawSlack {
				if slackMap, ok := v.(map[string]interface{}); ok {
					slackDetails := godo.SlackDetails{
						URL:     slackMap["URL"].(string),
						Channel: slackMap["Channel"].(string),
					}
					alerts.Slack = append(alerts.Slack, slackDetails)
				}
			}
		}
	}

	enabled := true
	if v, ok := req.GetArguments()["Enabled"].(bool); ok {
		enabled = v
	}

	updateRequest := &godo.AlertPolicyUpdateRequest{
		Type:        alertType,
		Description: description,
		Compare:     compare,
		Value:       value,
		Window:      window,
		Entities:    entities,
		Tags:        tags,
		Alerts:      alerts,
		Enabled:     &enabled,
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	alertPolicy, _, err := client.Monitoring.UpdateAlertPolicy(ctx, uuid, updateRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonAlertPolicy, err := json.MarshalIndent(alertPolicy, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonAlertPolicy)), nil
}

// deleteAlertPolicy deletes an alert policy
func (a *AlertPolicyTool) deleteAlertPolicy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid, ok := req.GetArguments()["UUID"].(string)
	if !ok || uuid == "" {
		return mcp.NewToolResultError("Alert Policy UUID is required"), nil
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Monitoring.DeleteAlertPolicy(ctx, uuid)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Alert Policy deleted successfully"), nil
}

// Tools returns a list of tool functions
func (c *AlertPolicyTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: c.getAlertPolicy,
			Tool: mcp.NewTool("alert-policy-get",
				mcp.WithDescription("Get Alert Policy information by UUID"),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("UUID of the Alert Policy to retrieve (format: 00000000-0000-0000-0000-000000000000)")),
			),
		},
		{
			Handler: c.listAlertPolicies,
			Tool: mcp.NewTool("alert-policy-list",
				mcp.WithDescription("List all Alert Policies in your account with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultAlertPoliciesPage), mcp.Description("Page number for pagination (starts from 1)")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultAlertPoliciesPageSize), mcp.Description("Number of items per page (1-200, default 20)")),
			),
		},
		{
			Handler: c.createAlertPolicy,
			Tool: mcp.NewTool("alert-policy-create",
				mcp.WithDescription("Create a new Alert Policy"),
				mcp.WithString("Type", mcp.Required(), mcp.Description(`Type of the Alert Policy. Available types:
Droplet metrics:
- 'v1/insights/droplet/load_1'
- 'v1/insights/droplet/load_5'
- 'v1/insights/droplet/load_15'
- 'v1/insights/droplet/cpu'
- 'v1/insights/droplet/memory_utilization'
- 'v1/insights/droplet/disk_utilization'
- 'v1/insights/droplet/disk_read_rate'
- 'v1/insights/droplet/disk_write_rate'
- 'v1/insights/droplet/public_outbound_bandwidth'
- 'v1/insights/droplet/public_inbound_bandwidth'
Load Balancer metrics:
- 'v1/insights/lbaas/avg_cpu_utilization'
- 'v1/insights/lbaas/connection_utilization'
- 'v1/insights/lbaas/droplet_health'
- 'v1/insights/lbaas/tls_connections_per_second_utilization'
Database metrics:
- 'v1/insights/database/cpu'
- 'v1/insights/database/memory_utilization'
- 'v1/insights/database/disk_utilization'`)),
				mcp.WithString("Description", mcp.Required(), mcp.Description("Human-readable description of the alert policy")),
				mcp.WithString("Compare", mcp.Required(), mcp.Description("Comparison operator: 'GreaterThan' or 'LessThan'")),
				mcp.WithNumber("Value", mcp.Required(), mcp.Description("Threshold value for the alert (e.g., 80 for 80% CPU)")),
				mcp.WithString("Window", mcp.Required(), mcp.Description("Time window for the alert: '5m', '10m', '30m', '1h' (5 minutes, 10 minutes, 30 minutes, 1 hour)")),
				mcp.WithArray("Entities", mcp.Description("List of resource IDs to monitor (e.g., Droplet IDs: '12345678', '23456789')"),
					mcp.Items(map[string]any{
						"type": "string",
					})),
				mcp.WithArray("Tags", mcp.Description("List of tags to monitor resources with these tags (e.g., 'production', 'staging')"),
					mcp.Items(map[string]any{
						"type": "string",
					})),
				mcp.WithObject("Alerts", mcp.Description("Alert notification settings"),
					mcp.Properties(map[string]any{
						"Email": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"description": "List of email addresses to receive alert notifications",
						},
						"Slack": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"URL":     map[string]any{"type": "string", "description": "Slack webhook URL"},
									"Channel": map[string]any{"type": "string", "description": "Slack channel (e.g., '#alerts')"},
								},
							},
							"description": "List of Slack webhook configurations",
						},
					})),
				mcp.WithBoolean("Enabled", mcp.Description("Whether the alert policy is enabled (true) or disabled (false)")),
			),
		},
		{
			Handler: c.updateAlertPolicy,
			Tool: mcp.NewTool("alert-policy-update",
				mcp.WithDescription("Update an Alert Policy"),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("UUID of the Alert Policy to update")),
				mcp.WithString("Type", mcp.Required(), mcp.Description(`Type of the Alert Policy. Available types:
Droplet metrics:
- 'v1/insights/droplet/load_1'
- 'v1/insights/droplet/load_5'
- 'v1/insights/droplet/load_15'
- 'v1/insights/droplet/cpu'
- 'v1/insights/droplet/memory_utilization'
- 'v1/insights/droplet/disk_utilization'
- 'v1/insights/droplet/disk_read_rate'
- 'v1/insights/droplet/disk_write_rate'
- 'v1/insights/droplet/public_outbound_bandwidth'
- 'v1/insights/droplet/public_inbound_bandwidth'
Load Balancer metrics:
- 'v1/insights/lbaas/avg_cpu_utilization'
- 'v1/insights/lbaas/connection_utilization'
- 'v1/insights/lbaas/droplet_health'
- 'v1/insights/lbaas/tls_connections_per_second_utilization'
Database metrics:
- 'v1/insights/database/cpu'
- 'v1/insights/database/memory_utilization'
- 'v1/insights/database/disk_utilization'`)),
				mcp.WithString("Description", mcp.Required(), mcp.Description("Human-readable description of the alert policy")),
				mcp.WithString("Compare", mcp.Required(), mcp.Description("Comparison operator: 'GreaterThan' or 'LessThan'")),
				mcp.WithNumber("Value", mcp.Required(), mcp.Description("Threshold value for the alert (e.g., 80 for 80% CPU)")),
				mcp.WithString("Window", mcp.Required(), mcp.Description("Time window for the alert: '5m', '10m', '30m', '1h' (5 minutes, 10 minutes, 30 minutes, 1 hour)")),
				mcp.WithArray("Entities", mcp.Description("List of resource IDs to monitor (e.g., Droplet IDs: '12345678', '23456789')"),
					mcp.Items(map[string]any{
						"type": "string",
					})),
				mcp.WithArray("Tags", mcp.Description("List of tags to monitor resources with these tags (e.g., 'production', 'staging')"),
					mcp.Items(map[string]any{
						"type": "string",
					})),
				mcp.WithObject("Alerts", mcp.Description("Alert notification settings"),
					mcp.Properties(map[string]any{
						"Email": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"description": "List of email addresses to receive alert notifications",
						},
						"Slack": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"URL":     map[string]any{"type": "string", "description": "Slack webhook URL"},
									"Channel": map[string]any{"type": "string", "description": "Slack channel (e.g., '#alerts')"},
								},
							},
							"description": "List of Slack webhook configurations",
						},
					})),
				mcp.WithBoolean("Enabled", mcp.Description("Whether the alert policy is enabled (true) or disabled (false)")),
			),
		},
		{
			Handler: c.deleteAlertPolicy,
			Tool: mcp.NewTool("alert-policy-delete",
				mcp.WithDescription("Delete an Alert Policy permanently"),
				mcp.WithString("UUID", mcp.Required(), mcp.Description("UUID of the Alert Policy to delete (format: 00000000-0000-0000-0000-000000000000)")),
			),
		},
	}
}
