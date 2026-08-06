package functions

import (
	"context"
	_ "embed"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

//go:embed DEPLOY_SPEC.md
var deploymentGuideContent string

// DeploymentGuideTool exposes the authoritative agent-facing spec for
// deploying a DigitalOcean Functions project from a local directory via
// `doctl serverless deploy`.
//
// The tool returns the full spec as markdown. Agents are expected to call
// this tool once at the start of a deploy flow and follow it step by step.
type DeploymentGuideTool struct{}

func NewDeploymentGuideTool() *DeploymentGuideTool {
	return &DeploymentGuideTool{}
}

func (t *DeploymentGuideTool) getDeploymentGuide(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(deploymentGuideContent), nil
}

func (t *DeploymentGuideTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.getDeploymentGuide,
			Tool: mcp.NewTool("functions-deployment-guide",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription(
					"Return the authoritative step-by-step guide for deploying a DigitalOcean "+
						"Functions project from a local directory via `doctl serverless deploy` "+
						"(project-based path; needs `doctl`, a `project.yml`, and a remote build "+
						"for dependencies). Call it when the request involves multiple files or "+
						"functions, a dependency file, a build script, or an existing project "+
						"directory. Do NOT call it for a single-file, no-dependency function "+
						"(use `functions-create-or-update-action`).",
				),
			),
		},
	}
}
