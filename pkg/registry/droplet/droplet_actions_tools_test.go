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

func setupDropletActionsToolWithMocks(actions *MockDropletActionsService) *DropletActionsTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{DropletActions: actions}, nil
	}

	return NewDropletActionsTool(client)
}

func TestDropletActionsTool_rebootDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testAction := &godo.Action{ID: 2001, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful reboot",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().Reboot(gomock.Any(), 123).Return(testAction, nil, nil).Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().Reboot(gomock.Any(), 456).Return(nil, nil, errors.New("api error")).Times(1)
			},
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.rebootDroplet(context.Background(), req)
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

func TestDropletActionsTool_powerCycleByTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{{ID: 3001, Status: "in-progress"}}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful power cycle by tag",
			args: map[string]any{"Tag": "tag1"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerCycleByTag(gomock.Any(), "tag1").
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Tag": "fail-tag"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerCycleByTag(gomock.Any(), "fail-tag").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.powerCycleByTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outActions []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outActions))
			require.Equal(t, testActions[0].ID, outActions[0].ID)
		})
	}
}

func TestDropletActionsTool_powerOnByTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{{ID: 3002, Status: "in-progress"}}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful power on by tag",
			args: map[string]any{"Tag": "tag2"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerOnByTag(gomock.Any(), "tag2").
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Tag": "fail-tag"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerOnByTag(gomock.Any(), "fail-tag").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.powerOnByTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outActions []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outActions))
			require.Equal(t, testActions[0].ID, outActions[0].ID)
		})
	}
}

func TestDropletActionsTool_powerOffByTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{{ID: 3003, Status: "in-progress"}}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful power off by tag",
			args: map[string]any{"Tag": "tag3"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerOffByTag(gomock.Any(), "tag3").
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Tag": "fail-tag"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					PowerOffByTag(gomock.Any(), "fail-tag").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.powerOffByTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outActions []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outActions))
			require.Equal(t, testActions[0].ID, outActions[0].ID)
		})
	}
}

func TestDropletActionsTool_shutdownByTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{{ID: 3004, Status: "in-progress"}}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful shutdown by tag",
			args: map[string]any{"Tag": "tag4"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					ShutdownByTag(gomock.Any(), "tag4").
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Tag": "fail-tag"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					ShutdownByTag(gomock.Any(), "fail-tag").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.shutdownByTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outActions []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outActions))
			require.Equal(t, testActions[0].ID, outActions[0].ID)
		})
	}
}

func TestDropletActionsTool_enableBackupsByTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{{ID: 3005, Status: "in-progress"}}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful enable backups by tag",
			args: map[string]any{"Tag": "tag5"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnableBackupsByTag(gomock.Any(), "tag5").
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Tag": "fail-tag"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnableBackupsByTag(gomock.Any(), "fail-tag").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.enableBackupsByTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outActions []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outActions))
			require.Equal(t, testActions[0].ID, outActions[0].ID)
		})
	}
}

func TestDropletActionsTool_disableBackupsByTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{{ID: 3006, Status: "in-progress"}}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful disable backups by tag",
			args: map[string]any{"Tag": "tag6"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					DisableBackupsByTag(gomock.Any(), "tag6").
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Tag": "fail-tag"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					DisableBackupsByTag(gomock.Any(), "fail-tag").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.disableBackupsByTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outActions []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outActions))
			require.Equal(t, testActions[0].ID, outActions[0].ID)
		})
	}
}

func TestDropletActionsTool_snapshotByTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{{ID: 3007, Status: "in-progress"}}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful snapshot by tag",
			args: map[string]any{"Tag": "tag7", "Name": "snap-by-tag"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					SnapshotByTag(gomock.Any(), "tag7", "snap-by-tag").
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Tag": "fail-tag", "Name": "fail"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					SnapshotByTag(gomock.Any(), "fail-tag", "fail").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.snapshotByTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outActions []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outActions))
			require.Equal(t, testActions[0].ID, outActions[0].ID)
		})
	}
}

