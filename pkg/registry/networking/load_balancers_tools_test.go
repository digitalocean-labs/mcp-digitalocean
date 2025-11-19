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

func setupLoadBalancersToolWithMock(loadBalancers *MockLoadBalancersService) *LoadBalancersTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{LoadBalancers: loadBalancers}, nil
	}
	return NewLoadBalancersTool(client)
}

func TestLoadBalancersTool_createLoadBalancer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLoadBalancerWithDropletIDs := &godo.LoadBalancer{
		ID:         "12345",
		Region:     &godo.Region{Slug: "nyc3"},
		DropletIDs: []int{111, 222},
	}
	forwardingRulesArg := []any{
		map[string]any{
			"EntryProtocol":  "http",
			"EntryPort":      float64(80),
			"TargetProtocol": "http",
			"TargetPort":     float64(80),
		},
		map[string]any{
			"EntryProtocol":  "https",
			"EntryPort":      float64(443),
			"TargetProtocol": "https",
			"TargetPort":     float64(443),
			"TlsPassthrough": true,
		},
	}
	mockForwardingRules := []godo.ForwardingRule{
		{
			EntryProtocol:  "http",
			EntryPort:      80,
			TargetProtocol: "http",
			TargetPort:     80,
		},
		{
			EntryProtocol:  "https",
			EntryPort:      443,
			TargetProtocol: "https",
			TargetPort:     443,
			TlsPassthrough: true,
		},
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful create with DropletIDs",
			args: map[string]any{
				"Region":          "nyc3",
				"Name":            "example-lb",
				"DropletIDs":      []any{float64(111), float64(222)},
				"ForwardingRules": forwardingRulesArg,
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.LoadBalancerRequest{
						Region:          "nyc3",
						Name:            "example-lb",
						DropletIDs:      []int{111, 222},
						ForwardingRules: mockForwardingRules,
					}).
					Return(testLoadBalancerWithDropletIDs, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create with Tag",
			args: map[string]any{
				"Region":          "nyc3",
				"Name":            "example-lb",
				"Tag":             "example-tag",
				"ForwardingRules": forwardingRulesArg,
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.LoadBalancerRequest{
						Region:          "nyc3",
						Name:            "example-lb",
						Tag:             "example-tag",
						ForwardingRules: mockForwardingRules,
					}).
					Return(&godo.LoadBalancer{
						ID:     "12345",
						Region: &godo.Region{Slug: "nyc3"},
						Tag:    "example-tag",
					}, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create with optional arguments provided",
			args: map[string]any{
				"Region":          "nyc3",
				"Name":            "example-lb",
				"DropletIDs":      []any{float64(111), float64(222)},
				"ForwardingRules": forwardingRulesArg,
				"Type":            "REGIONAL_NETWORK",
				"Network":         "INTERNAL",
				"SizeUnit":        float64(4),
				"NetworkStack":    "DUALSTACK",
				"ProjectID":       "example-project-id",
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.LoadBalancerRequest{
						Region:          "nyc3",
						Name:            "example-lb",
						DropletIDs:      []int{111, 222},
						ForwardingRules: mockForwardingRules,
						SizeUnit:        4,
						Type:            "REGIONAL_NETWORK",
						Network:         "INTERNAL",
						NetworkStack:    "DUALSTACK",
						ProjectID:       "example-project-id",
					}).
					Return(&godo.LoadBalancer{
						ID:           "12345",
						Region:       &godo.Region{Slug: "nyc3"},
						DropletIDs:   []int{111, 222},
						SizeUnit:     4,
						Type:         "REGIONAL_NETWORK",
						Network:      "INTERNAL",
						NetworkStack: "DUALSTACK",
						ProjectID:    "example-project-id",
					}, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create Global Load Balancer",
			args: map[string]any{
				"Name": "example-global-lb",
				"Tag":  "example-tag",
				"Type": "GLOBAL",
				"GLBSettings": map[string]any{
					"TargetPort":        float64(80),
					"TargetProtocol":    "http",
					"RegionPriorities":  map[string]any{"dev1": float64(1), "dev2": float64(2)},
					"FailoverThreshold": float64(10),
					"CDN": map[string]any{
						"IsEnabled": true,
					},
				},
				"TargetLoadBalancerIDs": []string{"target-lb-1", "target-lb-2"},
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.LoadBalancerRequest{
						Name: "example-global-lb",
						Tag:  "example-tag",
						Type: "GLOBAL",
						GLBSettings: &godo.GLBSettings{
							TargetPort:        80,
							TargetProtocol:    "http",
							RegionPriorities:  map[string]uint32{"dev1": 1, "dev2": 2},
							FailoverThreshold: 10,
							CDN: &godo.CDNSettings{
								IsEnabled: true,
							},
						},
						TargetLoadBalancerIDs: []string{"target-lb-1", "target-lb-2"},
					}).
					Return(&godo.LoadBalancer{
						ID:   "12345",
						Name: "example-global-lb",
						Type: "GLOBAL",
						GLBSettings: &godo.GLBSettings{
							TargetPort:        80,
							TargetProtocol:    "http",
							RegionPriorities:  map[string]uint32{"dev1": 1, "dev2": 2},
							FailoverThreshold: 10,
							CDN: &godo.CDNSettings{
								IsEnabled: true,
							},
						},
						TargetLoadBalancerIDs: []string{"target-lb-1", "target-lb-2"},
					}, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing Region argument",
			args: map[string]any{
				"Name":            "example-lb",
				"DropletIDs":      []any{float64(111), float64(222)},
				"ForwardingRules": forwardingRulesArg,
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "Region is required for REGIONAL and REGIONAL_NETWORK load balancers",
		},
		{
			name: "Missing Name argument",
			args: map[string]any{
				"Region":          "nyc3",
				"DropletIDs":      []any{float64(111), float64(222)},
				"ForwardingRules": forwardingRulesArg,
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "Name is required",
		},
		{
			name: "Both DropletIDs and Tag arguments provided",
			args: map[string]any{
				"Region":          "nyc3",
				"Name":            "example-lb",
				"DropletIDs":      []any{float64(111), float64(222)},
				"Tag":             "web-servers",
				"ForwardingRules": forwardingRulesArg,
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "Only one target identifier (e.g. tag, droplets) can be specified",
		},
		{
			name: "Missing ForwardingRules argument",
			args: map[string]any{
				"Region":     "nyc3",
				"Name":       "example-lb",
				"DropletIDs": []any{float64(111), float64(222)},
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "At least one forwarding rule must be provided",
		},
		{
			name: "API error",
			args: map[string]any{
				"Region":          "nyc3",
				"Name":            "example-lb",
				"DropletIDs":      []any{float64(111), float64(222)},
				"ForwardingRules": forwardingRulesArg,
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.LoadBalancerRequest{
						Region:          "nyc3",
						Name:            "example-lb",
						DropletIDs:      []int{111, 222},
						ForwardingRules: mockForwardingRules,
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createLoadBalancer(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				if tc.expectText != "" {
					require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outLoadBalancer godo.LoadBalancer
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outLoadBalancer))
			require.Equal(t, testLoadBalancerWithDropletIDs.ID, outLoadBalancer.ID)
		})
	}
}

func TestLoadBalancersTool_deleteLoadBalancer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{
				"LoadBalancerID": "12345",
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Delete(gomock.Any(), "12345").
					Return(nil, nil).
					Times(1)
			},
			expectText: "Load Balancer deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"LoadBalancerID": "12345",
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Delete(gomock.Any(), "12345").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteLoadBalancer(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Equal(t, tc.expectText, resp.Content[0].(mcp.TextContent).Text)
		})
	}
}

func TestLoadBalancersTool_deleteLoadBalancerCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete cache",
			args: map[string]any{
				"LoadBalancerID": "12345",
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					PurgeCache(gomock.Any(), "12345").
					Return(nil, nil).
					Times(1)
			},
			expectText: "Load Balancer cache deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"LoadBalancerID": "12345",
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					PurgeCache(gomock.Any(), "12345").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteLoadBalancerCache(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Equal(t, tc.expectText, resp.Content[0].(mcp.TextContent).Text)
		})
	}
}

func TestLoadBalancersTool_getLoadBalancer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLoadBalancer := &godo.LoadBalancer{
		ID:         "12345",
		Region:     &godo.Region{Slug: "nyc3"},
		DropletIDs: []int{111, 222},
	}
	tests := []struct {
		name        string
		lbID        string
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
	}{
		{
			name: "Successful get",
			lbID: "12345",
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Get(gomock.Any(), "12345").
					Return(testLoadBalancer, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			lbID: "12345",
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Get(gomock.Any(), "12345").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing ID argument",
			lbID:        "",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			args := map[string]any{}
			if tc.name != "Missing ID argument" {
				args["LoadBalancerID"] = tc.lbID
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getLoadBalancer(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outLoadBalancer godo.LoadBalancer
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outLoadBalancer))
			require.Equal(t, testLoadBalancer.ID, outLoadBalancer.ID)
		})
	}
}

