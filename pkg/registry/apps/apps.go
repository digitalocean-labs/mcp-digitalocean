package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	_ "embed"
)

//go:embed spec/app-create-schema.json
var appCreateSchemaJSON []byte

//go:embed spec/app-update-schema.json
var appUpdateSchemaJSON []byte

const (
	defaultPageSize = 20 // Default page size for listing apps
	defaultPage     = 1
)

type AppPlatformTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewAppPlatformTool creates a new AppsTool instance
func NewAppPlatformTool(client func(ctx context.Context) (*godo.Client, error)) (*AppPlatformTool, error) {
	return &AppPlatformTool{client: client}, nil
}

func (a *AppPlatformTool) createAppFromAppSpec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(req.GetArguments())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request arguments for app creation: %w", err)
	}

	var create godo.AppCreateRequest
	if err := json.Unmarshal(jsonBytes, &create); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to parse app creation request", err), nil
	}

	if create.Spec == nil {
		return mcp.NewToolResultError("App spec is required"), nil
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	app, _, err := client.Apps.Create(ctx, &create)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create app", err), nil
	}

	appJSON, err := json.MarshalIndent(app, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format created app response: %w", err)
	}

	return mcp.NewToolResultText(string(appJSON)), nil
}

type AppSummary struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Url       string          `json:"url"`
	ProjectID string          `json:"project_id,omitempty"` // Optional field, not always present
	Region    *godo.AppRegion `json:"region"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Tier      string          `json:"tier"`
}

func toAppSummary(app *godo.App) *AppSummary {
	return &AppSummary{
		ID:        app.GetID(),
		Name:      app.GetSpec().GetName(),
		Url:       app.GetLiveURL(),
		ProjectID: app.GetProjectID(),
		Region:    app.GetRegion(),
		CreatedAt: app.GetCreatedAt(),
		UpdatedAt: app.GetUpdatedAt(),
		Tier:      app.GetTierSlug(),
	}
}

// listApps lists all apps on the DigitalOcean App Platform
func (a *AppPlatformTool) listApps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = defaultPage
	}

	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = defaultPageSize
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apps, _, err := client.Apps.List(ctx, &godo.ListOptions{Page: int(page), PerPage: int(perPage)})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to retrieve apps list", err), nil
	}

	// create a slice of app summaries
	summaries := make([]*AppSummary, len(apps))
	for i, app := range apps {
		// Convert each app to a summary format
		summaries[i] = toAppSummary(app)
	}

	appsJSON, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format apps list response: %w", err)
	}

	return mcp.NewToolResultText(string(appsJSON)), nil
}

// deleteApp deletes an existing app by its ID
func (a *AppPlatformTool) deleteApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, ok := req.GetArguments()["AppID"].(string)
	if !ok {
		return mcp.NewToolResultError("App ID is required"), nil
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Apps.Delete(ctx, appID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr(fmt.Sprintf("failed to delete app %s", appID), err), nil
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

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	deployments, _, err := client.Apps.ListDeployments(ctx, appID, &godo.ListOptions{Page: 1, PerPage: defaultPageSize})
	if err != nil {
		return mcp.NewToolResultErrorFromErr(fmt.Sprintf("failed to list deployments for app %s", appID), err), nil
	}

	if len(deployments) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("no deployments found for app %s", appID)), nil
	}

	client, err = a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Get the health status of the deployment
	health, _, err := client.Apps.GetAppHealth(ctx, appID)
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
		return nil, fmt.Errorf("failed to marshal deployment status: %w", err)
	}

	return mcp.NewToolResultText(string(activeDeploymentJSON)), nil
}

// getAppInfo retrieves an app by its ID
func (a *AppPlatformTool) getAppInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, ok := req.GetArguments()["AppID"].(string)
	if !ok {
		return mcp.NewToolResultError("App ID is required"), nil
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	app, _, err := client.Apps.Get(ctx, appID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr(fmt.Sprintf("failed to get app %s", appID), err), nil
	}

	appJSON, err := json.MarshalIndent(app.Spec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal app spec: %w", err)
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
		return nil, fmt.Errorf("failed to marshal update request: %w", err)
	}

	var update AppUpdate
	if err := json.Unmarshal(jsonBytes, &update); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to parse app update request", err), nil
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	if update.Update.Request == nil {
		deployment, _, err := client.Apps.CreateDeployment(ctx, update.Update.AppID, &godo.DeploymentCreateRequest{
			ForceBuild: true,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr(fmt.Sprintf("failed to create deployment for app %s", update.Update.AppID), err), nil
		}

		deploymentJSON, err := json.MarshalIndent(deployment, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal deployment: %w", err)
		}

		return mcp.NewToolResultText(string(deploymentJSON)), nil
	}

	app, _, err := client.Apps.Update(ctx, update.Update.AppID, update.Update.Request)
	if err != nil {
		return mcp.NewToolResultErrorFromErr(fmt.Sprintf("failed to update app %s", update.Update.AppID), err), nil
	}

	appJSON, err := json.MarshalIndent(app, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated app: %w", err)
	}

	return mcp.NewToolResultText(string(appJSON)), nil
}

// getAppLogs retrieves logs for an app deployment
func (a *AppPlatformTool) getAppLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, ok := req.GetArguments()["AppID"].(string)
	if !ok {
		return mcp.NewToolResultError("App ID is required and must be a string"), nil
	}

	deploymentID, ok := req.GetArguments()["DeploymentID"].(string)
	if !ok {
		return mcp.NewToolResultError("Deployment ID is required and must be a string"), nil
	}

	component, ok := req.GetArguments()["Component"].(string)
	if !ok {
		return mcp.NewToolResultError("Component is required and must be a string"), nil
	}

	logTypeStr, ok := req.GetArguments()["LogType"].(string)
	if !ok {
		return mcp.NewToolResultError("LogType is required and must be a string"), nil
	}
	logType := godo.AppLogType(logTypeStr)

	// Optional parameters
	follow := false
	if followVal, ok := req.GetArguments()["Follow"].(bool); ok {
		follow = followVal
	}

	tailLines := 100 // Default to 100 lines
	if tailVal, ok := req.GetArguments()["TailLines"].(int); ok {
		tailLines = int(tailVal)
	}
	client, err := a.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch client: %w", err)
	}

	//Call Godo.getLogs function
	logs, _, err := client.Apps.GetLogs(ctx, appID, deploymentID, component, logType, follow, tailLines)
	if err != nil {
		return mcp.NewToolResultErrorFromErr(fmt.Sprintf("failed to get logs for app %s, deployment %s, component %s", appID, deploymentID, component), err), nil
	}

	logsJSON, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal app logs: %w", err)
	}

	return mcp.NewToolResultText(string(logsJSON)), nil
}

func (a *AppPlatformTool) Tools() []server.ServerTool {
	tools := []server.ServerTool{
		{
			Handler: a.getDeploymentStatus,
			Tool: mcp.NewTool("apps-get-deployment-status",
				mcp.WithDescription("Retrieves the active deployment for an application on DigitalOcean App Platform. This is useful for getting the current state of an app's latest deployment and it's health status."),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID of the app to retrieve active deployment for"))),
		},
		{
			Handler: a.listApps,
			Tool: mcp.NewTool("apps-list",
				mcp.WithDescription("List all applications on DigitalOcean App Platform. By default, we only return a summary of the apps. To get detailed information about an app, use the `apps-get-info` with the app id."),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultPage), mcp.Description("The page number to retrieve (default is 1)")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultPageSize), mcp.Description("The number of items per page (default is 200)")),
			),
		},
		{
			Handler: a.deleteApp,
			Tool: mcp.NewTool("apps-delete",
				mcp.WithDescription("Delete an existing app on DigitalOcean App Platform. This is a destructive operation and cannot be undone."),
				mcp.WithDestructiveHintAnnotation(true),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID of the app we want to delete.")),
			),
		},
		{
			Handler: a.getAppInfo,
			Tool: mcp.NewTool("apps-get-info",
				mcp.WithDescription("Get information about an application on DigitalOcean App Platform"),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID of the app to retrieve information for")),
			),
		},
		{
			Handler: a.createAppFromAppSpec,
			Tool: mcp.NewToolWithRawSchema(
				"apps-create-app-from-spec",
				"Creates an application from a given app spec. Within the app spec, a source has to be provided. The source can be a Git repository, a Dockerfile, or a container image.",
				appCreateSchemaJSON,
			),
		},
		{
			Handler: a.updateApp,
			Tool: mcp.NewToolWithRawSchema(
				"apps-update",
				"Updates an existing application on DigitalOcean App Platform. The app ID and the AppSpec must be provided in the request.",
				appUpdateSchemaJSON,
			),
		},
		{
			Handler: a.getAppLogs,
			Tool: mcp.NewTool("apps-get-logs",
				mcp.WithDescription("Retrieves app logs for a specific app deployment and component on DigitalOcean App Platform. Returns both live and historic log URLs."),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID")),
				mcp.WithString("DeploymentID", mcp.Required(), mcp.Description("The deployment ID")),
				mcp.WithString("Component", mcp.Required(), mcp.Description("The component name to get logs for")),
				mcp.WithString("LogType", mcp.Required(), mcp.Enum("BUILD", "RUN", "DEPLOY", "RUN_RESTARTED"), mcp.Description("The type of logs to retrieve.")),
				mcp.WithBoolean("Follow", mcp.DefaultBool(false), mcp.Description("Whether to follow logs in real-time (default: false)")),
				mcp.WithNumber("TailLines", mcp.DefaultNumber(100), mcp.Description("Number of lines to retrieve from the end of logs (default: 100)")),
			),
		},
	}

	return tools
}
