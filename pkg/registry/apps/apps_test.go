package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type getClientFn func(ctx context.Context) (*godo.Client, error)

func setupMock(t *testing.T) (getClientFn, *MockAppsService) {
	ctrl := gomock.NewController(t)
	appService := NewMockAppsService(ctrl)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Apps: appService,
		}, nil
	}

	return client, appService
}

func equalsToolResult[T any](t *testing.T, expected T, actual *mcp.CallToolResult) {
	require.NotNil(t, actual)
	require.Len(t, actual.Content, 1)

	var toolResult T
	err := json.Unmarshal([]byte(actual.Content[0].(mcp.TextContent).Text), &toolResult)

	require.NoError(t, err)
	require.Equal(t, expected, toolResult)
}

func TestUpdateApp(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		mock         func(*MockAppsService)
		mcpResult    *mcp.CallToolResult
		expectError  bool
		handlerError bool
	}{
		{
			name: "Force build (no spec)",
			args: func() map[string]any {
				update := AppUpdate{
					Update: AppUpdateRequest{
						Request: nil,
						AppID:   "app-123",
					},
				}
				b, _ := json.Marshal(update)
				var m map[string]any
				_ = json.Unmarshal(b, &m)
				return m
			}(),
			mock: func(app *MockAppsService) {
				app.EXPECT().CreateDeployment(gomock.Any(), "app-123", gomock.Any()).
					Return(&godo.Deployment{ID: "deploy-1"}, nil, nil).Times(1)
			},
			mcpResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: toJSONString(&godo.Deployment{ID: "deploy-1"}),
					},
				},
			},
		},
		{
			name: "With spec",
			args: func() map[string]any {
				update := AppUpdate{
					Update: AppUpdateRequest{
						Request: &godo.AppUpdateRequest{},
						AppID:   "app-123",
					},
				}
				b, _ := json.Marshal(update)
				var m map[string]any
				_ = json.Unmarshal(b, &m)
				return m
			}(),
			mock: func(app *MockAppsService) {
				app.EXPECT().Update(gomock.Any(), "app-123", &godo.AppUpdateRequest{}).
					Return(&godo.App{Spec: &godo.AppSpec{Name: "updated-app"}}, nil, nil).Times(1)
			},
			mcpResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: toJSONString(&godo.App{Spec: &godo.AppSpec{Name: "updated-app"}}),
					},
				},
			},
		},
		{
			name:         "Invalid JSON",
			args:         map[string]any{"invalid": make(chan int)},
			expectError:  true,
			handlerError: true,
		},
		{
			name: "API error on force build",
			args: func() map[string]any {
				update := AppUpdate{
					Update: AppUpdateRequest{
						Request: nil,
						AppID:   "app-123",
					},
				}
				b, _ := json.Marshal(update)
				var m map[string]any
				_ = json.Unmarshal(b, &m)
				return m
			}(),
			mock: func(app *MockAppsService) {
				app.EXPECT().CreateDeployment(gomock.Any(), "app-123", gomock.Any()).
					Return(nil, nil, fmt.Errorf("api error")).Times(1)
			},
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, appService := setupMock(t)
			tool := &AppPlatformTool{client: client}
			if tc.mock != nil {
				tc.mock(appService)
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateApp(context.Background(), req)
			if tc.expectError {
				if tc.handlerError {
					require.Error(t, err)
					require.Nil(t, resp)
				} else {
					require.NoError(t, err)
					require.NotNil(t, resp)
					require.True(t, resp.IsError)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tc.mcpResult != nil {
				require.Equal(t, tc.mcpResult, resp)
			}
		})
	}
}

