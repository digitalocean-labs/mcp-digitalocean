package harnessruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	triggersPath   = "v2/agents/triggers"
	executionsPath = "v2/agents/triggers/executions"
)

// TriggerOutput configures what happens with the trigger's output (none, email, slack).
type TriggerOutput struct {
	Mode   string `json:"mode,omitempty"`
	Prompt string `json:"prompt,omitempty"`
}

// TriggerWebhook holds webhook-specific trigger configuration.
type TriggerWebhook struct {
	Provider   string `json:"provider,omitempty"`
	WebhookURL string `json:"webhook_url,omitempty"`
}

// TriggerCron holds cron-specific trigger configuration.
type TriggerCron struct {
	CronExpr string `json:"cron_expr,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

// Trigger is the user-facing record of a trigger that can launch or drive agent sessions.
type Trigger struct {
	TriggerID       string          `json:"trigger_id"`
	Kind            string          `json:"kind,omitempty"`
	Name            string          `json:"name,omitempty"`
	Status          string          `json:"status,omitempty"`
	ConfigID        string          `json:"config_id,omitempty"`
	SessionMode     string          `json:"session_mode,omitempty"`
	AgentKind       string          `json:"agent_kind,omitempty"`
	PromptTemplate  string          `json:"prompt_template,omitempty"`
	Output          *TriggerOutput  `json:"output,omitempty"`
	SessionTemplate string          `json:"session_template,omitempty"`
	BoundSessionID  string          `json:"bound_session_id,omitempty"`
	Webhook         *TriggerWebhook `json:"webhook,omitempty"`
	Cron            *TriggerCron    `json:"cron,omitempty"`
	MaxConcurrency  int             `json:"max_concurrency,omitempty"`
	CreatedAt       *time.Time      `json:"created_at,omitempty"`
	UpdatedAt       *time.Time      `json:"updated_at,omitempty"`
}

// TriggerResponse is the response of a single-trigger endpoint.
type TriggerResponse struct {
	Trigger Trigger `json:"trigger"`
}

// CreateTriggerResponse is the response of POST /v2/agents/triggers.
// It includes the webhook_secret returned only at creation time.
type CreateTriggerResponse struct {
	Trigger       Trigger `json:"trigger"`
	WebhookSecret string  `json:"webhook_secret,omitempty"`
}

// ListTriggersResponse is the response of GET /v2/agents/triggers.
type ListTriggersResponse struct {
	Triggers      []Trigger `json:"triggers"`
	NextPageToken string    `json:"next_page_token,omitempty"`
}

// RotateTriggerSecretResponse is the response of POST .../triggers/{id}/rotate-secret.
type RotateTriggerSecretResponse struct {
	WebhookSecret string `json:"webhook_secret,omitempty"`
}

// TriggerExecution is the user-facing record of a single trigger execution.
type TriggerExecution struct {
	ID          string     `json:"id"`
	TriggerID   string     `json:"trigger_id,omitempty"`
	SessionID   string     `json:"session_id,omitempty"`
	Status      string     `json:"status,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// TriggerExecutionResponse is the response of a single-execution endpoint.
type TriggerExecutionResponse struct {
	Execution TriggerExecution `json:"execution"`
}

// ListTriggerExecutionsResponse is the response of GET .../triggers/{id}/executions.
type ListTriggerExecutionsResponse struct {
	Executions    []TriggerExecution `json:"executions"`
	NextPageToken string             `json:"next_page_token,omitempty"`
}

// CancelExecutionResponse is the response of POST .../executions/{eid}/cancel.
type CancelExecutionResponse struct {
	Execution TriggerExecution `json:"execution"`
}

// ReusableSession is a session eligible for reuse by a trigger.
type ReusableSession struct {
	SessionID string `json:"session_id"`
	Name      string `json:"name,omitempty"`
	Status    string `json:"status,omitempty"`
}

// ListReusableSessionsResponse is the response of GET .../triggers/{id}/reusable-sessions.
type ListReusableSessionsResponse struct {
	Sessions      []ReusableSession `json:"sessions"`
	NextPageToken string            `json:"next_page_token,omitempty"`
}

// listTriggers enumerates the caller's triggers.
func (t *Tool) listTriggers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	query := url.Values{}

	if v, ok := args["PageSize"].(float64); ok {
		if v != math.Trunc(v) || v < 1 || v > maxPageSize {
			return mcp.NewToolResultError(fmt.Sprintf("PageSize must be an integer between 1 and %d", maxPageSize)), nil
		}
		query.Set("page_size", strconv.Itoa(int(v)))
	}
	if v, _ := args["PageToken"].(string); v != "" {
		query.Set("page_token", v)
	}
	if v, _ := args["ConfigID"].(string); v != "" {
		query.Set("config_id", v)
	}

	path := triggersPath
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListTriggersResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list triggers", err), nil
	}
	if out.Triggers == nil {
		out.Triggers = []Trigger{}
	}
	return triggerListOut.Result(out)
}

