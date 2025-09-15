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

func setupNATGatewayToolWithMock(ngs *MockVPCNATGatewaysService) *NATGatewayTool {
	client := &godo.Client{}
	client.VPCNATGateways = ngs
	return NewNATGatewayTool(client)
}

func TestNewNATGatewayTool_createNATGateway(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testNG := &godo.VPCNATGateway{}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(service *MockVPCNATGatewaysService)
		expectError error
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":   "my-ng",
				"Region": "nyc3",
				"Size":   uint32(1),
				"VPCs":   []string{"vpc-123", "my-vpc"},
			},
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VPCNATGatewayRequest{
						Name:   "my-ng",
						Type:   "PUBLIC",
						Region: "nyc3",
						Size:   1,
						VPCs:   []*godo.IngressVPC{{VpcUUID: "vpc-123"}, {VpcUUID: "my-vpc"}},
					}).
					Return(testNG, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create with timeout options",
			args: map[string]any{
				"Name":                   "ng-options",
				"Region":                 "nyc3",
				"Size":                   uint32(1),
				"VPCs":                   []string{"vpc-123", "my-vpc"},
				"TCP Timeout (Seconds)":  uint32(31),
				"UDP Timeout (Seconds)":  uint32(32),
				"ICMP Timeout (Seconds)": uint32(33),
			},
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VPCNATGatewayRequest{
						Name:               "ng-options",
						Type:               "PUBLIC",
						Region:             "nyc3",
						Size:               uint32(1),
						VPCs:               []*godo.IngressVPC{{VpcUUID: "vpc-123"}, {VpcUUID: "my-vpc"}},
						TCPTimeoutSeconds:  uint32(31),
						UDPTimeoutSeconds:  uint32(32),
						ICMPTimeoutSeconds: uint32(33),
					}).
					Return(testNG, nil, nil).
					Times(1)
			},
		},
		{
			name: "api error",
			args: map[string]any{
				"Name":   "fail-ng",
				"Region": "sfo2",
				"Size":   uint32(1),
				"VPCs":   []string{"vpc-123", "my-vpc"},
			},
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.VPCNATGatewayRequest{
						Name:   "fail-ng",
						Type:   "PUBLIC",
						Region: "sfo2",
						Size:   1,
						VPCs:   []*godo.IngressVPC{{VpcUUID: "vpc-123"}, {VpcUUID: "my-vpc"}},
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: errors.New("api error"),
		},
		{
			name: "Missing required argument",
			args: map[string]any{
				"Name":   "incomplete-ng",
				"Size":   uint32(1),
				"Region": "",
				"VPCs":   []string{"vpc-123", "my-vpc"},
			},
			mockSetup:   nil,
			expectError: errors.New("missing required argument"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNGs := NewMockVPCNATGatewaysService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNGs)
			}
			tool := setupNATGatewayToolWithMock(mockNGs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createNATGateway(context.Background(), req)
			if tc.expectError != nil {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outNG godo.VPCNATGateway
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outNG))
			require.Equal(t, testNG.ID, outNG.ID)
		})
	}
}

func TestNATGatewayTool_getNATGateway(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testNG := &godo.VPCNATGateway{
		ID:   "ng-123",
		Name: "my-ng",
	}
	tests := []struct {
		name        string
		id          string
		mockSetup   func(service *MockVPCNATGatewaysService)
		expectError bool
	}{
		{
			name: "Successful get",
			id:   "ng-123",
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Get(gomock.Any(), "ng-123").
					Return(testNG, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   "ng-456",
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Get(gomock.Any(), "ng-456").
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
			mockNGs := NewMockVPCNATGatewaysService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNGs)
			}
			tool := setupNATGatewayToolWithMock(mockNGs)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getNATGateway(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var jsonNG godo.VPCNATGateway
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &jsonNG))
			require.Equal(t, testNG.ID, jsonNG.ID)
		})
	}
}

