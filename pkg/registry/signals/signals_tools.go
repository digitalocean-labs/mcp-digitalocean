package signals

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// Tool provides Signals investigation GETs, consent list/get/set, and export create/list/get.
type Tool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewTool creates a Signals MCP tool set.
func NewTool(client func(ctx context.Context) (*godo.Client, error)) *Tool {
	return &Tool{client: client}
}

func (t *Tool) doClient(ctx context.Context) (*godo.Client, error) {
	return t.client(ctx)
}

func argString(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func argBool(args map[string]any, key string) (bool, bool) {
	v, ok := args[key].(bool)
	return v, ok
}

func argInt(args map[string]any, key string) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func argInt64Ptr(args map[string]any, key string) *int64 {
	switch v := args[key].(type) {
	case float64:
		n := int64(v)
		return &n
	case int:
		n := int64(v)
		return &n
	case int64:
		return &v
	default:
		return nil
	}
}

func argStringSlice(args map[string]any, key string) []string {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok || s == "" {
				continue
			}
			out = append(out, s)
		}
		return out
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	default:
		return nil
	}
}

func (t *Tool) listConsents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Signals.ListConsents(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list consents", err), nil
	}
	return consentsListOut.Result(out)
}

func (t *Tool) getAgentConsent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	agentID := argString(req.GetArguments(), "AgentID")
	if agentID == "" {
		return mcp.NewToolResultError("AgentID is required"), nil
	}
	out, _, err := client.Signals.GetAgentConsent(ctx, agentID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get agent consent", err), nil
	}
	return consentOut.Result(out)
}

func (t *Tool) setAgentConsent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	args := req.GetArguments()
	agentID := argString(args, "AgentID")
	if agentID == "" {
		return mcp.NewToolResultError("AgentID is required"), nil
	}
	enabled, ok := argBool(args, "Enabled")
	if !ok {
		return mcp.NewToolResultError("Enabled is required"), nil
	}
	out, _, err := client.Signals.SetAgentConsent(ctx, agentID, enabled)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to set agent consent", err), nil
	}
	return consentSetOut.Result(out)
}

func (t *Tool) listAgentSessions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	args := req.GetArguments()
	agentID := argString(args, "AgentID")
	if agentID == "" {
		return mcp.NewToolResultError("AgentID is required"), nil
	}
	opts := &godo.SignalsListAgentSessionsOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
			Limit: argInt(args, "Limit"),
			After: argString(args, "After"),
		},
		StartTime:  argInt64Ptr(args, "StartTime"),
		EndTime:    argInt64Ptr(args, "EndTime"),
		SignalType: argStringSlice(args, "SignalType"),
	}
	out, _, err := client.Signals.ListAgentSessions(ctx, agentID, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list agent sessions", err), nil
	}
	return sessionsOut.Result(out)
}

func (t *Tool) listSessionDialogues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	args := req.GetArguments()
	sessionID := argString(args, "SessionID")
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	opts := &godo.SignalsListDialoguesOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
			Limit: argInt(args, "Limit"),
			After: argString(args, "After"),
		},
		Before:     argString(args, "Before"),
		SignalType: argStringSlice(args, "SignalType"),
		StartTime:  argInt64Ptr(args, "StartTime"),
		EndTime:    argInt64Ptr(args, "EndTime"),
	}
	if v, ok := argBool(args, "ContinueSession"); ok {
		opts.ContinueSession = v
	}
	out, _, err := client.Signals.ListSessionDialogues(ctx, sessionID, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list session dialogues", err), nil
	}
	return dialoguesOut.Result(out)
}

func (t *Tool) createExport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	args := req.GetArguments()
	agentID := argString(args, "AgentID")
	if agentID == "" {
		return mcp.NewToolResultError("AgentID is required"), nil
	}
	createReq := &godo.SignalsCreateExportRequest{
		AgentID:    agentID,
		SignalType: argStringSlice(args, "SignalType"),
		StartTime:  argInt64Ptr(args, "StartTime"),
		EndTime:    argInt64Ptr(args, "EndTime"),
	}
	out, _, err := client.Signals.CreateExport(ctx, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create export", err), nil
	}
	return exportJobOut.Result(out)
}

func (t *Tool) listExports(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	args := req.GetArguments()
	opts := &godo.SignalsListExportsOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
			Limit: argInt(args, "Limit"),
			After: argString(args, "After"),
		},
		AgentID: argString(args, "AgentID"),
	}
	out, _, err := client.Signals.ListExports(ctx, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list exports", err), nil
	}
	return exportsOut.Result(out)
}

func (t *Tool) getExport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	exportID := argString(req.GetArguments(), "ExportID")
	if exportID == "" {
		return mcp.NewToolResultError("ExportID is required"), nil
	}
	out, _, err := client.Signals.GetExport(ctx, exportID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get export", err), nil
	}
	return exportJobOut.Result(out)
}

func (t *Tool) getExportDownload(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	exportID := argString(req.GetArguments(), "ExportID")
	if exportID == "" {
		return mcp.NewToolResultError("ExportID is required"), nil
	}
	out, _, err := client.Signals.GetExportDownload(ctx, exportID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get export download URL", err), nil
	}
	return exportDownloadOut.Result(out)
}

func (t *Tool) getExportOptions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Signals.GetExportOptions(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get export options", err), nil
	}
	return exportOptionsOut.Result(out)
}

