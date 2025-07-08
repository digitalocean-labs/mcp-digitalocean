package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultPageSize = 30 // Default page size for listing apps
	defaultPage     = 1
)

type AppPlatformTool struct {
	client *godo.Client
}

// NewAppPlatformTool creates a new AppsTool instance
func NewAppPlatformTool(client *godo.Client) (*AppPlatformTool, error) {
	return &AppPlatformTool{client: client}, nil
}

func (a *AppPlatformTool) createAppFromAppSpec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(req.GetArguments())
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	var create godo.AppCreateRequest
	if err := json.Unmarshal(jsonBytes, &create); err != nil {
		return mcp.NewToolResultErrorFromErr("parse app spec", err), nil
	}

	if create.Spec == nil {
		return mcp.NewToolResultError("App spec is required"), nil
	}

	app, _, err := a.client.Apps.Create(ctx, &create)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	appJSON, err := json.MarshalIndent(app, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(appJSON)), nil
}

// listApps lists all apps on the DigitalOcean App Platform
func (a *AppPlatformTool) listApps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(int)
	if !ok {
		page = defaultPage
	}

	perPage, ok := req.GetArguments()["PerPage"].(int)
	if !ok {
		perPage = defaultPageSize
	}

	apps, _, err := a.client.Apps.List(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	appsJSON, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(appsJSON)), nil
}

// deleteApp deletes an existing app by its ID
func (a *AppPlatformTool) deleteApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, ok := req.GetArguments()["AppID"].(string)
	if !ok {
		return mcp.NewToolResultError("App ID is required"), nil
	}

	_, err := a.client.Apps.Delete(ctx, appID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("App deleted successfully"), nil
}

// DeploymentStatus represents the status of a deployment, including health and deployment details.
type DeploymentStatus struct {
	Health     *godo.AppHealth  `json:"health"`
	Deployment *godo.Deployment `json:"deployment"`
}

// getDeploymentStatus retrieves the deployment status of an app by its ID.
func (a *AppPlatformTool) getDeploymentStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, ok := req.GetArguments()["AppID"].(string)
	if !ok {
		return mcp.NewToolResultError("App ID is required"), nil
	}

	deployments, _, err := a.client.Apps.ListDeployments(ctx, appID, &godo.ListOptions{Page: 1, PerPage: defaultPageSize})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	if len(deployments) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("there are no deployments found for AppID %s", appID)), nil
	}

	// Get the health status of the deployment
	health, _, err := a.client.Apps.GetAppHealth(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get health status for app %s: %w", appID, err)
	}

	// Combine these two into a single response.
	deploymentStatus := DeploymentStatus{
		Health:     health,
		Deployment: deployments[0],
	}

	activeDeploymentJSON, err := json.MarshalIndent(deploymentStatus, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(activeDeploymentJSON)), nil
}

// getAppInfo retrieves an app by its ID
func (a *AppPlatformTool) getAppInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, ok := req.GetArguments()["AppID"].(string)
	if !ok {
		return mcp.NewToolResultError("App ID is required"), nil
	}

	app, _, err := a.client.Apps.Get(ctx, appID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	appJSON, err := json.MarshalIndent(app.Spec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(appJSON)), nil
}

type AppUpdate struct {
	Update AppUpdateRequest `json:"update"`
}

// AppUpdateRequest represents the request structure for updating an app
type AppUpdateRequest struct {
	// Request contains the app update request details
	Request *godo.AppUpdateRequest `json:"request"`
	// AppID is the ID of the app to update
	AppID string `json:"app_id"`
}

// updateApp updates an existing app by its ID. If the spec is not provided, this simply forces a re-deploy of the app.
func (a *AppPlatformTool) updateApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(req.GetArguments())
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	var update AppUpdate
	if err := json.Unmarshal(jsonBytes, &update); err != nil {
		return mcp.NewToolResultErrorFromErr("parse app spec", err), nil
	}

	if update.Update.Request == nil {
		deployment, _, err := a.client.Apps.CreateDeployment(ctx, update.Update.AppID, &godo.DeploymentCreateRequest{
			ForceBuild: true,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("api error", err), nil
		}

		deploymentJSON, err := json.MarshalIndent(deployment, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal error: %w", err)
		}

		return mcp.NewToolResultText(string(deploymentJSON)), nil
	}

	app, _, err := a.client.Apps.Update(ctx, update.Update.AppID, update.Update.Request)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	appJSON, err := json.MarshalIndent(app, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(appJSON)), nil
}

func (a *AppPlatformTool) Tools() []server.ServerTool {
	tools := []server.ServerTool{
		{
			Handler: a.getDeploymentStatus,
			Tool: mcp.NewTool("digitalocean-apps-get-deployment-status",
				mcp.WithDescription("Retrieves the active deployment for an application on DigitalOcean App Platform. This is useful for getting the current state of an app's latest deployment and it's health status."),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID of the app to retrieve active deployment for"))),
		},
		{
			Handler: a.listApps,
			Tool: mcp.NewTool("digitalocean-apps-list",
				mcp.WithDescription("List all applications on DigitalOcean App Platform"),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultPage), mcp.Description("The page number to retrieve (default is 1)")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultPageSize), mcp.Description("The number of items per page (default is 200)")),
			),
		},
		{
			Handler: a.deleteApp,
			Tool: mcp.NewTool("digitalocean-apps-delete",
				mcp.WithDescription("Delete an existing app on DigitalOcean App Platform"),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID of the app we want to delete.")),
			),
		},
		{
			Handler: a.getAppInfo,
			Tool: mcp.NewTool("digitalocean-apps-get-info",
				mcp.WithDescription("Get information about an application on DigitalOcean App Platform"),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID of the app to retrieve information for")),
			),
		},
	}

	appCreateSchema, err := loadSchema("app-create-schema.json")
	if err != nil {
		panic(fmt.Errorf("failed to generate app create schema: %w", err))
	}

	appCreateTool := server.ServerTool{
		Handler: a.createAppFromAppSpec,
		Tool: mcp.NewToolWithRawSchema(
			"digitalocean-apps-create-app-from-spec",
			"Creates an application from a given app spec. Within the app spec, a source has to be provided. The source can be a Git repository, a Dockerfile, or a container image.",
			appCreateSchema,
		),
	}

	appUpdateSchema, err := loadSchema("app-update-schema.json")
	if err != nil {
		panic(fmt.Errorf("failed to generate app create schema: %w", err))
	}

	appUpdateTool := server.ServerTool{
		Handler: a.updateApp,
		Tool: mcp.NewToolWithRawSchema(
			"digitalocean-apps-update",
			"Updates an existing application on DigitalOcean App Platform. The app ID and the AppSpec must be provided in the request.",
			appUpdateSchema,
		),
	}

	return append(tools, appCreateTool, appUpdateTool)
}

// loadSchema attempts to load the JSON schema from the specified file.
func loadSchema(file string) ([]byte, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	executableDir := filepath.Dir(executablePath)

	schema, err := os.ReadFile(filepath.Join(executableDir, file))
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", file, err)
	}
	return schema, nil
}
