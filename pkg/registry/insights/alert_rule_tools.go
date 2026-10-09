package insights

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

type AlertRuleTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewAlertRuleTool(client func(ctx context.Context) (*godo.Client, error)) *AlertRuleTool {
	return &AlertRuleTool{client: client}
}

func (t *AlertRuleTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := stringArg(req.GetArguments(), "ID")
	if id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	rule, _, err := client.Insights.GetAlertRule(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return alertRuleOut.Result(rule)
}

func (t *AlertRuleTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	rules, _, err := client.Insights.ListAlertRules(ctx, &godo.AlertRuleListOptions{
		Page:        intArg(args, "Page", defaultAlertPoliciesPage),
		PerPage:     intArg(args, "PerPage", defaultAlertPoliciesPageSize),
		ResourceURN: stringArg(args, "ResourceURN"),
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return alertRulesOut.Result(rules)
}

func (t *AlertRuleTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	if stringArg(args, "Name") == "" || stringArg(args, "Metric") == "" || stringArg(args, "Operator") == "" {
		return mcp.NewToolResultError("Name, Metric, and Operator are required"), nil
	}
	if notificationBindings(args) == nil {
		return mcp.NewToolResultError("NotificationChannels is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	rule, _, err := client.Insights.CreateAlertRule(ctx, alertRuleRequestFromArgs(args))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return alertRuleOut.Result(rule)
}

func (t *AlertRuleTool) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id := stringArg(args, "ID")
	if id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}
	if stringArg(args, "Name") == "" || stringArg(args, "Metric") == "" || stringArg(args, "Operator") == "" {
		return mcp.NewToolResultError("Name, Metric, and Operator are required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	rule, _, err := client.Insights.UpdateAlertRule(ctx, id, alertRuleRequestFromArgs(args))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return alertRuleOut.Result(rule)
}

func (t *AlertRuleTool) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := stringArg(req.GetArguments(), "ID")
	if id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	if _, err := client.Insights.DeleteAlertRule(ctx, id); err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Alert rule deleted successfully"), nil
}

func alertRuleSpecArgs() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("Name", mcp.Required(), mcp.Description("Alert rule name")),
		mcp.WithString("Metric", mcp.Required(), mcp.Description("Dotted OpenTelemetry metric, e.g. do.droplets.cpu_utilization")),
		mcp.WithString("Operator", mcp.Required(), mcp.Description("THRESHOLD_OPERATOR_EQUAL | LESS_THAN | LESS_THAN_OR_EQUAL | GREATER_THAN | GREATER_THAN_OR_EQUAL | NOT_EQUAL")),
		mcp.WithNumber("Warning", mcp.Description("Warning threshold")),
		mcp.WithNumber("Critical", mcp.Description("Critical threshold")),
		mcp.WithString("Window", mcp.Description("EVALUATION_WINDOW_1M | 5M | 10M | 15M | 30M | 1H")),
		mcp.WithArray("ResourceURNs", mcp.Description("Resource URNs to evaluate, e.g. do:droplet:12345"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithArray("Tags", mcp.Description("Resource tags"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithArray("Filters", mcp.Description("Label filters"), mcp.Items(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"Field":    map[string]any{"type": "string"},
				"Operator": map[string]any{"type": "string", "description": "FILTER_OPERATOR_*"},
				"Value":    map[string]any{"type": "string"},
			},
		})),
		mcp.WithArray("NotificationChannels", mcp.Description("Channel bindings. Required on create."), mcp.Items(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"NotificationChannelID": map[string]any{"type": "string"},
				"NotifyOn":              map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "SEVERITY_WARNING and/or SEVERITY_CRITICAL"},
			},
		})),
		mcp.WithString("ReAlertDuration", mcp.Description("RE_ALERT_DURATION_30M | 1H | 4H | NEVER")),
		mcp.WithString("Status", mcp.Description("ALERT_RULE_STATUS_ACTIVE or ALERT_RULE_STATUS_PAUSED")),
	}
}

func (t *AlertRuleTool) Tools() []server.ServerTool {
	createOpts := append([]mcp.ToolOption{
		common.WithHints(common.HintsCreate),
		common.WithRisk(common.RiskLow),
		alertRuleOut.Schema(),
		mcp.WithDescription("PREFERRED — Create an Insights v2 alert rule (POST /v2/insights/alert-rules). Do not use alert-policy-create."),
	}, alertRuleSpecArgs()...)
	updateOpts := append([]mcp.ToolOption{
		common.WithHints(common.HintsToggle),
		common.WithRisk(common.RiskLow),
		alertRuleOut.Schema(),
		mcp.WithDescription("PREFERRED — Update an Insights v2 alert rule (PUT /v2/insights/alert-rules/{id}). Do not use alert-policy-update."),
		mcp.WithString("ID", mcp.Required(), mcp.Description("Alert rule ID")),
	}, alertRuleSpecArgs()...)

	return []server.ServerTool{
		{Handler: t.get, Tool: mcp.NewTool("insights-alert-rule-get",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), alertRuleOut.Schema(),
			mcp.WithDescription("PREFERRED — Get an Insights v2 alert rule. Do not use alert-policy-get."),
			mcp.WithString("ID", mcp.Required(), mcp.Description("Alert rule ID")),
		)},
		{Handler: t.list, Tool: mcp.NewTool("insights-alert-rule-list",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), alertRulesOut.Schema(),
			mcp.WithDescription("PREFERRED — List Insights v2 alert rules. Do not use alert-policy-list."),
			mcp.WithNumber("Page", mcp.DefaultNumber(defaultAlertPoliciesPage), mcp.Description("Page number")),
			mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultAlertPoliciesPageSize), mcp.Description("Items per page")),
			mcp.WithString("ResourceURN", mcp.Description("Filter by resource URN")),
		)},
		{Handler: t.create, Tool: mcp.NewTool("insights-alert-rule-create", createOpts...)},
		{Handler: t.update, Tool: mcp.NewTool("insights-alert-rule-update", updateOpts...)},
		{Handler: t.delete, Tool: mcp.NewTool("insights-alert-rule-delete",
			common.WithHints(common.HintsDelete), common.WithRisk(common.RiskMedium),
			mcp.WithDescription("PREFERRED — Delete an Insights v2 alert rule. Do not use alert-policy-delete."),
			mcp.WithString("ID", mcp.Required(), mcp.Description("Alert rule ID")),
		)},
	}
}
