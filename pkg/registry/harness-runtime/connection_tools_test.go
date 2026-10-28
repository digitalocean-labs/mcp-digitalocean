package harnessruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testConnectionID = "069fb39c-14d9-7080-933e-b9b90e25acdf"
	testProvider     = "github"
)

func connectionJSON(id, provider, status string) string {
	return `{"id":"` + id + `","provider":"` + provider + `","user_id":"actor-1","status":"` + status + `","deletion_protected":false,"created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-01T12:00:00Z"}`
}

// ---- ListConnections ----

func TestListConnections(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connections":[` + connectionJSON(testConnectionID, testProvider, "active") + `],"pagination":{"page":1,"per_page":20,"total":1}}`))
	})

	res, err := tool.listConnections(context.Background(), request(map[string]any{
		"Provider": testProvider,
		"ActorID":  "actor-1",
		"Status":   "active",
		"Page":     float64(1),
		"PerPage":  float64(20),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/auth/github/connections", gotPath)
	require.Equal(t, "actor-1", gotQuery.Get("user_id"))
	require.Equal(t, "active", gotQuery.Get("status"))
	require.Equal(t, "1", gotQuery.Get("page"))
	require.Equal(t, "20", gotQuery.Get("per_page"))
	text := callText(t, res)
	require.Contains(t, text, testConnectionID)
	require.Contains(t, text, "github")
}

func TestListConnectionsMinimalArgs(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connections":[],"pagination":{"page":1,"per_page":20,"total":0}}`))
	})

	res, err := tool.listConnections(context.Background(), request(map[string]any{
		"Provider": testProvider,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
}

func TestListConnectionsValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing provider":     {},
		"invalid provider":     {"Provider": "bitbucket"},
		"page not integer":     {"Provider": testProvider, "Page": 1.5},
		"page zero":            {"Provider": testProvider, "Page": float64(0)},
		"per_page too large":   {"Provider": testProvider, "PerPage": float64(101)},
		"per_page zero":        {"Provider": testProvider, "PerPage": float64(0)},
		"per_page not integer": {"Provider": testProvider, "PerPage": 1.5},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listConnections(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListConnectionsAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"id":"unauthorized","message":"Unable to authenticate you"}`))
	})

	res, err := tool.listConnections(context.Background(), request(map[string]any{
		"Provider": testProvider,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list connections")
}

// ---- CreateConnection ----

func TestCreateConnection(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connection":` + connectionJSON(testConnectionID, testProvider, "active") + `}`))
	})

	res, err := tool.createConnection(context.Background(), request(map[string]any{
		"Provider": testProvider,
		"ActorID":  "actor-1",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/auth/github/connections", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, gotContentType, "application/json")
	require.Equal(t, "actor-1", gotBody["user_id"])
	text := callText(t, res)
	require.Contains(t, text, testConnectionID)
}

func TestCreateConnectionWithScopes(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connection":` + connectionJSON(testConnectionID, testProvider, "pending") + `,"authorization":{"status":"pending","connect_url":"https://github.com/login/device","verification_code":"ABCD-1234"}}`))
	})

	res, err := tool.createConnection(context.Background(), request(map[string]any{
		"Provider": testProvider,
		"ActorID":  "actor-1",
		"Scopes":   []any{"repo", "read:org"},
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "actor-1", gotBody["user_id"])
	scopes, ok := gotBody["scopes"].([]any)
	require.True(t, ok)
	require.Len(t, scopes, 2)
	text := callText(t, res)
	require.Contains(t, text, "pending")
	require.Contains(t, text, "connect_url")
}

func TestCreateConnectionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing provider": {},
		"missing actor":    {"Provider": testProvider},
		"invalid provider": {"Provider": "bitbucket", "ActorID": "actor-1"},
		"empty actor":      {"Provider": testProvider, "ActorID": ""},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.createConnection(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCreateConnectionAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"connection already exists"}`))
	})

	res, err := tool.createConnection(context.Background(), request(map[string]any{
		"Provider": testProvider,
		"ActorID":  "actor-1",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to create connection")
}

// ---- GetConnection ----

func TestGetConnection(t *testing.T) {
	var gotPath string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connection":` + connectionJSON(testConnectionID, testProvider, "active") + `}`))
	})

	res, err := tool.getConnection(context.Background(), request(map[string]any{
		"Provider":     testProvider,
		"ConnectionID": testConnectionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/auth/github/connections/"+testConnectionID, gotPath)
	text := callText(t, res)
	require.Contains(t, text, testConnectionID)
	require.Contains(t, text, "active")
}

func TestGetConnectionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing provider":        {},
		"invalid provider":        {"Provider": "bitbucket", "ConnectionID": testConnectionID},
		"missing connection id":   {"Provider": testProvider},
		"empty connection id":     {"Provider": testProvider, "ConnectionID": ""},
		"invalid connection uuid": {"Provider": testProvider, "ConnectionID": "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getConnection(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetConnectionNotFound(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"connection not found"}`))
	})

	res, err := tool.getConnection(context.Background(), request(map[string]any{
		"Provider":     testProvider,
		"ConnectionID": testConnectionID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get connection")
}

// ---- DeleteConnection ----

func TestDeleteConnection(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connection":` + connectionJSON(testConnectionID, testProvider, "revoked") + `}`))
	})

	res, err := tool.deleteConnection(context.Background(), request(map[string]any{
		"Provider":     testProvider,
		"ConnectionID": testConnectionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/auth/github/connections/"+testConnectionID, gotPath)
	require.Equal(t, http.MethodDelete, gotMethod)
	text := callText(t, res)
	require.Contains(t, text, testConnectionID)
	require.Contains(t, text, "revoked")
}

func TestDeleteConnectionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing provider":        {},
		"invalid provider":        {"Provider": "bitbucket", "ConnectionID": testConnectionID},
		"missing connection id":   {"Provider": testProvider},
		"invalid connection uuid": {"Provider": testProvider, "ConnectionID": "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.deleteConnection(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestDeleteConnectionAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"connection is deletion protected"}`))
	})

	res, err := tool.deleteConnection(context.Background(), request(map[string]any{
		"Provider":     testProvider,
		"ConnectionID": testConnectionID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to delete connection")
}

// ---- StartProviderAuth ----

func TestStartProviderAuth(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"provider":"github","status":"pending","connect_url":"https://github.com/login/device","poll_url":"https://api.github.com/poll","verification_code":"ABCD-1234","expires_at":"2026-08-01T12:15:00Z"}`))
	})

	res, err := tool.startProviderAuth(context.Background(), request(map[string]any{
		"Provider": testProvider,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/auth/github", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	text := callText(t, res)
	require.Contains(t, text, "pending")
	require.Contains(t, text, "connect_url")
	require.Contains(t, text, "ABCD-1234")
}

func TestStartProviderAuthAlreadyConnected(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"provider":"github","status":"success"}`))
	})

	res, err := tool.startProviderAuth(context.Background(), request(map[string]any{
		"Provider": testProvider,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	text := callText(t, res)
	require.Contains(t, text, "success")
}

func TestStartProviderAuthValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing provider": {},
		"invalid provider": {"Provider": "bitbucket"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.startProviderAuth(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestStartProviderAuthAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"id":"server_error","message":"internal error"}`))
	})

	res, err := tool.startProviderAuth(context.Background(), request(map[string]any{
		"Provider": testProvider,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to start provider auth")
}

// ---- PollProviderAuth ----

func TestPollProviderAuth(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"provider":"github","status":"success"}`))
	})

	res, err := tool.pollProviderAuth(context.Background(), request(map[string]any{
		"Provider": testProvider,
		"PollURL":  "https://api.github.com/poll",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/auth/github/poll", gotPath)
	require.Equal(t, "https://api.github.com/poll", gotQuery.Get("poll_url"))
	text := callText(t, res)
	require.Contains(t, text, "success")
}

func TestPollProviderAuthPending(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"provider":"github","status":"pending","expires_at":"2026-08-01T12:15:00Z"}`))
	})

	res, err := tool.pollProviderAuth(context.Background(), request(map[string]any{
		"Provider": testProvider,
		"PollURL":  "https://api.github.com/poll",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	text := callText(t, res)
	require.Contains(t, text, "pending")
}

func TestPollProviderAuthValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing provider": {},
		"invalid provider": {"Provider": "bitbucket", "PollURL": "https://example.com"},
		"missing poll url": {"Provider": testProvider},
		"empty poll url":   {"Provider": testProvider, "PollURL": ""},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.pollProviderAuth(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestPollProviderAuthAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"no pending auth"}`))
	})

	res, err := tool.pollProviderAuth(context.Background(), request(map[string]any{
		"Provider": testProvider,
		"PollURL":  "https://api.github.com/poll",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to poll provider auth")
}

// ---- ListWebhookProviders ----

func TestListWebhookProviders(t *testing.T) {
	var gotPath string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"providers":[{"key":"github","display_name":"GitHub","description":"GitHub webhooks","docs_url":"https://docs.github.com","paste_hint":"Paste your webhook URL","signature":{"header":"X-Hub-Signature-256","scheme":"hmac-sha256"}}]}`))
	})

	res, err := tool.listWebhookProviders(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/webhook-providers", gotPath)
	text := callText(t, res)
	require.Contains(t, text, "github")
	require.Contains(t, text, "GitHub")
	require.Contains(t, text, "X-Hub-Signature-256")
}

func TestListWebhookProvidersEmpty(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listWebhookProviders(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Regexp(t, `"providers":\s*\[\]`, callText(t, res))
}

func TestListWebhookProvidersAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"id":"server_error","message":"internal error"}`))
	})

	res, err := tool.listWebhookProviders(context.Background(), request(nil))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list webhook providers")
}

// ---- Tools registration (updated list) ----

func TestToolsRegisteredWithConnections(t *testing.T) {
	names := []string{}
	for _, st := range NewTool(nil).Tools() {
		names = append(names, st.Tool.Name)
	}
	// Verify new tools are registered at the end.
	require.Contains(t, names, "harness-runtime-list-connections")
	require.Contains(t, names, "harness-runtime-create-connection")
	require.Contains(t, names, "harness-runtime-get-connection")
	require.Contains(t, names, "harness-runtime-delete-connection")
	require.Contains(t, names, "harness-runtime-start-provider-auth")
	require.Contains(t, names, "harness-runtime-poll-provider-auth")
	require.Contains(t, names, "harness-runtime-list-webhook-providers")
}
