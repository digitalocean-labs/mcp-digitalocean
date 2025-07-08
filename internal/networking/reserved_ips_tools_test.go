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

func setupReservedIPToolWithMocks(
	ipv4 *MockReservedIPsService,
	ipv6 *MockReservedIPV6sService,
	ipv4Actions *MockReservedIPActionsService,
	ipv6Actions *MockReservedIPV6ActionsService,
) *ReservedIPTool {
	client := &godo.Client{}
	client.ReservedIPs = ipv4
	client.ReservedIPV6s = ipv6
	client.ReservedIPActions = ipv4Actions
	client.ReservedIPV6Actions = ipv6Actions
	return NewReservedIPTool(client)
}

func TestReservedIPTool_reserveIP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testIPv4 := &godo.ReservedIP{IP: "192.0.2.1", Region: &godo.Region{Slug: "nyc3"}}
	testIPv6 := &godo.ReservedIPV6{IP: "2001:db8::1", RegionSlug: "nyc3"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockReservedIPsService, *MockReservedIPV6sService)
		expectError bool
		expectIP    string
	}{
		{
			name: "Reserve IPv4 success",
			args: map[string]any{"Region": "nyc3", "Type": "ipv4"},
			mockSetup: func(ipv4 *MockReservedIPsService, ipv6 *MockReservedIPV6sService) {
				ipv4.EXPECT().
					Create(gomock.Any(), &godo.ReservedIPCreateRequest{Region: "nyc3"}).
					Return(testIPv4, nil, nil).
					Times(1)
			},
			expectIP: "192.0.2.1",
		},
		{
			name: "Reserve IPv6 success",
			args: map[string]any{"Region": "nyc3", "Type": "ipv6"},
			mockSetup: func(ipv4 *MockReservedIPsService, ipv6 *MockReservedIPV6sService) {
				ipv6.EXPECT().
					Create(gomock.Any(), &godo.ReservedIPV6CreateRequest{Region: "nyc3"}).
					Return(testIPv6, nil, nil).
					Times(1)
			},
			expectIP: "2001:db8::1",
		},
		{
			name:        "Invalid type error",
			args:        map[string]any{"Region": "nyc3", "Type": "badtype"},
			mockSetup:   func(ipv4 *MockReservedIPsService, ipv6 *MockReservedIPV6sService) {},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"Region": "nyc3", "Type": "ipv4"},
			mockSetup: func(ipv4 *MockReservedIPsService, ipv6 *MockReservedIPV6sService) {
				ipv4.EXPECT().
					Create(gomock.Any(), &godo.ReservedIPCreateRequest{Region: "nyc3"}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockIPv4 := NewMockReservedIPsService(ctrl)
			mockIPv6 := NewMockReservedIPV6sService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockIPv4, mockIPv6)
			}
			tool := setupReservedIPToolWithMocks(mockIPv4, mockIPv6, nil, nil)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.reserveIP(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var out map[string]any
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectIP)
		})
	}
}

func TestReservedIPTool_releaseIP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockReservedIPsService, *MockReservedIPV6sService)
		expectError bool
		expectText  string
	}{
		{
			name: "Release IPv4 success",
			args: map[string]any{"IP": "192.0.2.1", "Type": "ipv4"},
			mockSetup: func(ipv4 *MockReservedIPsService, ipv6 *MockReservedIPV6sService) {
				ipv4.EXPECT().
					Delete(gomock.Any(), "192.0.2.1").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "reserved IP released successfully",
		},
		{
			name: "Release IPv6 success",
			args: map[string]any{"IP": "2001:db8::1", "Type": "ipv6"},
			mockSetup: func(ipv4 *MockReservedIPsService, ipv6 *MockReservedIPV6sService) {
				ipv6.EXPECT().
					Delete(gomock.Any(), "2001:db8::1").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "reserved IP released successfully",
		},
		{
			name:        "Invalid type error",
			args:        map[string]any{"IP": "bad", "Type": "badtype"},
			mockSetup:   func(ipv4 *MockReservedIPsService, ipv6 *MockReservedIPV6sService) {},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"IP": "192.0.2.1", "Type": "ipv4"},
			mockSetup: func(ipv4 *MockReservedIPsService, ipv6 *MockReservedIPV6sService) {
				ipv4.EXPECT().
					Delete(gomock.Any(), "192.0.2.1").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockIPv4 := NewMockReservedIPsService(ctrl)
			mockIPv6 := NewMockReservedIPV6sService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockIPv4, mockIPv6)
			}
			tool := setupReservedIPToolWithMocks(mockIPv4, mockIPv6, nil, nil)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.releaseIP(context.Background(), req)
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

func TestReservedIPTool_assignIP_unassignIP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 123, Status: "in-progress"}

	// assignIP
	t.Run("Assign IPv4 success", func(t *testing.T) {
		mockIPv4Actions := NewMockReservedIPActionsService(ctrl)
		mockIPv6Actions := NewMockReservedIPV6ActionsService(ctrl)
		mockIPv4Actions.EXPECT().
			Assign(gomock.Any(), "192.0.2.1", 42).
			Return(testAction, nil, nil).
			Times(1)
		tool := setupReservedIPToolWithMocks(nil, nil, mockIPv4Actions, mockIPv6Actions)
		args := map[string]any{"IP": "192.0.2.1", "DropletID": float64(42), "Type": "ipv4"}
		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
		resp, err := tool.assignIP(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError)
		var outAction godo.Action
		require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outAction))
		require.Equal(t, testAction.ID, outAction.ID)
	})

	t.Run("Assign IPv6 error", func(t *testing.T) {
		mockIPv4Actions := NewMockReservedIPActionsService(ctrl)
		mockIPv6Actions := NewMockReservedIPV6ActionsService(ctrl)
		mockIPv6Actions.EXPECT().
			Assign(gomock.Any(), "2001:db8::1", 99).
			Return(nil, nil, errors.New("api error")).
			Times(1)
		tool := setupReservedIPToolWithMocks(nil, nil, mockIPv4Actions, mockIPv6Actions)
		args := map[string]any{"IP": "2001:db8::1", "DropletID": float64(99), "Type": "ipv6"}
		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
		resp, err := tool.assignIP(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError)
	})

	// unassignIP
	t.Run("Unassign IPv4 success", func(t *testing.T) {
		mockIPv4Actions := NewMockReservedIPActionsService(ctrl)
		mockIPv6Actions := NewMockReservedIPV6ActionsService(ctrl)
		mockIPv4Actions.EXPECT().
			Unassign(gomock.Any(), "192.0.2.1").
			Return(testAction, nil, nil).
			Times(1)
		tool := setupReservedIPToolWithMocks(nil, nil, mockIPv4Actions, mockIPv6Actions)
		args := map[string]any{"IP": "192.0.2.1", "Type": "ipv4"}
		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
		resp, err := tool.unassignIP(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError)
		var outAction godo.Action
		require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outAction))
		require.Equal(t, testAction.ID, outAction.ID)
	})

	t.Run("Unassign IPv6 error", func(t *testing.T) {
		mockIPv4Actions := NewMockReservedIPActionsService(ctrl)
		mockIPv6Actions := NewMockReservedIPV6ActionsService(ctrl)
		mockIPv6Actions.EXPECT().
			Unassign(gomock.Any(), "2001:db8::1").
			Return(nil, nil, errors.New("api error")).
			Times(1)
		tool := setupReservedIPToolWithMocks(nil, nil, mockIPv4Actions, mockIPv6Actions)
		args := map[string]any{"IP": "2001:db8::1", "Type": "ipv6"}
		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
		resp, err := tool.unassignIP(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError)
	})
}
