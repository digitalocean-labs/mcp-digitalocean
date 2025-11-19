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

func setupUptimeAlertToolWithMock(mockChecks *MockUptimeChecksService) *UptimeCheckAlertTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			UptimeChecks: mockChecks,
		}, nil
	}

	return NewUptimeCheckAlertTool(client)
}

func TestUptimeCheckAlertTool_getUptimeCheckAlert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testAlert := &godo.UptimeAlert{ID: "alert1", Name: "alert"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
	}{
		{
			name:        "missing CheckID",
			args:        map[string]any{"AlertID": "alert1"},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name:        "missing AlertID",
			args:        map[string]any{"CheckID": "check1"},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"CheckID": "check1", "AlertID": "alert1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().GetAlert(gomock.Any(), "check1", "alert1").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"CheckID": "check1", "AlertID": "alert1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().GetAlert(gomock.Any(), "check1", "alert1").Return(testAlert, nil, nil)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeAlertToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getUptimeCheckAlert(context.Background(), req)
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

func TestUptimeCheckAlertTool_listUptimeCheckAlerts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testAlerts := []godo.UptimeAlert{{ID: "alert1"}, {ID: "alert2"}}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
	}{
		{
			name:        "missing CheckID",
			args:        map[string]any{},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"CheckID": "check1", "Page": float64(1), "PerPage": float64(2)},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().ListAlerts(gomock.Any(), "check1", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"CheckID": "check1", "Page": float64(1), "PerPage": float64(2)},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().ListAlerts(gomock.Any(), "check1", gomock.Any()).DoAndReturn(
					func(_ context.Context, checkID string, opts *godo.ListOptions) ([]godo.UptimeAlert, *godo.Response, error) {
						require.Equal(t, "check1", checkID)
						return testAlerts, nil, nil
					},
				)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeAlertToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listUptimeCheckAlerts(context.Background(), req)
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

func TestUptimeCheckAlertTool_createUptimeCheckAlert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testAlert := &godo.UptimeAlert{ID: "alert1", Name: "alert"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
	}{
		{
			name:        "missing CheckID",
			args:        map[string]any{"Name": "alert"},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"CheckID": "check1", "Name": "alert", "Type": "latency", "Threshold": float64(100), "Period": "2m", "Comparison": "greater_than", "Emails": []string{"a@b.com"}, "SlackDetails": []map[string]any{{"channel": "alerts", "url": "https://slack"}}},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().CreateAlert(gomock.Any(), "check1", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"CheckID": "check1", "Name": "alert", "Type": "latency", "Threshold": float64(100), "Period": "2m", "Comparison": "greater_than", "Emails": []string{"a@b.com"}, "SlackDetails": []map[string]any{{"channel": "alerts", "url": "https://slack"}}},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().CreateAlert(gomock.Any(), "check1", gomock.Any()).DoAndReturn(
					func(_ context.Context, checkID string, req *godo.CreateUptimeAlertRequest) (*godo.UptimeAlert, *godo.Response, error) {
						require.Equal(t, "check1", checkID)
						return testAlert, nil, nil
					},
				)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeAlertToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createUptimeCheckAlert(context.Background(), req)
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

func TestUptimeCheckAlertTool_updateUptimeCheckAlert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testAlert := &godo.UptimeAlert{ID: "alert1", Name: "alert"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
	}{
		{
			name:        "missing CheckID",
			args:        map[string]any{"AlertID": "alert1"},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name:        "missing AlertID",
			args:        map[string]any{"CheckID": "check1"},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"CheckID": "check1", "AlertID": "alert1", "Name": "alert", "Type": "latency", "Threshold": float64(100), "Period": "2m", "Comparison": "greater_than", "Emails": []string{"a@b.com"}, "SlackDetails": []map[string]any{{"channel": "alerts", "url": "https://slack"}}},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().UpdateAlert(gomock.Any(), "check1", "alert1", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"CheckID": "check1", "AlertID": "alert1", "Name": "alert", "Type": "latency", "Threshold": float64(100), "Period": "2m", "Comparison": "greater_than", "Emails": []string{"a@b.com"}, "SlackDetails": []map[string]any{{"channel": "alerts", "url": "https://slack"}}},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().UpdateAlert(gomock.Any(), "check1", "alert1", gomock.Any()).DoAndReturn(
					func(_ context.Context, checkID, alertID string, req *godo.UpdateUptimeAlertRequest) (*godo.UptimeAlert, *godo.Response, error) {
						require.Equal(t, "check1", checkID)
						return testAlert, nil, nil
					},
				)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeAlertToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateUptimeCheckAlert(context.Background(), req)
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

func TestUptimeCheckAlertTool_deleteUptimeCheckAlert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockUptimeChecksService)
		expectError bool
	}{
		{
			name:        "missing CheckID",
			args:        map[string]any{"AlertID": "alert1"},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name:        "missing AlertID",
			args:        map[string]any{"CheckID": "check1"},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"CheckID": "check1", "AlertID": "alert1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().DeleteAlert(gomock.Any(), "check1", "alert1").Return(nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"CheckID": "check1", "AlertID": "alert1"},
			mockSetup: func(m *MockUptimeChecksService) {
				m.EXPECT().DeleteAlert(gomock.Any(), "check1", "alert1").Return(nil, nil)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockChecks := NewMockUptimeChecksService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockChecks)
			}
			tool := setupUptimeAlertToolWithMock(mockChecks)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteUptimeCheckAlert(context.Background(), req)
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
