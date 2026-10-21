package harnessruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testCheckpointID = "cp_9f2c1a4b"
)

// checkpointJSON returns a minimal Checkpoint JSON for test servers to emit.
func checkpointJSON(checkpointID, sessionID, status, kind string) string {
	return `{"checkpoint_id":"` + checkpointID + `","session_id":"` + sessionID +
		`","status":"` + status + `","kind":"` + kind + `","size_bytes":1024,"created_at":"2026-08-01T12:00:00Z"}`
}

// ---- ListCheckpoints ----

func TestListCheckpoints(t *testing.T) {
	var gotPath, gotMethod string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checkpoints":[` + checkpointJSON(testCheckpointID, testSessionID, "READY", "explicit") + `],"next_page_token":"tok2"}`))
	})

	res, err := tool.listCheckpoints(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/checkpoints", gotPath)
	require.Equal(t, http.MethodGet, gotMethod)

	text := callText(t, res)
	require.Contains(t, text, testCheckpointID)
	require.Contains(t, text, "READY")
	require.Contains(t, text, "tok2")
}

func TestListCheckpointsPagination(t *testing.T) {
	var gotQuery string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checkpoints":[]}`))
	})

	res, err := tool.listCheckpoints(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"PageSize":  float64(25),
		"PageToken": "cursor123",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Contains(t, gotQuery, "page_size=25")
	require.Contains(t, gotQuery, "page_token=cursor123")
}

func TestListCheckpointsValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id": {},
		"invalid session id": {"SessionID": "not-a-uuid"},
		"invalid page size":  {"SessionID": testSessionID, "PageSize": float64(0)},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listCheckpoints(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListCheckpointsEmptySlice(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listCheckpoints(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	text := callText(t, res)
	require.Contains(t, text, "[]")
}

// ---- GetCheckpoint ----

func TestGetCheckpoint(t *testing.T) {
	var gotPath string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checkpoint":` + checkpointJSON(testCheckpointID, testSessionID, "READY", "explicit") + `}`))
	})

	res, err := tool.getCheckpoint(context.Background(), request(map[string]any{
		"SessionID":    testSessionID,
		"CheckpointID": testCheckpointID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/checkpoints/"+testCheckpointID, gotPath)

	text := callText(t, res)
	require.Contains(t, text, testCheckpointID)
	require.Contains(t, text, "READY")
}

func TestGetCheckpointValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":    {"CheckpointID": testCheckpointID},
		"invalid session id":    {"SessionID": "not-a-uuid", "CheckpointID": testCheckpointID},
		"missing checkpoint id": {"SessionID": testSessionID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getCheckpoint(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetCheckpointAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"checkpoint not found"}`))
	})

	res, err := tool.getCheckpoint(context.Background(), request(map[string]any{
		"SessionID":    testSessionID,
		"CheckpointID": testCheckpointID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
}

// ---- CreateCheckpoint ----

func TestCreateCheckpoint(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checkpoint":` + checkpointJSON(testCheckpointID, testSessionID, "READY", "explicit") + `}`))
	})

	res, err := tool.createCheckpoint(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Label":     "before-refactor",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/checkpoints", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "before-refactor", gotBody["label"])

	text := callText(t, res)
	require.Contains(t, text, testCheckpointID)
	require.Contains(t, text, "READY")
}

func TestCreateCheckpointNoLabel(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checkpoint":` + checkpointJSON(testCheckpointID, testSessionID, "READY", "explicit") + `}`))
	})

	res, err := tool.createCheckpoint(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	// label key absent when not provided
	_, hasLabel := gotBody["label"]
	require.False(t, hasLabel)
}

func TestCreateCheckpointValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id": {},
		"invalid session id": {"SessionID": "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.createCheckpoint(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCreateCheckpointAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"active turn in flight"}`))
	})

	res, err := tool.createCheckpoint(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
}

// ---- DeleteCheckpoint ----

func TestDeleteCheckpoint(t *testing.T) {
	var gotPath, gotMethod string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checkpoint_id":"` + testCheckpointID + `","deleted":true}`))
	})

	res, err := tool.deleteCheckpoint(context.Background(), request(map[string]any{
		"SessionID":    testSessionID,
		"CheckpointID": testCheckpointID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/checkpoints/"+testCheckpointID, gotPath)
	require.Equal(t, http.MethodDelete, gotMethod)

	text := callText(t, res)
	require.Contains(t, text, testCheckpointID)
	require.Contains(t, text, "true")
}

func TestDeleteCheckpointValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":    {"CheckpointID": testCheckpointID},
		"invalid session id":    {"SessionID": "not-a-uuid", "CheckpointID": testCheckpointID},
		"missing checkpoint id": {"SessionID": testSessionID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.deleteCheckpoint(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

// ---- RollbackToCheckpoint ----

func TestRollbackToCheckpoint(t *testing.T) {
	var gotPath, gotMethod string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session":` + sessionJSON(testSessionID, "my-agent", "SESSION_STATUS_READY") + `}`))
	})

	res, err := tool.rollbackToCheckpoint(context.Background(), request(map[string]any{
		"SessionID":    testSessionID,
		"CheckpointID": testCheckpointID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/checkpoints/"+testCheckpointID+"/rollback", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)

	text := callText(t, res)
	require.Contains(t, text, testSessionID)
	require.Contains(t, text, "SESSION_STATUS_READY")
}

func TestRollbackToCheckpointValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":    {"CheckpointID": testCheckpointID},
		"invalid session id":    {"SessionID": "not-a-uuid", "CheckpointID": testCheckpointID},
		"missing checkpoint id": {"SessionID": testSessionID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.rollbackToCheckpoint(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestRollbackToCheckpointAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"checkpoint not READY"}`))
	})

	res, err := tool.rollbackToCheckpoint(context.Background(), request(map[string]any{
		"SessionID":    testSessionID,
		"CheckpointID": testCheckpointID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
}