func TestLoadBalancersTool_listLoadBalancers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLoadBalancers := []godo.LoadBalancer{
		{
			ID:         "12345",
			Region:     &godo.Region{Slug: "nyc3"},
			DropletIDs: []int{111, 222},
		},
		{
			ID:         "67890",
			Region:     &godo.Region{Slug: "sfo2"},
			DropletIDs: []int{333, 444},
		},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testLoadBalancers, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:    "Default pagination",
			page:    0,
			perPage: 0,
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testLoadBalancers, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listLoadBalancers(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outLoadBalancers []godo.LoadBalancer
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outLoadBalancers))
			require.GreaterOrEqual(t, len(testLoadBalancers), len(outLoadBalancers))
		})
	}
}

func TestLoadBalancersTool_addDroplets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		lbID        string
		dropletIDs  []any
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
		expectText  string
	}{
		{
			name:       "Successful add droplets",
			lbID:       "12345",
			dropletIDs: []any{float64(111), float64(222)},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					AddDroplets(gomock.Any(), "12345", []int{111, 222}).
					Return(nil, nil).
					Times(1)
			},
			expectText: "Droplets added successfully",
		},
		{
			name:       "API error",
			lbID:       "12345",
			dropletIDs: []any{float64(111), float64(222)},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					AddDroplets(gomock.Any(), "12345", []int{111, 222}).
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing load balancer ID argument",
			lbID:        "",
			dropletIDs:  []any{float64(111), float64(222)},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name:        "Missing droplet IDs argument",
			lbID:        "12345",
			dropletIDs:  nil,
			mockSetup:   nil,
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			args := map[string]any{}
			if tc.lbID != "" {
				args["LoadBalancerID"] = tc.lbID
			}
			if tc.dropletIDs != nil {
				args["DropletIDs"] = tc.dropletIDs
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.addDroplets(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Equal(t, tc.expectText, resp.Content[0].(mcp.TextContent).Text)
		})
	}
}

