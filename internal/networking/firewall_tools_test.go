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

func setupFirewallToolWithMock(firewalls *MockFirewallsService) *FirewallTool {
	client := &godo.Client{}
	client.Firewalls = firewalls
	return NewFirewallTool(client)
}

func TestFirewallTool_createFirewall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testFirewall := &godo.Firewall{
		ID:   "fw-123",
		Name: "test-fw",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockFirewallsService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":                "test-fw",
				"InboundProtocol":     "tcp",
				"InboundPortRange":    "80",
				"InboundSource":       "0.0.0.0/0",
				"OutboundProtocol":    "udp",
				"OutboundPortRange":   "53",
				"OutboundDestination": "8.8.8.8/32",
				"DropletIDs":          []float64{123, 456},
				"Tags":                []string{"web", "prod"},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.FirewallRequest{
						Name: "test-fw",
						InboundRules: []godo.InboundRule{
							{
								Protocol:  "tcp",
								PortRange: "80",
								Sources:   &godo.Sources{Addresses: []string{"0.0.0.0/0"}},
							},
						},
						OutboundRules: []godo.OutboundRule{
							{
								Protocol:     "udp",
								PortRange:    "53",
								Destinations: &godo.Destinations{Addresses: []string{"8.8.8.8/32"}},
							},
						},
						DropletIDs: []int{123, 456},
						Tags:       []string{"web", "prod"},
					}).
					Return(testFirewall, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":                "fail-fw",
				"InboundProtocol":     "tcp",
				"InboundPortRange":    "22",
				"InboundSource":       "10.0.0.0/8",
				"OutboundProtocol":    "tcp",
				"OutboundPortRange":   "443",
				"OutboundDestination": "0.0.0.0/0",
				"DropletIDs":          []float64{},
				"Tags":                []string{},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.FirewallRequest{
						Name: "fail-fw",
						InboundRules: []godo.InboundRule{
							{
								Protocol:  "tcp",
								PortRange: "22",
								Sources:   &godo.Sources{Addresses: []string{"10.0.0.0/8"}},
							},
						},
						OutboundRules: []godo.OutboundRule{
							{
								Protocol:     "tcp",
								PortRange:    "443",
								Destinations: &godo.Destinations{Addresses: []string{"0.0.0.0/0"}},
							},
						},
						DropletIDs: []int{},
						Tags:       []string{},
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockFirewalls := NewMockFirewallsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockFirewalls)
			}
			tool := setupFirewallToolWithMock(mockFirewalls)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createFirewall(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outFirewall godo.Firewall
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outFirewall))
			require.Equal(t, testFirewall.ID, outFirewall.ID)
		})
	}
}

func TestFirewallTool_deleteFirewall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockFirewallsService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": "fw-123"},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					Delete(gomock.Any(), "fw-123").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Firewall deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{"ID": "fw-456"},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					Delete(gomock.Any(), "fw-456").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockFirewalls := NewMockFirewallsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockFirewalls)
			}
			tool := setupFirewallToolWithMock(mockFirewalls)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteFirewall(context.Background(), req)
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
