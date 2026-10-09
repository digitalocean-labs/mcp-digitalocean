package insights

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

type AlertInstanceTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewAlertInstanceTool(client func(ctx context.Context) (*godo.Client, error)) *AlertInstanceTool {
	return &AlertInstanceTool{client: client}
}

func (t *AlertInstanceTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := stringArg(req.GetArguments(), "ID")
	if id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	inst, _, err := client.Insights.GetAlertInstance(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return alertInstanceOut.Result(inst)
}

func (t *AlertInstanceTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	instances, _, err := client.Insights.ListAlertInstances(ctx, &godo.AlertInstanceListOptions{
		Page:        intArg(args, "Page", defaultAlertPoliciesPage),
		PerPage:     intArg(args, "PerPage", defaultAlertPoliciesPageSize),
		Status:      stringArg(args, "Status"),
		RuleID:      stringArg(args, "RuleID"),
		ResourceURN: stringArg(args, "ResourceURN"),
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return alertInstancesOut.Result(instances)
}

func (t *AlertInstanceTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{Handler: t.get, Tool: mcp.NewTool("insights-alert-instance-get",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), alertInstanceOut.Schema(),
			mcp.WithDescription("PREFERRED — Get a firing Insights v2 alert instance."),
			mcp.WithString("ID", mcp.Required(), mcp.Description("Alert instance ID")),
		)},
		{Handler: t.list, Tool: mcp.NewTool("insights-alert-instance-list",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), alertInstancesOut.Schema(),
			mcp.WithDescription("PREFERRED — List Insights v2 alert instances (newest trigger first)."),
			mcp.WithNumber("Page", mcp.DefaultNumber(defaultAlertPoliciesPage), mcp.Description("Page number")),
			mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultAlertPoliciesPageSize), mcp.Description("Items per page")),
			mcp.WithString("Status", mcp.Description("active or resolved")),
			mcp.WithString("RuleID", mcp.Description("Filter by alert rule ID")),
			mcp.WithString("ResourceURN", mcp.Description("Filter by resource URN")),
		)},
	}
}