// createTrigger creates a new trigger.
func (t *Tool) createTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()

	configID, _ := args["ConfigID"].(string)
	if configID == "" {
		return mcp.NewToolResultError("ConfigID is required"), nil
	}
	if _, err := uuid.Parse(configID); err != nil {
		return mcp.NewToolResultError("ConfigID must be a UUID"), nil
	}

	name, _ := args["Name"].(string)
	if name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}

	kind, _ := args["Kind"].(string)
	if kind == "" {
		return mcp.NewToolResultError("Kind is required"), nil
	}
	if kind != "webhook" && kind != "cron" {
		return mcp.NewToolResultError("Kind must be one of: webhook, cron"), nil
	}

	sessionMode, _ := args["SessionMode"].(string)
	if sessionMode == "" {
		return mcp.NewToolResultError("SessionMode is required"), nil
	}
	if sessionMode != "fresh" && sessionMode != "reuse" {
		return mcp.NewToolResultError("SessionMode must be one of: fresh, reuse"), nil
	}

	outputMode, _ := args["OutputMode"].(string)
	if outputMode == "" {
		outputMode = "none"
	}
	if outputMode != "none" && outputMode != "email" && outputMode != "slack" {
		return mcp.NewToolResultError("OutputMode must be one of: none, email, slack"), nil
	}

	body := map[string]any{
		"config_id":    configID,
		"name":         name,
		"kind":         kind,
		"session_mode": sessionMode,
		"output":       map[string]any{"mode": outputMode},
	}

	if kind == "cron" {
		cronExpr, _ := args["CronExpr"].(string)
		if cronExpr == "" {
			return mcp.NewToolResultError("CronExpr is required when Kind is cron"), nil
		}
		timezone, _ := args["Timezone"].(string)
		if timezone == "" {
			timezone = "UTC"
		}
		body["cron"] = map[string]any{
			"cron_expr": cronExpr,
			"timezone":  timezone,
		}
	}

	if kind == "webhook" {
		body["webhook"] = map[string]any{}
	}

	if sessionMode == "fresh" {
		tmpl, _ := args["SessionTemplate"].(string)
		if tmpl == "" {
			return mcp.NewToolResultError("SessionTemplate is required when SessionMode is fresh"), nil
		}
		body["session_template"] = tmpl
	}

	if sessionMode == "reuse" {
		boundID, _ := args["BoundSessionID"].(string)
		if boundID == "" {
			return mcp.NewToolResultError("BoundSessionID is required when SessionMode is reuse"), nil
		}
		if _, err := uuid.Parse(boundID); err != nil {
			return mcp.NewToolResultError("BoundSessionID must be a UUID"), nil
		}
		body["bound_session_id"] = boundID
	}

	if v, _ := args["Prompt"].(string); v != "" {
		output := body["output"].(map[string]any)
		output["prompt"] = v
	}

	if v, ok := args["MaxConcurrency"].(float64); ok {
		body["max_concurrency"] = int(v)
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, triggersPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out CreateTriggerResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create trigger", err), nil
	}
	return createTriggerOut.Result(out)
}

// getTrigger returns a single trigger by ID.
func (t *Tool) getTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	triggerID, _ := req.GetArguments()["TriggerID"].(string)
	if triggerID == "" {
		return mcp.NewToolResultError("TriggerID is required"), nil
	}
	if _, err := uuid.Parse(triggerID); err != nil {
		return mcp.NewToolResultError("TriggerID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, triggersPath+"/"+triggerID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out TriggerResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get trigger", err), nil
	}
	return triggerOut.Result(out)
}

