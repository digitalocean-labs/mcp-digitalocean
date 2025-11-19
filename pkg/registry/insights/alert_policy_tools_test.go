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

func setupAlertPolicyToolWithMock(mockMonitoring *MockMonitoringService) *AlertPolicyTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Monitoring: mockMonitoring,
		}, nil
	}

	return NewAlertPolicyTool(client)
}

func TestAlertPolicyTool_getAlertPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testPolicy := &godo.AlertPolicy{UUID: "id1", Description: "test policy"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockMonitoringService)
		expectError bool
	}{
		{
			name:        "missing UUID",
			args:        map[string]any{},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"UUID": "id1"},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().GetAlertPolicy(gomock.Any(), "id1").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"UUID": "id1"},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().GetAlertPolicy(gomock.Any(), "id1").Return(testPolicy, nil, nil)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockMonitoring := NewMockMonitoringService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockMonitoring)
			}
			tool := setupAlertPolicyToolWithMock(mockMonitoring)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getAlertPolicy(context.Background(), req)
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

func TestAlertPolicyTool_listAlertPolicies(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testPolicies := []godo.AlertPolicy{{UUID: "id1"}, {UUID: "id2"}}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockMonitoringService)
		expectError bool
	}{
		{
			name: "api error",
			args: map[string]any{"Page": float64(1), "PerPage": float64(2)},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().ListAlertPolicies(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"Page": float64(1), "PerPage": float64(2)},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().ListAlertPolicies(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, opts *godo.ListOptions) ([]godo.AlertPolicy, *godo.Response, error) {
						require.Equal(t, 1, opts.Page)
						require.Equal(t, 2, opts.PerPage)
						return testPolicies, nil, nil
					},
				)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockMonitoring := NewMockMonitoringService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockMonitoring)
			}
			tool := setupAlertPolicyToolWithMock(mockMonitoring)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listAlertPolicies(context.Background(), req)
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

func TestAlertPolicyTool_createAlertPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testPolicy := &godo.AlertPolicy{UUID: "id1", Description: "test policy"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockMonitoringService)
		expectError bool
	}{
		{
			name: "api error",
			args: map[string]any{
				"Type":        "v1/insights/droplet/cpu",
				"Description": "High CPU usage",
				"Compare":     "GreaterThan",
				"Value":       float64(80),
				"Window":      "5m",
				"Alerts": map[string]any{
					"Email": []string{"test@example.com"},
					"Slack": []map[string]any{{
						"URL":     "https://hooks.slack.com/services/xxx",
						"Channel": "#alerts",
					}},
				},
				"Enabled": true,
			},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().CreateAlertPolicy(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{
				"Type":        "v1/insights/droplet/cpu",
				"Description": "High CPU usage",
				"Compare":     "GreaterThan",
				"Value":       float64(80),
				"Window":      "5m",
				"Alerts": map[string]any{
					"Email": []string{"test@example.com"},
					"Slack": []map[string]any{{
						"URL":     "https://hooks.slack.com/services/xxx",
						"Channel": "#alerts",
					}},
				},
				"Enabled": true,
			},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().CreateAlertPolicy(gomock.Any(), gomock.Any()).Return(testPolicy, nil, nil)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockMonitoring := NewMockMonitoringService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockMonitoring)
			}
			tool := setupAlertPolicyToolWithMock(mockMonitoring)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createAlertPolicy(context.Background(), req)
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

func TestAlertPolicyTool_updateAlertPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testPolicy := &godo.AlertPolicy{UUID: "id1", Description: "test policy"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockMonitoringService)
		expectError bool
	}{
		{
			name:        "missing UUID",
			args:        map[string]any{},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{
				"UUID":        "id1",
				"Type":        "v1/insights/droplet/cpu",
				"Description": "High CPU usage",
				"Compare":     "GreaterThan",
				"Value":       float64(80),
				"Window":      "5m",
				"Alerts": map[string]any{
					"Email": []string{"test@example.com"},
					"Slack": []map[string]any{{
						"URL":     "https://hooks.slack.com/services/xxx",
						"Channel": "#alerts",
					}},
				},
				"Enabled": true,
			},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().UpdateAlertPolicy(gomock.Any(), "id1", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{
				"UUID":        "id1",
				"Type":        "v1/insights/droplet/cpu",
				"Description": "High CPU usage",
				"Compare":     "GreaterThan",
				"Value":       float64(80),
				"Window":      "5m",
				"Alerts": map[string]any{
					"Email": []string{"test@example.com"},
					"Slack": []map[string]any{{
						"URL":     "https://hooks.slack.com/services/xxx",
						"Channel": "#alerts",
					}},
				},
				"Enabled": true,
			},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().UpdateAlertPolicy(gomock.Any(), "id1", gomock.Any()).Return(testPolicy, nil, nil)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockMonitoring := NewMockMonitoringService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockMonitoring)
			}
			tool := setupAlertPolicyToolWithMock(mockMonitoring)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateAlertPolicy(context.Background(), req)
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

func TestAlertPolicyTool_deleteAlertPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockMonitoringService)
		expectError bool
	}{
		{
			name:        "missing UUID",
			args:        map[string]any{},
			mockSetup:   nil,
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"UUID": "id1"},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().DeleteAlertPolicy(gomock.Any(), "id1").Return(nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"UUID": "id1"},
			mockSetup: func(m *MockMonitoringService) {
				m.EXPECT().DeleteAlertPolicy(gomock.Any(), "id1").Return(nil, nil)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockMonitoring := NewMockMonitoringService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockMonitoring)
			}
			tool := setupAlertPolicyToolWithMock(mockMonitoring)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteAlertPolicy(context.Background(), req)
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
