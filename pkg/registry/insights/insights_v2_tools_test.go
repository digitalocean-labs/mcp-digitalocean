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

func setupInsightsTools(mock *MockInsightsService) (*AlertRuleTool, *NotificationChannelTool, *AlertInstanceTool, *QueryTool) {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Insights: mock}, nil
	}
	return NewAlertRuleTool(client), NewNotificationChannelTool(client), NewAlertInstanceTool(client), NewQueryTool(client)
}

func TestAlertRuleTool_get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockInsightsService(ctrl)
	tool, _, _, _ := setupInsightsTools(mock)

	resp, err := tool.get(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().GetAlertRule(gomock.Any(), "id1").Return(nil, nil, errors.New("api error"))
	resp, err = tool.get(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"ID": "id1"}}})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().GetAlertRule(gomock.Any(), "id1").Return(&godo.AlertRule{ID: "id1"}, nil, nil)
	resp, err = tool.get(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"ID": "id1"}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestAlertRuleTool_list(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockInsightsService(ctrl)
	tool, _, _, _ := setupInsightsTools(mock)

	mock.EXPECT().ListAlertRules(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, opt *godo.AlertRuleListOptions) ([]godo.AlertRule, *godo.Response, error) {
			require.Equal(t, 2, opt.Page)
			require.Equal(t, 10, opt.PerPage)
			require.Equal(t, "do:droplet:1", opt.ResourceURN)
			return []godo.AlertRule{{ID: "id1"}}, nil, nil
		},
	)
	resp, err := tool.list(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Page": float64(2), "PerPage": float64(10), "ResourceURN": "do:droplet:1",
	}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestAlertRuleTool_create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockInsightsService(ctrl)
	tool, _, _, _ := setupInsightsTools(mock)

	args := map[string]any{
		"Name":     "High CPU",
		"Metric":   "do.droplets.cpu_utilization",
		"Operator": godo.InsightsThresholdOperatorGreaterThan,
		"Critical": float64(95),
		"Window":   godo.InsightsEvaluationWindow5m,
		"NotificationChannels": []any{map[string]any{
			"NotificationChannelID": "ch1",
			"NotifyOn":              []any{godo.InsightsSeverityCritical},
		}},
	}

	resp, err := tool.create(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"Name": "x"}}})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().CreateAlertRule(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *godo.AlertRuleRequest) (*godo.AlertRule, *godo.Response, error) {
			require.Equal(t, "High CPU", req.Spec.Name)
			require.Equal(t, "do.droplets.cpu_utilization", req.Spec.Query.Metric)
			require.NotNil(t, req.Spec.NotificationChannels)
			require.Len(t, *req.Spec.NotificationChannels, 1)
			return &godo.AlertRule{ID: "rule1"}, nil, nil
		},
	)
	resp, err = tool.create(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestAlertRuleTool_updateAndDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockInsightsService(ctrl)
	tool, _, _, _ := setupInsightsTools(mock)

	resp, err := tool.update(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().UpdateAlertRule(gomock.Any(), "rule1", gomock.Any()).Return(&godo.AlertRule{ID: "rule1"}, nil, nil)
	resp, err = tool.update(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"ID": "rule1", "Name": "n", "Metric": "do.droplets.cpu_utilization", "Operator": godo.InsightsThresholdOperatorGreaterThan,
	}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	mock.EXPECT().DeleteAlertRule(gomock.Any(), "rule1").Return(nil, nil)
	resp, err = tool.delete(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"ID": "rule1"}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestNotificationChannelTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockInsightsService(ctrl)
	_, tool, _, _ := setupInsightsTools(mock)

	resp, err := tool.create(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"Name": "ops"}}})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().CreateNotificationChannel(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *godo.NotificationChannelRequest) (*godo.NotificationChannel, *godo.Response, error) {
			require.Equal(t, "ops", req.Name)
			require.Equal(t, "ops@example.com", req.Email.To)
			return &godo.NotificationChannel{ID: "ch1"}, nil, nil
		},
	)
	resp, err = tool.create(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Name": "ops", "EmailTo": "ops@example.com",
	}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	mock.EXPECT().ListNotificationChannels(gomock.Any(), gomock.Any()).Return([]godo.NotificationChannel{{ID: "ch1"}}, nil, nil)
	resp, err = tool.list(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestAlertInstanceTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockInsightsService(ctrl)
	_, _, tool, _ := setupInsightsTools(mock)

	mock.EXPECT().ListAlertInstances(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, opt *godo.AlertInstanceListOptions) ([]godo.AlertInstance, *godo.Response, error) {
			require.Equal(t, "active", opt.Status)
			require.Equal(t, "rule1", opt.RuleID)
			return []godo.AlertInstance{{ID: "i1"}}, nil, nil
		},
	)
	resp, err := tool.list(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Status": "active", "RuleID": "rule1",
	}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	mock.EXPECT().GetAlertInstance(gomock.Any(), "i1").Return(&godo.AlertInstance{ID: "i1"}, nil, nil)
	resp, err = tool.get(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"ID": "i1"}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestQueryTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockInsightsService(ctrl)
	_, _, _, tool := setupInsightsTools(mock)

	resp, err := tool.query(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().Query(gomock.Any(), "nyc3", gomock.Any()).Return(&godo.PromQueryResponse{Status: "success"}, nil, nil)
	resp, err = tool.query(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Region": "nyc3", "Query": "up",
	}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	mock.EXPECT().PostQuery(gomock.Any(), "nyc3", gomock.Any()).Return(&godo.PromQueryResponse{Status: "success"}, nil, nil)
	resp, err = tool.query(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Region": "nyc3", "Query": "up", "UsePOST": true,
	}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestQueryTool_searchLogs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockInsightsService(ctrl)
	_, _, _, tool := setupInsightsTools(mock)

	resp, err := tool.searchLogs(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().SearchLogs(gomock.Any(), "nyc3", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, req *godo.LogsSearchRequest) (*godo.LogsSearchResponse, *godo.Response, error) {
			require.Equal(t, "now-1h", req.TimeRange.From.Relative)
			require.Equal(t, "now", req.TimeRange.To.Relative)
			require.NotNil(t, req.Filter)
			require.NotNil(t, req.Filter.TextSearch)
			require.Equal(t, "timeout", req.Filter.TextSearch.Query)
			require.NotNil(t, req.Pagination)
			require.Equal(t, 50, req.Pagination.Limit)
			return &godo.LogsSearchResponse{Data: []godo.InsightsLogRecord{{Body: "timeout"}}}, nil, nil
		},
	)
	resp, err = tool.searchLogs(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"Region": "nyc3", "From": "now-1h", "To": "now", "Query": "timeout", "Limit": float64(50),
	}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestAlertPolicyDeprecationNotice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMonitoring := NewMockMonitoringService(ctrl)
	mockMonitoring.EXPECT().GetAlertPolicy(gomock.Any(), "id1").Return(&godo.AlertPolicy{UUID: "id1"}, nil, nil)
	tool := setupAlertPolicyToolWithMock(mockMonitoring)
	resp, err := tool.getAlertPolicy(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"UUID": "id1"}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.GreaterOrEqual(t, len(resp.Content), 2)
	notice, ok := resp.Content[1].(mcp.TextContent)
	require.True(t, ok)
	require.Contains(t, notice.Text, "DEPRECATED:")
	require.Contains(t, notice.Text, "insights-alert-rule-")
}

func TestAlertPolicyAndRuleToolTitles(t *testing.T) {
	clientFn := func(context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test-token"), nil
	}
	for _, st := range NewAlertPolicyTool(clientFn).Tools() {
		require.Contains(t, st.Tool.Title, "Deprecated")
		require.Contains(t, st.Tool.Title, "v1")
		require.Equal(t, st.Tool.Title, st.Tool.Annotations.Title)
	}
	for _, st := range NewAlertRuleTool(clientFn).Tools() {
		require.Contains(t, st.Tool.Title, "Insights v2")
		require.Equal(t, st.Tool.Title, st.Tool.Annotations.Title)
	}
}
