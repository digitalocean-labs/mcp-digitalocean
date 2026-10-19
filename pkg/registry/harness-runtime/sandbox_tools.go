package harnessruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	sandboxSizesPath = "v2/agents/sessions/sandbox/sizes"
)

// SandboxSize describes the compute resources available for a sandbox.
type SandboxSize struct {
	Slug     string `json:"slug"`
	Name     string `json:"name,omitempty"`
	VCPUs    int    `json:"vcpus,omitempty"`
	MemoryMB int    `json:"memory_mb,omitempty"`
	DiskGB   int    `json:"disk_gb,omitempty"`
}

// ListSandboxSizesResponse is the response of GET /v2/agents/sandbox-sizes.
type ListSandboxSizesResponse struct {
	SandboxSizes []SandboxSize `json:"sandbox_sizes"`
}

// PolicyViolation is a single policy rule violation returned by ValidatePolicy.
type PolicyViolation struct {
	Rule    string `json:"rule,omitempty"`
	Message string `json:"message,omitempty"`
}

// ValidatePolicyResponse is the response of POST /v2/agents/sessions/{id}/policy/validate.
type ValidatePolicyResponse struct {
	Valid      bool              `json:"valid"`
	Violations []PolicyViolation `json:"violations,omitempty"`
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
	if out.SandboxSizes == nil {
		out.SandboxSizes = []SandboxSize{}
	}
	return sandboxSizesOut.Result(out)
}

// validatePolicy checks whether a policy document is syntactically and
// semantically valid for the given session without committing it.
func (t *Tool) validatePolicy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	sessionID, _ := args["SessionID"].(string)
	if sessionID == "" {
		return mcp.NewToolResultError("SessionID is required"), nil
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return mcp.NewToolResultError("SessionID must be a UUID"), nil
	}

	policy, hasPolicy := args["Policy"].(map[string]any)
	if !hasPolicy || len(policy) == 0 {
		return mcp.NewToolResultError("Policy is required and must be a non-empty object"), nil
	}

	body := map[string]any{
		"policy": policy,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	path := sessionsPath + "/" + sessionID + "/policy/validate"
	apiReq, err := client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

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
				mcp.WithDescription("Validate a policy document against a session without committing it. Returns valid=true when the policy is syntactically and semantically correct, or valid=false with a list of violations describing what needs to be fixed."),
				mcp.WithString("SessionID", mcp.Required(), mcp.Description("The session UUID to validate the policy against")),
				mcp.WithObject("Policy", mcp.Required(), mcp.Description("The policy document to validate as a JSON object")),
			),
		},
	}
}
