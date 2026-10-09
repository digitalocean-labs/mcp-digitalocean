package harnessruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func newTestTool(t *testing.T, handler http.HandlerFunc) *Tool {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return NewTool(func(context.Context) (*godo.Client, error) {
		c := godo.NewFromToken("test")
		c.BaseURL = base
		return c, nil
	})
}

func callText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	return text.Text
}

func request(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

func TestToolsRegistered(t *testing.T) {
	names := []string{}
	for _, st := range NewTool(nil).Tools() {
		names = append(names, st.Tool.Name)
	}
	require.Equal(t, []string{"harness-runtime-list-agent-configs"}, names)
}

func TestListAgentConfigs(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"configs":[{"id":"019fb39c-14d9-7080-933e-b9b90e25acda","name":"support-agent","agentspec_schema_version":"agents.digitalocean.com/v1alpha1","content_hash":"abc","created_by":"user-1","created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-01T12:00:00Z"}],"next_page_token":"tok2"}`))
	})

	res, err := tool.listAgentConfigs(context.Background(), request(map[string]any{
		"PageSize":  float64(10),
		"PageToken": "tok1",
		"Search":    "support",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/configs", gotPath)
	require.Equal(t, "10", gotQuery.Get("page_size"))
	require.Equal(t, "tok1", gotQuery.Get("page_token"))
	require.Equal(t, "support", gotQuery.Get("search"))
	text := callText(t, res)
	require.Contains(t, text, "support-agent")
	require.Contains(t, text, "019fb39c-14d9-7080-933e-b9b90e25acda")
	require.Contains(t, text, "tok2")
}

func TestListAgentConfigsNoArgsOmitsQuery(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listAgentConfigs(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"configs":\s*\[\]`, callText(t, res))
}

func TestListAgentConfigsValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"page size too small": {"PageSize": float64(0)},
		"page size too large": {"PageSize": float64(201)},
		"search too long":     {"Search": string(make([]byte, 65))},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listAgentConfigs(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListAgentConfigsAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"id":"unauthorized","message":"Unable to authenticate you"}`))
	})

	res, err := tool.listAgentConfigs(context.Background(), request(nil))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list agent configs")
}
