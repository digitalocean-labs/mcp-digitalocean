package projects

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mcp-digitalocean/pkg/registry/common"
)

const (
	defaultProjectsPageSize = 30
	defaultProjectsPage     = 1
)

// ProjectTool provides DigitalOcean project management tools.
type ProjectTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewProjectTool creates a new ProjectTool.
func NewProjectTool(client func(ctx context.Context) (*godo.Client, error)) *ProjectTool {
	return &ProjectTool{
		client: client,
	}
}

func (p *ProjectTool) listProjects(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultProjectsPage
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultProjectsPageSize
	}

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	projects, _, err := client.Projects.List(ctx, &godo.ListOptions{Page: int(page), PerPage: int(perPage)})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return projectsOut.Result(projects)
}

func (p *ProjectTool) getProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Project ID is required"), nil
	}

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	project, _, err := client.Projects.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return projectOut.Result(project)
}

func (p *ProjectTool) getDefaultProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	project, _, err := client.Projects.GetDefault(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return projectOut.Result(project)
}

func (p *ProjectTool) createProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	name, ok := args["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}

	createReq := &godo.CreateProjectRequest{Name: name}
	if purpose, ok := args["Purpose"].(string); ok && purpose != "" {
		createReq.Purpose = purpose
	}
	if description, ok := args["Description"].(string); ok {
		createReq.Description = description
	}
	if environment, ok := args["Environment"].(string); ok {
		createReq.Environment = environment
	}

	client, err := p.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	project, _, err := client.Projects.Create(ctx, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return projectOut.Result(project)
}

// Tools returns the project management tools.
func (p *ProjectTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: p.listProjects,
			Tool: mcp.NewTool("project-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				projectsOut.Schema(),
				mcp.WithDescription("List DigitalOcean projects with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultProjectsPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultProjectsPageSize), mcp.Description("Items per page")),
			),
		},
		{
			Handler: p.getProject,
			Tool: mcp.NewTool("project-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				projectOut.Schema(),
				mcp.WithDescription("Get a DigitalOcean project by ID. Pass \"default\" to get the default project."),
				mcp.WithString("ID", mcp.Required(), mcp.Description("Project UUID, or \"default\" for the account default project")),
			),
		},
		{
			Handler: p.getDefaultProject,
			Tool: mcp.NewTool("project-get-default",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				projectOut.Schema(),
				mcp.WithDescription("Get the account's default DigitalOcean project. Same call as project-get with ID \"default\": both read /v2/projects/default."),
			),
		},
		{
			Handler: p.createProject,
			Tool: mcp.NewTool("project-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskLow),
				projectOut.Schema(),
				mcp.WithDescription("Create a new DigitalOcean project"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the project")),
				mcp.WithString("Purpose", mcp.Description("Optional purpose of the project. Preferred values: \"Just trying out DigitalOcean\", \"Class project / Educational purposes\", \"Website or blog\", \"Web Application\", \"Service or API\", \"Mobile Application\", \"Machine learning / AI / Data processing\", \"IoT\", \"Operational / Developer tooling\". Other values are stored as \"Other: <value>\".")),
				mcp.WithString("Description", mcp.Description("Optional description of the project")),
				mcp.WithString("Environment", mcp.Enum("Development", "Staging", "Production"), mcp.Description("Environment of the project's resources")),
			),
		},
	}
}
