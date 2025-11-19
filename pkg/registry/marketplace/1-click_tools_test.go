package marketplace

import (
	"context"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestOneClickTool_Tools(t *testing.T) {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{}, nil
	}

	tool := NewOneClickTool(client)

	tools := tool.Tools()
	assert.Len(t, tools, 2)

	// Check tool names
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Tool.Name
	}

	assert.Contains(t, toolNames, "1-click-list")
	assert.Contains(t, toolNames, "1-click-kubernetes-app-install")
}

func setupOneClickToolWithMock(mockOneClick *MockOneClickService) *OneClickTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{OneClick: mockOneClick}, nil
	}
	return NewOneClickTool(client)
}

func TestOneClickTool_listOneClickApps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOneClick := NewMockOneClickService(ctrl)
	tool := setupOneClickToolWithMock(mockOneClick)

	testApps := []*godo.OneClick{
		{
			Slug: "wordpress",
			Type: "droplet",
		},
		{
			Slug: "mysql",
			Type: "droplet",
		},
	}

	tests := []struct {
		name        string
		args        map[string]interface{}
		mockSetup   func(*MockOneClickService)
		expectError bool
	}{
		{
			name: "Successful list with default type",
			args: map[string]interface{}{},
			mockSetup: func(m *MockOneClickService) {
				m.EXPECT().
					List(gomock.Any(), "droplet").
					Return(testApps, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful list with kubernetes type",
			args: map[string]interface{}{
				"Type": "kubernetes",
			},
			mockSetup: func(m *MockOneClickService) {
				m.EXPECT().
					List(gomock.Any(), "kubernetes").
					Return(testApps, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]interface{}{},
			mockSetup: func(m *MockOneClickService) {
				m.EXPECT().
					List(gomock.Any(), "droplet").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name: "Empty type parameter uses default",
			args: map[string]interface{}{
				"Type": "",
			},
			mockSetup: func(m *MockOneClickService) {
				m.EXPECT().
					List(gomock.Any(), "droplet").
					Return(testApps, nil, nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockOneClick)

			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.args,
				},
			}

			result, err := tool.listOneClickApps(context.Background(), req)
			require.NoError(t, err)

			if tt.expectError {
				assert.True(t, result.IsError)
				assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Failed to list 1-click apps")
			} else {
				assert.False(t, result.IsError)
				assert.NotEmpty(t, result.Content[0].(mcp.TextContent).Text)
			}
		})
	}
}

func TestOneClickTool_installKubernetesApps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOneClick := NewMockOneClickService(ctrl)
	tool := setupOneClickToolWithMock(mockOneClick)

	testResponse := &godo.InstallKubernetesAppsResponse{
		Message: "Apps installed successfully",
	}

	tests := []struct {
		name        string
		args        map[string]interface{}
		mockSetup   func(*MockOneClickService)
		expectError bool
		errorMsg    string
	}{
		{
			name: "Successful install",
			args: map[string]interface{}{
				"ClusterUUID": "k8s-1234567890abcdef",
				"AppSlugs":    []interface{}{"wordpress", "mysql"},
			},
			mockSetup: func(m *MockOneClickService) {
				expectedRequest := &godo.InstallKubernetesAppsRequest{
					Slugs:       []string{"wordpress", "mysql"},
					ClusterUUID: "k8s-1234567890abcdef",
				}
				m.EXPECT().
					InstallKubernetes(gomock.Any(), expectedRequest).
					Return(testResponse, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing ClusterUUID",
			args: map[string]interface{}{
				"AppSlugs": []interface{}{"wordpress"},
			},
			mockSetup:   func(m *MockOneClickService) {},
			expectError: true,
			errorMsg:    "ClusterUUID parameter is required",
		},
		{
			name: "Missing AppSlugs",
			args: map[string]interface{}{
				"ClusterUUID": "k8s-1234567890abcdef",
			},
			mockSetup:   func(m *MockOneClickService) {},
			expectError: true,
			errorMsg:    "AppSlugs parameter is required",
		},
		{
			name: "Empty ClusterUUID",
			args: map[string]interface{}{
				"ClusterUUID": "",
				"AppSlugs":    []interface{}{"wordpress"},
			},
			mockSetup:   func(m *MockOneClickService) {},
			expectError: true,
			errorMsg:    "ClusterUUID cannot be empty",
		},
		{
			name: "Empty AppSlugs",
			args: map[string]interface{}{
				"ClusterUUID": "k8s-1234567890abcdef",
				"AppSlugs":    []interface{}{},
			},
			mockSetup:   func(m *MockOneClickService) {},
			expectError: true,
			errorMsg:    "AppSlugs cannot be empty",
		},
		{
			name: "Invalid ClusterUUID type",
			args: map[string]interface{}{
				"ClusterUUID": 123,
				"AppSlugs":    []interface{}{"wordpress"},
			},
			mockSetup:   func(m *MockOneClickService) {},
			expectError: true,
			errorMsg:    "ClusterUUID must be a string",
		},
		{
			name: "Invalid AppSlugs type",
			args: map[string]interface{}{
				"ClusterUUID": "k8s-1234567890abcdef",
				"AppSlugs":    "wordpress",
			},
			mockSetup:   func(m *MockOneClickService) {},
			expectError: true,
			errorMsg:    "AppSlugs must be an array",
		},
		{
			name: "Invalid app slug type in array",
			args: map[string]interface{}{
				"ClusterUUID": "k8s-1234567890abcdef",
				"AppSlugs":    []interface{}{"wordpress", 123},
			},
			mockSetup:   func(m *MockOneClickService) {},
			expectError: true,
			errorMsg:    "all AppSlugs must be strings",
		},
		{
			name: "API error",
			args: map[string]interface{}{
				"ClusterUUID": "k8s-1234567890abcdef",
				"AppSlugs":    []interface{}{"wordpress"},
			},
			mockSetup: func(m *MockOneClickService) {
				m.EXPECT().
					InstallKubernetes(gomock.Any(), gomock.Any()).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
			errorMsg:    "Failed to install Kubernetes apps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockOneClick)

			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.args,
				},
			}

			result, err := tool.installKubernetesApps(context.Background(), req)
			require.NoError(t, err)

			if tt.expectError {
				assert.True(t, result.IsError)
				assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.errorMsg)
			} else {
				assert.False(t, result.IsError)
				assert.NotEmpty(t, result.Content[0].(mcp.TextContent).Text)
			}
		})
	}
}
