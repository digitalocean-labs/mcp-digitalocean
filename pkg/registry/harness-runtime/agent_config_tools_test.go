package harnessruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
	require.Equal(t, []string{
		"harness-runtime-list-agent-configs",
		"harness-runtime-get-agent-config",
		"harness-runtime-create-agent-config",
		"harness-runtime-delete-agent-config",
		"harness-runtime-clone-agent-config",
		"harness-runtime-list-config-sessions",
		"harness-runtime-create-session",
		"harness-runtime-get-session",
		"harness-runtime-list-sessions",
		"harness-runtime-destroy-session",
		"harness-runtime-send-input",
		"harness-runtime-pause-session",
		"harness-runtime-resume-session",
		"harness-runtime-patch-session",
		"harness-runtime-resolve-hitl",
		"harness-runtime-list-sandbox-sizes",
		"harness-runtime-validate-policy",
	}, names)
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
		"page size fraction":  {"PageSize": 1.9},
		"search too long":     {"Search": string(make([]byte, 65))},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listAgentConfigs(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListAgentConfigsSearchCountsRunes(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	// 64 multi-byte runes is valid even though it is well over 64 bytes.
	res, err := tool.listAgentConfigs(context.Background(), request(map[string]any{"Search": strings.Repeat("é", 64)}))
	require.NoError(t, err)
	require.False(t, res.IsError)
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

const testConfigID = "019fb39c-14d9-7080-933e-b9b90e25acda"

func TestGetAgentConfig(t *testing.T) {
	var gotPath string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"config":{"id":"` + testConfigID + `","name":"support-agent","agentspec_schema_version":"agents.digitalocean.com/flat.v1","content_hash":"abc","created_by":"user-1","created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-01T12:00:00Z","manifest":{"spec":{"runtime":"python"}},"credentials":[{"name":"OPENAI_API_KEY","source":"tenantSecret","provider":"openai"}]}}`))
	})

	res, err := tool.getAgentConfig(context.Background(), request(map[string]any{"ConfigID": testConfigID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/configs/"+testConfigID, gotPath)
	text := callText(t, res)
	require.Contains(t, text, "support-agent")
	require.Contains(t, text, "python")
	require.Contains(t, text, "OPENAI_API_KEY")
}

func TestGetAgentConfigValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":   {},
		"not uuid":  {"ConfigID": "support-agent"},
		"traversal": {"ConfigID": "../sessions"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getAgentConfig(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetAgentConfigNotFound(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"config not found"}`))
	})

	res, err := tool.getAgentConfig(context.Background(), request(map[string]any{"ConfigID": testConfigID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get agent config")
}

// ---- CreateAgentConfig ----

const testManifestJSON = `{"spec":{"runtime":"python3.11","entrypoint":"agent.py","system_prompt":"You are a helpful agent."}}`

func testManifest() map[string]any {
	var m map[string]any
	_ = json.Unmarshal([]byte(testManifestJSON), &m)
	return m
}

func configJSON(id, name string) string {
	return `{"id":"` + id + `","name":"` + name + `","agentspec_schema_version":"agents.digitalocean.com/flat.v1","content_hash":"abc","created_by":"user-1","created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-01T12:00:00Z","manifest":` + testManifestJSON + `}`
}

func TestCreateAgentConfig(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"config":` + configJSON(testConfigID, "my-new-agent") + `}`))
	})

	res, err := tool.createAgentConfig(context.Background(), request(map[string]any{
		"Name":     "my-new-agent",
		"Manifest": testManifest(),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/configs", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, gotContentType, "application/json")

	require.Equal(t, "my-new-agent", gotBody["name"])
	manifestYAML, ok := gotBody["manifest_yaml"].(string)
	require.True(t, ok, "request body should have a 'manifest_yaml' string key")
	require.Contains(t, manifestYAML, "python3.11")
	require.Contains(t, manifestYAML, "agent.py")

	text := callText(t, res)
	require.Contains(t, text, testConfigID)
	require.Contains(t, text, "my-new-agent")
}

func TestCreateAgentConfigWithManageKey(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"config":` + configJSON(testConfigID, "my-agent") + `}`))
	})

	res, err := tool.createAgentConfig(context.Background(), request(map[string]any{
		"Name":                "my-agent",
		"Manifest":            testManifest(),
		"ManageModelAccessKey": true,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, true, gotBody["manage_model_access_key"])
}

func TestCreateAgentConfigValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing name and manifest": {},
		"missing name":              {"Manifest": testManifest()},
		"missing manifest":          {"Name": "my-agent"},
		"empty manifest object":     {"Name": "my-agent", "Manifest": map[string]any{}},
		"empty manifest string":     {"Name": "my-agent", "Manifest": ""},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.createAgentConfig(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCreateAgentConfigAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"id":"unprocessable_entity","message":"invalid manifest"}`))
	})

	res, err := tool.createAgentConfig(context.Background(), request(map[string]any{
		"Name":     "my-agent",
		"Manifest": testManifest(),
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to create agent config")
}

// ---- DeleteAgentConfig ----

func TestDeleteAgentConfig(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})

	res, err := tool.deleteAgentConfig(context.Background(), request(map[string]any{"ConfigID": testConfigID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/configs/"+testConfigID, gotPath)
	require.Equal(t, http.MethodDelete, gotMethod)
	require.Contains(t, callText(t, res), testConfigID)
	require.Contains(t, callText(t, res), "deleted")
}

func TestDeleteAgentConfigValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"ConfigID": "support-agent"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.deleteAgentConfig(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestDeleteAgentConfigAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"config not found"}`))
	})

	res, err := tool.deleteAgentConfig(context.Background(), request(map[string]any{"ConfigID": testConfigID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to delete agent config")
}

// ---- CloneAgentConfig ----

const testCloneConfigID = "119fb39c-14d9-7080-933e-b9b90e25acdb"

func TestCloneAgentConfig(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"config":` + configJSON(testCloneConfigID, "support-agent-copy") + `}`))
	})

	res, err := tool.cloneAgentConfig(context.Background(), request(map[string]any{
		"ConfigID": testConfigID,
		"Name":     "support-agent-copy",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/configs/"+testConfigID+"/clone", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, gotContentType, "application/json")
	require.Equal(t, "support-agent-copy", gotBody["name"])

	text := callText(t, res)
	require.Contains(t, text, testCloneConfigID)
	require.Contains(t, text, "support-agent-copy")
}

func TestCloneAgentConfigNoName(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"config":` + configJSON(testCloneConfigID, "support-agent-clone") + `}`))
	})

	res, err := tool.cloneAgentConfig(context.Background(), request(map[string]any{
		"ConfigID": testConfigID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	// No name in body when not provided.
	_, hasName := gotBody["name"]
	require.False(t, hasName)
	require.Contains(t, callText(t, res), testCloneConfigID)
}

func TestCloneAgentConfigValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"ConfigID": "support-agent"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.cloneAgentConfig(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCloneAgentConfigAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"config not found"}`))
	})

	res, err := tool.cloneAgentConfig(context.Background(), request(map[string]any{"ConfigID": testConfigID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to clone agent config")
}

// ---- ListConfigSessions ----

func TestListConfigSessions(t *testing.T) {
	var gotPath string
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sessions":[{"session_id":"` + testSessionID + `","name":"agent-1","status":"SESSION_STATUS_READY","config_id":"` + testConfigID + `","created_at":"2026-08-01T12:00:00Z"}],"next_page_token":"tok2"}`))
	})

	res, err := tool.listConfigSessions(context.Background(), request(map[string]any{
		"ConfigID":  testConfigID,
		"PageSize":  float64(5),
		"PageToken": "tok1",
		"Status":    "SESSION_STATUS_READY",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/configs/"+testConfigID+"/sessions", gotPath)
	require.Contains(t, gotRawQuery, "page_size=5")
	require.Contains(t, gotRawQuery, "page_token=tok1")
	require.Contains(t, gotRawQuery, "status=SESSION_STATUS_READY")

	text := callText(t, res)
	require.Contains(t, text, testSessionID)
	require.Contains(t, text, "tok2")
}

func TestListConfigSessionsNoQueryParams(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listConfigSessions(context.Background(), request(map[string]any{
		"ConfigID": testConfigID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"sessions":\s*\[\]`, callText(t, res))
}

func TestListConfigSessionsValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing config id":   {},
		"not uuid":            {"ConfigID": "support-agent"},
		"page size too small": {"ConfigID": testConfigID, "PageSize": float64(0)},
		"page size too large": {"ConfigID": testConfigID, "PageSize": float64(201)},
		"page size fraction":  {"ConfigID": testConfigID, "PageSize": 1.7},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listConfigSessions(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListConfigSessionsAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"config not found"}`))
	})

	res, err := tool.listConfigSessions(context.Background(), request(map[string]any{"ConfigID": testConfigID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list config sessions")
}
