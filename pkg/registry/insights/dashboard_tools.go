package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// specHelp is the error returned when Spec is missing or is not a JSON object.
// godo validates the same thing before any request is sent, but its message is
// aimed at Go callers; this one tells the model what to do next.
const specHelp = "Spec must be a string containing a JSON object (not double-encoded). Call insights-dashboard-get-schema for the required shape."

type DashboardTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewDashboardTool(client func(ctx context.Context) (*godo.Client, error)) *DashboardTool {
	return &DashboardTool{client: client}
}

// dashboardSpecArg returns the Spec argument as the JSON document godo expects,
// or "" when it is absent or not a JSON object.
func dashboardSpecArg(args map[string]any) json.RawMessage {
	spec := strings.TrimSpace(stringArg(args, "Spec"))
	if spec == "" || spec[0] != '{' || !json.Valid([]byte(spec)) {
		return nil
	}
	return json.RawMessage(spec)
}

func (t *DashboardTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid := stringArg(req.GetArguments(), "UUID")
	if uuid == "" {
		return mcp.NewToolResultError("UUID is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Insights.GetDashboard(ctx, uuid)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return dashboardOut.Result(dashboardFromGodo(out))
}

func (t *DashboardTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Insights.ListDashboards(ctx, &godo.DashboardListOptions{
		Page:          intArg(args, "Page", defaultAlertPoliciesPage),
		PerPage:       intArg(args, "PerPage", defaultAlertPoliciesPageSize),
		Sort:          stringArg(args, "Sort"),
		SortDirection: stringArg(args, "SortDirection"),
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return dashboardsOut.Result(dashboardListFromGodo(out))
}

func (t *DashboardTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	spec := dashboardSpecArg(req.GetArguments())
	if spec == nil {
		return mcp.NewToolResultError(specHelp), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Insights.CreateDashboard(ctx, &godo.DashboardRequest{Spec: spec})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return dashboardOut.Result(dashboardFromGodo(out))
}

func (t *DashboardTool) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	uuid := stringArg(args, "UUID")
	if uuid == "" {
		return mcp.NewToolResultError("UUID is required"), nil
	}
	spec := dashboardSpecArg(args)
	if spec == nil {
		return mcp.NewToolResultError(specHelp), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Insights.UpdateDashboard(ctx, uuid, &godo.DashboardRequest{Spec: spec})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return dashboardOut.Result(dashboardFromGodo(out))
}

func (t *DashboardTool) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid := stringArg(req.GetArguments(), "UUID")
	if uuid == "" {
		return mcp.NewToolResultError("UUID is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	if _, err := client.Insights.DeleteDashboard(ctx, uuid); err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Dashboard deleted successfully"), nil
}

func (t *DashboardTool) getSchema(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	schema, _, err := client.Insights.GetDashboardSchema(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	// Pass the upstream document through verbatim rather than decoding into a
	// Go type, which would reorder keys and could drop schema vocabulary this
	// server does not model. json.Indent reformats the bytes without decoding.
	text := string(schema)
	var indented bytes.Buffer
	if err := json.Indent(&indented, schema, "", "  "); err == nil {
		text = indented.String()
	}
	return dashboardSchemaOut.ResultRaw(text, schema), nil
}

func (t *DashboardTool) Tools() []server.ServerTool {
	specArg := mcp.WithString("Spec", mcp.Required(), mcp.Description(
		"Perses dashboard document as a string containing a JSON object (pass the object text itself, not a JSON-encoded string of it). "+
			"Must satisfy the JSON Schema from insights-dashboard-get-schema. name and description are derived by the server from spec.display."))

	return []server.ServerTool{
		{Handler: t.get, Tool: mcp.NewTool("insights-dashboard-get",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), dashboardOut.Schema(),
			mcp.WithDescription("PREFERRED — Get an Insights dashboard (GET /v2/insights/dashboards/{uuid}). spec is returned as a JSON-encoded string. 404 for unknown, deleted, or other-team UUIDs."),
			mcp.WithString("UUID", mcp.Required(), mcp.Description("Dashboard UUID")),
		)},
		{Handler: t.list, Tool: mcp.NewTool("insights-dashboard-list",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), dashboardsOut.Schema(),
			mcp.WithDescription("PREFERRED — List Insights dashboards (GET /v2/insights/dashboards). Each spec is a JSON-encoded string. Without Sort, results are newest first."),
			mcp.WithNumber("Page", mcp.DefaultNumber(defaultAlertPoliciesPage), mcp.Description("Page number")),
			mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultAlertPoliciesPageSize), mcp.Description("Items per page (default 20, max 200)")),
			mcp.WithString("Sort", mcp.Description("name | created_at | updated_at")),
			mcp.WithString("SortDirection", mcp.Description("asc or desc. Only applied when Sort is set; defaults to asc")),
		)},
		{Handler: t.create, Tool: mcp.NewTool("insights-dashboard-create",
			common.WithHints(common.HintsCreate), common.WithRisk(common.RiskLow), dashboardOut.Schema(),
			mcp.WithDescription("PREFERRED — Create an Insights dashboard (POST /v2/insights/dashboards). Call insights-dashboard-get-schema first. "+
				"409 when another dashboard in the team already uses spec.display.name; 429 when the team's dashboard limit or the per-dashboard chart limit is reached."),
			specArg,
		)},
		{Handler: t.update, Tool: mcp.NewTool("insights-dashboard-update",
			// Idempotent PUT, so HintsToggle; but a full replace of a user-authored
			// document with no server-side history, so the risk sits with delete.
			common.WithHints(common.HintsToggle), common.WithRisk(common.RiskMedium), dashboardOut.Schema(),
			mcp.WithDescription("PREFERRED — Replace an Insights dashboard's spec in full (PUT /v2/insights/dashboards/{uuid}). name and description are recomputed from the new spec. "+
				"409 when the new spec.display.name is already used by another dashboard in the team."),
			mcp.WithString("UUID", mcp.Required(), mcp.Description("Dashboard UUID")),
			specArg,
		)},
		{Handler: t.delete, Tool: mcp.NewTool("insights-dashboard-delete",
			common.WithHints(common.HintsDelete), common.WithRisk(common.RiskMedium),
			mcp.WithDescription("PREFERRED — Delete an Insights dashboard (DELETE /v2/insights/dashboards/{uuid})."),
			mcp.WithString("UUID", mcp.Required(), mcp.Description("Dashboard UUID")),
		)},
		{Handler: t.getSchema, Tool: mcp.NewTool("insights-dashboard-get-schema",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), dashboardSchemaOut.Schema(),
			mcp.WithDescription("PREFERRED — Get the JSON Schema (draft 2020-12) that a dashboard Spec must satisfy (GET /v2/insights/dashboards/schema). Call before insights-dashboard-create or -update."),
		)},
	}
}
