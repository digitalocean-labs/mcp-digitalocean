package networking

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupBYOIPPrefixToolWithMocks(
	byoipService *MockBYOIPPrefixesService,
) *BYOIPPrefixTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			BYOIPPrefixes: byoipService,
		}, nil
	}
	return NewBYOIPPrefixTool(client)
}

func TestBYOIPPrefixTool_getBYOIPPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPrefix := &godo.BYOIPPrefix{
		Prefix: "5.42.203.0/24",
		Status: "active",
		UUID:   "a60caef1-f11e-481f-9f40-5313658e7523",
		Region: "syd1",
	}

	tests := []struct {
		name        string
		uuid        string
		mockSetup   func(*MockBYOIPPrefixesService)
		expectError bool
	}{
		{
			name: "Successful get BYOIP prefix",
			uuid: "a60caef1-f11e-481f-9f40-5313658e7523",
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					Get(gomock.Any(), "a60caef1-f11e-481f-9f40-5313658e7523").
					Return(testPrefix, nil, nil).
					Times(1)
			},
		},
		{
			name:        "Missing UUID argument",
			uuid:        "",
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "API error",
			uuid: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					Get(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockBYOIP := NewMockBYOIPPrefixesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockBYOIP)
			}
			tool := setupBYOIPPrefixToolWithMocks(mockBYOIP)
			args := map[string]any{}
			if tc.name != "Missing UUID argument" {
				args["UUID"] = tc.uuid
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getBYOIPPrefix(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outPrefix godo.BYOIPPrefix
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outPrefix))
			require.Equal(t, testPrefix.Prefix, outPrefix.Prefix)
			require.Equal(t, testPrefix.UUID, outPrefix.UUID)
		})
	}
}

func TestBYOIPPrefixTool_listBYOIPPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPrefixes := []*godo.BYOIPPrefix{
		{
			Prefix: "5.42.203.0/24",
			Status: "active",
			UUID:   "a60caef1-f11e-481f-9f40-5313658e7523",
			Region: "syd1",
		},
		{
			Prefix: "129.212.128.0/24",
			Status: "active",
			UUID:   "002cf550-969f-4af9-90d9-f460ad2132ae",
			Region: "blr1",
		},
	}

	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*MockBYOIPPrefixesService)
		expectError    bool
		expectPrefixes []string
	}{
		{
			name: "List BYOIP prefixes default pagination",
			args: map[string]any{},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testPrefixes, nil, nil).
					Times(1)
			},
			expectPrefixes: []string{"5.42.203.0/24", "129.212.128.0/24"},
		},
		{
			name: "List BYOIP prefixes custom pagination",
			args: map[string]any{"Page": float64(2), "PerPage": float64(1)},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testPrefixes[:1], nil, nil).
					Times(1)
			},
			expectPrefixes: []string{"5.42.203.0/24"},
		},
		{
			name: "API error",
			args: map[string]any{},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockBYOIP := NewMockBYOIPPrefixesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockBYOIP)
			}
			tool := setupBYOIPPrefixToolWithMocks(mockBYOIP)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listBYOIPPrefix(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out []*godo.BYOIPPrefix
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			gotPrefixes := make([]string, len(out))
			for i, prefix := range out {
				gotPrefixes[i] = prefix.Prefix
			}
			require.Equal(t, tc.expectPrefixes, gotPrefixes)
		})
	}
}

