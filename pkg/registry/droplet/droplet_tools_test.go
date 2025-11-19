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
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Droplets:       droplets,
			DropletActions: actions,
		}, nil
	}
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

func TestDropletTool_getDropletByID(t *testing.T) {
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
			name: "Successful get",
			args: map[string]any{"ID": float64(123)},
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().
					Get(gomock.Any(), 123).
					Return(testDroplet, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"ID": float64(456)},
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().
					Get(gomock.Any(), 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing ID argument",
			args:        map[string]any{},
			mockSetup:   nil,
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
			resp, err := tool.getDropletByID(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outDroplet godo.Droplet
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outDroplet))
			require.Equal(t, testDroplet.ID, outDroplet.ID)
		})
	}
}

func TestDropletTool_getDropletActionByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{
		ID:     789,
		Status: "completed",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletActionsService)
		expectError bool
	}{
		{
			name: "Successful get action",
			args: map[string]any{"DropletID": float64(123), "ActionID": float64(789)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Get(gomock.Any(), 123, 789).
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"DropletID": float64(456), "ActionID": float64(999)},
			mockSetup: func(m *MockDropletActionsService) {
				m.EXPECT().
					Get(gomock.Any(), 456, 999).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing DropletID argument",
			args:        map[string]any{"ActionID": float64(789)},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name:        "Missing ActionID argument",
			args:        map[string]any{"DropletID": float64(123)},
			mockSetup:   nil,
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
			resp, err := tool.getDropletActionByID(context.Background(), req)
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

func TestDropletTool_getDroplets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDroplet := godo.Droplet{
		ID:               123,
		Name:             "test-droplet",
		Memory:           2048,
		Vcpus:            2,
		Disk:             50,
		Region:           &godo.Region{Slug: "nyc1", Name: "New York 1"},
		Image:            &godo.Image{ID: 456, Name: "ubuntu-20-04-x64", Distribution: "Ubuntu"},
		Size:             &godo.Size{Slug: "s-1vcpu-2gb", Memory: 2048, Vcpus: 2, Disk: 50},
		SizeSlug:         "s-1vcpu-2gb",
		BackupIDs:        []int{1, 2},
		NextBackupWindow: &godo.BackupWindow{},
		SnapshotIDs:      []int{3, 4},
		Features:         []string{"ipv6", "private_networking"},
		Locked:           false,
		Status:           "active",
		Networks:         &godo.Networks{},
		Created:          "2023-01-01T00:00:00Z",
		Kernel:           &godo.Kernel{ID: 789, Name: "kernel-1", Version: "1.0.0"},
		Tags:             []string{"web", "prod"},
		VolumeIDs:        []string{"vol-1", "vol-2"},
		VPCUUID:          "vpc-uuid-123",
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDropletsService)
		expectError bool
	}{
		{
			name: "Successful list",
			args: map[string]any{"Page": float64(1), "PerPage": float64(1)},
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 1}).Return([]godo.Droplet{testDroplet}, nil, nil).Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{"Page": float64(1), "PerPage": float64(1)},
			mockSetup: func(m *MockDropletsService) {
				m.EXPECT().List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 1}).Return(nil, nil, errors.New("api error")).Times(1)
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
			resp, err := tool.getDroplets(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outDroplets []map[string]any
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outDroplets))
			require.Len(t, outDroplets, 1)
			out := outDroplets[0]
			// Check that all expected fields are present
			for _, field := range []string{
				"id", "name", "memory", "vcpus", "disk", "region", "image", "size", "size_slug", "backup_ids", "next_backup_window", "snapshot_ids", "features", "locked", "status", "networks", "created_at", "kernel", "tags", "volume_ids", "vpc_uuid",
			} {
				require.Contains(t, out, field)
			}
			// Spot check a few values
			require.Equal(t, float64(testDroplet.ID), out["id"])
			require.Equal(t, testDroplet.Name, out["name"])
			require.Equal(t, testDroplet.SizeSlug, out["size_slug"])
		})
	}
}
