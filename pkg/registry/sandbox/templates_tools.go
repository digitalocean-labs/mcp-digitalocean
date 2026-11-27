package sandbox

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

// ---- Templates ----

func (t *Tool) listTemplates(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var out templateListResponse
	if err := t.do(ctx, http.MethodGet, withQuery(sandboxPath("templates"), pageQuery(req.GetArguments())), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list templates", err), nil
	}
	if out.Templates == nil {
		out.Templates = []Template{}
	}
	return templateListOut.Result(out)
}

func (t *Tool) getTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errRes := requireString(req.GetArguments(), "TemplateID")
	if errRes != nil {
		return errRes, nil
	}

	var out templateResponse
	if err := t.do(ctx, http.MethodGet, sandboxPath("templates", id), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get template", err), nil
	}
	return templateOut.Result(out)
}

func (t *Tool) createTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	name, errRes := requireString(args, "Name")
	if errRes != nil {
		return errRes, nil
	}
	base, errRes := requireString(args, "BaseTemplate")
	if errRes != nil {
		return errRes, nil
	}
	ociRef, errRes := requireString(args, "SourceOCIRef")
	if errRes != nil {
		return errRes, nil
	}

	body := templateCreateRequest{Name: name, BaseTemplate: base, SourceOCIRef: ociRef}
	var out templateResponse
	if err := t.do(ctx, http.MethodPost, sandboxPath("templates"), body, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create template", err), nil
	}
	return templateOut.Result(out)
}

func (t *Tool) updateTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "TemplateID")
	if errRes != nil {
		return errRes, nil
	}

	body := templateUpdateRequest{
		BaseTemplate: argString(args, "BaseTemplate"),
		SourceOCIRef: argString(args, "SourceOCIRef"),
	}
	if body.BaseTemplate == "" && body.SourceOCIRef == "" {
		return mcp.NewToolResultError("Provide BaseTemplate, SourceOCIRef, or both"), nil
	}

	var out templateResponse
	if err := t.do(ctx, http.MethodPut, sandboxPath("templates", id), body, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to update template", err), nil
	}
	return templateOut.Result(out)
}

func (t *Tool) deleteTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, errRes := requireString(req.GetArguments(), "TemplateID")
	if errRes != nil {
		return errRes, nil
	}

	if err := t.do(ctx, http.MethodDelete, sandboxPath("templates", id), nil, nil); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete template", err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Template %s deleted.", id)), nil
}

// ---- Template builds ----

func (t *Tool) listTemplateBuilds(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, errRes := requireString(args, "TemplateID")
	if errRes != nil {
		return errRes, nil
	}

	var out templateBuildListResponse
	if err := t.do(ctx, http.MethodGet, withQuery(sandboxPath("templates", id, "builds"), pageQuery(args)), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list template builds", err), nil
	}
	if out.Builds == nil {
		out.Builds = []TemplateBuild{}
	}
	return buildListOut.Result(out)
}

func (t *Tool) getTemplateBuild(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	templateID, errRes := requireString(args, "TemplateID")
	if errRes != nil {
		return errRes, nil
	}
	buildID, errRes := requireString(args, "BuildID")
	if errRes != nil {
		return errRes, nil
	}

	var out templateBuildResponse
	if err := t.do(ctx, http.MethodGet, sandboxPath("templates", templateID, "builds", buildID), nil, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get template build", err), nil
	}
	return buildOut.Result(out)
}

// templateTools returns the template and template-build tools.
func (t *Tool) templateTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.listTemplates,
			Tool: mcp.NewTool("sandbox-template-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				templateListOut.Schema(),
				mcp.WithDescription("List the team's sandbox templates. Keyset-paginated. Requires the sandbox_templates:read scope."),
				mcp.WithNumber("PageSize", mcp.Description("Page size")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
			),
		},
		{
			Handler: t.getTemplate,
			Tool: mcp.NewTool("sandbox-template-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				templateOut.Schema(),
				mcp.WithDescription("Get one sandbox template by ID, including its spec and build status. Requires the sandbox_templates:read scope."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template ID")),
			),
		},
		{
			Handler: t.createTemplate,
			Tool: mcp.NewTool("sandbox-template-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				templateOut.Schema(),
				mcp.WithDescription("Create a sandbox template from a base template and an OCI image. The build runs asynchronously; poll sandbox-template-get until status is TEMPLATE_STATUS_READY. Requires the sandbox_templates:create scope."),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Template name")),
				mcp.WithString("BaseTemplate", mcp.Required(), mcp.Description("Base template key to layer the image on")),
				mcp.WithString("SourceOCIRef", mcp.Required(), mcp.Description("OCI image reference to build from")),
			),
		},
		{
			Handler: t.updateTemplate,
			Tool: mcp.NewTool("sandbox-template-update",
				common.WithHints(common.HintsOverwrite),
				common.WithRisk(common.RiskMedium),
				templateOut.Schema(),
				mcp.WithDescription("Update a sandbox template's base template or OCI image. Triggers a new build; poll sandbox-template-get until READY. Requires the sandbox_templates:update scope."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template ID")),
				mcp.WithString("BaseTemplate", mcp.Description("New base template key")),
				mcp.WithString("SourceOCIRef", mcp.Description("New OCI image reference")),
			),
		},
		{
			Handler: t.deleteTemplate,
			Tool: mcp.NewTool("sandbox-template-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Permanently delete a sandbox template. Destructive. Requires the sandbox_templates:delete scope."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template ID to delete")),
			),
		},
		{
			Handler: t.listTemplateBuilds,
			Tool: mcp.NewTool("sandbox-template-build-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				buildListOut.Schema(),
				mcp.WithDescription("List builds for a sandbox template. Keyset-paginated. Requires the sandbox_templates:read scope."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template ID")),
				mcp.WithNumber("PageSize", mcp.Description("Page size")),
				mcp.WithString("PageToken", mcp.Description("Opaque cursor from the previous response's next_page_token")),
			),
		},
		{
			Handler: t.getTemplateBuild,
			Tool: mcp.NewTool("sandbox-template-build-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				buildOut.Schema(),
				mcp.WithDescription("Get one build of a sandbox template, including its status and error message on failure. Requires the sandbox_templates:read scope."),
				mcp.WithString("TemplateID", mcp.Required(), mcp.Description("The template ID")),
				mcp.WithString("BuildID", mcp.Required(), mcp.Description("The build ID")),
			),
		},
	}
}
