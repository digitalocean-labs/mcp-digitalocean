package harnessruntime

import (
	"context"
	"fmt"
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

// Tools returns the Harness Runtime MCP tools.
func (t *Tool) Tools() []server.ServerTool {
	return []server.ServerTool{
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
	}
}
