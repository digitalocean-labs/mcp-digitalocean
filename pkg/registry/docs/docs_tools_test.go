package docs

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupDocsToolWithMock(mock DocsService) *DocsTool {
	return &DocsTool{client: mock}
}

func TestSearchDocs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testIndex := &DocsIndex{
		Entries: []DocsEntry{
			{Title: "How to Create a Droplet", URL: "https://docs.digitalocean.com/products/droplets/how-to/create/", Description: "Create Droplets from the control panel.", Section: "Droplet How-Tos"},
			{Title: "Droplet Quickstart", URL: "https://docs.digitalocean.com/products/droplets/getting-started/quickstart/", Description: "Get started with Droplets.", Section: "Getting Started"},
		},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDocsService)
		expectError bool
		expectText  string
	}{
		{
			name:        "missing query",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "index fetch error",
			args: map[string]any{"Query": "droplet"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().GetDocsIndex().Return(nil, errors.New("network error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"Query": "droplet"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().GetDocsIndex().Return(testIndex, nil)
			},
			expectText: "result(s) for \"droplet\"",
		},
		{
			name: "no results",
			args: map[string]any{"Query": "nonexistent-term-xyz"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().GetDocsIndex().Return(testIndex, nil)
			},
			expectText: "No results found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDocsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupDocsToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.searchDocs(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.expectError {
				require.True(t, resp.IsError)
				return
			}
			require.False(t, resp.IsError)
			text := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, text, tc.expectText)
		})
	}
}

func TestGetDoc(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDocsService)
		expectError bool
		expectText  string
	}{
		{
			name:        "missing URL",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "fetch error",
			args: map[string]any{"URL": "https://docs.digitalocean.com/notfound/"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().FetchDocPage("https://docs.digitalocean.com/notfound/").Return("", errors.New("HTTP 404"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"URL": "https://docs.digitalocean.com/products/droplets/"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().FetchDocPage("https://docs.digitalocean.com/products/droplets/").Return("# Droplets\n\nDroplets are virtual machines.", nil)
			},
			expectText: "# Droplets",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDocsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupDocsToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getDoc(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.expectError {
				require.True(t, resp.IsError)
				return
			}
			require.False(t, resp.IsError)
			text := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, text, tc.expectText)
		})
	}
}

func TestFindDocsForService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serviceIndex := &DocsIndex{
		Entries: []DocsEntry{
			{Title: "Droplet Quickstart", URL: "https://docs.digitalocean.com/products/droplets/quickstart/", Section: "Getting Started"},
		},
	}

	mainIndex := &DocsIndex{
		Entries: []DocsEntry{
			{Title: "Kubernetes Overview", URL: "https://docs.digitalocean.com/products/kubernetes/", Description: "Managed Kubernetes clusters.", Section: "Products"},
		},
		Sections: []string{"Products"},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDocsService)
		expectError bool
		expectText  string
	}{
		{
			name:        "missing service",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "service index found",
			args: map[string]any{"Service": "droplets"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().GetServiceIndex("droplets").Return(serviceIndex, nil)
			},
			expectText: "Documentation for \"droplets\"",
		},
		{
			name: "fallback to main index",
			args: map[string]any{"Service": "kubernetes"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().GetServiceIndex("kubernetes").Return(nil, nil)
				m.EXPECT().GetDocsIndex().Return(mainIndex, nil)
			},
			expectText: "Kubernetes Overview",
		},
		{
			name: "main index error",
			args: map[string]any{"Service": "kubernetes"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().GetServiceIndex("kubernetes").Return(nil, nil)
				m.EXPECT().GetDocsIndex().Return(nil, errors.New("network error"))
			},
			expectError: true,
		},
		{
			name: "no results anywhere",
			args: map[string]any{"Service": "nonexistent"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().GetServiceIndex("nonexistent").Return(nil, nil)
				m.EXPECT().GetDocsIndex().Return(&DocsIndex{Sections: []string{"Products"}}, nil)
			},
			expectText: "No documentation found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDocsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupDocsToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.findDocsForService(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.expectError {
				require.True(t, resp.IsError)
				return
			}
			require.False(t, resp.IsError)
			text := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, text, tc.expectText)
		})
	}
}

func TestGetQuickstart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDocsService)
		expectError bool
		expectText  string
	}{
		{
			name:        "missing service",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "quickstart not found",
			args: map[string]any{"Service": "nonexistent"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().FindQuickstart("nonexistent").Return("", "", errors.New("no quickstart found"))
			},
			expectText: "No quickstart guide found",
		},
		{
			name: "success",
			args: map[string]any{"Service": "droplets"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().FindQuickstart("droplets").Return(
					"https://docs.digitalocean.com/products/droplets/quickstart/",
					"# Droplet Quickstart\n\nCreate a Droplet in minutes.",
					nil,
				)
			},
			expectText: "# Quickstart: droplets",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDocsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupDocsToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getQuickstart(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.expectError {
				require.True(t, resp.IsError)
				return
			}
			require.False(t, resp.IsError)
			text := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, text, tc.expectText)
		})
	}
}

