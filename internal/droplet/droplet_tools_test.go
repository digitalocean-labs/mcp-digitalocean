package droplet

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

func setupDropletToolWithMocks(droplets *MockDropletsService, actions *MockDropletActionsService) *DropletTool {
	client := &godo.Client{}
	client.Droplets = droplets
	client.DropletActions = actions
	return NewDropletTool(client)
}

func TestDropletTool_createDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDroplet := &godo.Droplet{
		ID:   123,
		Name: "test-droplet",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletsService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":       "test-droplet",
				"Size":       "s-1vcpu-1gb",
				"ImageID":    float64(456),
				"Region":     "nyc1",
				"Backup":     true,
				"Monitoring": false,
			},
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.DropletCreateRequest{
						Name:       "test-droplet",
						Region:     "nyc1",
						Size:       "s-1vcpu-1gb",
						Image:      godo.DropletCreateImage{ID: 456},
						Backups:    true,
						Monitoring: false,
					}).
					Return(testDroplet, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":       "fail-droplet",
				"Size":       "s-1vcpu-1gb",
				"ImageID":    float64(789),
				"Region":     "nyc3",
				"Backup":     false,
				"Monitoring": true,
			},
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.DropletCreateRequest{
						Name:       "fail-droplet",
						Region:     "nyc3",
						Size:       "s-1vcpu-1gb",
						Image:      godo.DropletCreateImage{ID: 789},
						Backups:    false,
						Monitoring: true,
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDroplets := NewMockDropletsService(ctrl)
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDroplets)
			}
			tool := setupDropletToolWithMocks(mockDroplets, mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createDroplet(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
		})
	}
}

func TestDropletTool_deleteDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletsService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().
					Delete(gomock.Any(), 123).
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Droplet deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().
					Delete(gomock.Any(), 456).
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDroplets := NewMockDropletsService(ctrl)
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDroplets)
			}
			tool := setupDropletToolWithMocks(mockDroplets, mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteDroplet(context.Background(), req)
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

func TestDropletTool_powerCycleDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 111, Status: "in-progress"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful power cycle",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerCycle(gomock.Any(), 123).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerCycle(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDroplets := NewMockDropletsService(ctrl)
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletToolWithMocks(mockDroplets, mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.powerCycleDroplet(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outAction godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outAction))
			require.Equal(t, testAction.ID, outAction.ID)
		})
	}
}

func TestDropletTool_powerOnDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 222, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful power on",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerOn(gomock.Any(), 123).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerOn(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDroplets := NewMockDropletsService(ctrl)
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletToolWithMocks(mockDroplets, mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.powerOnDroplet(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outAction godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outAction))
			require.Equal(t, testAction.ID, outAction.ID)
		})
	}
}

func TestDropletTool_powerOffDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 333, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful power off",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerOff(gomock.Any(), 123).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerOff(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDroplets := NewMockDropletsService(ctrl)
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletToolWithMocks(mockDroplets, mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.powerOffDroplet(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outAction godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outAction))
			require.Equal(t, testAction.ID, outAction.ID)
		})
	}
}
