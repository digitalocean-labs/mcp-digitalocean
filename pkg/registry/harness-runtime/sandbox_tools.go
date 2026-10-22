package harnessruntime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	sandboxSizesPath = "v2/agents/sessions/sandbox/sizes"
)

// SandboxSize describes the compute resources available for a sandbox.
type SandboxSize struct {
	Slug             string  `json:"slug"`
	VCPUs            int     `json:"vcpus,omitempty"`
	MemoryMB         int     `json:"memory_mb,omitempty"`
	PricePerHourUSD  float64 `json:"price_per_hour_usd,omitempty"`
	PricePerMonthUSD float64 `json:"price_per_month_usd,omitempty"`
}

// ListSandboxSizesResponse is the response of GET /v2/agents/sessions/sandbox/sizes.
type ListSandboxSizesResponse struct {
	Sizes []SandboxSize `json:"sizes"`
}

// RuleVerdict is a per-rule verdict from ValidatePolicy.
type RuleVerdict struct {
	Rule    string `json:"rule,omitempty"`
	Verdict string `json:"verdict,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// ValidatePolicyResponse is the response of POST /v2/agents/sessions/policy/validate.
type ValidatePolicyResponse struct {
	Agent                 string        `json:"agent,omitempty"`
	Version               string        `json:"version,omitempty"`
	OK                    bool          `json:"ok"`
	DefaultAction         string        `json:"defaultAction,omitempty"`
	DefaultActionVerdict  string        `json:"defaultActionVerdict,omitempty"`
	DefaultActionRendered string        `json:"defaultActionRendered,omitempty"`
	DefaultActionReason   string        `json:"defaultActionReason,omitempty"`
	Verdicts              []RuleVerdict `json:"verdicts,omitempty"`
}

// listSandboxSizes returns the catalog of available sandbox compute sizes.
func (t *Tool) listSandboxSizes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, sandboxSizesPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListSandboxSizesResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list sandbox sizes", err), nil
	}
	if out.Sizes == nil {
		out.Sizes = []SandboxSize{}
	}
	return sandboxSizesOut.Result(out)
}

// validatePolicy checks a manifest's tool-permission policy against the
// agent's capabilities without creating a session. Accepts the same manifest
// as create-agent-config (JSON object → YAML, or a pre-formatted YAML string).
func (t *Tool) validatePolicy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()

	// Accept Manifest as JSON object (→ YAML) or raw YAML string.
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

	path := sessionsPath + "/policy/validate"
	apiReq, err := client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/x-yaml")
	apiReq.Body = io.NopCloser(bytes.NewReader([]byte(manifestYAML)))
	apiReq.ContentLength = int64(len(manifestYAML))

	var out ValidatePolicyResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to validate policy", err), nil
	}
	return validatePolicyOut.Result(out)
}

// sandboxTools returns the sandbox MCP tools to be merged into Tools().
func (t *Tool) sandboxTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listSandboxSizes,
			Tool: mcp.NewTool("harness-runtime-list-sandbox-sizes",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				sandboxSizesOut.Schema(),
				mcp.WithDescription("List the available sandbox compute sizes (slug, name, vCPUs, memory, disk) that can be referenced when creating agent configs or sessions. The returned slugs are the valid values for sandbox size fields in manifests."),
			),
		},
		{
			Handler: t.validatePolicy,
			Tool: mcp.NewTool("harness-runtime-validate-policy",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				validatePolicyOut.Schema(),
				mcp.WithDescription("Stateless preflight: check a manifest's tool-permission policy against the agent's capabilities without creating a session. Returns per-rule verdicts (exact, degraded, unsupported, unverified). ok=false means CreateSession would reject this policy. Accepts the same manifest as create-agent-config."),
				mcp.WithObject("Manifest", mcp.Required(), mcp.Description("The agent manifest as a JSON object (same shape as create-agent-config's Manifest — must include a permissions block to validate). Will be serialized to YAML.")),
			),
		},
	}
}
