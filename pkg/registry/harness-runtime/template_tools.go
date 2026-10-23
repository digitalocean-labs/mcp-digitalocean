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
	templatesPath = "v2/agents/templates"
)

// SandboxTemplateImageSource describes the OCI image that backs a template.
type SandboxTemplateImageSource struct {
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest,omitempty"`
}

// SandboxTemplateSpec is the resolved build specification of a template.
type SandboxTemplateSpec struct {
	BaseTemplate string                      `json:"base_template,omitempty"`
	Image        *SandboxTemplateImageSource `json:"image,omitempty"`
}

// SandboxTemplate is the full representation of a custom sandbox template.
type SandboxTemplate struct {
	TemplateID string               `json:"template_id"`
	Name       string               `json:"name"`
	Spec       *SandboxTemplateSpec `json:"spec,omitempty"`
	Status     string               `json:"status"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

// ListTemplatesResponse is the response of GET /v2/agents/templates.
type ListTemplatesResponse struct {
	Templates     []SandboxTemplate `json:"templates"`
	NextPageToken string            `json:"next_page_token,omitempty"`
}

// CreateTemplateResponse is the response of POST /v2/agents/templates and PUT .../templates/{id}.
type CreateTemplateResponse struct {
	Template SandboxTemplate `json:"template"`
}

// GetTemplateResponse is the response of GET /v2/agents/templates/{id}.
type GetTemplateResponse struct {
	Template SandboxTemplate `json:"template"`
}

// DeleteTemplateResponse is the response of DELETE /v2/agents/templates/{id}.
type DeleteTemplateResponse struct {
	TemplateID string `json:"template_id"`
	Deleted    bool   `json:"deleted"`
}

// SandboxTemplateBuild records one build attempt of a template.
type SandboxTemplateBuild struct {
	BuildID    string               `json:"build_id"`
	TemplateID string               `json:"template_id,omitempty"`
	Name       string               `json:"name,omitempty"`
	Spec       *SandboxTemplateSpec `json:"spec,omitempty"`
	Status     string               `json:"status"`
	Error      string               `json:"error,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

// ListTemplateBuildsResponse is the response of GET .../templates/{id}/builds.
type ListTemplateBuildsResponse struct {
	Builds        []SandboxTemplateBuild `json:"builds"`
	NextPageToken string                 `json:"next_page_token,omitempty"`
}

// GetTemplateBuildResponse is the response of GET .../templates/{id}/builds/{build_id}.
type GetTemplateBuildResponse struct {
	Build SandboxTemplateBuild `json:"build"`
}

// listTemplates returns the team's custom sandbox templates.
func (t *Tool) listTemplates(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	path := templatesPath
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListTemplatesResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list templates", err), nil
	}
	if out.Templates == nil {
		out.Templates = []SandboxTemplate{}
	}
	return templateListOut.Result(out)
}

// getTemplate returns a single template by UUID.
func (t *Tool) getTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	templateID, _ := req.GetArguments()["TemplateID"].(string)
	if templateID == "" {
		return mcp.NewToolResultError("TemplateID is required"), nil
	}
	if _, err := uuid.Parse(templateID); err != nil {
		return mcp.NewToolResultError("TemplateID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, templatesPath+"/"+templateID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out GetTemplateResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get template", err), nil
	}
	return templateOut.Result(out)
}

// createTemplate registers a new custom sandbox template.
func (t *Tool) createTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	name, _ := args["Name"].(string)
	if name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}
	baseTemplate, _ := args["BaseTemplate"].(string)
	if baseTemplate == "" {
		return mcp.NewToolResultError("BaseTemplate is required"), nil
	}
	sourceOCIRef, _ := args["SourceOCIRef"].(string)
	if sourceOCIRef == "" {
		return mcp.NewToolResultError("SourceOCIRef is required"), nil
	}

	body := map[string]any{
		"name":           name,
		"base_template":  baseTemplate,
		"source_oci_ref": sourceOCIRef,
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPost, templatesPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out CreateTemplateResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create template", err), nil
	}
	return templateCreateOut.Result(out)
}

// updateTemplate updates an existing custom sandbox template (triggers a new build).
func (t *Tool) updateTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	templateID, _ := args["TemplateID"].(string)
	if templateID == "" {
		return mcp.NewToolResultError("TemplateID is required"), nil
	}
	if _, err := uuid.Parse(templateID); err != nil {
		return mcp.NewToolResultError("TemplateID must be a UUID"), nil
	}

	sourceOCIRef, hasSource := args["SourceOCIRef"].(string)
	baseTemplate, hasBase := args["BaseTemplate"].(string)
	if !hasSource && !hasBase {
		return mcp.NewToolResultError("At least one of SourceOCIRef or BaseTemplate must be provided"), nil
	}

	body := map[string]any{}
	if hasSource && sourceOCIRef != "" {
		body["source_oci_ref"] = sourceOCIRef
	}
	if hasBase && baseTemplate != "" {
		body["base_template"] = baseTemplate
	}
	if len(body) == 0 {
		return mcp.NewToolResultError("At least one of SourceOCIRef or BaseTemplate must be non-empty"), nil
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodPut, templatesPath+"/"+templateID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out CreateTemplateResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to update template", err), nil
	}
	return templateCreateOut.Result(out)
}