func TestListApps(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		expectedApps []*godo.App
		mock         func(app *MockAppsService, apps []*godo.App)
		expectError  bool
		handlerError bool
	}{
		{
			name:         "Successful list",
			args:         map[string]any{"Page": float64(1), "PerPage": float64(2)},
			expectedApps: []*godo.App{{ID: "1"}, {ID: "2"}},
			mock: func(app *MockAppsService, apps []*godo.App) {
				app.EXPECT().List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(apps, nil, nil).Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Page": float64(1), "PerPage": float64(2)},
			mock: func(app *MockAppsService, apps []*godo.App) {
				app.EXPECT().List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(nil, nil, fmt.Errorf("api error")).Times(1)
			},
			expectError: true,
		},
		{
			name:         "page size not provided",
			expectedApps: []*godo.App{},
			mock: func(app *MockAppsService, apps []*godo.App) {
				app.EXPECT().List(gomock.Any(), &godo.ListOptions{Page: defaultPage, PerPage: defaultPageSize}).
					Return(apps, nil, nil).Times(1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, appService := setupMock(t)
			tool := &AppPlatformTool{client: client}
			if tc.mock != nil {
				tc.mock(appService, tc.expectedApps)
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listApps(context.Background(), req)
			if tc.expectError {
				if tc.handlerError {
					require.Error(t, err)
					require.Nil(t, resp)
				} else {
					require.NoError(t, err)
					require.NotNil(t, resp)
					require.True(t, resp.IsError)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			equalsToolResult(t, tc.expectedApps, resp)
		})
	}
}

func toJSONString(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("error marshaling to JSON: %v", err)
	}
	return string(data)
}

func TestCreateAppFromAppSpec(t *testing.T) {
	baseSpec := &godo.AppSpec{
		Name: "test-app",
		Services: []*godo.AppServiceSpec{
			{
				Name: "test-service",
				Git: &godo.GitSourceSpec{
					RepoCloneURL: "https://repo-clone-url.com/test/repo.git",
					Branch:       "main",
				},
			},
		},
	}

	tests := []struct {
		name         string
		spec         *godo.AppSpec
		mock         func(app *MockAppsService)
		mcpResult    *mcp.CallToolResult
		expectError  bool
		handlerError bool
	}{
		{
			name: "Successful create",
			spec: &godo.AppSpec{
				Name: "test-app",
				Services: []*godo.AppServiceSpec{
					{
						Name: "test-service",
						Git: &godo.GitSourceSpec{
							RepoCloneURL: "https://repo-clone-url.com/test/repo.git",
							Branch:       "main",
						},
					},
				},
			},
			mock: func(app *MockAppsService) {
				app.EXPECT().Create(gomock.Any(), &godo.AppCreateRequest{Spec: baseSpec}).Return(&godo.App{Spec: baseSpec}, nil, nil).Times(1)
			},
			mcpResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: toJSONString(&godo.App{Spec: baseSpec}),
					},
				},
			},
		},
		{
			name: "Invalid arguments (marshal error)",
			mcpResult: &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "App spec is required",
					},
				},
			},
		},
		{
			name: "API error",
			spec: baseSpec,
			mock: func(app *MockAppsService) {
				app.EXPECT().Create(gomock.Any(), &godo.AppCreateRequest{Spec: baseSpec}).Return(nil, nil, fmt.Errorf("api error")).Times(1)
			},
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, appService := setupMock(t)
			tool := &AppPlatformTool{client: client}
			if tc.mock != nil {
				tc.mock(appService)
			}

			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"spec": tc.spec}}}
			resp, err := tool.createAppFromAppSpec(context.Background(), req)
			if tc.expectError {
				if tc.handlerError {
					require.Error(t, err)
					require.Nil(t, resp)
				} else {
					require.NoError(t, err)
					require.NotNil(t, resp)
					require.True(t, resp.IsError)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tc.mcpResult != nil {
				require.Equal(t, tc.mcpResult, resp)
			}
		})
	}
}

func TestDeleteApp(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		mock         func(app *MockAppsService)
		expectError  bool
		expectMcp    string
		handlerError bool
	}{
		{
			name: "Successful delete",
			args: map[string]any{"AppID": "app-123"},
			mock: func(app *MockAppsService) {
				app.EXPECT().Delete(gomock.Any(), "app-123").Return(nil, nil).Times(1)
			},
		},
		{
			name:      "Missing AppID",
			args:      map[string]any{},
			expectMcp: "App ID is required",
		},
		{
			name: "API error",
			args: map[string]any{"AppID": "app-123"},
			mock: func(app *MockAppsService) {
				app.EXPECT().Delete(gomock.Any(), "app-123").Return(nil, fmt.Errorf("api error")).Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, appService := setupMock(t)
			tool := &AppPlatformTool{client: client}
			if tc.mock != nil {
				tc.mock(appService)
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteApp(context.Background(), req)
			if tc.expectError {
				if tc.handlerError {
					require.Error(t, err)
					require.Nil(t, resp)
				} else {
					require.NoError(t, err)
					require.NotNil(t, resp)
					require.True(t, resp.IsError)
				}
				return
			}
			if tc.expectMcp != "" {
				require.True(t, resp.IsError)
				require.Equal(t, tc.expectMcp, resp.Content[0].(mcp.TextContent).Text)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "App deleted successfully")
		})
	}
}