func TestLoadBalancersTool_removeDroplets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		lbID        string
		dropletIDs  []any
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
		expectText  string
	}{
		{
			name:       "Successful remove droplets",
			lbID:       "12345",
			dropletIDs: []any{float64(111), float64(222)},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					RemoveDroplets(gomock.Any(), "12345", []int{111, 222}).
					Return(nil, nil).
					Times(1)
			},
			expectText: "Droplets removed successfully",
		},
		{
			name:       "API error",
			lbID:       "12345",
			dropletIDs: []any{float64(111), float64(222)},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					RemoveDroplets(gomock.Any(), "12345", []int{111, 222}).
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing load balancer ID argument",
			lbID:        "",
			dropletIDs:  []any{float64(111), float64(222)},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name:        "Missing droplet IDs argument",
			lbID:        "12345",
			dropletIDs:  nil,
			mockSetup:   nil,
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			args := map[string]any{}
			if tc.lbID != "" {
				args["LoadBalancerID"] = tc.lbID
			}
			if tc.dropletIDs != nil {
				args["DropletIDs"] = tc.dropletIDs
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.removeDroplets(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Equal(t, tc.expectText, resp.Content[0].(mcp.TextContent).Text)
		})
	}
}

func TestLoadBalancersTool_updateLoadBalancer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLoadBalancer := &godo.LoadBalancer{
		ID:         "12345",
		Name:       "example-lb-updated",
		Region:     &godo.Region{Slug: "nyc3"},
		DropletIDs: []int{111, 222},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful update",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"Name":           "example-lb-updated",
				"Type":           "REGIONAL",
				"Region":         "nyc3",
				"DropletIDs":     []any{float64(111), float64(222)},
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Update(gomock.Any(), "12345", &godo.LoadBalancerRequest{
						Region:     "nyc3",
						Name:       "example-lb-updated",
						Type:       "REGIONAL",
						DropletIDs: []int{111, 222},
						ForwardingRules: []godo.ForwardingRule{
							{
								EntryProtocol:  "http",
								EntryPort:      80,
								TargetProtocol: "http",
								TargetPort:     80,
							},
						},
					}).
					Return(testLoadBalancer, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful update Global Load Balancer",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"Name":           "example-global-lb-updated",
				"Type":           "GLOBAL",
				"Tag":            "example-tag-updated",
				"GLBSettings": map[string]any{
					"TargetPort":        float64(20),
					"TargetProtocol":    "http",
					"RegionPriorities":  map[string]any{"dev1": float64(2), "dev2": float64(1)},
					"FailoverThreshold": float64(50),
					"CDN": map[string]any{
						"IsEnabled": true,
					},
				},
				"TargetLoadBalancerIDs": []string{"target-lb-3", "target-lb-4"},
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Update(gomock.Any(), "12345", &godo.LoadBalancerRequest{
						Name: "example-global-lb-updated",
						Tag:  "example-tag-updated",
						Type: "GLOBAL",
						GLBSettings: &godo.GLBSettings{
							TargetPort:        20,
							TargetProtocol:    "http",
							RegionPriorities:  map[string]uint32{"dev1": 2, "dev2": 1},
							FailoverThreshold: 50,
							CDN: &godo.CDNSettings{
								IsEnabled: true,
							},
						},
						TargetLoadBalancerIDs: []string{"target-lb-3", "target-lb-4"},
					}).
					Return(&godo.LoadBalancer{
						ID:   "12345",
						Name: "example-global-lb-updated",
						Type: "GLOBAL",
						Tag:  "example-tag-updated",
						GLBSettings: &godo.GLBSettings{
							TargetPort:        20,
							TargetProtocol:    "http",
							RegionPriorities:  map[string]uint32{"dev1": 2, "dev2": 1},
							FailoverThreshold: 50,
							CDN: &godo.CDNSettings{
								IsEnabled: true,
							},
						},
						TargetLoadBalancerIDs: []string{"target-lb-3", "target-lb-4"},
					}, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful update with optional arguments provided",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"Name":           "example-lb-updated",
				"Type":           "REGIONAL_NETWORK",
				"Region":         "nyc3",
				"DropletIDs":     []any{float64(111), float64(222)},
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
				"Network":      "INTERNAL",
				"SizeUnit":     float64(4),
				"NetworkStack": "DUALSTACK",
				"ProjectID":    "example-project-id",
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Update(gomock.Any(), "12345", &godo.LoadBalancerRequest{
						Region:     "nyc3",
						Name:       "example-lb-updated",
						Type:       "REGIONAL_NETWORK",
						DropletIDs: []int{111, 222},
						ForwardingRules: []godo.ForwardingRule{
							{
								EntryProtocol:  "http",
								EntryPort:      80,
								TargetProtocol: "http",
								TargetPort:     80,
							},
						},
						SizeUnit:     4,
						Network:      "INTERNAL",
						NetworkStack: "DUALSTACK",
						ProjectID:    "example-project-id",
					}).
					Return(&godo.LoadBalancer{
						ID:           "12345",
						Region:       &godo.Region{Slug: "nyc3"},
						Name:         "example-lb-updated",
						Type:         "REGIONAL_NETWORK",
						DropletIDs:   []int{111, 222},
						SizeUnit:     4,
						Network:      "INTERNAL",
						NetworkStack: "DUALSTACK",
						ProjectID:    "example-project-id",
					}, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing LoadBalancerID argument",
			args: map[string]any{
				"Region":     "nyc3",
				"Name":       "example-lb",
				"Type":       "REGIONAL",
				"DropletIDs": []any{float64(111), float64(222)},
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "Load Balancer ID is required",
		},
		{
			name: "Missing Name argument",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"DropletIDs":     []any{float64(111), float64(222)},
				"Region":         "nyc3",
				"Type":           "REGIONAL",
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "Name is required",
		},
		{
			name: "Missing Type argument",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"Name":           "example-lb",
				"Region":         "nyc3",
				"DropletIDs":     []any{float64(111), float64(222)},
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "Type is required",
		},
		{
			name: "Missing Region argument for REGIONAL type",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"Name":           "example-lb",
				"Type":           "REGIONAL",
				"DropletIDs":     []any{float64(111), float64(222)},
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "Region is required for REGIONAL and REGIONAL_NETWORK load balancers",
		},
		{
			name: "Both DropletIDs and Tag arguments cannot be provided",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"Region":         "nyc3",
				"Name":           "example-lb",
				"Type":           "REGIONAL",
				"DropletIDs":     []any{float64(111), float64(222)},
				"Tag":            "web-servers",
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "Only one target identifier (e.g. tag, droplets) can be specified",
		},
		{
			name: "API error",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"Name":           "example-lb",
				"Region":         "nyc3",
				"Type":           "REGIONAL",
				"DropletIDs":     []any{float64(111), float64(222)},
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					Update(gomock.Any(), "12345", &godo.LoadBalancerRequest{
						Region:     "nyc3",
						Name:       "example-lb",
						Type:       "REGIONAL",
						DropletIDs: []int{111, 222},
						ForwardingRules: []godo.ForwardingRule{
							{
								EntryProtocol:  "http",
								EntryPort:      80,
								TargetProtocol: "http",
								TargetPort:     80,
							},
						},
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateLoadBalancer(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				if tc.expectText != "" {
					require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outLoadBalancer godo.LoadBalancer
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outLoadBalancer))
			require.Equal(t, testLoadBalancer.ID, outLoadBalancer.ID)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
		})
	}
}

func TestLoadBalancersTool_addForwardingRules(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful add forwarding rules",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					AddForwardingRules(gomock.Any(), "12345", []godo.ForwardingRule{
						{
							EntryProtocol:  "http",
							EntryPort:      80,
							TargetProtocol: "http",
							TargetPort:     80,
						},
					}).
					Times(1)
			},
			expectText: "Forwarding rules added successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					AddForwardingRules(gomock.Any(), "12345", []godo.ForwardingRule{
						{
							EntryProtocol:  "http",
							EntryPort:      80,
							TargetProtocol: "http",
							TargetPort:     80,
						},
					}).
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name: "Missing LoadBalancerID argument",
			args: map[string]any{
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "Missing ForwardingRules argument",
			args: map[string]any{
				"LoadBalancerID": "12345",
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "Forwarding Rules are required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.addForwardingRules(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Equal(t, tc.expectText, resp.Content[0].(mcp.TextContent).Text)
		})
	}
}

