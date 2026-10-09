package signals

import (
	"context"
	"errors"
	"testing"

	"mcp-digitalocean/pkg/registry/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupToolWithMock(mock godo.SignalsService) *Tool {
	return NewTool(func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Signals: mock}, nil
	})
}

func TestToolsRegistered(t *testing.T) {
	names := make([]string, 0)
	for _, st := range NewTool(func(context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test"), nil
	}).Tools() {
		names = append(names, st.Tool.Name)
	}
	require.Equal(t, []string{
		"signals-list-consents",
		"signals-get-agent-consent",
		"signals-set-agent-consent",
		"signals-list-agent-sessions",
		"signals-list-session-segments",
		"signals-list-session-dialogues",
		"signals-create-export",
		"signals-list-exports",
		"signals-get-export",
		"signals-get-export-download",
		"signals-get-export-options",
	}, names)
}

func TestListConsents(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockSignalsService(ctrl)
	tool := setupToolWithMock(mock)

	mock.EXPECT().ListConsents(gomock.Any()).Return(&godo.SignalsListConsentsResponse{
		TeamID: 123,
		Consents: []godo.SignalsConsentRecord{
			{ID: 1, TeamID: 123, AgentID: "agent-1", Enabled: true},
		},
	}, &godo.Response{}, nil)

	res, err := tool.listConsents(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Contains(t, callText(res), "agent-1")
}

func TestCreateExport(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockSignalsService(ctrl)
	tool := setupToolWithMock(mock)

	t.Run("missing AgentID", func(t *testing.T) {
		res, err := tool.createExport(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{}},
		})
		require.NoError(t, err)
		require.True(t, res.IsError)
	})

	t.Run("success", func(t *testing.T) {
		start := int64(1727740800)
		mock.EXPECT().CreateExport(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, req *godo.SignalsCreateExportRequest) (*godo.SignalsExportJob, *godo.Response, error) {
				require.Equal(t, "agent-1", req.AgentID)
				require.Equal(t, []string{"Looping"}, req.SignalType)
				require.Equal(t, &start, req.StartTime)
				return &godo.SignalsExportJob{ExportID: "export-1", AgentID: "agent-1", Status: "queued"}, &godo.Response{}, nil
			},
		)
		res, err := tool.createExport(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{
				"AgentID":    "agent-1",
				"SignalType": []any{"Looping"},
				"StartTime":  float64(1727740800),
			}},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)
		require.Contains(t, callText(res), "export-1")
	})
}

func TestGetAgentConsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockSignalsService(ctrl)
	tool := setupToolWithMock(mock)

	t.Run("missing AgentID", func(t *testing.T) {
		res, err := tool.getAgentConsent(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{}},
		})
		require.NoError(t, err)
		require.True(t, res.IsError)
	})

	t.Run("success", func(t *testing.T) {
		mock.EXPECT().GetAgentConsent(gomock.Any(), "agent-1").Return(&godo.SignalsAgentConsent{
			TeamID: 123, AgentID: "agent-1", Enabled: true, Allowed: true,
		}, &godo.Response{}, nil)
		res, err := tool.getAgentConsent(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{"AgentID": "agent-1"}},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)
		require.Contains(t, callText(res), "agent-1")
	})
}

func TestSetAgentConsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockSignalsService(ctrl)
	tool := setupToolWithMock(mock)

	t.Run("missing Enabled", func(t *testing.T) {
		res, err := tool.setAgentConsent(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{"AgentID": "agent-1"}},
		})
		require.NoError(t, err)
		require.True(t, res.IsError)
	})

	t.Run("success", func(t *testing.T) {
		mock.EXPECT().SetAgentConsent(gomock.Any(), "agent-1", true).Return(&godo.SignalsConsentRecord{
			ID: 1, TeamID: 123, AgentID: "agent-1", Enabled: true,
		}, &godo.Response{}, nil)
		res, err := tool.setAgentConsent(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{"AgentID": "agent-1", "Enabled": true}},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)
		require.Contains(t, callText(res), "agent-1")
	})
}

func TestListAgentSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockSignalsService(ctrl)
	tool := setupToolWithMock(mock)

	start := int64(1727740800)
	mock.EXPECT().ListAgentSessions(gomock.Any(), "agent-1", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, opts *godo.SignalsListAgentSessionsOptions) (*godo.SignalsListAgentSessionsResponse, *godo.Response, error) {
			require.Equal(t, 20, opts.Limit)
			require.Equal(t, "c1", opts.After)
			require.Equal(t, &start, opts.StartTime)
			require.Equal(t, []string{"Looping"}, opts.SignalType)
			return &godo.SignalsListAgentSessionsResponse{
				Edges: []godo.SignalsSessionEdge{
					{Cursor: "c1", Node: godo.SignalsSession{SessionID: "sess-1", TotalTurns: 12}},
				},
				PageInfo: godo.SignalsPageInfo{HasNextPage: true, EndCursor: "c1"},
			}, &godo.Response{}, nil
		},
	)

	res, err := tool.listAgentSessions(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{
			"AgentID":    "agent-1",
			"Limit":      float64(20),
			"After":      "c1",
			"StartTime":  float64(1727740800),
			"SignalType": []any{"Looping"},
		}},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Contains(t, callText(res), "sess-1")
}

func TestGetExportOptionsAPIError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockSignalsService(ctrl)
	tool := setupToolWithMock(mock)
	mock.EXPECT().GetExportOptions(gomock.Any()).Return(nil, nil, errors.New("403"))
	res, err := tool.getExportOptions(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.True(t, res.IsError)
}

func TestClientError(t *testing.T) {
	tool := NewTool(func(ctx context.Context) (*godo.Client, error) {
		return nil, errors.New("auth failed")
	})
	_, err := tool.getAgentConsent(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"AgentID": "agent-1"}},
	})
	require.Error(t, err)
}

func callText(res *mcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

var expectedAnnotations = map[string]struct {
	readOnly, destructive, idempotent, openWorld bool
	operation                                    common.Operation
	risk                                         common.Risk
	parallelizable, streamingSafe                bool
}{
	"signals-list-consents":          {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"signals-get-agent-consent":      {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"signals-set-agent-consent":      {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"signals-list-agent-sessions":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"signals-list-session-segments":  {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"signals-list-session-dialogues": {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"signals-create-export":          {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"signals-list-exports":           {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"signals-get-export":             {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"signals-get-export-download":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"signals-get-export-options":     {true, false, true, false, common.OpRead, common.RiskLow, false, false},
}

func TestToolAnnotations(t *testing.T) {
	all := NewTool(func(context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test-token"), nil
	}).Tools()
	if len(all) != len(expectedAnnotations) {
		t.Fatalf("tool count mismatch: registered=%d, expected=%d", len(all), len(expectedAnnotations))
	}
	for _, st := range all {
		want, ok := expectedAnnotations[st.Tool.Name]
		if !ok {
			t.Errorf("unexpected tool %q", st.Tool.Name)
			continue
		}
		a := st.Tool.Annotations
		if a.ReadOnlyHint == nil || a.DestructiveHint == nil || a.IdempotentHint == nil || a.OpenWorldHint == nil {
			t.Errorf("tool %q is missing one or more annotation hints", st.Tool.Name)
			continue
		}
		if got := *a.ReadOnlyHint; got != want.readOnly {
			t.Errorf("tool %q readOnlyHint = %v, want %v", st.Tool.Name, got, want.readOnly)
		}
		if got := *a.DestructiveHint; got != want.destructive {
			t.Errorf("tool %q destructiveHint = %v, want %v", st.Tool.Name, got, want.destructive)
		}
		if got := *a.IdempotentHint; got != want.idempotent {
			t.Errorf("tool %q idempotentHint = %v, want %v", st.Tool.Name, got, want.idempotent)
		}
		if got := *a.OpenWorldHint; got != want.openWorld {
			t.Errorf("tool %q openWorldHint = %v, want %v", st.Tool.Name, got, want.openWorld)
		}
		if st.Tool.Meta == nil || st.Tool.Meta.AdditionalFields == nil {
			t.Errorf("tool %q is missing _meta", st.Tool.Name)
			continue
		}
		reg, ok := st.Tool.Meta.AdditionalFields[common.RegistryMetaKey].(map[string]any)
		if !ok {
			t.Errorf("tool %q is missing registry metadata", st.Tool.Name)
			continue
		}
		if got, _ := reg["operation"].(string); got != string(want.operation) {
			t.Errorf("tool %q operation = %q, want %q", st.Tool.Name, got, want.operation)
		}
		if got, _ := reg["risk"].(string); got != string(want.risk) {
			t.Errorf("tool %q risk = %q, want %q", st.Tool.Name, got, want.risk)
		}
	}
}
