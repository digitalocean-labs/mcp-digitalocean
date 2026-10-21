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
	"unicode/utf8"

	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	agentConfigsPath = "v2/agents/configs"
	maxPageSize      = 200
	maxSearchLen     = 64
)

// AgentConfigSummary is the lightweight list representation of an agent config.
// The manifest is intentionally not part of list responses.
type AgentConfigSummary struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	AgentSpecSchemaVersion string    `json:"agentspec_schema_version"`
	ContentHash            string    `json:"content_hash"`
	CreatedBy              string    `json:"created_by"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// ListAgentConfigsResponse is the response of GET /v2/agents/configs.
type ListAgentConfigsResponse struct {
	Configs       []AgentConfigSummary `json:"configs"`
	NextPageToken string               `json:"next_page_token,omitempty"`
}

// AgentConfigCredentialSlot is redacted credential slot metadata; secret values are never returned.
type AgentConfigCredentialSlot struct {
	Name     string `json:"name"`
	Source   string `json:"source,omitempty"`
	Provider string `json:"provider,omitempty"`
}

// AgentConfig is a full agent config including its sanitized manifest.
type AgentConfig struct {
	AgentConfigSummary
	// Manifest is the canonical runnable environment spec; its shape depends on
	// the manifest format (flat or legacy v1alpha1), so it is passed through.
	Manifest    map[string]any              `json:"manifest"`
	Insights    map[string]any              `json:"insights,omitempty"`
	Credentials []AgentConfigCredentialSlot `json:"credentials,omitempty"`
}

// AgentConfigResponse is the response of GET /v2/agents/configs/{config_id}.
type AgentConfigResponse struct {
	Config AgentConfig `json:"config"`
}

// Tool provides Harness Runtime (hosted agents) MCP tools.
type Tool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewTool creates a Harness Runtime MCP tool set.
func NewTool(client func(ctx context.Context) (*godo.Client, error)) *Tool {
	return &Tool{client: client}
}

func (t *Tool) listAgentConfigs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if v, _ := args["Search"].(string); v != "" {
		if utf8.RuneCountInString(v) > maxSearchLen {
			return mcp.NewToolResultError(fmt.Sprintf("Search must be at most %d characters", maxSearchLen)), nil
		}
		query.Set("search", v)
	}

	path := agentConfigsPath
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListAgentConfigsResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list agent configs", err), nil
	}
	if out.Configs == nil {
		out.Configs = []AgentConfigSummary{}
	}
	return agentConfigListOut.Result(out)
}

func (t *Tool) getAgentConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	configID, _ := req.GetArguments()["ConfigID"].(string)
	if configID == "" {
		return mcp.NewToolResultError("ConfigID is required"), nil
	}
	if _, err := uuid.Parse(configID); err != nil {
		return mcp.NewToolResultError("ConfigID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, agentConfigsPath+"/"+configID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out AgentConfigResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get agent config", err), nil
	}
	return agentConfigOut.Result(out)
}

func (t *Tool) createAgentConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	name, _ := args["Name"].(string)
	if name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}

	// The API expects manifest_yaml as a YAML string. Accept either a
	// pre-formatted YAML string or a JSON object that we convert to YAML.
	var manifestYAML string
	switch v := args["Manifest"].(type) {
	case string:
		if v == "" {
			return mcp.NewToolResultError("Manifest must not be empty"), nil
		}
		manifestYAML = v
	case map[string]any:
		if len(v) == 0 {
			return mcp.NewToolResultError("Manifest must not be empty"), nil
		}
		yamlBytes, err := yaml.Marshal(v)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("Failed to convert manifest to YAML", err), nil
		}
		manifestYAML = string(yamlBytes)
	default:
		return mcp.NewToolResultError("Manifest is required (provide a YAML string or a JSON object)"), nil
	}

	body := map[string]any{
		"name":          name,
		"manifest_yaml": manifestYAML,
	}
	if v, ok := args["ManageModelAccessKey"].(bool); ok {
		body["manage_model_access_key"] = v
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, agentConfigsPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out AgentConfigResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create agent config", err), nil
	}
	return agentConfigOut.Result(out)
}

func (t *Tool) deleteAgentConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	configID, _ := req.GetArguments()["ConfigID"].(string)
	if configID == "" {
		return mcp.NewToolResultError("ConfigID is required"), nil
	}
	if _, err := uuid.Parse(configID); err != nil {
		return mcp.NewToolResultError("ConfigID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodDelete, agentConfigsPath+"/"+configID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	if _, err := client.Do(ctx, apiReq, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete agent config", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Agent config %s deleted successfully.", configID)), nil
}

func (t *Tool) cloneAgentConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		return mcp.NewToolResultError("Name is required for clone"), nil
	}
	body := map[string]any{
		"name": name,
	}
	if v, ok := args["Secrets"].(map[string]any); ok && len(v) > 0 {
		body["secrets"] = v
	}
	if v, ok := args["OAuthAssignments"].(map[string]any); ok && len(v) > 0 {
		body["oauth_assignments"] = v
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, agentConfigsPath+"/"+configID+"/clone", nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out AgentConfigResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to clone agent config", err), nil
	}
	return agentConfigOut.Result(out)
}

func (t *Tool) listConfigSessions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if v, _ := args["Status"].(string); v != "" {
		query.Set("status", v)
	}

	path := agentConfigsPath + "/" + configID + "/sessions"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListSessionsResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list config sessions", err), nil
	}
	if out.Sessions == nil {
		out.Sessions = []Session{}
	}
	return sessionListOut.Result(out)
}

// Tools returns the Harness Runtime MCP tools.
func (t *Tool) Tools() []server.ServerTool {
	tools := []server.ServerTool{
		{
			Handler: t.listAgentConfigs,
			Tool: mcp.NewTool("harness-runtime-list-agent-configs",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				agentConfigListOut.Schema(),
				mcp.WithDescription("List the active hosted agent configs (Environment Configs) for the team. Returns lightweight metadata (id, name, content hash, creator, timestamps) without manifests; use the returned id as the agent config id for sessions and simulations. Results are keyset-paginated: pass next_page_token back as PageToken to fetch the next page."),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
				mcp.WithString("Search", mcp.Description("Optional case-insensitive substring filter on config name (max 64 characters)")),
			),
		},
		{
			Handler: t.getAgentConfig,
			Tool: mcp.NewTool("harness-runtime-get-agent-config",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				agentConfigOut.Schema(),
				mcp.WithDescription("Get one active hosted agent config (Environment Config) by id, including its sanitized environment spec (manifest) and redacted credential slot metadata. Secret values are never returned."),
				mcp.WithString("ConfigID", mcp.Required(), mcp.Description("The agent config UUID, as returned by harness-runtime-list-agent-configs")),
			),
		},
		{
			Handler: t.createAgentConfig,
			Tool: mcp.NewTool("harness-runtime-create-agent-config",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				agentConfigOut.Schema(),
				mcp.WithDescription("Create a new hosted agent config from a name and an agents.yaml manifest. The manifest can be provided as a YAML string or a JSON object (which will be converted to YAML). Configs are immutable after creation; clone to change content. Returns the created config including its assigned UUID. SECURITY WARNING: if the manifest's top-level `secrets` map declares any credentials (e.g. API keys), those values are write-only — extracted into Secrets Manager and never persisted or echoed back by this or any other tool. Never commit real secret values to files or source control, and avoid passing them through a terminal that logs history; prefer environment variables or the DO Console."),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Config name (1-64 chars, alphanumeric/dots/hyphens/underscores, unique among the team's active configs)")),
				mcp.WithObject("Manifest", mcp.Required(), mcp.Description("The agent manifest as a JSON object (e.g. {\"agent\":\"codex\",\"size\":\"mars-2vcpu-4gb\",\"env\":{...},\"secrets\":{...}}) — will be serialized to YAML for the API. Alternatively pass a pre-formatted YAML string. Any values under `secrets` are treated as sensitive write-only credentials (see security warning above).")),
				mcp.WithBoolean("ManageModelAccessKey", mcp.Description("Ask the platform to provision a DO serverless-inference key for the agent (default false)")),
			),
		},
		{
			Handler: t.deleteAgentConfig,
			Tool: mcp.NewTool("harness-runtime-delete-agent-config",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Permanently delete a hosted agent config by id. Any sessions that reference this config will remain but cannot be cloned. Returns 204 on success."),
				mcp.WithString("ConfigID", mcp.Required(), mcp.Description("The agent config UUID to delete")),
			),
		},
		{
			Handler: t.cloneAgentConfig,
			Tool: mcp.NewTool("harness-runtime-clone-agent-config",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				agentConfigOut.Schema(),
				mcp.WithDescription("Clone an existing hosted agent config into a new config with a fresh UUID. Useful for creating a variant of an existing environment spec without modifying the original. Returns the newly created clone. SECURITY WARNING: credentials are never copied from the source config. If the source manifest declares tenantSecret or oauth slots, their values are required here (via Secrets / OAuthAssignments) or the clone is rejected. Secrets values are write-only — never persisted or echoed back; avoid passing real secret values through a terminal that logs history."),
				mcp.WithString("ConfigID", mcp.Required(), mcp.Description("The agent config UUID to clone")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name for the clone (1-64 chars, alphanumeric/dots/hyphens/underscores, unique among the team's active configs)")),
				mcp.WithObject("Secrets", mcp.Description("Write-only map of tenantSecret slot name -> plaintext value, required when the source config declares those slots (never copied from the source; never returned by any tool)")),
				mcp.WithObject("OAuthAssignments", mcp.Description("Map of oauth slot name -> OAuth assignment UUID, required when the source config declares those slots (never copied from the source)")),
			),
		},
		{
			Handler: t.listConfigSessions,
			Tool: mcp.NewTool("harness-runtime-list-config-sessions",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				sessionListOut.Schema(),
				mcp.WithDescription("List all sessions that were created from a specific agent config. Results are keyset-paginated; pass next_page_token back as PageToken for the next page."),
				mcp.WithString("ConfigID", mcp.Required(), mcp.Description("The agent config UUID whose sessions to list")),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
				mcp.WithString("Status", mcp.Description("Optional status filter: SESSION_STATUS_UNSPECIFIED, SESSION_STATUS_PROVISIONING, SESSION_STATUS_READY, SESSION_STATUS_DETACHED, SESSION_STATUS_DESTROYING, SESSION_STATUS_DESTROYED, SESSION_STATUS_FAILED, SESSION_STATUS_PAUSED, SESSION_STATUS_ALL")),
			),
		},
	}
	tools = append(tools, t.sessionTools()...)
	tools = append(tools, t.sandboxTools()...)
	return tools
}
