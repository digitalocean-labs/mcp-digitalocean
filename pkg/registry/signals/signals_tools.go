package signals

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// InferenceAgentID is the agent ID under which Signals stores serverless inference sessions and exports.
const InferenceAgentID = "00000000-0000-0000-0000-000000000000"

const (
	environmentIDDesc              = "Environment ID"
	environmentIDWithInferenceDesc = "Environment ID. Use " + InferenceAgentID + " for serverless inference."
	agentIDDeprecatedDesc          = "Deprecated: use EnvironmentID. Environment ID"
	agentIDDeprecatedInferenceDesc = "Deprecated: use EnvironmentID. " + environmentIDWithInferenceDesc
	agentIDDeprecationNotice       = "DEPRECATED: AgentID is deprecated; use EnvironmentID."
)

// Tool provides Signals investigation GETs, consent list/get/set, and export create/list/get.
type Tool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

type resolvedEnvironmentID struct {
	id         string
	deprecated bool
}

func resolveEnvironmentID(args map[string]any, required bool) (resolvedEnvironmentID, *mcp.CallToolResult) {
	agentID := argString(args, "AgentID")
	envID := argString(args, "EnvironmentID")
	switch {
	case agentID != "" && envID != "" && agentID != envID:
		return resolvedEnvironmentID{}, mcp.NewToolResultError("AgentID and EnvironmentID must be equal when both are set")
	case envID != "":
		return resolvedEnvironmentID{id: envID, deprecated: agentID != ""}, nil
	case agentID != "":
		return resolvedEnvironmentID{id: agentID, deprecated: true}, nil
	case required:
		return resolvedEnvironmentID{}, mcp.NewToolResultError("EnvironmentID or AgentID is required")
	default:
		return resolvedEnvironmentID{}, nil
	}
}

func withAgentIDDeprecation(res *mcp.CallToolResult, err error, deprecated bool) (*mcp.CallToolResult, error) {
	if !deprecated || err != nil || res == nil || res.IsError {
		return res, err
	}
	res.Content = append(res.Content, mcp.TextContent{Type: "text", Text: agentIDDeprecationNotice})
	return res, nil
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
	return resultDualEmit(consentsListOut, out)
}

func (t *Tool) getAgentConsent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	resolved, errRes := resolveEnvironmentID(req.GetArguments(), true)
	if errRes != nil {
		return errRes, nil
	}
	out, _, err := client.Signals.GetAgentConsent(ctx, resolved.id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get agent consent", err), nil
	}
	res, err := resultDualEmit(consentOut, out)
	return withAgentIDDeprecation(res, err, resolved.deprecated)
}

func (t *Tool) setAgentConsent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	args := req.GetArguments()
	resolved, errRes := resolveEnvironmentID(args, true)
	if errRes != nil {
		return errRes, nil
	}
	enabled, ok := argBool(args, "Enabled")
	if !ok {
		return mcp.NewToolResultError("Enabled is required"), nil
	}
	out, _, err := client.Signals.SetAgentConsent(ctx, resolved.id, enabled)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to set agent consent", err), nil
	}
	res, err := resultDualEmit(consentSetOut, out)
	return withAgentIDDeprecation(res, err, resolved.deprecated)
}

func (t *Tool) getInferenceConsent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	out, _, err := client.Signals.GetInferenceConsent(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get inference consent", err), nil
	}
	return resultDualEmit(inferenceConsentOut, out)
}

func (t *Tool) setInferenceConsent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	enabled, ok := argBool(req.GetArguments(), "Enabled")
	if !ok {
		return mcp.NewToolResultError("Enabled is required"), nil
	}
	out, _, err := client.Signals.SetInferenceConsent(ctx, enabled)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to set inference consent", err), nil
	}
	return resultDualEmit(inferenceConsentOut, out)
}

func (t *Tool) listAgentSessions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	args := req.GetArguments()
	resolved, errRes := resolveEnvironmentID(args, true)
	if errRes != nil {
		return errRes, nil
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
	out, _, err := client.Signals.ListAgentSessions(ctx, resolved.id, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list agent sessions", err), nil
	}
	res, err := resultDualEmit(sessionsOut, out)
	return withAgentIDDeprecation(res, err, resolved.deprecated)
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
	return resultDualEmit(dialoguesOut, out)
}

func (t *Tool) createExport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	args := req.GetArguments()
	resolved, errRes := resolveEnvironmentID(args, true)
	if errRes != nil {
		return errRes, nil
	}
	createReq := &godo.SignalsCreateExportRequest{
		AgentID:    resolved.id,
		SignalType: argStringSlice(args, "SignalType"),
		StartTime:  argInt64Ptr(args, "StartTime"),
		EndTime:    argInt64Ptr(args, "EndTime"),
	}
	out, _, err := client.Signals.CreateExport(ctx, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create export", err), nil
	}
	res, err := resultDualEmit(exportJobOut, out)
	return withAgentIDDeprecation(res, err, resolved.deprecated)
}

func (t *Tool) listExports(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.doClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	args := req.GetArguments()
	resolved, errRes := resolveEnvironmentID(args, false)
	if errRes != nil {
		return errRes, nil
	}
	opts := &godo.SignalsListExportsOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
			Limit: argInt(args, "Limit"),
			After: argString(args, "After"),
		},
		AgentID: resolved.id,
	}
	out, _, err := client.Signals.ListExports(ctx, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list exports", err), nil
	}
	res, err := resultDualEmit(exportsOut, out)
	return withAgentIDDeprecation(res, err, resolved.deprecated)
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
	return resultDualEmit(exportJobOut, out)
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
	return resultDualEmit(exportDownloadOut, out)
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
	return resultDualEmit(exportOptionsOut, out)
}

