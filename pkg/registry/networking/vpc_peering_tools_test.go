package networking

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

func setupVPCPeeringToolWithMock(vpcs *MockVPCsService) *VPCPeeringTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{VPCs: vpcs}, nil
	}

	return NewVPCPeeringTool(client)
}

func TestVPCPeeringTool_getVPCPeering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPeering := &godo.VPCPeering{
		ID:   "peer-123",
		Name: "test-peering",
	}
	tests := []struct {
		name        string
		id          string
		mockSetup   func(*MockVPCsService)
		expectError bool
	}{
		{
			name: "Successful get",
			id:   "peer-123",
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					GetVPCPeering(gomock.Any(), "peer-123").
					Return(testPeering, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   "peer-456",
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					GetVPCPeering(gomock.Any(), "peer-456").
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
			mockVPC := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPC)
			}
			tool := setupVPCPeeringToolWithMock(mockVPC)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getVPCPeering(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outPeering godo.VPCPeering
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outPeering))
			require.Equal(t, testPeering.ID, outPeering.ID)
		})
	}
}

func TestVPCPeeringTool_listVPCPeerings(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPeerings := []*godo.VPCPeering{
		{ID: "peer-1", Name: "peer1"},
		{ID: "peer-2", Name: "peer2"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockVPCsService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					ListVPCPeerings(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testPeerings, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					ListVPCPeerings(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:    "Default pagination",
			page:    0,
			perPage: 0,
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					ListVPCPeerings(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testPeerings, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVPC := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPC)
			}
			tool := setupVPCPeeringToolWithMock(mockVPC)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listVPCPeerings(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			var outPeerings []godo.VPCPeering
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outPeerings))
			require.GreaterOrEqual(t, len(outPeerings), 1)
		})
	}
}

func TestVPCPeeringTool_createPeering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPeering := &godo.VPCPeering{
		ID:   "peer-123",
		Name: "test-peering",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVPCsService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name": "test-peering",
				"Vpc1": "vpc-1",
				"Vpc2": "vpc-2",
			},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					CreateVPCPeering(gomock.Any(), &godo.VPCPeeringCreateRequest{
						Name:   "test-peering",
						VPCIDs: []string{"vpc-1", "vpc-2"},
					}).
					Return(testPeering, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Name": "fail-peering",
				"Vpc1": "vpc-x",
				"Vpc2": "vpc-y",
			},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					CreateVPCPeering(gomock.Any(), &godo.VPCPeeringCreateRequest{
						Name:   "fail-peering",
						VPCIDs: []string{"vpc-x", "vpc-y"},
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVPC := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPC)
			}
			tool := setupVPCPeeringToolWithMock(mockVPC)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createPeering(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outPeering godo.VPCPeering
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outPeering))
			require.Equal(t, testPeering.ID, outPeering.ID)
		})
	}
}

func TestVPCPeeringTool_deletePeering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVPCsService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": "peer-123"},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					DeleteVPCPeering(gomock.Any(), "peer-123").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "VPC peering connection deleted",
		},
		{
			name: "API error",
			args: map[string]any{"ID": "peer-456"},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					DeleteVPCPeering(gomock.Any(), "peer-456").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVPC := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPC)
			}
			tool := setupVPCPeeringToolWithMock(mockVPC)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deletePeering(context.Background(), req)
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