func TestNewNATGatewayTool_listNATGateways(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testNGs := []*godo.VPCNATGateway{
		{ID: "ng-1", Name: "ng1"},
		{ID: "ng-2", Name: "ng2"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockVPCNATGatewaysService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					List(gomock.Any(), &godo.VPCNATGatewaysListOptions{ListOptions: godo.ListOptions{Page: 2, PerPage: 1}}).
					Return(testNGs, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					List(gomock.Any(), &godo.VPCNATGatewaysListOptions{ListOptions: godo.ListOptions{Page: 1, PerPage: 2}}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:    "Default pagination",
			page:    0,
			perPage: 0,
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					List(gomock.Any(), &godo.VPCNATGatewaysListOptions{ListOptions: godo.ListOptions{Page: 1, PerPage: 20}}).
					Return(testNGs, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNGs := NewMockVPCNATGatewaysService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNGs)
			}
			tool := setupNATGatewayToolWithMock(mockNGs)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listNATGateways(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			var jsonNGs []godo.VPCNATGateway
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &jsonNGs))
			require.GreaterOrEqual(t, len(jsonNGs), 1)
		})
	}
}

func TestNATGatewayTool_updateNATGateway(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testNG := &godo.VPCNATGateway{
		ID:     "ng-123",
		Name:   "my-ng",
		Type:   "PUBLIC",
		Region: "nyc3",
		Size:   2,
		VPCs:   []*godo.IngressVPC{{VpcUUID: "vpc-123"}},
	}
	updatedNG := &godo.VPCNATGateway{
		ID:     "ng-123",
		Name:   "my-ng",
		Type:   "PUBLIC",
		Region: "nyc3",
		Size:   3,
		VPCs:   []*godo.IngressVPC{{VpcUUID: "vpc-123"}},
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVPCNATGatewaysService)
		expectError error
	}{
		{
			name: "Successfully update size",
			args: map[string]any{
				"ID":   "ng-123",
				"Name": "my-ng",
				"Size": uint32(3),
			},
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Get(gomock.Any(), "ng-123").
					Return(testNG, nil, nil).
					Times(1)
				m.EXPECT().
					Update(gomock.Any(), "ng-123", &godo.VPCNATGatewayRequest{
						Name:   "my-ng",
						Type:   "PUBLIC",
						Region: "nyc3",
						Size:   3,
						VPCs:   []*godo.IngressVPC{{VpcUUID: "vpc-123"}},
					}).
					Return(updatedNG, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"ID":   "ng-123",
				"Name": "my-ng",
				"Size": uint32(1),
			},
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Get(gomock.Any(), "ng-123").
					Return(testNG, nil, nil).
					Times(1)
				m.EXPECT().
					Update(gomock.Any(), "ng-123", &godo.VPCNATGatewayRequest{
						Name:   "my-ng",
						Type:   "PUBLIC",
						Region: "nyc3",
						Size:   1,
						VPCs:   []*godo.IngressVPC{{VpcUUID: "vpc-123"}},
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: errors.New("api error"),
		},
		{
			name: "Name and ID mismatch",
			args: map[string]any{
				"ID":   "ng-123",
				"Name": "wrong-name",
			},
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Get(gomock.Any(), "ng-123").
					Return(testNG, nil, nil).
					Times(1)
			},
			expectError: errors.New("NAT Gateway Name does not ID, and cannot be updated"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNGs := NewMockVPCNATGatewaysService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNGs)
			}
			tool := setupNATGatewayToolWithMock(mockNGs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateNATGateway(context.Background(), req)
			if tc.expectError != nil {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var jsonNG godo.VPCNATGateway
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &jsonNG))
			require.Equal(t, updatedNG.Size, jsonNG.Size)
		})
	}
}
func TestNATGatewayTool_deleteNATGateway(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockVPCNATGatewaysService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": "ng-123"},
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Delete(gomock.Any(), "ng-123").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "VPC NAT Gateway deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{"ID": "ng-456"},
			mockSetup: func(m *MockVPCNATGatewaysService) {
				m.EXPECT().
					Delete(gomock.Any(), "ng-456").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNGs := NewMockVPCNATGatewaysService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNGs)
			}
			tool := setupNATGatewayToolWithMock(mockNGs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteNATGateway(context.Background(), req)
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
