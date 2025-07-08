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

func setupVPCToolWithMock(vpcs *MockVPCsService) *VPCTool {
	client := &godo.Client{}
	client.VPCs = vpcs
	return NewVPCTool(client)
}

func TestVPCTool_createVPC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testVPC := &godo.VPC{
		ID:         "vpc-123",
		Name:       "private-net",
		RegionSlug: "nyc3",
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
				"Name":   "private-net",
				"Region": "nyc3",
			},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VPCCreateRequest{
						Name:       "private-net",
						RegionSlug: "nyc3",
					}).
					Return(testVPC, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":   "fail-vpc",
				"Region": "sfo2",
			},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VPCCreateRequest{
						Name:       "fail-vpc",
						RegionSlug: "sfo2",
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVPCs := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPCs)
			}
			tool := setupVPCToolWithMock(mockVPCs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createVPC(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outVPC godo.VPC
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outVPC))
			require.Equal(t, testVPC.ID, outVPC.ID)
		})
	}
}

func TestVPCTool_listVPCMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testMembers := []*godo.VPCMember{
		{URN: "do:droplet:123"},
		{URN: "do:droplet:456"},
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVPCsService)
		expectError bool
	}{
		{
			name: "Successful list members",
			args: map[string]any{"ID": "vpc-123"},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					ListMembers(gomock.Any(), "vpc-123", nil, nil).
					Return(testMembers, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": "vpc-456"},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					ListMembers(gomock.Any(), "vpc-456", nil, nil).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVPCs := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPCs)
			}
			tool := setupVPCToolWithMock(mockVPCs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listVPCMembers(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outMembers []*godo.VPCMember
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outMembers))
			require.Len(t, outMembers, len(testMembers))
		})
	}
}

func TestVPCTool_deleteVPC(t *testing.T) {
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
			args: map[string]any{"ID": "vpc-123"},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Delete(gomock.Any(), "vpc-123").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "VPC deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{"ID": "vpc-456"},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Delete(gomock.Any(), "vpc-456").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVPCs := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPCs)
			}
			tool := setupVPCToolWithMock(mockVPCs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteVPC(context.Background(), req)
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
