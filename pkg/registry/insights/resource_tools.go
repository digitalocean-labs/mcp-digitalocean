package insights

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

type ResourceTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewResourceTool(client func(ctx context.Context) (*godo.Client, error)) *ResourceTool {
	return &ResourceTool{client: client}
}

func (t *ResourceTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	urn := stringArg(req.GetArguments(), "ResourceURN")
	if urn == "" {
		return mcp.NewToolResultError("ResourceURN is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Insights.GetResource(ctx, urn)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return resourceOut.Result(out)
}

func (t *ResourceTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Insights.ListResources(ctx, &godo.InsightsResourceListOptions{
		ProductTypes:  stringSliceArg(args, "ProductTypes"),
		Regions:       stringSliceArg(args, "Regions"),
		Page:          intArg(args, "Page", defaultAlertPoliciesPage),
		PerPage:       intArg(args, "PerPage", defaultAlertPoliciesPageSize),
		Sort:          stringArg(args, "Sort"),
		SortDirection: stringArg(args, "SortDirection"),
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return resourcesOut.Result(out)
}

func (t *ResourceTool) updateSignals(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	urn := stringArg(args, "ResourceURN")
	metrics := boolPtrArg(args, "Metrics")
	logs := boolPtrArg(args, "Logs")
	traces := boolPtrArg(args, "Traces")
	if urn == "" {
		return mcp.NewToolResultError("ResourceURN is required"), nil
	}
	if metrics == nil && logs == nil && traces == nil {
		return mcp.NewToolResultError("Set at least one of Metrics, Logs, or Traces"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Insights.UpdateResourceSignals(ctx, urn, &godo.InsightsResourceSignalsRequest{
		Region:  stringArg(args, "Region"),
		Metrics: metrics,
		Logs:    logs,
		Traces:  traces,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return resourceOut.Result(out)
}

func (t *ResourceTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{Handler: t.get, Tool: mcp.NewTool("insights-resource-get",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), resourceOut.Schema(),
			mcp.WithDescription("PREFERRED — Get Insights telemetry enablement for one resource (GET /v2/insights/resources/{urn}). 404 if none is recorded."),
			mcp.WithString("ResourceURN", mcp.Required(), mcp.Description("Resource URN, e.g. do:droplet:123")),
		)},
		{Handler: t.list, Tool: mcp.NewTool("insights-resource-list",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), resourcesOut.Schema(),
			mcp.WithDescription("PREFERRED — List resources with recorded Insights telemetry enablement (GET /v2/insights/resources)."),
			mcp.WithArray("ProductTypes", mcp.Description("Filter by product type, e.g. droplet"), mcp.Items(map[string]any{"type": "string"})),
			mcp.WithArray("Regions", mcp.Description("Filter by region slug"), mcp.Items(map[string]any{"type": "string"})),
			mcp.WithNumber("Page", mcp.DefaultNumber(defaultAlertPoliciesPage), mcp.Description("Page number")),
			mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultAlertPoliciesPageSize), mcp.Description("Items per page (default 20, max 200)")),
			mcp.WithString("Sort", mcp.Description("resource_urn or effective_at")),
			mcp.WithString("SortDirection", mcp.Description("asc or desc")),
		)},
		{Handler: t.updateSignals, Tool: mcp.NewTool("insights-resource-update-signals",
			common.WithHints(common.HintsToggle), common.WithRisk(common.RiskLow), resourceOut.Schema(),
			mcp.WithDescription("PREFERRED — Enable or disable Insights telemetry signals for a resource (PUT /v2/insights/resources/{urn}/signals). Omitted signals stay unchanged."),
			mcp.WithString("ResourceURN", mcp.Required(), mcp.Description("Resource URN, e.g. do:droplet:123")),
			mcp.WithString("Region", mcp.Description("Required only when the resource has no existing enablement")),
			mcp.WithBoolean("Metrics", mcp.Description("Enable or disable metrics")),
			mcp.WithBoolean("Logs", mcp.Description("Enable or disable logs")),
			mcp.WithBoolean("Traces", mcp.Description("Enable or disable traces")),
		)},
	}
}
