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
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{VPCs: vpcs}, nil
	}
	return NewVPCTool(client)
}

func TestVPCTool_getVPC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testVPC := &godo.VPC{
		ID:         "vpc-123",
		Name:       "private-net",
		RegionSlug: "nyc3",
	}
	tests := []struct {
		name        string
		id          string
		mockSetup   func(*MockVPCsService)
		expectError bool
	}{
		{
			name: "Successful get",
			id:   "vpc-123",
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Get(gomock.Any(), "vpc-123").
					Return(testVPC, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   "vpc-456",
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Get(gomock.Any(), "vpc-456").
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
			mockVPCs := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPCs)
			}
			tool := setupVPCToolWithMock(mockVPCs)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getVPC(context.Background(), req)
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

func TestVPCTool_listVPCs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testVPCs := []*godo.VPC{
		{ID: "vpc-1", Name: "vpc1"},
		{ID: "vpc-2", Name: "vpc2"},
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
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testVPCs, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockVPCsService) {
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
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testVPCs, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVPCs := NewMockVPCsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVPCs)
			}
			tool := setupVPCToolWithMock(mockVPCs)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listVPCs(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			var outVPCs []godo.VPC
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outVPCs))
			require.GreaterOrEqual(t, len(outVPCs), 1)
		})
	}
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
			name: "Successful create with subnet",
			args: map[string]any{
				"Name":   "private-net",
				"Region": "nyc3",
				"Subnet": "10.10.0.0/20",
			},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VPCCreateRequest{
						Name:       "private-net",
						RegionSlug: "nyc3",
						IPRange:    "10.10.0.0/20",
					}).
					Return(testVPC, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create with empty subnet",
			args: map[string]any{
				"Name":   "private-net",
				"Region": "nyc3",
				"Subnet": "",
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
			name: "Successful create with description",
			args: map[string]any{
				"Name":        "private-net",
				"Region":      "nyc3",
				"Description": "My private network",
			},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VPCCreateRequest{
						Name:        "private-net",
						RegionSlug:  "nyc3",
						Description: "My private network",
					}).
					Return(testVPC, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create with description and subnet",
			args: map[string]any{
				"Name":        "private-net",
				"Region":      "nyc3",
				"Subnet":      "10.10.0.0/20",
				"Description": "My private network with custom subnet",
			},
			mockSetup: func(m *MockVPCsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VPCCreateRequest{
						Name:        "private-net",
						RegionSlug:  "nyc3",
						IPRange:     "10.10.0.0/20",
						Description: "My private network with custom subnet",
					}).
					Return(testVPC, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create with empty description",
			args: map[string]any{
				"Name":        "private-net",
				"Region":      "nyc3",
				"Description": "",
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