func TestDropletActionsTool_enableIPv6ByTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{{ID: 3008, Status: "in-progress"}}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful enable IPv6 by tag",
			args: map[string]any{"Tag": "tag8"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnableIPv6ByTag(gomock.Any(), "tag8").
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Tag": "fail-tag"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnableIPv6ByTag(gomock.Any(), "fail-tag").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.enableIPv6ByTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outActions []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outActions))
			require.Equal(t, testActions[0].ID, outActions[0].ID)
		})
	}
}

func TestDropletActionsTool_enablePrivateNetworkingByTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{{ID: 3009, Status: "in-progress"}}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful enable private networking by tag",
			args: map[string]any{"Tag": "tag9"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnablePrivateNetworkingByTag(gomock.Any(), "tag9").
					Return(testActions, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Tag": "fail-tag"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnablePrivateNetworkingByTag(gomock.Any(), "fail-tag").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.enablePrivateNetworkingByTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outActions []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outActions))
			require.Equal(t, testActions[0].ID, outActions[0].ID)
		})
	}
}

func TestDropletActionsTool_powerCycleDroplet(t *testing.T) {
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
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
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

func TestDropletActionsTool_powerOnDroplet(t *testing.T) {
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
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
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

func TestDropletActionsTool_powerOffDroplet(t *testing.T) {
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
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
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

func TestDropletActionsTool_shutdownDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 444, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful shutdown",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Shutdown(gomock.Any(), 123).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Shutdown(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.shutdownDroplet(context.Background(), req)
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

func TestDropletActionsTool_restoreDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 555, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful restore",
			args: map[string]any{"ID": float64(123), "ImageID": float64(789)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Restore(gomock.Any(), 123, 789).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456), "ImageID": float64(101)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Restore(gomock.Any(), 456, 101).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.restoreDroplet(context.Background(), req)
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

func TestDropletActionsTool_resizeDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 666, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful resize",
			args: map[string]any{"ID": float64(123), "Size": "s-1vcpu-1gb", "ResizeDisk": true},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Resize(gomock.Any(), 123, "s-1vcpu-1gb", true).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456), "Size": "s-2vcpu-2gb", "ResizeDisk": false},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Resize(gomock.Any(), 456, "s-2vcpu-2gb", false).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.resizeDroplet(context.Background(), req)
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

func TestDropletActionsTool_rebuildDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 777, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful rebuild",
			args: map[string]any{"ID": float64(123), "ImageID": float64(789)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					RebuildByImageID(gomock.Any(), 123, 789).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456), "ImageID": float64(101)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					RebuildByImageID(gomock.Any(), 456, 101).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.rebuildDroplet(context.Background(), req)
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

func TestDropletActionsTool_renameDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 888, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful rename",
			args: map[string]any{"ID": float64(123), "Name": "new-name"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Rename(gomock.Any(), 123, "new-name").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456), "Name": "fail-name"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Rename(gomock.Any(), 456, "fail-name").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.renameDroplet(context.Background(), req)
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

func TestDropletActionsTool_changeKernel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 999, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful change kernel",
			args: map[string]any{"ID": float64(123), "KernelID": float64(321)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					ChangeKernel(gomock.Any(), 123, 321).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456), "KernelID": float64(654)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					ChangeKernel(gomock.Any(), 456, 654).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.changeKernel(context.Background(), req)
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

func TestDropletActionsTool_enableIPv6(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 1001, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful enable IPv6",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnableIPv6(gomock.Any(), 123).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnableIPv6(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.enableIPv6(context.Background(), req)
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

func TestDropletActionsTool_enableBackups(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 1002, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful enable backups",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnableBackups(gomock.Any(), 123).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					EnableBackups(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.enableBackups(context.Background(), req)
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

func TestDropletActionsTool_disableBackups(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 1003, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful disable backups",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					DisableBackups(gomock.Any(), 123).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					DisableBackups(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.disableBackups(context.Background(), req)
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

func TestDropletActionsTool_snapshotDroplet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 1004, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful snapshot",
			args: map[string]any{"ID": float64(123), "Name": "snap1"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Snapshot(gomock.Any(), 123, "snap1").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456), "Name": "fail"},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Snapshot(gomock.Any(), 456, "fail").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockDropletActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupDropletActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.snapshotDroplet(context.Background(), req)
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
