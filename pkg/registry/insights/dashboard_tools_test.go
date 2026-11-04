package insights

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testDashboardSpec = `{"kind":"Dashboard","apiVersion":"perses.dev/v1alpha1","spec":{"display":{"name":"CPU"},"duration":"1h","layouts":[],"panels":{}}}`

func setupDashboardTool(t *testing.T) (*MockInsightsService, *DashboardTool) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mock := NewMockInsightsService(ctrl)
	tool := NewDashboardTool(func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Insights: mock}, nil
	})
	return mock, tool
}

func callArgs(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

func TestDashboardTool_get(t *testing.T) {
	mock, tool := setupDashboardTool(t)

	resp, err := tool.get(context.Background(), callArgs(map[string]any{}))
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().GetDashboard(gomock.Any(), "d1").Return(nil, nil, errors.New("api error"))
	resp, err = tool.get(context.Background(), callArgs(map[string]any{"UUID": "d1"}))
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().GetDashboard(gomock.Any(), "d1").Return(&godo.Dashboard{UUID: "d1", Name: "CPU", Spec: json.RawMessage(testDashboardSpec)}, nil, nil)
	resp, err = tool.get(context.Background(), callArgs(map[string]any{"UUID": "d1"}))
	require.NoError(t, err)
	require.False(t, resp.IsError)

	// spec travels as a JSON-encoded string, exactly as the API emits it.
	var out struct {
		UUID string `json:"uuid"`
		Spec string `json:"spec"`
	}
	require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
	require.Equal(t, "d1", out.UUID)
	require.JSONEq(t, testDashboardSpec, out.Spec)
}

func TestDashboardTool_list(t *testing.T) {
	mock, tool := setupDashboardTool(t)

	mock.EXPECT().ListDashboards(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, opt *godo.DashboardListOptions) (*godo.DashboardList, *godo.Response, error) {
			require.Equal(t, 2, opt.Page)
			require.Equal(t, 50, opt.PerPage)
			require.Equal(t, godo.InsightsDashboardSortName, opt.Sort)
			require.Equal(t, godo.InsightsDashboardSortDescending, opt.SortDirection)
			return &godo.DashboardList{
				Dashboards: []godo.Dashboard{{UUID: "d1"}},
				Pagination: &godo.DashboardsPagination{Page: 2, Pages: 3, PerPage: 50, Total: 120},
			}, nil, nil
		},
	)
	resp, err := tool.list(context.Background(), callArgs(map[string]any{
		"Page": float64(2), "PerPage": float64(50), "Sort": "name", "SortDirection": "desc",
	}))
	require.NoError(t, err)
	require.False(t, resp.IsError)

	// Defaults apply when pagination arguments are omitted.
	mock.EXPECT().ListDashboards(gomock.Any(), &godo.DashboardListOptions{
		Page: defaultAlertPoliciesPage, PerPage: defaultAlertPoliciesPageSize,
	}).Return(&godo.DashboardList{}, nil, nil)
	resp, err = tool.list(context.Background(), callArgs(map[string]any{}))
	require.NoError(t, err)
	require.False(t, resp.IsError)

	mock.EXPECT().ListDashboards(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
	resp, err = tool.list(context.Background(), callArgs(map[string]any{}))
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestDashboardTool_create(t *testing.T) {
	mock, tool := setupDashboardTool(t)

	// Spec shape is checked before any API call; the mock has no expectation.
	for name, spec := range map[string]any{
		"missing":        nil,
		"empty":          "",
		"not json":       "{kind: Dashboard}",
		"array":          `[{"kind":"Dashboard"}]`,
		"double-encoded": `"{\"kind\":\"Dashboard\"}"`,
	} {
		args := map[string]any{}
		if spec != nil {
			args["Spec"] = spec
		}
		resp, err := tool.create(context.Background(), callArgs(args))
		require.NoError(t, err, name)
		require.True(t, resp.IsError, name)
		require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "insights-dashboard-get-schema", name)
	}

	mock.EXPECT().CreateDashboard(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *godo.DashboardRequest) (*godo.Dashboard, *godo.Response, error) {
			require.Equal(t, testDashboardSpec, string(req.Spec), "spec must reach godo byte-for-byte")
			return &godo.Dashboard{UUID: "d1", Name: "CPU", Spec: req.Spec}, nil, nil
		},
	)
	resp, err := tool.create(context.Background(), callArgs(map[string]any{"Spec": testDashboardSpec}))
	require.NoError(t, err)
	require.False(t, resp.IsError)

	// Surrounding whitespace is tolerated and trimmed.
	mock.EXPECT().CreateDashboard(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *godo.DashboardRequest) (*godo.Dashboard, *godo.Response, error) {
			require.Equal(t, testDashboardSpec, string(req.Spec))
			return &godo.Dashboard{UUID: "d1"}, nil, nil
		},
	)
	resp, err = tool.create(context.Background(), callArgs(map[string]any{"Spec": "\n  " + testDashboardSpec + "\n"}))
	require.NoError(t, err)
	require.False(t, resp.IsError)

	mock.EXPECT().CreateDashboard(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("409"))
	resp, err = tool.create(context.Background(), callArgs(map[string]any{"Spec": testDashboardSpec}))
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestDashboardTool_update(t *testing.T) {
	mock, tool := setupDashboardTool(t)

	resp, err := tool.update(context.Background(), callArgs(map[string]any{"Spec": testDashboardSpec}))
	require.NoError(t, err)
	require.True(t, resp.IsError, "UUID is required")

	resp, err = tool.update(context.Background(), callArgs(map[string]any{"UUID": "d1"}))
	require.NoError(t, err)
	require.True(t, resp.IsError, "Spec is required")

	mock.EXPECT().UpdateDashboard(gomock.Any(), "d1", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, req *godo.DashboardRequest) (*godo.Dashboard, *godo.Response, error) {
			require.Equal(t, testDashboardSpec, string(req.Spec))
			return &godo.Dashboard{UUID: "d1", Spec: req.Spec}, nil, nil
		},
	)
	resp, err = tool.update(context.Background(), callArgs(map[string]any{"UUID": "d1", "Spec": testDashboardSpec}))
	require.NoError(t, err)
	require.False(t, resp.IsError)

	mock.EXPECT().UpdateDashboard(gomock.Any(), "d1", gomock.Any()).Return(nil, nil, errors.New("api error"))
	resp, err = tool.update(context.Background(), callArgs(map[string]any{"UUID": "d1", "Spec": testDashboardSpec}))
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestDashboardTool_delete(t *testing.T) {
	mock, tool := setupDashboardTool(t)

	resp, err := tool.delete(context.Background(), callArgs(map[string]any{}))
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().DeleteDashboard(gomock.Any(), "d1").Return(nil, errors.New("api error"))
	resp, err = tool.delete(context.Background(), callArgs(map[string]any{"UUID": "d1"}))
	require.NoError(t, err)
	require.True(t, resp.IsError)

	mock.EXPECT().DeleteDashboard(gomock.Any(), "d1").Return(nil, nil)
	resp, err = tool.delete(context.Background(), callArgs(map[string]any{"UUID": "d1"}))
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Equal(t, "Dashboard deleted successfully", resp.Content[0].(mcp.TextContent).Text)
}