func TestLoadBalancersTool_removeForwardingRules(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(m *MockLoadBalancersService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful remove forwarding rules",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					RemoveForwardingRules(gomock.Any(), "12345", []godo.ForwardingRule{
						{
							EntryProtocol:  "http",
							EntryPort:      80,
							TargetProtocol: "http",
							TargetPort:     80,
						},
					}).
					Times(1)
			},
			expectText: "Forwarding rules removed successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"LoadBalancerID": "12345",
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      float64(80),
						"TargetProtocol": "http",
						"TargetPort":     float64(80),
					},
				},
			},
			mockSetup: func(m *MockLoadBalancersService) {
				m.EXPECT().
					RemoveForwardingRules(gomock.Any(), "12345", []godo.ForwardingRule{
						{
							EntryProtocol:  "http",
							EntryPort:      80,
							TargetProtocol: "http",
							TargetPort:     80,
						},
					}).
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name: "Missing LoadBalancerID argument",
			args: map[string]any{
				"ForwardingRules": []any{
					map[string]any{
						"EntryProtocol":  "http",
						"EntryPort":      80,
						"TargetProtocol": "http",
						"TargetPort":     80,
					},
				},
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "LoadBalancerID is required",
		},
		{
			name: "Missing ForwardingRules argument",
			args: map[string]any{
				"LoadBalancerID": "12345",
			},
			mockSetup:   nil,
			expectError: true,
			expectText:  "At least one forwarding rule must be provided",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockLoadBalancers := NewMockLoadBalancersService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockLoadBalancers)
			}
			tool := setupLoadBalancersToolWithMock(mockLoadBalancers)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.removeForwardingRules(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Equal(t, tc.expectText, resp.Content[0].(mcp.TextContent).Text)
		})
	}
}