func TestTroubleshoot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	supportIndex := &DocsIndex{
		Entries: []DocsEntry{
			{Title: "Why am I receiving 520 status codes from my app?", URL: "https://docs.digitalocean.com/support/520-status-codes/", Description: "Your app may have crashed.", Section: "App Platform Support"},
			{Title: "My app deployment failed because of a health check", URL: "https://docs.digitalocean.com/support/health-check-failed/", Description: "Your app is likely unavailable on the port.", Section: "App Platform Support"},
			{Title: "How to Create a Droplet", URL: "https://docs.digitalocean.com/products/droplets/how-to/create/", Description: "Create Droplets.", Section: "Droplet How-Tos"},
		},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDocsService)
		expectError bool
		expectText  string
	}{
		{
			name:        "missing symptom",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "search error",
			args: map[string]any{"Symptom": "520 error"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().FindTroubleshootPage("520 error").Return(nil, errors.New("network error"))
			},
			expectError: true,
		},
		{
			name: "no results",
			args: map[string]any{"Symptom": "xyznonexistent"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().FindTroubleshootPage("xyznonexistent").Return([]DocsEntry{}, nil)
			},
			expectText: "No troubleshooting pages found",
		},
		{
			name: "success with content fetch",
			args: map[string]any{"Symptom": "520 status code"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().FindTroubleshootPage("520 status code").Return(supportIndex.Entries[:2], nil)
				m.EXPECT().FetchDocPage("https://docs.digitalocean.com/support/520-status-codes/").Return("# 520 Status Codes\n\nYour app crashed.", nil)
			},
			expectText: "520 Status Codes",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDocsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupDocsToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.troubleshoot(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.expectError {
				require.True(t, resp.IsError)
				return
			}
			require.False(t, resp.IsError)
			text := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, text, tc.expectText)
		})
	}
}

func TestGetRelatedLinks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDocsService)
		expectError bool
		expectText  string
	}{
		{
			name:        "missing URL",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "fetch error",
			args: map[string]any{"URL": "https://docs.digitalocean.com/notfound/"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().ExtractRelatedLinks("https://docs.digitalocean.com/notfound/").Return(nil, errors.New("HTTP 404"))
			},
			expectError: true,
		},
		{
			name: "no links found",
			args: map[string]any{"URL": "https://docs.digitalocean.com/products/droplets/"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().ExtractRelatedLinks("https://docs.digitalocean.com/products/droplets/").Return([]RelatedLink{}, nil)
			},
			expectText: "No related documentation links found",
		},
		{
			name: "success with categorized links",
			args: map[string]any{"URL": "https://docs.digitalocean.com/products/app-platform/"},
			mockSetup: func(m *MockDocsService) {
				m.EXPECT().ExtractRelatedLinks("https://docs.digitalocean.com/products/app-platform/").Return([]RelatedLink{
					{URL: "https://docs.digitalocean.com/products/app-platform/how-to/create-apps/", Title: "How to Create Apps", Category: "how-to"},
					{URL: "https://docs.digitalocean.com/products/app-platform/reference/app-spec/", Title: "App Spec Reference", Category: "reference"},
				}, nil)
			},
			expectText: "how-to",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDocsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupDocsToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getRelatedLinks(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.expectError {
				require.True(t, resp.IsError)
				return
			}
			require.False(t, resp.IsError)
			text := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, text, tc.expectText)
		})
	}
}

func TestDocsTool_Tools(t *testing.T) {
	tool := NewDocsTool()
	tools := tool.Tools()

	require.Len(t, tools, 6)

	toolNames := make([]string, len(tools))
	for i, st := range tools {
		toolNames[i] = st.Tool.Name
	}

	require.Contains(t, toolNames, "docs-search")
	require.Contains(t, toolNames, "docs-get-page")
	require.Contains(t, toolNames, "docs-find-for-service")
	require.Contains(t, toolNames, "docs-get-quickstart")
	require.Contains(t, toolNames, "docs-troubleshoot")
	require.Contains(t, toolNames, "docs-get-related")

	for _, st := range tools {
		require.NotNil(t, st.Handler, "tool %s should have a handler", st.Tool.Name)
		require.NotNil(t, st.Tool.Annotations.ReadOnlyHint, "tool %s should have ReadOnlyHint", st.Tool.Name)
		require.True(t, *st.Tool.Annotations.ReadOnlyHint, "tool %s should be read-only", st.Tool.Name)
		require.NotNil(t, st.Tool.Annotations.IdempotentHint, "tool %s should have IdempotentHint", st.Tool.Name)
		require.True(t, *st.Tool.Annotations.IdempotentHint, "tool %s should be idempotent", st.Tool.Name)
	}
}
