package docr

import (
	"context"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupGarbageCollectionToolWithMock(mockRegistries godo.RegistriesService) *GarbageCollectionTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Registries: mockRegistries,
		}, nil
	}
	return NewGarbageCollectionTool(client)
}

func TestGarbageCollectionTool_startGarbageCollection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testGC := &godo.GarbageCollection{UUID: "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01", Status: "requested"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().StartGarbageCollection(gomock.Any(), "my-registry", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success without type",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().StartGarbageCollection(gomock.Any(), "my-registry", gomock.Any()).Return(testGC, nil, nil)
			},
		},
		{
			name: "success with type",
			args: map[string]any{"RegistryName": "my-registry", "Type": "untagged manifests and unreferenced blobs"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().StartGarbageCollection(gomock.Any(), "my-registry", gomock.Any()).Return(testGC, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupGarbageCollectionToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.startGarbageCollection(context.Background(), req)
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

func TestGarbageCollectionTool_getGarbageCollection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testGC := &godo.GarbageCollection{UUID: "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01", Status: "running"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().GetGarbageCollection(gomock.Any(), "my-registry").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().GetGarbageCollection(gomock.Any(), "my-registry").Return(testGC, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupGarbageCollectionToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getGarbageCollection(context.Background(), req)
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

func TestGarbageCollectionTool_listGarbageCollections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testGCs := []*godo.GarbageCollection{{UUID: "gc-1"}, {UUID: "gc-2"}}
	testGCsMeta := &godo.Meta{Total: 2, Page: 1, Pages: 1}

	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*MockRegistriesService)
		expectError    bool
		expectContains []string
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ListGarbageCollections(gomock.Any(), "my-registry", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry", "Page": float64(1), "PerPage": float64(10)},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ListGarbageCollections(gomock.Any(), "my-registry", gomock.Any()).DoAndReturn(
					func(_ context.Context, name string, opts *godo.ListOptions) ([]*godo.GarbageCollection, *godo.Response, error) {
						require.Equal(t, 1, opts.Page)
						require.Equal(t, 10, opts.PerPage)
						return testGCs, &godo.Response{Meta: testGCsMeta}, nil
					},
				)
			},
			expectContains: []string{`"meta"`, `"total": 2`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupGarbageCollectionToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listGarbageCollections(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			for _, s := range tc.expectContains {
				require.Contains(t, textContent.Text, s)
			}
		})
	}
}

func TestGarbageCollectionTool_updateGarbageCollection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testGC := &godo.GarbageCollection{UUID: "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01", Status: "cancelled"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{"GarbageCollectionUUID": "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01", "Cancel": true},
			expectError: true,
		},
		{
			name:        "missing GarbageCollectionUUID",
			args:        map[string]any{"RegistryName": "my-registry", "Cancel": true},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry", "GarbageCollectionUUID": "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01", "Cancel": true},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().UpdateGarbageCollection(gomock.Any(), "my-registry", "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry", "GarbageCollectionUUID": "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01", "Cancel": true},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().UpdateGarbageCollection(gomock.Any(), "my-registry", "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01", gomock.Any()).DoAndReturn(
					func(_ context.Context, regName, gcUUID string, req *godo.UpdateGarbageCollectionRequest) (*godo.GarbageCollection, *godo.Response, error) {
						require.True(t, req.Cancel)
						return testGC, nil, nil
					},
				)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupGarbageCollectionToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateGarbageCollection(context.Background(), req)
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
