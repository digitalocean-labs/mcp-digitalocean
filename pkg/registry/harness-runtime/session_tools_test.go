package harnessruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testSessionID  = "029ab39c-14d9-7080-933e-b9b90e25acda"
	testSessionID2 = "039ab39c-14d9-7080-933e-b9b90e25acdb"
	testHITLReqID  = "hitl-req-001"
)

// sessionJSON returns a minimal Session JSON for test servers to emit.
func sessionJSON(sessionID, name, status string) string {
	return `{"session_id":"` + sessionID + `","name":"` + name + `","status":"` + status + `","config_id":"` + testConfigID + `","created_at":"2026-08-01T12:00:00Z"}`
}

// ---- CreateSession ----

func TestCreateSession(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session":` + sessionJSON(testSessionID, "my-agent", "SESSION_STATUS_PROVISIONING") + `}`))
	})

	res, err := tool.createSession(context.Background(), request(map[string]any{
		"ConfigID": testConfigID,
		"Name":     "my-agent",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, gotContentType, "application/json")
	require.Equal(t, testConfigID, gotBody["config_id"])
	require.Equal(t, "my-agent", gotBody["name"])

	text := callText(t, res)
	require.Contains(t, text, testSessionID)
	require.Contains(t, text, "my-agent")
	require.Contains(t, text, "SESSION_STATUS_PROVISIONING")
}

func TestCreateSessionWithOptionalParams(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session":` + sessionJSON(testSessionID, "ws-agent", "SESSION_STATUS_PROVISIONING") + `}`))
	})

	res, err := tool.createSession(context.Background(), request(map[string]any{
		"ConfigID":       testConfigID,
		"Name":           "ws-agent",
		"ResumeOnTopoff": true,
		"WorkspaceID":    "my-workspace",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, true, gotBody["resume_on_topoff"])
	require.Equal(t, "my-workspace", gotBody["workspace_id"])
}

func TestCreateSessionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing config id": {"Name": "my-agent"},
		"invalid config id": {"ConfigID": "not-a-uuid", "Name": "my-agent"},
		"missing name":      {"ConfigID": testConfigID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.createSession(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCreateSessionAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"session name already exists"}`))
	})

	res, err := tool.createSession(context.Background(), request(map[string]any{
		"ConfigID": testConfigID,
		"Name":     "existing-agent",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to create session")
}

// ---- GetSession ----

func TestGetSession(t *testing.T) {
	var gotPath string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session":` + sessionJSON(testSessionID, "support-agent", "SESSION_STATUS_READY") + `}`))
	})

	res, err := tool.getSession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID, gotPath)

	text := callText(t, res)
	require.Contains(t, text, testSessionID)
	require.Contains(t, text, "SESSION_STATUS_READY")
}

func TestGetSessionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":   {},
		"not uuid":  {"SessionID": "my-session"},
		"traversal": {"SessionID": "../configs"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getSession(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetSessionNotFound(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"session not found"}`))
	})

	res, err := tool.getSession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get session")
}

// ---- ListSessions ----

func TestListSessions(t *testing.T) {
	var gotPath string
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sessions":[` + sessionJSON(testSessionID, "agent-1", "SESSION_STATUS_READY") + `],"next_page_token":"tok2"}`))
	})

	res, err := tool.listSessions(context.Background(), request(map[string]any{
		"PageSize":  float64(10),
		"PageToken": "tok1",
		"Status":    "SESSION_STATUS_READY",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions", gotPath)
	require.Contains(t, gotRawQuery, "page_size=10")
	require.Contains(t, gotRawQuery, "page_token=tok1")
	require.Contains(t, gotRawQuery, "status=SESSION_STATUS_READY")

	text := callText(t, res)
	require.Contains(t, text, "agent-1")
	require.Contains(t, text, "tok2")
}

func TestListSessionsNoArgsOmitsQuery(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listSessions(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"sessions":\s*\[\]`, callText(t, res))
}

func TestListSessionsNameAndSearchFilters(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sessions":[]}`))
	})

	res, err := tool.listSessions(context.Background(), request(map[string]any{
		"Name":   "support-agent",
		"Search": "support",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Contains(t, gotRawQuery, "name=support-agent")
	require.Contains(t, gotRawQuery, "search=support")
}

func TestListSessionsValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"page size too small": {"PageSize": float64(0)},
		"page size too large": {"PageSize": float64(201)},
		"page size fraction":  {"PageSize": 1.5},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listSessions(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

// ---- DestroySession ----

func TestDestroySession(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})

	res, err := tool.destroySession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID, gotPath)
	require.Equal(t, http.MethodDelete, gotMethod)
	require.Contains(t, callText(t, res), testSessionID)
	require.Contains(t, callText(t, res), "destroyed")
}

func TestDestroySessionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"SessionID": "my-session"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.destroySession(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestDestroySessionAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"checkpoint in progress"}`))
	})

	res, err := tool.destroySession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to destroy session")
}

// ---- SendInput ----

func TestSendInput(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"run_id":"run-abc123"}`))
	})

	res, err := tool.sendInput(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Text":      "Hello, agent!",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/input", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "Hello, agent!", gotBody["text"])
	require.Contains(t, callText(t, res), "run-abc123")
}