// updateTrigger replaces a trigger's mutable fields.
func (t *Tool) updateTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	triggerID, _ := args["TriggerID"].(string)
	if triggerID == "" {
		return mcp.NewToolResultError("TriggerID is required"), nil
	}
	if _, err := uuid.Parse(triggerID); err != nil {
		return mcp.NewToolResultError("TriggerID must be a UUID"), nil
	}

	body := map[string]any{}
	if v, _ := args["Name"].(string); v != "" {
		body["name"] = v
	}
	if v, _ := args["ConfigID"].(string); v != "" {
		if _, err := uuid.Parse(v); err != nil {
			return mcp.NewToolResultError("ConfigID must be a UUID"), nil
		}
		body["config_id"] = v
	}
	if v, _ := args["SessionMode"].(string); v != "" {
		body["session_mode"] = v
	}
	if v, _ := args["SessionTemplate"].(string); v != "" {
		body["session_template"] = v
	}
	if v, _ := args["BoundSessionID"].(string); v != "" {
		if _, err := uuid.Parse(v); err != nil {
			return mcp.NewToolResultError("BoundSessionID must be a UUID"), nil
		}
		body["bound_session_id"] = v
	}
	if v, ok := args["MaxConcurrency"].(float64); ok {
		body["max_concurrency"] = int(v)
	}

	// Build output sub-object if any output fields are provided.
	outputMode, hasOutputMode := args["OutputMode"].(string)
	prompt, hasPrompt := args["Prompt"].(string)
	if (hasOutputMode && outputMode != "") || (hasPrompt && prompt != "") {
		out := map[string]any{}
		if hasOutputMode && outputMode != "" {
			out["mode"] = outputMode
		}
		if hasPrompt && prompt != "" {
			out["prompt"] = prompt
		}
		body["output"] = out
	}

	// Build cron sub-object if any cron fields are provided.
	cronExpr, hasCron := args["CronExpr"].(string)
	timezone, hasTZ := args["Timezone"].(string)
	if (hasCron && cronExpr != "") || (hasTZ && timezone != "") {
		c := map[string]any{}
		if hasCron && cronExpr != "" {
			c["cron_expr"] = cronExpr
		}
		if hasTZ && timezone != "" {
			c["timezone"] = timezone
		}
		body["cron"] = c
	}

	if len(body) == 0 {
		return mcp.NewToolResultError("At least one updatable field must be provided"), nil
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPatch, triggersPath+"/"+triggerID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out TriggerResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to update trigger", err), nil
	}
	return triggerOut.Result(out)
}

// deleteTrigger permanently removes a trigger.
func (t *Tool) deleteTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	triggerID, _ := req.GetArguments()["TriggerID"].(string)
	if triggerID == "" {
		return mcp.NewToolResultError("TriggerID is required"), nil
	}
	if _, err := uuid.Parse(triggerID); err != nil {
		return mcp.NewToolResultError("TriggerID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodDelete, triggersPath+"/"+triggerID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	if _, err := client.Do(ctx, apiReq, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete trigger", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Trigger %s deleted successfully.", triggerID)), nil
}

// setTriggerStatus sets a trigger's status via PUT (used by pause and resume).
func (t *Tool) setTriggerStatus(ctx context.Context, triggerID, status, action string) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	body := map[string]any{"status": status}
	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPatch, triggersPath+"/"+triggerID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out TriggerResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr(fmt.Sprintf("Failed to %s trigger", action), err), nil
	}
	return triggerOut.Result(out)
}

// pauseTrigger pauses an active trigger by setting status=paused.
func (t *Tool) pauseTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	triggerID, _ := req.GetArguments()["TriggerID"].(string)
	if triggerID == "" {
		return mcp.NewToolResultError("TriggerID is required"), nil
	}
	if _, err := uuid.Parse(triggerID); err != nil {
		return mcp.NewToolResultError("TriggerID must be a UUID"), nil
	}
	return t.setTriggerStatus(ctx, triggerID, "paused", "pause")
}

// resumeTrigger resumes a paused trigger by setting status=active.
func (t *Tool) resumeTrigger(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	triggerID, _ := req.GetArguments()["TriggerID"].(string)
	if triggerID == "" {
		return mcp.NewToolResultError("TriggerID is required"), nil
	}
	if _, err := uuid.Parse(triggerID); err != nil {
		return mcp.NewToolResultError("TriggerID must be a UUID"), nil
	}
	return t.setTriggerStatus(ctx, triggerID, "active", "resume")
}

