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
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Firewalls: firewalls}, nil
	}

	return NewFirewallTool(client)
}

func TestFirewallTool_getFirewall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testFirewall := &godo.Firewall{
		ID:   "fw-123",
		Name: "test-fw",
	}
	tests := []struct {
		name        string
		id          string
		mockSetup   func(*MockFirewallsService)
		expectError bool
	}{
		{
			name: "Successful get",
			id:   "fw-123",
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					Get(gomock.Any(), "fw-123").
					Return(testFirewall, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			id:   "fw-456",
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					Get(gomock.Any(), "fw-456").
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
			mockFirewalls := NewMockFirewallsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockFirewalls)
			}
			tool := setupFirewallToolWithMock(mockFirewalls)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["ID"] = tc.id
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getFirewall(context.Background(), req)
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

func TestFirewallTool_listFirewalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testFirewalls := []godo.Firewall{
		{ID: "fw-1", Name: "fw1"},
		{ID: "fw-2", Name: "fw2"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockFirewallsService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testFirewalls, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockFirewallsService) {
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
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testFirewalls, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockFirewalls := NewMockFirewallsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockFirewalls)
			}
			tool := setupFirewallToolWithMock(mockFirewalls)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listFirewalls(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			var outFirewalls []godo.Firewall
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outFirewalls))
			require.GreaterOrEqual(t, len(outFirewalls), 1)
		})
	}
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
				"DropletIDs":          []any{float64(123), float64(456)},
				"Tags":                []any{"web", "prod"},
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
				"DropletIDs":          []any{},
				"Tags":                []any{},
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

func TestFirewallTool_addTags(t *testing.T) {
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
			name: "Successful add tags",
			args: map[string]any{
				"ID":   "fw-123",
				"Tags": []any{"web", "prod"},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					AddTags(gomock.Any(), "fw-123", "web", "prod").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Tag(s) added to firewall successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"ID":   "fw-456",
				"Tags": []any{"fail"},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					AddTags(gomock.Any(), "fw-456", "fail").
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
			resp, err := tool.addTags(context.Background(), req)
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

func TestFirewallTool_removeTags(t *testing.T) {
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
			name: "Successful remove tags",
			args: map[string]any{
				"ID":   "fw-123",
				"Tags": []any{"web", "prod"},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					RemoveTags(gomock.Any(), "fw-123", "web", "prod").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Tag(s) removed from firewall successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"ID":   "fw-456",
				"Tags": []any{"fail"},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					RemoveTags(gomock.Any(), "fw-456", "fail").
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
			resp, err := tool.removeTags(context.Background(), req)
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

func TestFirewallTool_addDroplets(t *testing.T) {
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
			name: "Successful add droplets",
			args: map[string]any{
				"ID":         "fw-123",
				"DropletIDs": []any{float64(101), float64(202)},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					AddDroplets(gomock.Any(), "fw-123", 101, 202).
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Droplet(s) added to firewall successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"ID":         "fw-456",
				"DropletIDs": []any{float64(303)},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					AddDroplets(gomock.Any(), "fw-456", 303).
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
			resp, err := tool.addDroplets(context.Background(), req)
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

func TestFirewallTool_removeDroplets(t *testing.T) {
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
			name: "Successful remove droplets",
			args: map[string]any{
				"ID":         "fw-123",
				"DropletIDs": []any{float64(101), float64(202)},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					RemoveDroplets(gomock.Any(), "fw-123", 101, 202).
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Droplet(s) removed from firewall successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"ID":         "fw-456",
				"DropletIDs": []any{float64(303)},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					RemoveDroplets(gomock.Any(), "fw-456", 303).
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
			resp, err := tool.removeDroplets(context.Background(), req)
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

func TestFirewallTool_addRules(t *testing.T) {
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
			name: "Successful add inbound and outbound rules",
			args: map[string]any{
				"ID": "fw-123",
				"InboundRules": []any{
					map[string]any{
						"Protocol":  "tcp",
						"PortRange": "80",
						"Sources":   []any{"0.0.0.0/0", "10.0.0.0/8"},
					},
					map[string]any{
						"Protocol":  "tcp",
						"PortRange": "443",
						"Sources":   []any{"0.0.0.0/0"},
					},
				},
				"OutboundRules": []any{
					map[string]any{
						"Protocol":     "udp",
						"PortRange":    "53",
						"Destinations": []any{"8.8.8.8/32", "1.1.1.1/32"},
					},
				},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					AddRules(gomock.Any(), "fw-123", gomock.Any()).
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Rule(s) added to firewall successfully",
		},
		{
			name: "Successful add inbound rules only",
			args: map[string]any{
				"ID": "fw-456",
				"InboundRules": []any{
					map[string]any{
						"Protocol":  "tcp",
						"PortRange": "22",
						"Sources":   []any{"192.168.1.0/24"},
					},
				},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					AddRules(gomock.Any(), "fw-456", gomock.Any()).
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Rule(s) added to firewall successfully",
		},
		{
			name: "No rules provided",
			args: map[string]any{
				"ID": "fw-789",
			},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ID": "fw-error",
				"InboundRules": []any{
					map[string]any{
						"Protocol":  "tcp",
						"PortRange": "80",
						"Sources":   []any{"0.0.0.0/0"},
					},
				},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					AddRules(gomock.Any(), "fw-error", gomock.Any()).
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
			resp, err := tool.addRules(context.Background(), req)
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

func TestFirewallTool_removeRules(t *testing.T) {
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
			name: "Successful remove inbound and outbound rules",
			args: map[string]any{
				"ID": "fw-123",
				"InboundRules": []any{
					map[string]any{
						"Protocol":  "tcp",
						"PortRange": "80",
						"Sources":   []any{"0.0.0.0/0"},
					},
				},
				"OutboundRules": []any{
					map[string]any{
						"Protocol":     "udp",
						"PortRange":    "53",
						"Destinations": []any{"8.8.8.8/32"},
					},
				},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					RemoveRules(gomock.Any(), "fw-123", gomock.Any()).
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Rule(s) removed from firewall successfully",
		},
		{
			name: "Successful remove outbound rules only",
			args: map[string]any{
				"ID": "fw-456",
				"OutboundRules": []any{
					map[string]any{
						"Protocol":     "tcp",
						"PortRange":    "443",
						"Destinations": []any{"192.168.1.0/24", "10.0.0.0/8"},
					},
				},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					RemoveRules(gomock.Any(), "fw-456", gomock.Any()).
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Rule(s) removed from firewall successfully",
		},
		{
			name: "No rules provided",
			args: map[string]any{
				"ID": "fw-789",
			},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ID": "fw-error",
				"InboundRules": []any{
					map[string]any{
						"Protocol":  "tcp",
						"PortRange": "22",
						"Sources":   []any{"192.168.1.0/24"},
					},
				},
			},
			mockSetup: func(m *MockFirewallsService) {
				m.EXPECT().
					RemoveRules(gomock.Any(), "fw-error", gomock.Any()).
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
			resp, err := tool.removeRules(context.Background(), req)
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