func TestGetDeploymentStatus(t *testing.T) {
	testAppId := "test-app-id"
	tcs := []struct {
		name                string
		expectedDeployments []*godo.Deployment
		toolRequest         mcp.CallToolRequest
		expectErr           bool
		mcpResult           *mcp.CallToolResult
		mock                func(app *MockAppsService, deployments []*godo.Deployment)
		handlerError        bool
	}{
		{
			name: "Deployments found, returns the latest one",
			expectedDeployments: []*godo.Deployment{
				{ID: "deployment-1", Cause: "manual", Phase: godo.DeploymentPhase_Deploying},
				{ID: "deployment-2", Cause: "app spec change", Phase: godo.DeploymentPhase_Active}},
			mock: func(app *MockAppsService, deployments []*godo.Deployment) {
				app.EXPECT().ListDeployments(gomock.Any(), testAppId, gomock.Any()).
					Return(deployments, nil, nil).Times(1)
				app.EXPECT().GetAppHealth(gomock.Any(), testAppId).
					Return(&godo.AppHealth{
						Components: []*godo.ComponentHealth{
							{
								Name:               "web",
								CPUUsagePercent:    90,
								MemoryUsagePercent: 50,
								ReplicasDesired:    1,
								ReplicasReady:      1,
								State:              "HEALTHY",
							},
						},
					}, nil, nil).Times(1)
			},
			toolRequest: mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: map[string]any{"AppID": testAppId}},
			},
			mcpResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: toJSONString(&DeploymentStatus{
							Health: &godo.AppHealth{
								Components: []*godo.ComponentHealth{
									{
										Name:               "web",
										CPUUsagePercent:    90,
										MemoryUsagePercent: 50,
										ReplicasDesired:    1,
										ReplicasReady:      1,
										State:              "HEALTHY",
									},
								},
							},
							Deployment: &godo.Deployment{
								ID: "deployment-1", Cause: "manual", Phase: godo.DeploymentPhase_Deploying,
							},
						}),
					},
				},
			},
		},
		{
			name:                "No deployments found for user, returns mcp error message",
			expectedDeployments: []*godo.Deployment{},
			mock: func(app *MockAppsService, deployments []*godo.Deployment) {
				app.EXPECT().ListDeployments(gomock.Any(), testAppId, gomock.Any()).Return([]*godo.Deployment{}, nil, nil).Times(1)
			},
			toolRequest: mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: map[string]any{"AppID": testAppId}},
			},
			mcpResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("no deployments found for app %s", testAppId),
					},
				},
			},
		},
		{
			name:      "Error retrieving deployments, returns error",
			expectErr: true,
			mock: func(app *MockAppsService, deployments []*godo.Deployment) {
				app.EXPECT().ListDeployments(gomock.Any(), testAppId, gomock.Any()).Return(nil, nil, fmt.Errorf("authentication error")).Times(1)
			},
			toolRequest: mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: map[string]any{"AppID": testAppId}},
			},
		},
		{
			name: "get deployment status is called with an empty app ID, returns an mcp error message",
			mcpResult: &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "App ID is required",
					},
				},
			},
			toolRequest: mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: map[string]any{"NotAppID": testAppId}},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			client, appService := setupMock(t)
			tool := &AppPlatformTool{client: client}

			if tc.mock != nil {
				tc.mock(appService, tc.expectedDeployments)
			}

			resp, err := tool.getDeploymentStatus(context.Background(), tc.toolRequest)

			// unexpected error from the http client should return an error
			if tc.expectErr {
				if tc.handlerError {
					require.Error(t, err)
					require.Nil(t, resp)
				} else {
					require.NoError(t, err)
					require.NotNil(t, resp)
					require.True(t, resp.IsError)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tc.mcpResult != nil {
				require.Equal(t, tc.mcpResult, resp)
			}
		})
	}
}