func TestDashboardTool_getSchema(t *testing.T) {
	mock, tool := setupDashboardTool(t)

	mock.EXPECT().GetDashboardSchema(gomock.Any()).Return(nil, nil, errors.New("api error"))
	resp, err := tool.getSchema(context.Background(), callArgs(nil))
	require.NoError(t, err)
	require.True(t, resp.IsError)

	// Compact upstream bytes with deliberately non-alphabetical keys: the text
	// half is indented, the structured half is the upstream document unchanged.
	raw := json.RawMessage(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"kind":{"const":"Dashboard"}},"required":["kind"]}`)
	mock.EXPECT().GetDashboardSchema(gomock.Any()).Return(raw, nil, nil)
	resp, err = tool.getSchema(context.Background(), callArgs(nil))
	require.NoError(t, err)
	require.False(t, resp.IsError)

	text := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, text, "\n  \"type\": \"object\"", "text is indented")
	require.JSONEq(t, string(raw), text)

	structured, ok := resp.StructuredContent.(map[string]json.RawMessage)
	require.True(t, ok, "structuredContent is the schema envelope")
	require.Equal(t, string(raw), string(structured["schema"]), "upstream bytes pass through verbatim")
}

// TestDashboardStructuredOutputSatisfiesDeclaredSchema drives the two tools
// whose payloads diverge from plain godo reflection through a validating
// server: get (spec is a json:"-" field emitted as a string by a custom
// MarshalJSON) and get-schema (an arbitrary JSON document published under a
// map[string]any schema via ResultRaw). Handler unit tests cannot catch a
// payload the advertised outputSchema rejects; only the server can.
func TestDashboardStructuredOutputSatisfiesDeclaredSchema(t *testing.T) {
	mock, tool := setupDashboardTool(t)

	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	mock.EXPECT().GetDashboard(gomock.Any(), "d1").Return(&godo.Dashboard{
		UUID: "d1", Name: "CPU", Description: "cpu board",
		Spec: json.RawMessage(testDashboardSpec), CreatedAt: now, UpdatedAt: now,
	}, nil, nil)
	mock.EXPECT().ListDashboards(gomock.Any(), gomock.Any()).Return(&godo.DashboardList{
		Dashboards: []godo.Dashboard{{UUID: "d1", Spec: json.RawMessage(testDashboardSpec)}},
		Pagination: &godo.DashboardsPagination{Page: 1, Pages: 1, PerPage: 20, Total: 1},
	}, nil, nil)
	mock.EXPECT().GetDashboardSchema(gomock.Any()).Return(json.RawMessage(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","$defs":{"x":{"type":"string"}}}`), nil, nil)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(tool.Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	call := func(name, args string) map[string]json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(svr.HandleMessage(ctx,
			[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"`+name+`","arguments":`+args+`}}`)))
		require.NoError(t, err)
		var resp struct {
			Error  *struct{ Message string } `json:"error"`
			Result struct {
				IsError           bool                       `json:"isError"`
				StructuredContent map[string]json.RawMessage `json:"structuredContent"`
			} `json:"result"`
		}
		require.NoError(t, json.Unmarshal(raw, &resp))
		require.Nil(t, resp.Error, "%s: server rejected the call: %s", name, raw)
		require.False(t, resp.Result.IsError, "%s: tool returned an error result: %s", name, raw)
		require.NotEmpty(t, resp.Result.StructuredContent, "%s: structuredContent missing (schema validation dropped it?)", name)
		return resp.Result.StructuredContent
	}

	got := call("insights-dashboard-get", `{"UUID":"d1"}`)
	var dash struct {
		UUID string `json:"uuid"`
		Spec string `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(got["dashboard"], &dash))
	require.Equal(t, "d1", dash.UUID)
	require.JSONEq(t, testDashboardSpec, dash.Spec)

	got = call("insights-dashboard-list", `{}`)
	require.Contains(t, got, "dashboards")
	require.Contains(t, got, "pagination")

	got = call("insights-dashboard-get-schema", `{}`)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(got["schema"], &schema))
	require.Equal(t, "object", schema["type"])
	require.Contains(t, schema, "$defs")
}
