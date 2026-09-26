package projects

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupProjectToolWithMock(mockProjects *MockProjectsService) *ProjectTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Projects: mockProjects,
		}, nil
	}

	return NewProjectTool(client)
}

func TestProjectTool_listProjects(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testProjects := []godo.Project{
		{ID: "proj-1", Name: "alpha"},
		{ID: "proj-2", Name: "beta"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockProjectsService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(testProjects, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:    "Default pagination",
			page:    0,
			perPage: 0,
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 30}).
					Return(testProjects, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockProjects := NewMockProjectsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockProjects)
			}
			tool := setupProjectToolWithMock(mockProjects)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listProjects(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			if tc.name == "Successful list" {
				structured, ok := resp.StructuredContent.(map[string]json.RawMessage)
				require.True(t, ok)
				require.Contains(t, string(structured["projects"]), `"id": "proj-1"`)
			}
		})
	}
}

func TestProjectTool_getProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testProject := &godo.Project{ID: "proj-1", Name: "alpha"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockProjectsService)
		expectError bool
	}{
		{
			name: "Successful get",
			args: map[string]any{"ID": "proj-1"},
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					Get(gomock.Any(), "proj-1").
					Return(testProject, nil, nil).
					Times(1)
			},
		},
		{
			name: "Get default alias",
			args: map[string]any{"ID": "default"},
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					Get(gomock.Any(), "default").
					Return(testProject, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": "missing"},
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					Get(gomock.Any(), "missing").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing ID argument",
			args:        map[string]any{},
			mockSetup:   func(m *MockProjectsService) {},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockProjects := NewMockProjectsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockProjects)
			}
			tool := setupProjectToolWithMock(mockProjects)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getProject(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			if tc.name == "Successful get" {
				structured, ok := resp.StructuredContent.(map[string]json.RawMessage)
				require.True(t, ok)
				require.Contains(t, string(structured["project"]), `"id": "proj-1"`)
			}
		})
	}
}

func TestProjectTool_getDefaultProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testProject := &godo.Project{ID: "default-id", Name: "default", IsDefault: true}
	tests := []struct {
		name        string
		mockSetup   func(*MockProjectsService)
		expectError bool
	}{
		{
			name: "Successful get default",
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					GetDefault(gomock.Any()).
					Return(testProject, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					GetDefault(gomock.Any()).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockProjects := NewMockProjectsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockProjects)
			}
			tool := setupProjectToolWithMock(mockProjects)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
			resp, err := tool.getDefaultProject(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestProjectTool_createProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testProject := &godo.Project{
		ID:          "proj-new",
		Name:        "my-web-api",
		Purpose:     "Service or API",
		Description: "My website API",
		Environment: "Production",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockProjectsService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":        "my-web-api",
				"Purpose":     "Service or API",
				"Description": "My website API",
				"Environment": "Production",
			},
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.CreateProjectRequest{
						Name:        "my-web-api",
						Purpose:     "Service or API",
						Description: "My website API",
						Environment: "Production",
					}).
					Return(testProject, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create with required fields only",
			args: map[string]any{
				"Name":    "minimal",
				"Purpose": "Web Application",
			},
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.CreateProjectRequest{
						Name:    "minimal",
						Purpose: "Web Application",
					}).
					Return(&godo.Project{ID: "proj-min", Name: "minimal"}, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":    "fail",
				"Purpose": "Web Application",
			},
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.CreateProjectRequest{
						Name:    "fail",
						Purpose: "Web Application",
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name: "Missing Name",
			args: map[string]any{
				"Purpose": "Web Application",
			},
			mockSetup:   func(m *MockProjectsService) {},
			expectError: true,
		},
		{
			name: "Name only omits purpose",
			args: map[string]any{
				"Name": "no-purpose",
			},
			mockSetup: func(m *MockProjectsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.CreateProjectRequest{
						Name: "no-purpose",
					}).
					Return(&godo.Project{ID: "proj-name", Name: "no-purpose"}, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockProjects := NewMockProjectsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockProjects)
			}
			tool := setupProjectToolWithMock(mockProjects)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createProject(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
		})
	}
}