func TestSendInputWithInputsMap(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"run_id":"run-xyz"}`))
	})

	res, err := tool.sendInput(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Text":      "Run the crew",
		"Inputs":    map[string]any{"topic": "AI safety", "depth": float64(3)},
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "Run the crew", gotBody["text"])
	inputs, ok := gotBody["inputs"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "AI safety", inputs["topic"])
}

func TestSendInputValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":      {"Text": "hello"},
		"invalid session id":      {"SessionID": "not-a-uuid", "Text": "hello"},
		"missing text and inputs": {"SessionID": testSessionID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.sendInput(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestSendInputAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"session not in a state that accepts input"}`))
	})

	res, err := tool.sendInput(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Text":      "hello",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to send input")
}

// ---- PauseSession ----

func TestPauseSession(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})

	res, err := tool.pauseSession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/pause", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, callText(t, res), testSessionID)
	require.Contains(t, callText(t, res), "paused")
}

func TestPauseSessionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"SessionID": "not-valid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.pauseSession(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestPauseSessionAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"session not in a pauseable state"}`))
	})

	res, err := tool.pauseSession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to pause session")
}

// ---- ResumeSession ----

func TestResumeSession(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})

	res, err := tool.resumeSession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/resume", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, callText(t, res), testSessionID)
	require.Contains(t, callText(t, res), "resumed")
}

func TestResumeSessionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"SessionID": "not-valid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.resumeSession(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestResumeSessionAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"session is not paused"}`))
	})

	res, err := tool.resumeSession(context.Background(), request(map[string]any{"SessionID": testSessionID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to resume session")
}

// ---- PatchSession ----

func TestPatchSession(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType := r.Header.Get("Content-Type")
		require.Contains(t, gotContentType, "application/json")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session":` + sessionJSON(testSessionID, "my-agent", "SESSION_STATUS_PAUSED") + `}`))
	})

	res, err := tool.patchSession(context.Background(), request(map[string]any{
		"SessionID":      testSessionID,
		"ResumeOnTopoff": true,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID, gotPath)
	require.Equal(t, http.MethodPatch, gotMethod)
	require.Equal(t, true, gotBody["resume_on_topoff"])

	text := callText(t, res)
	require.Contains(t, text, testSessionID)
}

func TestPatchSessionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":       {"ResumeOnTopoff": true},
		"invalid session id":       {"SessionID": "not-uuid", "ResumeOnTopoff": true},
		"missing patchable fields": {"SessionID": testSessionID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.patchSession(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestPatchSessionAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"id":"bad_request","message":"empty patch body"}`))
	})

	res, err := tool.patchSession(context.Background(), request(map[string]any{
		"SessionID":      testSessionID,
		"ResumeOnTopoff": false,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to patch session")
}

// ---- ResolveHITL ----

func TestResolveHITL(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	})

	res, err := tool.resolveHITL(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"RequestID": testHITLReqID,
		"Outcome":   "HITL_OUTCOME_APPROVE",
		"Reason":    "Looks good",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/hitl/"+testHITLReqID, gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "HITL_OUTCOME_APPROVE", gotBody["outcome"])
	require.Equal(t, "Looks good", gotBody["reason"])

	text := callText(t, res)
	require.Contains(t, text, testSessionID)
	require.Contains(t, text, testHITLReqID)
	require.Contains(t, text, "HITL_OUTCOME_APPROVE")
}

func TestResolveHITLWithContent(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	})

	res, err := tool.resolveHITL(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"RequestID": testHITLReqID,
		"Outcome":   "HITL_OUTCOME_APPROVE",
		"Content":   map[string]any{"site_url": "https://acme.atlassian.net"},
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	content, ok := gotBody["content"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://acme.atlassian.net", content["site_url"])
}

func TestResolveHITLValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id": {"RequestID": testHITLReqID, "Outcome": "HITL_OUTCOME_APPROVE"},
		"invalid session id": {"SessionID": "not-uuid", "RequestID": testHITLReqID, "Outcome": "HITL_OUTCOME_APPROVE"},
		"missing request id": {"SessionID": testSessionID, "Outcome": "HITL_OUTCOME_APPROVE"},
		"missing outcome":    {"SessionID": testSessionID, "RequestID": testHITLReqID},
		"invalid outcome":    {"SessionID": testSessionID, "RequestID": testHITLReqID, "Outcome": "INVALID_OUTCOME"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.resolveHITL(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestResolveHITLAllOutcomes(t *testing.T) {
	for _, outcome := range []string{"HITL_OUTCOME_APPROVE", "HITL_OUTCOME_REJECT", "HITL_OUTCOME_DEFER"} {
		t.Run(outcome, func(t *testing.T) {
			tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			res, err := tool.resolveHITL(context.Background(), request(map[string]any{
				"SessionID": testSessionID,
				"RequestID": testHITLReqID,
				"Outcome":   outcome,
			}))
			require.NoError(t, err)
			require.False(t, res.IsError)
		})
	}
}

func TestResolveHITLAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"hitl request not found"}`))
	})

	res, err := tool.resolveHITL(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"RequestID": testHITLReqID,
		"Outcome":   "HITL_OUTCOME_APPROVE",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to resolve HITL request")
}