func withLimitAfter(opts ...mcp.ToolOption) []mcp.ToolOption {
	return append(opts,
		mcp.WithNumber("Limit", mcp.Description("Page size (default 20, max 100)")),
		mcp.WithString("After", mcp.Description("Cursor from page_info.end_cursor")),
	)
}

func signalTypeOpt() mcp.ToolOption {
	return mcp.WithArray("SignalType", mcp.Description("Optional signal_type filter. Repeatable (OR)."), mcp.Items(map[string]any{"type": "string"}))
}

// Tools returns MCP tools for Signals.
func (t *Tool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listConsents,
			Tool: mcp.NewTool("signals-list-consents",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				consentsListOut.Schema(),
				mcp.WithDescription("List Signals collection consent records for the team (GET /v1/consent)."),
			),
		},
		{
			Handler: t.getAgentConsent,
			Tool: mcp.NewTool("signals-get-agent-consent",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				consentOut.Schema(),
				mcp.WithDescription("Get Signals collection consent for one agent. Missing row is default deny (enabled=false, allowed=false)."),
				mcp.WithString("AgentID", mcp.Required(), mcp.Description("Agent id (Mars SPI config_id / Agent Config UUID)")),
			),
		},
		{
			Handler: t.setAgentConsent,
			Tool: mcp.NewTool("signals-set-agent-consent",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				consentSetOut.Schema(),
				mcp.WithDescription("Enable or disable Signals collection for one agent. PUT /v1/consent/{agent_id} on consent-gateway. Requires signals:update. Ingest cache can take up to 15 minutes."),
				mcp.WithString("AgentID", mcp.Required(), mcp.Description("Agent id to update")),
				mcp.WithBoolean("Enabled", mcp.Required(), mcp.Description("true to collect, false to deny")),
			),
		},
		{
			Handler: t.listAgentSessions,
			Tool: mcp.NewTool("signals-list-agent-sessions",
				withLimitAfter(
					common.WithHints(common.HintsRead),
					common.WithRisk(common.RiskLow),
					sessionsOut.Schema(),
					mcp.WithDescription("List sessions for an agent (cursor page of session summaries)."),
					mcp.WithString("AgentID", mcp.Required(), mcp.Description("Agent id whose sessions to list")),
					mcp.WithNumber("StartTime", mcp.Description("Optional lower bound, Unix epoch seconds")),
					mcp.WithNumber("EndTime", mcp.Description("Optional upper bound, Unix epoch seconds")),
					signalTypeOpt(),
				)...,
			),
		},
		{
			Handler: t.listSessionDialogues,
			Tool: mcp.NewTool("signals-list-session-dialogues",
				withLimitAfter(
					common.WithHints(common.HintsRead),
					common.WithRisk(common.RiskLow),
					dialoguesOut.Schema(),
					mcp.WithDescription("List dialogue turns for a session."),
					mcp.WithString("SessionID", mcp.Required(), mcp.Description("Session id")),
					mcp.WithString("Before", mcp.Description("Upper cursor bound")),
					signalTypeOpt(),
					mcp.WithNumber("StartTime", mcp.Description("Optional Unix epoch seconds lower bound")),
					mcp.WithNumber("EndTime", mcp.Description("Optional Unix epoch seconds upper bound")),
					mcp.WithBoolean("ContinueSession", mcp.Description("When true, continue from the current session cursor")),
				)...,
			),
		},
		{
			Handler: t.createExport,
			Tool: mcp.NewTool("signals-create-export",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				exportJobOut.Schema(),
				mcp.WithDescription("Create a Signals export job for an agent (POST /v1/signals/exports). Poll with signals-get-export, then download with signals-get-export-download when completed."),
				mcp.WithString("AgentID", mcp.Required(), mcp.Description("Agent id whose signal data to export")),
				signalTypeOpt(),
				mcp.WithNumber("StartTime", mcp.Description("Optional lower bound, Unix epoch seconds")),
				mcp.WithNumber("EndTime", mcp.Description("Optional upper bound, Unix epoch seconds")),
			),
		},
		{
			Handler: t.listExports,
			Tool: mcp.NewTool("signals-list-exports",
				withLimitAfter(
					common.WithHints(common.HintsRead),
					common.WithRisk(common.RiskLow),
					exportsOut.Schema(),
					mcp.WithDescription("List prior export jobs for the team. Optional AgentID filter."),
					mcp.WithString("AgentID", mcp.Description("If set, only exports for this agent")),
				)...,
			),
		},
		{
			Handler: t.getExport,
			Tool: mcp.NewTool("signals-get-export",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				exportJobOut.Schema(),
				mcp.WithDescription("Poll one export job until status is completed or failed."),
				mcp.WithString("ExportID", mcp.Required(), mcp.Description("Export job id")),
			),
		},
		{
			Handler: t.getExportDownload,
			Tool: mcp.NewTool("signals-get-export-download",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				exportDownloadOut.Schema(),
				mcp.WithDescription("Get a 15-minute presigned Spaces URL for a completed export. Do not persist the URL."),
				mcp.WithString("ExportID", mcp.Required(), mcp.Description("Export job id")),
			),
		},
		{
			Handler: t.getExportOptions,
			Tool: mcp.NewTool("signals-get-export-options",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				exportOptionsOut.Schema(),
				mcp.WithDescription("Catalog of exportable entity types. Unauthenticated on the API; MCP still sends the team token."),
			),
		},
	}
}