// rotateTriggerSecret rotates the webhook secret for a trigger.
func (t *Tool) rotateTriggerSecret(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	triggerID, _ := req.GetArguments()["TriggerID"].(string)
	if triggerID == "" {
		return mcp.NewToolResultError("TriggerID is required"), nil
	}
	if _, err := uuid.Parse(triggerID); err != nil {
		return mcp.NewToolResultError("TriggerID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, triggersPath+"/"+triggerID+"/rotate-secret", nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out RotateTriggerSecretResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to rotate trigger secret", err), nil
	}
	return rotateTriggerSecretOut.Result(out)
}

// listTriggerExecutions lists executions for a specific trigger.
func (t *Tool) listTriggerExecutions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	triggerID, _ := args["TriggerID"].(string)
	if triggerID == "" {
		return mcp.NewToolResultError("TriggerID is required"), nil
	}
	if _, err := uuid.Parse(triggerID); err != nil {
		return mcp.NewToolResultError("TriggerID must be a UUID"), nil
	}

	query := url.Values{}
	if v, ok := args["PageSize"].(float64); ok {
		if v != math.Trunc(v) || v < 1 || v > maxPageSize {
			return mcp.NewToolResultError(fmt.Sprintf("PageSize must be an integer between 1 and %d", maxPageSize)), nil
		}
		query.Set("page_size", strconv.Itoa(int(v)))
	}
	if v, _ := args["PageToken"].(string); v != "" {
		query.Set("page_token", v)
	}

	path := triggersPath + "/" + triggerID + "/executions"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListTriggerExecutionsResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list trigger executions", err), nil
	}
	if out.Executions == nil {
		out.Executions = []TriggerExecution{}
	}
	return triggerExecutionListOut.Result(out)
}

// getTriggerExecution returns a single trigger execution by ID.
func (t *Tool) getTriggerExecution(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	executionID, _ := req.GetArguments()["ExecutionID"].(string)
	if executionID == "" {
		return mcp.NewToolResultError("ExecutionID is required"), nil
	}
	if _, err := uuid.Parse(executionID); err != nil {
		return mcp.NewToolResultError("ExecutionID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, executionsPath+"/"+executionID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out TriggerExecutionResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get trigger execution", err), nil
	}
	return triggerExecutionOut.Result(out)
}

// cancelExecution cancels a running trigger execution.
func (t *Tool) cancelExecution(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	executionID, _ := req.GetArguments()["ExecutionID"].(string)
	if executionID == "" {
		return mcp.NewToolResultError("ExecutionID is required"), nil
	}
	if _, err := uuid.Parse(executionID); err != nil {
		return mcp.NewToolResultError("ExecutionID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, executionsPath+"/"+executionID+"/cancel", nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out CancelExecutionResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to cancel execution", err), nil
	}
	return cancelExecutionOut.Result(out)
}

// getTriggerBySession returns the trigger associated with a specific session.
func (t *Tool) getTriggerBySession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	sessionID, _ := req.GetArguments()["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, triggersPath+"/by-session/"+sessionID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out TriggerResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get trigger by session", err), nil
	}
	return triggerOut.Result(out)
}

// listReusableSessions lists sessions that a trigger can reuse.
func (t *Tool) listReusableSessions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	triggerID, _ := args["TriggerID"].(string)
	if triggerID == "" {
		return mcp.NewToolResultError("TriggerID is required"), nil
	}
	if _, err := uuid.Parse(triggerID); err != nil {
		return mcp.NewToolResultError("TriggerID must be a UUID"), nil
	}

	query := url.Values{}
	if v, ok := args["PageSize"].(float64); ok {
		if v != math.Trunc(v) || v < 1 || v > maxPageSize {
			return mcp.NewToolResultError(fmt.Sprintf("PageSize must be an integer between 1 and %d", maxPageSize)), nil
		}
		query.Set("page_size", strconv.Itoa(int(v)))
	}
	if v, _ := args["PageToken"].(string); v != "" {
		query.Set("page_token", v)
	}

	path := triggersPath + "/" + triggerID + "/reusable-sessions"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListReusableSessionsResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list reusable sessions", err), nil
	}
	if out.Sessions == nil {
		out.Sessions = []ReusableSession{}
	}
	return reusableSessionListOut.Result(out)
}