func TestGetAppLogs(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		mock         func(app *MockAppsService)
		expectedLogs *godo.AppLogs
		expectError  bool
		expectMcp    string
		handlerError bool
	}{
		{
			name: "Successful get logs",
			args: map[string]any{"AppID": "app-123", "DeploymentID": "deployment-123", "Component": "web", "LogType": "BUILD"},
			mock: func(app *MockAppsService) {
				app.EXPECT().GetLogs(gomock.Any(), "app-123", "deployment-123", "web", godo.AppLogTypeBuild, false, 100).Return(&godo.AppLogs{LiveURL: "live-url", HistoricURLs: []string{"historic-url"}}, nil, nil).Times(1)
			},
		},
		{
			name: "Error getting logs",
			args: map[string]any{"AppID": "app-123", "DeploymentID": "deployment-123", "Component": "web", "LogType": "BUILD", "Follow": false, "TailLines": 100},
			mock: func(app *MockAppsService) {
				app.EXPECT().GetLogs(gomock.Any(), "app-123", "deployment-123", "web", godo.AppLogTypeBuild, false, 100).Return(nil, nil, fmt.Errorf("api error")).Times(1)
			},
			expectError: true,
		},
		{
			name:      "Missing AppID",
			args:      map[string]any{"DeploymentID": "deployment-123", "Component": "web", "LogType": "BUILD", "Follow": false, "TailLines": 100},
			expectMcp: "App ID is required and must be a string",
		},
		{
			name:      "Wrong type for AppID",
			args:      map[string]any{"AppID": 123, "Component": "web", "LogType": "BUILD", "Follow": false, "TailLines": 100},
			expectMcp: "App ID is required and must be a string",
		},
		{
			name: "Invalid LogType",
			args: map[string]any{"AppID": "app-123", "DeploymentID": "deployment-123", "Component": "web", "LogType": "INVALID_LOG_TYPE"},
			mock: func(app *MockAppsService) {
				app.EXPECT().GetLogs(gomock.Any(), "app-123", "deployment-123", "web", godo.AppLogType("INVALID_LOG_TYPE"), false, 100).
					Return(nil, nil, fmt.Errorf("invalid log_type: INVALID_LOG_TYPE")).Times(1)
			},
			expectError: true,
		},
		{
			name: "With optional parameters - Follow true, custom TailLines",
			args: map[string]any{"AppID": "app-123", "DeploymentID": "deployment-123", "Component": "web", "LogType": "RUN", "Follow": true, "TailLines": 50},
			mock: func(app *MockAppsService) {
				app.EXPECT().GetLogs(gomock.Any(), "app-123", "deployment-123", "web", godo.AppLogTypeRun, true, 50).Return(&godo.AppLogs{LiveURL: "live-url", HistoricURLs: []string{"historic-url"}}, nil, nil).Times(1)
			},
			expectedLogs: &godo.AppLogs{LiveURL: "live-url", HistoricURLs: []string{"historic-url"}},
		},
		{
			name: "Wrong type for Follow parameter",
			args: map[string]any{"AppID": "app-123", "DeploymentID": "deployment-123", "Component": "web", "LogType": "RUN", "Follow": "true", "TailLines": 100},
			mock: func(app *MockAppsService) {
				app.EXPECT().GetLogs(gomock.Any(), "app-123", "deployment-123", "web", godo.AppLogTypeRun, false, 100).Return(&godo.AppLogs{LiveURL: "live-url", HistoricURLs: []string{"historic-url"}}, nil, nil).Times(1)
			},
			expectedLogs: &godo.AppLogs{LiveURL: "live-url", HistoricURLs: []string{"historic-url"}},
		},
		{
			name: "Wrong type for TailLines parameter",
			args: map[string]any{"AppID": "app-123", "DeploymentID": "deployment-123", "Component": "web", "LogType": "RUN", "Follow": false, "TailLines": "100"},
			mock: func(app *MockAppsService) {
				app.EXPECT().GetLogs(gomock.Any(), "app-123", "deployment-123", "web", godo.AppLogTypeRun, false, 100).Return(&godo.AppLogs{LiveURL: "live-url", HistoricURLs: []string{"historic-url"}}, nil, nil).Times(1)
			},
			expectedLogs: &godo.AppLogs{LiveURL: "live-url", HistoricURLs: []string{"historic-url"}},
		},
		{
			name:      "Empty arguments map",
			args:      map[string]any{},
			expectMcp: "App ID is required and must be a string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, appService := setupMock(t)
			tool := &AppPlatformTool{client: client}
			if tc.mock != nil {
				tc.mock(appService)
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getAppLogs(context.Background(), req)
			if tc.expectError {
				if tc.handlerError {
					require.Error(t, err)
					require.Nil(t, resp)
				} else {
					require.NoError(t, err)
					require.NotNil(t, resp)
					require.True(t, resp.IsError)
				}
				return
			}
			if tc.expectMcp != "" {
				require.True(t, resp.IsError)
				require.Equal(t, tc.expectMcp, resp.Content[0].(mcp.TextContent).Text)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.expectedLogs != nil {
				equalsToolResult(t, tc.expectedLogs, resp)
			}

		})
	}

}
