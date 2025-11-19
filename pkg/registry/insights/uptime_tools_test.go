package insights

import (
	"context"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupUptimeToolWithMock(mockChecks godo.UptimeChecksService) *UptimeTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			UptimeChecks: mockChecks,
		}, nil
	}

	return NewUptimeTool(client)
}

func TestUptimeTool_getUptimeCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testUptimeCheck := &godo.UptimeCheck{ID: "id1", Name: "test"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
		expect      *godo.UptimeCheck
	}{
		{
			name:        "missing ID",
			args:        map[string]any{},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"ID": "id1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().Get(gomock.Any(), "id1").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"ID": "id1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().Get(gomock.Any(), "id1").Return(testUptimeCheck, nil, nil)
			},
			expect: testUptimeCheck,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getUptimeCheck(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestUptimeTool_getUptimeCheckState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testState := &godo.UptimeCheckState{
		Regions: map[string]godo.UptimeRegion{
			"nyc": {
				Status:                    "UP",
				StatusChangedAt:           "2025-07-17T11:26:26Z",
				ThirtyDayUptimePercentage: 100,
			},
		}}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
		expect      *godo.UptimeCheckState
	}{
		{
			name:        "missing ID",
			args:        map[string]any{},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"ID": "id1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().GetState(gomock.Any(), "id1").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"ID": "id1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().GetState(gomock.Any(), "id1").Return(testState, nil, nil)
			},
			expect: testState,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getUptimeCheckState(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestUptimeTool_listUptimeChecks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testChecks := []godo.UptimeCheck{{ID: "id1"}, {ID: "id2"}}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
		expect      []godo.UptimeCheck
	}{
		{
			name: "api error",
			args: map[string]any{"Page": float64(1), "PerPage": float64(2)},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"Page": float64(1), "PerPage": float64(2)},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, opts *godo.ListOptions) ([]godo.UptimeCheck, *godo.Response, error) {
						require.Equal(t, 1, opts.Page)
						require.Equal(t, 2, opts.PerPage)
						return testChecks, nil, nil
					},
				)
			},
			expect: testChecks,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listUptimeChecks(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestUptimeTool_createUptimeCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testUptimeCheck := &godo.UptimeCheck{ID: "id1", Name: "n"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
		expect      *godo.UptimeCheck
	}{
		{
			name: "api error",
			args: map[string]any{"Name": "n", "Type": "t", "Target": "x", "Regions": []string{"nyc"}, "Enabled": true},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"Name": "n", "Type": "t", "Target": "x", "Regions": []string{"nyc"}, "Enabled": true},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *godo.CreateUptimeCheckRequest) (*godo.UptimeCheck, *godo.Response, error) {
						require.Equal(t, "n", req.Name)
						return testUptimeCheck, nil, nil
					},
				)
			},
			expect: testUptimeCheck,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createUptimeCheck(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestUptimeTool_updateUptimeCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testUptimeCheck := &godo.UptimeCheck{ID: "id1", Name: "n"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
		expect      *godo.UptimeCheck
	}{
		{
			name:        "missing ID",
			args:        map[string]any{},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"ID": "id1", "Name": "n", "Type": "t", "Target": "x", "Regions": []string{"nyc"}, "Enabled": true},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().Update(gomock.Any(), "id1", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"ID": "id1", "Name": "n", "Type": "t", "Target": "x", "Regions": []string{"nyc"}, "Enabled": true},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().Update(gomock.Any(), "id1", gomock.Any()).DoAndReturn(
					func(_ context.Context, id string, req *godo.UpdateUptimeCheckRequest) (*godo.UptimeCheck, *godo.Response, error) {
						require.Equal(t, "id1", id)
						return testUptimeCheck, nil, nil
					},
				)
			},
			expect: testUptimeCheck,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateUptimeCheck(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestUptimeTool_deleteUptimeCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
		expectText  string
	}{
		{
			name: "api error",
			args: map[string]any{"ID": "id1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().Delete(gomock.Any(), "id1").Return(nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"ID": "id1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().Delete(gomock.Any(), "id1").Return(nil, nil)
			},
			expectText: "deleted successfully",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteUptimeCheck(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}