func TestBYOIPPrefixTool_createBYOIPPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPrefixResp := &godo.BYOIPPrefixCreateResp{
		Status: "pending",
		UUID:   "new-uuid-123",
		Region: "nyc3",
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockBYOIPPrefixesService)
		expectError bool
		expectUUID  string
	}{
		{
			name: "Create BYOIP prefix success",
			args: map[string]any{
				"prefix":    "192.0.2.0/24",
				"signature": "test-signature-abc123",
				"region":    "nyc3",
			},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.BYOIPPrefixCreateReq{
						Prefix:    "192.0.2.0/24",
						Signature: "test-signature-abc123",
						Region:    "nyc3",
					}).
					Return(testPrefixResp, nil, nil).
					Times(1)
			},
			expectUUID: "new-uuid-123",
		},
		{
			name: "Missing prefix argument",
			args: map[string]any{
				"signature": "test-signature-abc123",
				"region":    "nyc3",
			},
			mockSetup:   func(m *MockBYOIPPrefixesService) {},
			expectError: true,
		},
		{
			name: "Missing signature argument",
			args: map[string]any{
				"prefix": "192.0.2.0/24",
				"region": "nyc3",
			},
			mockSetup:   func(m *MockBYOIPPrefixesService) {},
			expectError: true,
		},
		{
			name: "Missing region argument",
			args: map[string]any{
				"prefix":    "192.0.2.0/24",
				"signature": "test-signature-abc123",
			},
			mockSetup:   func(m *MockBYOIPPrefixesService) {},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"prefix":    "192.0.2.0/24",
				"signature": "test-signature-abc123",
				"region":    "nyc3",
			},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockBYOIP := NewMockBYOIPPrefixesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockBYOIP)
			}
			tool := setupBYOIPPrefixToolWithMocks(mockBYOIP)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createBYOIPPrefix(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var out godo.BYOIPPrefixCreateResp
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, tc.expectUUID, out.UUID)
		})
	}
}

func TestBYOIPPrefixTool_getByOIPPrefixResources(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	time1, _ := time.Parse(time.RFC3339, "2025-10-13T12:52:14Z")
	time2, _ := time.Parse(time.RFC3339, "2025-10-14T10:30:00Z")

	testResources := []godo.BYOIPPrefixResource{
		{
			ID:         3,
			BYOIP:      "5.42.203.4",
			Resource:   "do:droplet:3dab8e0e-b9c3-4a48-be65-fd4b8d17cef3",
			Region:     "syd1",
			AssignedAt: time1,
		},
		{
			ID:         4,
			BYOIP:      "5.42.203.5",
			Resource:   "do:droplet:abc123-def456",
			Region:     "syd1",
			AssignedAt: time2,
		},
	}

	tests := []struct {
		name          string
		args          map[string]any
		mockSetup     func(*MockBYOIPPrefixesService)
		expectError   bool
		expectIPCount int
	}{
		{
			name: "Get BYOIP prefix resources default pagination",
			args: map[string]any{"UUID": "a60caef1-f11e-481f-9f40-5313658e7523"},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					GetResources(gomock.Any(), "a60caef1-f11e-481f-9f40-5313658e7523", &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testResources, nil, nil).
					Times(1)
			},
			expectIPCount: 2,
		},
		{
			name: "Get BYOIP prefix resources custom pagination",
			args: map[string]any{"UUID": "a60caef1-f11e-481f-9f40-5313658e7523", "Page": float64(2), "PerPage": float64(1)},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					GetResources(gomock.Any(), "a60caef1-f11e-481f-9f40-5313658e7523", &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testResources[:1], nil, nil).
					Times(1)
			},
			expectIPCount: 1,
		},
		{
			name:        "Missing UUID argument",
			args:        map[string]any{},
			mockSetup:   func(m *MockBYOIPPrefixesService) {},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"UUID": "a60caef1-f11e-481f-9f40-5313658e7523"},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					GetResources(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockBYOIP := NewMockBYOIPPrefixesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockBYOIP)
			}
			tool := setupBYOIPPrefixToolWithMocks(mockBYOIP)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getByOIPPrefixResources(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out []godo.BYOIPPrefixResource
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, tc.expectIPCount, len(out))
		})
	}
}

func TestBYOIPPrefixTool_deleteBYOIPPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockBYOIPPrefixesService)
		expectError bool
		expectText  string
	}{
		{
			name: "Delete BYOIP prefix success",
			args: map[string]any{"UUID": "a60caef1-f11e-481f-9f40-5313658e7523"},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					Delete(gomock.Any(), "a60caef1-f11e-481f-9f40-5313658e7523").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "BYOIP Prefix deleted",
		},
		{
			name:        "Missing UUID argument",
			args:        map[string]any{},
			mockSetup:   func(m *MockBYOIPPrefixesService) {},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"UUID": "a60caef1-f11e-481f-9f40-5313658e7523"},
			mockSetup: func(m *MockBYOIPPrefixesService) {
				m.EXPECT().
					Delete(gomock.Any(), "a60caef1-f11e-481f-9f40-5313658e7523").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockBYOIP := NewMockBYOIPPrefixesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockBYOIP)
			}
			tool := setupBYOIPPrefixToolWithMocks(mockBYOIP)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteBYOIPPrefix(context.Background(), req)
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