// deleteTemplate permanently removes a custom sandbox template.
func (t *Tool) deleteTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	templateID, _ := req.GetArguments()["TemplateID"].(string)
	if templateID == "" {
		return mcp.NewToolResultError("TemplateID is required"), nil
	}
	if _, err := uuid.Parse(templateID); err != nil {
		return mcp.NewToolResultError("TemplateID must be a UUID"), nil
	}

	apiReq, err := client.NewRequest(ctx, http.MethodDelete, templatesPath+"/"+templateID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out DeleteTemplateResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete template", err), nil
	}
	return templateDeleteOut.Result(out)
}

// listTemplateBuilds enumerates builds for a template.
func (t *Tool) listTemplateBuilds(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	templateID, _ := args["TemplateID"].(string)
	if templateID == "" {
		return mcp.NewToolResultError("TemplateID is required"), nil
	}
	if _, err := uuid.Parse(templateID); err != nil {
		return mcp.NewToolResultError("TemplateID must be a UUID"), nil
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

	path := templatesPath + "/" + templateID + "/builds"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListTemplateBuildsResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list template builds", err), nil
	}
	if out.Builds == nil {
		out.Builds = []SandboxTemplateBuild{}
	}
	return templateBuildListOut.Result(out)
}

// getTemplateBuild returns a single build of a template.
func (t *Tool) getTemplateBuild(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	templateID, _ := args["TemplateID"].(string)
	if templateID == "" {
		return mcp.NewToolResultError("TemplateID is required"), nil
	}
	if _, err := uuid.Parse(templateID); err != nil {
		return mcp.NewToolResultError("TemplateID must be a UUID"), nil
	}
	buildID, _ := args["BuildID"].(string)
	if buildID == "" {
		return mcp.NewToolResultError("BuildID is required"), nil
	}
	if _, err := uuid.Parse(buildID); err != nil {
		return mcp.NewToolResultError("BuildID must be a UUID"), nil
	}

	path := templatesPath + "/" + templateID + "/builds/" + buildID
	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out GetTemplateBuildResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get template build", err), nil
	}
	return templateBuildOut.Result(out)
}

// templateTools returns the template MCP tools to be merged into Tools().
func (t *Tool) templateTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listTemplates,
			Tool: mcp.NewTool("harness-runtime-list-templates",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				templateListOut.Schema(),
				mcp.WithDescription("List the team's custom sandbox templates. Returns template id, name, status, spec, and timestamps. Keyset-paginated: pass next_page_token back as PageToken for the next page. Only READY templates can be referenced in agent configs."),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
			),
		},
		{
			Handler: t.getTemplate,
			Tool: mcp.NewTool("harness-runtime-get-template",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				templateOut.Schema(),
				mcp.WithDescription("Get a single custom sandbox template by UUID, including its spec (base template + OCI image source) and current status."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template UUID")),
			),
		},
		{
			Handler: t.createTemplate,
			Tool: mcp.NewTool("harness-runtime-create-template",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				templateCreateOut.Schema(),
				mcp.WithDescription("Register a new custom sandbox template. The platform pulls the OCI image referenced by SourceOCIRef on top of the BaseTemplate and builds the template asynchronously. Poll harness-runtime-get-template until status is TEMPLATE_STATUS_READY. The template can then be referenced in agent config manifests. Returns the created template including its assigned UUID."),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Template name (human-readable label)")),
				mcp.WithString("BaseTemplate", mcp.Required(), mcp.Description("Base template key. One of: codex-base, opencode-base, claude-code-base, hermes-base, hermes-byoa-base, langgraph-base, coding-base, crewai")),
				mcp.WithString("SourceOCIRef", mcp.Required(), mcp.Description("OCI image reference to layer on top of the base template (e.g. registry.example.com/my-image:latest)")),
			),
		},
		{
			Handler: t.updateTemplate,
			Tool: mcp.NewTool("harness-runtime-update-template",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				templateCreateOut.Schema(),
				mcp.WithDescription("Update a custom sandbox template's OCI image ref and/or base template key. Triggers a new build; poll until TEMPLATE_STATUS_READY. Returns the updated template."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template UUID to update")),
				mcp.WithString("SourceOCIRef", mcp.Description("New OCI image reference")),
				mcp.WithString("BaseTemplate", mcp.Description("New base template key")),
			),
		},
		{
			Handler: t.deleteTemplate,
			Tool: mcp.NewTool("harness-runtime-delete-template",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				templateDeleteOut.Schema(),
				mcp.WithDescription("Permanently delete a custom sandbox template by UUID. Agent configs referencing this template will no longer be able to create sessions. Returns the template_id and a deleted flag."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template UUID to delete")),
			),
		},
		{
			Handler: t.listTemplateBuilds,
			Tool: mcp.NewTool("harness-runtime-list-template-builds",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				templateBuildListOut.Schema(),
				mcp.WithDescription("List builds for a custom sandbox template, newest first. Each create or update triggers a new build. Keyset-paginated: pass next_page_token back as PageToken for the next page."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template UUID whose builds to list")),
				mcp.WithNumber("PageSize", mcp.Description("Page size (1-200, default 50)")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
			),
		},
		{
			Handler: t.getTemplateBuild,
			Tool: mcp.NewTool("harness-runtime-get-template-build",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				templateBuildOut.Schema(),
				mcp.WithDescription("Get a single build of a custom sandbox template by template UUID and build UUID. Includes status (PENDING, BUILDING, SUCCEEDED, FAILED), error message on failure, and the spec that was built."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template UUID that owns the build")),
				mcp.WithString("BuildID", mcp.Required(), mcp.Description("The build UUID")),
			),
		},
	}
}
