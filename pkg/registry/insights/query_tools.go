package insights

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

type QueryTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewQueryTool(client func(ctx context.Context) (*godo.Client, error)) *QueryTool {
	return &QueryTool{client: client}
}

func (t *QueryTool) query(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	region := stringArg(args, "Region")
	query := stringArg(args, "Query")
	if region == "" || query == "" {
		return mcp.NewToolResultError("Region and Query are required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	opt := &godo.PromQueryOptions{Query: query, Time: stringArg(args, "Time"), Timeout: stringArg(args, "Timeout")}
	var out *godo.PromQueryResponse
	if boolArg(args, "UsePOST") {
		out, _, err = client.Insights.PostQuery(ctx, region, opt)
	} else {
		out, _, err = client.Insights.Query(ctx, region, opt)
	}
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return promQueryOut.Result(out)
}

func (t *QueryTool) queryRange(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	region := stringArg(args, "Region")
	if region == "" || stringArg(args, "Query") == "" || stringArg(args, "Start") == "" || stringArg(args, "End") == "" || stringArg(args, "Step") == "" {
		return mcp.NewToolResultError("Region, Query, Start, End, and Step are required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	opt := &godo.PromQueryRangeOptions{
		Query:   stringArg(args, "Query"),
		Start:   stringArg(args, "Start"),
		End:     stringArg(args, "End"),
		Step:    stringArg(args, "Step"),
		Timeout: stringArg(args, "Timeout"),
	}
	var out *godo.PromQueryRangeResponse
	if boolArg(args, "UsePOST") {
		out, _, err = client.Insights.PostQueryRange(ctx, region, opt)
	} else {
		out, _, err = client.Insights.QueryRange(ctx, region, opt)
	}
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return promQueryRangeOut.Result(out)
}

func (t *QueryTool) series(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	region := stringArg(args, "Region")
	if region == "" {
		return mcp.NewToolResultError("Region is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	opt := &godo.PromSelectorOptions{Match: stringSliceArg(args, "Match"), Start: stringArg(args, "Start"), End: stringArg(args, "End")}
	var out *godo.PromSeriesResponse
	if boolArg(args, "UsePOST") {
		out, _, err = client.Insights.PostSeries(ctx, region, opt)
	} else {
		out, _, err = client.Insights.Series(ctx, region, opt)
	}
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return promSeriesOut.Result(out)
}

func (t *QueryTool) labels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	region := stringArg(args, "Region")
	if region == "" {
		return mcp.NewToolResultError("Region is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	opt := &godo.PromSelectorOptions{Match: stringSliceArg(args, "Match"), Start: stringArg(args, "Start"), End: stringArg(args, "End")}
	var out *godo.PromLabelsResponse
	if boolArg(args, "UsePOST") {
		out, _, err = client.Insights.PostLabels(ctx, region, opt)
	} else {
		out, _, err = client.Insights.Labels(ctx, region, opt)
	}
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return promLabelsOut.Result(out)
}

func (t *QueryTool) labelValues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	region := stringArg(args, "Region")
	name := stringArg(args, "Label")
	if region == "" || name == "" {
		return mcp.NewToolResultError("Region and Label are required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Insights.LabelValues(ctx, region, name, &godo.PromSelectorOptions{
		Match: stringSliceArg(args, "Match"),
		Start: stringArg(args, "Start"),
		End:   stringArg(args, "End"),
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return promLabelsOut.Result(out)
}

func (t *QueryTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{Handler: t.query, Tool: mcp.NewTool("insights-query",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), promQueryOut.Schema(),
			mcp.WithDescription("PREFERRED — Instant PromQL query (GET/POST /v2/insights/query/{region}/prom/api/v1/query)."),
			mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug, e.g. nyc3")),
			mcp.WithString("Query", mcp.Required(), mcp.Description("PromQL expression")),
			mcp.WithString("Time", mcp.Description("Evaluation timestamp")),
			mcp.WithString("Timeout", mcp.Description("Query timeout, e.g. 30s")),
			mcp.WithBoolean("UsePOST", mcp.Description("Use POST for long expressions")),
		)},
		{Handler: t.queryRange, Tool: mcp.NewTool("insights-query-range",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), promQueryRangeOut.Schema(),
			mcp.WithDescription("PREFERRED — Range PromQL query (/v2/insights/query/{region}/prom/api/v1/query_range)."),
			mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug")),
			mcp.WithString("Query", mcp.Required(), mcp.Description("PromQL expression")),
			mcp.WithString("Start", mcp.Required(), mcp.Description("Range start")),
			mcp.WithString("End", mcp.Required(), mcp.Description("Range end")),
			mcp.WithString("Step", mcp.Required(), mcp.Description("Step, e.g. 15s")),
			mcp.WithString("Timeout", mcp.Description("Query timeout")),
			mcp.WithBoolean("UsePOST", mcp.Description("Use POST for long expressions")),
		)},
		{Handler: t.series, Tool: mcp.NewTool("insights-query-series",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), promSeriesOut.Schema(),
			mcp.WithDescription("PREFERRED — Find series matching PromQL selectors."),
			mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug")),
			mcp.WithArray("Match", mcp.Description("match[] selectors"), mcp.Items(map[string]any{"type": "string"})),
			mcp.WithString("Start", mcp.Description("Start time")),
			mcp.WithString("End", mcp.Description("End time")),
			mcp.WithBoolean("UsePOST", mcp.Description("Use POST for long selectors")),
		)},
		{Handler: t.labels, Tool: mcp.NewTool("insights-query-labels",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), promLabelsOut.Schema(),
			mcp.WithDescription("PREFERRED — List PromQL label names."),
			mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug")),
			mcp.WithArray("Match", mcp.Description("Optional match[] selectors"), mcp.Items(map[string]any{"type": "string"})),
			mcp.WithString("Start", mcp.Description("Start time")),
			mcp.WithString("End", mcp.Description("End time")),
			mcp.WithBoolean("UsePOST", mcp.Description("Use POST")),
		)},
		{Handler: t.labelValues, Tool: mcp.NewTool("insights-query-label-values",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), promLabelsOut.Schema(),
			mcp.WithDescription("PREFERRED — List values for one PromQL label."),
			mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug")),
			mcp.WithString("Label", mcp.Required(), mcp.Description("Label name, e.g. __name__")),
			mcp.WithArray("Match", mcp.Description("Optional match[] selectors"), mcp.Items(map[string]any{"type": "string"})),
			mcp.WithString("Start", mcp.Description("Start time")),
			mcp.WithString("End", mcp.Description("End time")),
		)},
	}
}