// triggerTools returns the trigger MCP tools to be merged into Tools().
func (t *Tool) triggerTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listTriggers,
			Tool: mcp.NewTool("harness-runtime-list-triggers",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				triggerListOut.Schema(),
				mcp.WithDescription("List the team's triggers. Triggers automate agent session launches on a schedule (cron) or via webhook. Results are keyset-paginated; pass next_page_token back as PageToken for the next page. Optionally filter by ConfigID."),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
				mcp.WithString("ConfigID", mcp.Description("Optional agent config UUID to filter triggers by")),
			),
		},
		{
			Handler: t.createTrigger,
			Tool: mcp.NewTool("harness-runtime-create-trigger",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				createTriggerOut.Schema(),
				mcp.WithDescription("Create a new trigger that automates agent session launches. A trigger references an agent config and fires on a schedule (cron) or via webhook. For webhook triggers, returns the webhook URL and signing secret. SessionMode controls whether each execution gets a fresh session (requires SessionTemplate with an agentspec YAML manifest including permissions.default set to allow or deny) or reuses an existing session (requires BoundSessionID)."),
				mcp.WithString("ConfigID", mcp.Required(), mcp.Description("Agent Config UUID the trigger will use")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Trigger name (must be unique among the team's triggers)")),
				mcp.WithString("Kind", mcp.Required(), mcp.Description("Trigger kind: 'webhook' or 'cron'")),
				mcp.WithString("SessionMode", mcp.Required(), mcp.Description("Session lifecycle: 'fresh' (new session per execution, requires SessionTemplate) or 'reuse' (reuses existing session, requires BoundSessionID)")),
				mcp.WithString("SessionTemplate", mcp.Description("YAML agentspec manifest for fresh sessions. Must include permissions.default set to 'allow' or 'deny'. Example: 'size: mars-2vcpu-4gb\\nagent: codex\\npermissions:\\n  default: allow\\n'")),
				mcp.WithString("BoundSessionID", mcp.Description("UUID of the session to reuse when SessionMode is 'reuse'")),
				mcp.WithString("CronExpr", mcp.Description("Cron expression for cron triggers (e.g. '0 0 * * *' for daily at midnight). Required when Kind is 'cron'.")),
				mcp.WithString("Timezone", mcp.Description("Timezone for cron triggers (e.g. 'UTC', 'America/New_York'). Defaults to 'UTC'. Only used when Kind is 'cron'.")),
				mcp.WithString("OutputMode", mcp.Description("Output delivery mode: 'none' (default), 'email', or 'slack'")),
				mcp.WithString("Prompt", mcp.Description("Prompt template to send to the agent when the trigger fires")),
				mcp.WithNumber("MaxConcurrency", mcp.Description("Maximum number of concurrent executions (default 1)")),
			),
		},
		{
			Handler: t.getTrigger,
			Tool: mcp.NewTool("harness-runtime-get-trigger",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				triggerOut.Schema(),
				mcp.WithDescription("Get a single trigger by its UUID, including its configuration, status, schedule, and webhook URL."),
				mcp.WithString("TriggerID", mcp.Required(), mcp.Description("The trigger UUID")),
			),
		},
		{
			Handler: t.updateTrigger,
			Tool: mcp.NewTool("harness-runtime-update-trigger",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				triggerOut.Schema(),
				mcp.WithDescription("Update a trigger's mutable fields. At least one field must be provided. Returns the updated trigger."),
				mcp.WithString("TriggerID", mcp.Required(), mcp.Description("The trigger UUID to update")),
				mcp.WithString("Name", mcp.Description("New trigger name")),
				mcp.WithString("ConfigID", mcp.Description("New agent config UUID")),
				mcp.WithString("SessionMode", mcp.Description("New session mode: 'fresh' or 'reuse'")),
				mcp.WithString("SessionTemplate", mcp.Description("New YAML agentspec manifest for fresh sessions")),
				mcp.WithString("BoundSessionID", mcp.Description("New bound session UUID for reuse mode")),
				mcp.WithString("CronExpr", mcp.Description("New cron expression")),
				mcp.WithString("Timezone", mcp.Description("New timezone for cron triggers")),
				mcp.WithString("OutputMode", mcp.Description("New output mode: 'none', 'email', or 'slack'")),
				mcp.WithString("Prompt", mcp.Description("New prompt template")),
				mcp.WithNumber("MaxConcurrency", mcp.Description("New maximum concurrent executions")),
			),
		},
		{
			Handler: t.deleteTrigger,
			Tool: mcp.NewTool("harness-runtime-delete-trigger",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Permanently delete a trigger by its UUID. Running executions may be cancelled. Returns 204 on success."),
				mcp.WithString("TriggerID", mcp.Required(), mcp.Description("The trigger UUID to delete")),
			),
		},
		{
			Handler: t.pauseTrigger,
			Tool: mcp.NewTool("harness-runtime-pause-trigger",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				triggerOut.Schema(),
				mcp.WithDescription("Pause an active trigger. A paused trigger will not fire until resumed. In-flight executions continue to completion. Returns the updated trigger with its new status."),
				mcp.WithString("TriggerID", mcp.Required(), mcp.Description("The trigger UUID to pause")),
			),
		},
		{
			Handler: t.resumeTrigger,
			Tool: mcp.NewTool("harness-runtime-resume-trigger",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				triggerOut.Schema(),
				mcp.WithDescription("Resume a paused trigger. The trigger will begin firing again according to its schedule or accepting webhooks. Returns the updated trigger with its new status."),
				mcp.WithString("TriggerID", mcp.Required(), mcp.Description("The trigger UUID to resume")),
			),
		},
		{
			Handler: t.rotateTriggerSecret,
			Tool: mcp.NewTool("harness-runtime-rotate-trigger-secret",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				rotateTriggerSecretOut.Schema(),
				mcp.WithDescription("Rotate the webhook signing secret for a webhook trigger. The old secret is immediately invalidated. Returns the new secret — this is the only time it is visible; store it securely."),
				mcp.WithString("TriggerID", mcp.Required(), mcp.Description("The trigger UUID whose webhook secret to rotate")),
			),
		},
		{
			Handler: t.listTriggerExecutions,
			Tool: mcp.NewTool("harness-runtime-list-trigger-executions",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				triggerExecutionListOut.Schema(),
				mcp.WithDescription("List executions for a specific trigger. An execution represents a single invocation of the trigger. Results are keyset-paginated; pass next_page_token back as PageToken for the next page."),
				mcp.WithString("TriggerID", mcp.Required(), mcp.Description("The trigger UUID whose executions to list")),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
			),
		},
		{
			Handler: t.getTriggerExecution,
			Tool: mcp.NewTool("harness-runtime-get-trigger-execution",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				triggerExecutionOut.Schema(),
				mcp.WithDescription("Get a single trigger execution by its UUID, including its status, associated session, timing, and any error details."),
				mcp.WithString("ExecutionID", mcp.Required(), mcp.Description("The execution UUID")),
			),
		},
		{
			Handler: t.cancelExecution,
			Tool: mcp.NewTool("harness-runtime-cancel-execution",
				common.WithHints(common.HintsAction),
				common.WithRisk(common.RiskMedium),
				cancelExecutionOut.Schema(),
				mcp.WithDescription("Cancel a running trigger execution. The associated session may be destroyed or returned to the reusable pool depending on the trigger's session mode. Returns the updated execution."),
				mcp.WithString("ExecutionID", mcp.Required(), mcp.Description("The execution UUID to cancel")),
			),
		},
		{
			Handler: t.getTriggerBySession,
			Tool: mcp.NewTool("harness-runtime-get-trigger-by-session",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				triggerOut.Schema(),
				mcp.WithDescription("Look up the trigger that created or is associated with a given session. Useful for tracing a session back to its automation source."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID to look up the trigger for")),
			),
		},
		{
			Handler: t.listReusableSessions,
			Tool: mcp.NewTool("harness-runtime-list-reusable-sessions",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				reusableSessionListOut.Schema(),
				mcp.WithDescription("List sessions eligible for reuse by a trigger. Only relevant for triggers with session_mode='reuse'. Returns sessions that are idle and can be claimed by the next trigger execution."),
				mcp.WithString("TriggerID", mcp.Required(), mcp.Description("The trigger UUID whose reusable sessions to list")),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
			),
		},
	}
}
