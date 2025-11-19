package spaces

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

func setupCDNToolWithMock(cdn *MockCDNService) *CDNTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{CDNs: cdn}, nil
	}

	return NewCDNTool(client)
}

// --- getCDN tool handler tests ---

func TestCDNTool_getCDN(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCDN := &godo.CDN{
		ID:     "cdn-123",
		Origin: "origin.example.com",
		TTL:    3600,
	}
	tests := []struct {
		name        string
		id          string
		mockSetup   func(*MockCDNService)
		expectError bool
	}{
		{
			name: "Successful get",
			id:   "cdn-123",
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					Get(gomock.Any(), "cdn-123").
					Return(testCDN, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   "cdn-456",
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					Get(gomock.Any(), "cdn-456").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing ID argument",
			id:          "",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCDN := NewMockCDNService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockCDN)
			}
			tool := setupCDNToolWithMock(mockCDN)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getCDN(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outCDN godo.CDN
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outCDN))
			require.Equal(t, testCDN.ID, outCDN.ID)
		})
	}
}

func TestCDNTool_listCDNs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCDNs := []godo.CDN{
		{ID: "cdn-1", Origin: "origin1.example.com", TTL: 3600},
		{ID: "cdn-2", Origin: "origin2.example.com", TTL: 7200},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockCDNService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testCDNs, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockCDNService) {
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
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testCDNs, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCDN := NewMockCDNService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockCDN)
			}
			tool := setupCDNToolWithMock(mockCDN)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listCDNs(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			var outCDNs []godo.CDN
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outCDNs))
			require.GreaterOrEqual(t, len(outCDNs), 1)
		})
	}
}

func TestCDNTool_createCDN(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCDN := &godo.CDN{
		ID:     "cdn-123",
		Origin: "origin.example.com",
		TTL:    3600,
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockCDNService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Origin": "origin.example.com",
				"TTL":    float64(3600),
			},
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.CDNCreateRequest{
						Origin: "origin.example.com",
						TTL:    3600,
					}).
					Return(testCDN, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Origin": "fail.example.com",
				"TTL":    float64(1800),
			},
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.CDNCreateRequest{
						Origin: "fail.example.com",
						TTL:    1800,
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCDN := NewMockCDNService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockCDN)
			}
			tool := setupCDNToolWithMock(mockCDN)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createCDN(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outCDN godo.CDN
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outCDN))
			require.Equal(t, testCDN.ID, outCDN.ID)
		})
	}
}

func TestCDNTool_deleteCDN(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockCDNService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": "cdn-123"},
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					Delete(gomock.Any(), "cdn-123").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "CDN deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{"ID": "cdn-456"},
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					Delete(gomock.Any(), "cdn-456").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCDN := NewMockCDNService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockCDN)
			}
			tool := setupCDNToolWithMock(mockCDN)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteCDN(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
		})
	}
}

func TestCDNTool_flushCDNCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockCDNService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful flush",
			args: map[string]any{
				"ID":    "cdn-123",
				"Files": []any{"/index.html", "/logo.png"},
			},
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					FlushCache(gomock.Any(), "cdn-123", &godo.CDNFlushCacheRequest{
						Files: []string{"/index.html", "/logo.png"},
					}).
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "CDN cache flushed successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"ID":    "cdn-456",
				"Files": []any{"/fail.js"},
			},
			mockSetup: func(m *MockCDNService) {
				m.EXPECT().
					FlushCache(gomock.Any(), "cdn-456", &godo.CDNFlushCacheRequest{
						Files: []string{"/fail.js"},
					}).
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCDN := NewMockCDNService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockCDN)
			}
			tool := setupCDNToolWithMock(mockCDN)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.flushCDNCache(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
		})
	}
}
