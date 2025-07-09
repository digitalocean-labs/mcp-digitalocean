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
	client := &godo.Client{}
	client.VPCs = vpcs
	return NewVPCPeeringTool(client)
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
