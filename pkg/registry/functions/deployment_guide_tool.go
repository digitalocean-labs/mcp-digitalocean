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
						"Functions project from a local directory using `doctl serverless deploy`. "+
						"This is the **project-based** deploy path — it requires `doctl`, a local "+
						"project directory with a `project.yml`, and (for anything with dependencies) "+
						"a remote build.\n\n"+
						"Call this tool when the user's request involves ANY of:\n"+
						"- Multiple source files in one function, or multiple functions in one deploy\n"+
						"- A dependency file (`package.json` with non-empty `dependencies`, "+
						"`requirements.txt`, `go.mod`, `composer.json`)\n"+
						"- A `build.sh` / `build.cmd` script\n"+
						"- An existing local project directory (with `project.yml`) the user pointed you at\n"+
						"- An explicit mention of `doctl`, `project.yml`, or remote build\n\n"+
						"Do NOT call this tool when the user just wants a single-file function with "+
						"no dependencies — in that case, skip `doctl` entirely and call "+
						"`functions-create-or-update-action` directly with the source inline. Also "+
						"do not call this tool for routine per-action CRUD (invoke / list / delete / "+
						"update-inline) or for trigger management — use the matching `functions-*` "+
						"tool.\n\n"+
						"The guide itself covers preflight (`doctl` install, auth, serverless plugin), "+
						"namespace selection and connect via `doctl serverless connect <hint>` "+
						"(with `doctl`/MCP-server account alignment when needed), project "+
						"scaffolding via `doctl serverless init`, the local-vs-remote-build decision, "+
						"post-deploy verification via the other `functions-*` MCP tools, supported "+
						"runtimes, stop conditions, and error-handling rules.\n\n"+
						"The returned content is markdown. Follow its instructions exactly; do not "+
						"paraphrase the steps to the user.",
				),
			),
		},
	}
}