// ---- ForkSession ----

func TestForkSession(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sessions":[` + sessionJSON(testSessionID2, "fork-1", "SESSION_STATUS_PROVISIONING") + `]}`))
	})

	res, err := tool.forkSession(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Count":     float64(1),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/fork", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, float64(1), gotBody["count"])

	text := callText(t, res)
	require.Contains(t, text, testSessionID2)
}

func TestForkSessionWithCheckpoint(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sessions":[` + sessionJSON(testSessionID2, "fork-from-cp", "SESSION_STATUS_PROVISIONING") + `]}`))
	})

	res, err := tool.forkSession(context.Background(), request(map[string]any{
		"SessionID":        testSessionID,
		"FromCheckpointID": "cp_9f2c1a4b",
		"Count":            float64(2),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "cp_9f2c1a4b", gotBody["from_checkpoint_id"])
	require.Equal(t, float64(2), gotBody["count"])
}

func TestForkSessionValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id": {},
		"invalid session id": {"SessionID": "not-a-uuid"},
		"count too low":      {"SessionID": testSessionID, "Count": float64(0)},
		"count too high":     {"SessionID": testSessionID, "Count": float64(5)},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.forkSession(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestForkSessionEmptyResponse(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.forkSession(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	text := callText(t, res)
	require.Contains(t, text, "[]")
}

// ---- ExecInSandbox ----

func TestExecInSandbox(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"exit_code":0,"stdout":"hello\n","stderr":"","duration_ms":42}`))
	})

	res, err := tool.execInSandbox(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Argv":      []any{"echo", "hello"},
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/sandbox/exec", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, []any{"echo", "hello"}, gotBody["argv"])

	text := callText(t, res)
	require.Contains(t, text, "hello")
}

func TestExecInSandboxWithOptionalParams(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"exit_code":0,"stdout":"ok\n"}`))
	})

	res, err := tool.execInSandbox(context.Background(), request(map[string]any{
		"SessionID":      testSessionID,
		"Argv":           []any{"cat", "input.txt"},
		"Workdir":        "/tmp",
		"Stdin":          "aGVsbG8=",
		"TimeoutSeconds": float64(30),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/tmp", gotBody["workdir"])
	require.Equal(t, "aGVsbG8=", gotBody["stdin"])
	require.Equal(t, float64(30), gotBody["timeout_seconds"]) // JSON numbers unmarshal as float64
}

func TestExecInSandboxValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":   {"Argv": []any{"ls"}},
		"invalid session id":   {"SessionID": "not-a-uuid", "Argv": []any{"ls"}},
		"missing argv":         {"SessionID": testSessionID},
		"empty argv":           {"SessionID": testSessionID, "Argv": []any{}},
		"non-string argv elem": {"SessionID": testSessionID, "Argv": []any{42}},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.execInSandbox(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestExecInSandboxAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"session not ready"}`))
	})

	res, err := tool.execInSandbox(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Argv":      []any{"ls"},
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
}

// ---- GetIngressURL ----

func TestGetIngressURL(t *testing.T) {
	var gotPath, gotMethod string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ingress_url":{"ingress_url_id":"iu_abc123","session_id":"` + testSessionID + `","port":8000,"url":"https://iu_abc123.nyc3.sandbox.ondigitalocean.com","state":"INGRESS_URL_STATE_READY"}}`))
	})

	res, err := tool.getIngressURL(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/ingress", gotPath)
	require.Equal(t, http.MethodGet, gotMethod)

	text := callText(t, res)
	require.Contains(t, text, "INGRESS_URL_STATE_READY")
	require.Contains(t, text, "nyc3.sandbox.ondigitalocean.com")
}

func TestGetIngressURLValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id": {},
		"invalid session id": {"SessionID": "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getIngressURL(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetIngressURLAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"session not running"}`))
	})

	res, err := tool.getIngressURL(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
}

// ---- CancelTurn ----

func TestCancelTurn(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"outcome":"acked","run_id":"run_xyz"}`))
	})

	res, err := tool.cancelTurn(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"RunID":     "run_xyz",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/cancel", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "run_xyz", gotBody["run_id"])

	text := callText(t, res)
	require.Contains(t, text, "acked")
	require.Contains(t, text, "run_xyz")
}

func TestCancelTurnNoRunID(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"outcome":"no_turn"}`))
	})

	res, err := tool.cancelTurn(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	_, hasRunID := gotBody["run_id"]
	require.False(t, hasRunID)
	text := callText(t, res)
	require.Contains(t, text, "no_turn")
}

func TestCancelTurnValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id": {},
		"invalid session id": {"SessionID": "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.cancelTurn(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCancelTurnAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"id":"forbidden","message":"cancel_turn not enabled for team"}`))
	})

	res, err := tool.cancelTurn(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
}