func withLimitAfter(opts ...mcp.ToolOption) []mcp.ToolOption {
	return append(opts,
		mcp.WithNumber("Limit", mcp.Description("Page size (default 20, max 100)")),
		mcp.WithString("After", mcp.Description("Cursor from page_info.end_cursor")),
	)
}

func signalTypeOpt() mcp.ToolOption {
	return mcp.WithArray("SignalType", mcp.Description("Filter by signal type"), mcp.Items(map[string]any{"type": "string"}))
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
				mcp.WithDescription("List Signals consent for the team."),
			),
		},
		{
			Handler: t.getAgentConsent,
			Tool: mcp.NewTool("signals-get-agent-consent",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				consentOut.Schema(),
				mcp.WithDescription("Get Signals consent for one environment. For serverless inference, use signals-get-inference-consent."),
				mcp.WithString("EnvironmentID", mcp.Description(environmentIDDesc)),
				mcp.WithString("AgentID", mcp.Description(agentIDDeprecatedDesc)),
			),
		},
		{
			Handler: t.setAgentConsent,
			Tool: mcp.NewTool("signals-set-agent-consent",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				consentSetOut.Schema(),
				mcp.WithDescription("Enable or disable Signals for one environment. For serverless inference, use signals-set-inference-consent."),
				mcp.WithString("EnvironmentID", mcp.Description(environmentIDDesc)),
				mcp.WithString("AgentID", mcp.Description(agentIDDeprecatedDesc)),
				mcp.WithBoolean("Enabled", mcp.Required(), mcp.Description("true to enable, false to disable")),
			),
		},
		{
			Handler: t.getInferenceConsent,
			Tool: mcp.NewTool("signals-get-inference-consent",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				inferenceConsentOut.Schema(),
				mcp.WithDescription("Get the team's Signals consent for serverless inference traffic. Returns enabled=false when the team has never set it."),
			),
		},
		{
			Handler: t.setInferenceConsent,
			Tool: mcp.NewTool("signals-set-inference-consent",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				inferenceConsentOut.Schema(),
				mcp.WithDescription("Enable or disable Signals for all serverless inference traffic of the team."),
				mcp.WithBoolean("Enabled", mcp.Required(), mcp.Description("true to enable, false to disable")),
			),
		},
		{
			Handler: t.listAgentSessions,
			Tool: mcp.NewTool("signals-list-agent-sessions",
				withLimitAfter(
					common.WithHints(common.HintsRead),
					common.WithRisk(common.RiskLow),
					sessionsOut.Schema(),
					mcp.WithDescription("List sessions for an environment, or for serverless inference."),
					mcp.WithString("EnvironmentID", mcp.Description(environmentIDWithInferenceDesc)),
					mcp.WithString("AgentID", mcp.Description(agentIDDeprecatedInferenceDesc)),
					mcp.WithNumber("StartTime", mcp.Description("Lower bound (Unix epoch seconds)")),
					mcp.WithNumber("EndTime", mcp.Description("Upper bound (Unix epoch seconds)")),
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
					mcp.WithDescription("List dialogues for a session."),
					mcp.WithString("SessionID", mcp.Required(), mcp.Description("Session ID")),
					mcp.WithString("Before", mcp.Description("Upper cursor bound")),
					signalTypeOpt(),
					mcp.WithNumber("StartTime", mcp.Description("Lower bound (Unix epoch seconds)")),
					mcp.WithNumber("EndTime", mcp.Description("Upper bound (Unix epoch seconds)")),
					mcp.WithBoolean("ContinueSession", mcp.Description("Continue from the session cursor")),
				)...,
			),
		},
		{
			Handler: t.createExport,
			Tool: mcp.NewTool("signals-create-export",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				exportJobOut.Schema(),
				mcp.WithDescription("Create an export for an environment, or for serverless inference."),
				mcp.WithString("EnvironmentID", mcp.Description(environmentIDWithInferenceDesc)),
				mcp.WithString("AgentID", mcp.Description(agentIDDeprecatedInferenceDesc)),
				signalTypeOpt(),
				mcp.WithNumber("StartTime", mcp.Description("Lower bound (Unix epoch seconds)")),
				mcp.WithNumber("EndTime", mcp.Description("Upper bound (Unix epoch seconds)")),
			),
		},
		{
			Handler: t.listExports,
			Tool: mcp.NewTool("signals-list-exports",
				withLimitAfter(
					common.WithHints(common.HintsRead),
					common.WithRisk(common.RiskLow),
					exportsOut.Schema(),
					mcp.WithDescription("List exports for the team."),
					mcp.WithString("EnvironmentID", mcp.Description(environmentIDWithInferenceDesc)),
					mcp.WithString("AgentID", mcp.Description(agentIDDeprecatedInferenceDesc)),
				)...,
			),
		},
		{
			Handler: t.getExport,
			Tool: mcp.NewTool("signals-get-export",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				exportJobOut.Schema(),
				mcp.WithDescription("Get an export by ID."),
				mcp.WithString("ExportID", mcp.Required(), mcp.Description("Export ID")),
			),
		},
		{
			Handler: t.getExportDownload,
			Tool: mcp.NewTool("signals-get-export-download",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				exportDownloadOut.Schema(),
				mcp.WithDescription("Get a short-lived download URL for a completed export."),
				mcp.WithString("ExportID", mcp.Required(), mcp.Description("Export ID")),
			),
		},
		{
			Handler: t.getExportOptions,
			Tool: mcp.NewTool("signals-get-export-options",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				exportOptionsOut.Schema(),
				mcp.WithDescription("List export filter options."),
			),
		},
	}
}
