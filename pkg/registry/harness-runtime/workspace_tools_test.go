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
	testWorkspaceID = "019fb39c-14d9-7080-933e-b9b90e25acde"
	testTransferID  = "xfer-abc123"
)

func workspaceJSON(workspaceID, name, state string) string {
	return `{"workspace_id":"` + workspaceID + `","name":"` + name + `","state":"` + state + `","size_gibibytes":10,"bytes_used":1024,"created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-01T12:00:00Z","created_by":"user-1","updated_by":"user-1"}`
}

// ---- CreateWorkspace ----

func TestCreateWorkspace(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"workspace":` + workspaceJSON(testWorkspaceID, "my-ws", "AVAILABLE") + `}`))
	})

	res, err := tool.createWorkspace(context.Background(), request(map[string]any{
		"SizeGibibytes": float64(10),
		"Name":          "my-ws",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/workspaces", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, float64(10), gotBody["size_gibibytes"])
	require.Equal(t, "my-ws", gotBody["name"])

	text := callText(t, res)
	require.Contains(t, text, testWorkspaceID)
	require.Contains(t, text, "AVAILABLE")
}

func TestCreateWorkspaceNoName(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspace":` + workspaceJSON(testWorkspaceID, "", "AVAILABLE") + `}`))
	})

	res, err := tool.createWorkspace(context.Background(), request(map[string]any{
		"SizeGibibytes": float64(5),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	_, hasName := gotBody["name"]
	require.False(t, hasName)
}

func TestCreateWorkspaceValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing size":  {},
		"size zero":     {"SizeGibibytes": float64(0)},
		"size too big":  {"SizeGibibytes": float64(101)},
		"size fraction": {"SizeGibibytes": 5.5},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.createWorkspace(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCreateWorkspaceAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"quota exceeded"}`))
	})

	res, err := tool.createWorkspace(context.Background(), request(map[string]any{
		"SizeGibibytes": float64(50),
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to create workspace")
}

// ---- ListWorkspaces ----

func TestListWorkspaces(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspaces":[` + workspaceJSON(testWorkspaceID, "my-ws", "AVAILABLE") + `],"next_page_token":"tok2"}`))
	})

	res, err := tool.listWorkspaces(context.Background(), request(map[string]any{
		"PageSize":  float64(10),
		"PageToken": "tok1",
		"State":     "AVAILABLE",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/workspaces", gotPath)
	require.Equal(t, "10", gotQuery.Get("page_size"))
	require.Equal(t, "tok1", gotQuery.Get("page_token"))
	require.Equal(t, "AVAILABLE", gotQuery.Get("state"))

	text := callText(t, res)
	require.Contains(t, text, testWorkspaceID)
	require.Contains(t, text, "tok2")
}

func TestListWorkspacesNoArgsOmitsQuery(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listWorkspaces(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"workspaces":\s*\[\]`, callText(t, res))
}

func TestListWorkspacesValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"page size too small": {"PageSize": float64(0)},
		"page size too large": {"PageSize": float64(201)},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listWorkspaces(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

// ---- GetWorkspace ----

func TestGetWorkspace(t *testing.T) {
	var gotPath string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspace":` + workspaceJSON(testWorkspaceID, "my-ws", "ATTACHED") + `}`))
	})

	res, err := tool.getWorkspace(context.Background(), request(map[string]any{"WorkspaceID": testWorkspaceID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/workspaces/"+testWorkspaceID, gotPath)

	text := callText(t, res)
	require.Contains(t, text, testWorkspaceID)
	require.Contains(t, text, "ATTACHED")
}

func TestGetWorkspaceValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"WorkspaceID": "my-ws"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getWorkspace(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetWorkspaceNotFound(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"workspace not found"}`))
	})

	res, err := tool.getWorkspace(context.Background(), request(map[string]any{"WorkspaceID": testWorkspaceID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get workspace")
}

// ---- DeleteWorkspace ----

func TestDeleteWorkspace(t *testing.T) {
	var gotPath, gotMethod string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})

	res, err := tool.deleteWorkspace(context.Background(), request(map[string]any{"WorkspaceID": testWorkspaceID}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/workspaces/"+testWorkspaceID, gotPath)
	require.Equal(t, http.MethodDelete, gotMethod)
	require.Contains(t, callText(t, res), testWorkspaceID)
	require.Contains(t, callText(t, res), "deleted")
}

func TestDeleteWorkspaceValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing":  {},
		"not uuid": {"WorkspaceID": "my-ws"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.deleteWorkspace(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestDeleteWorkspaceAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"workspace is attached"}`))
	})

	res, err := tool.deleteWorkspace(context.Background(), request(map[string]any{"WorkspaceID": testWorkspaceID}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to delete workspace")
}

// ---- CreateWorkspaceTransfer ----

func TestCreateWorkspaceTransfer(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"transfer_id":"` + testTransferID + `","direction":"upload","status":"pending","upload_id":"mp-123","part_size":5242880}`))
	})

	res, err := tool.createWorkspaceTransfer(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Direction": "upload",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/workspace/transfers", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "upload", gotBody["direction"])

	text := callText(t, res)
	require.Contains(t, text, testTransferID)
	require.Contains(t, text, "upload")
	require.Contains(t, text, "5242880")
}

func TestCreateWorkspaceTransferDownload(t *testing.T) {
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"transfer_id":"` + testTransferID + `","direction":"download","status":"pending"}`))
	})

	res, err := tool.createWorkspaceTransfer(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Direction": "download",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "download", gotBody["direction"])
}

func TestCreateWorkspaceTransferValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id": {"Direction": "upload"},
		"invalid session id": {"SessionID": "not-a-uuid", "Direction": "upload"},
		"missing direction":  {"SessionID": testSessionID},
		"invalid direction":  {"SessionID": testSessionID, "Direction": "sideways"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.createWorkspaceTransfer(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCreateWorkspaceTransferAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"transfer already in progress"}`))
	})

	res, err := tool.createWorkspaceTransfer(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Direction": "upload",
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to create workspace transfer")
}

// ---- GetWorkspaceTransfer ----

func TestGetWorkspaceTransfer(t *testing.T) {
	var gotPath string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transfer_id":"` + testTransferID + `","direction":"download","status":"completed","bytes_written":1048576,"sha256":"abc123","download_url":"https://example.com/dl"}`))
	})

	res, err := tool.getWorkspaceTransfer(context.Background(), request(map[string]any{
		"SessionID":  testSessionID,
		"TransferID": testTransferID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/workspace/transfers/"+testTransferID, gotPath)

	text := callText(t, res)
	require.Contains(t, text, "completed")
	require.Contains(t, text, "abc123")
	require.Contains(t, text, "example.com/dl")
}

func TestGetWorkspaceTransferValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":  {"TransferID": testTransferID},
		"invalid session id":  {"SessionID": "not-a-uuid", "TransferID": testTransferID},
		"missing transfer id": {"SessionID": testSessionID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getWorkspaceTransfer(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetWorkspaceTransferAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"transfer not found"}`))
	})

	res, err := tool.getWorkspaceTransfer(context.Background(), request(map[string]any{
		"SessionID":  testSessionID,
		"TransferID": testTransferID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get workspace transfer")
}

// ---- CancelWorkspaceTransfer ----

func TestCancelWorkspaceTransfer(t *testing.T) {
	var gotPath, gotMethod string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transfer_id":"` + testTransferID + `","aborted":true,"status":"aborted"}`))
	})

	res, err := tool.cancelWorkspaceTransfer(context.Background(), request(map[string]any{
		"SessionID":  testSessionID,
		"TransferID": testTransferID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/workspace/transfers/"+testTransferID+"/cancel", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)

	text := callText(t, res)
	require.Contains(t, text, testTransferID)
	require.Contains(t, text, "true")
	require.Contains(t, text, "aborted")
}

func TestCancelWorkspaceTransferValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":  {"TransferID": testTransferID},
		"invalid session id":  {"SessionID": "not-a-uuid", "TransferID": testTransferID},
		"missing transfer id": {"SessionID": testSessionID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.cancelWorkspaceTransfer(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCancelWorkspaceTransferAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"transfer not found"}`))
	})

	res, err := tool.cancelWorkspaceTransfer(context.Background(), request(map[string]any{
		"SessionID":  testSessionID,
		"TransferID": testTransferID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to cancel workspace transfer")
}

// ---- CommitWorkspaceUpload ----

func TestCommitWorkspaceUpload(t *testing.T) {
	var gotPath, gotMethod string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"transfer_id":"` + testTransferID + `","status":"completed","size_bytes":10485760}`))
	})

	res, err := tool.commitWorkspaceUpload(context.Background(), request(map[string]any{
		"SessionID":  testSessionID,
		"TransferID": testTransferID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/workspace/transfers/"+testTransferID+"/commit", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)

	text := callText(t, res)
	require.Contains(t, text, "completed")
	require.Contains(t, text, "10485760")
}

func TestCommitWorkspaceUploadValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":  {"TransferID": testTransferID},
		"invalid session id":  {"SessionID": "not-a-uuid", "TransferID": testTransferID},
		"missing transfer id": {"SessionID": testSessionID},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.commitWorkspaceUpload(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCommitWorkspaceUploadAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"parts not complete"}`))
	})

	res, err := tool.commitWorkspaceUpload(context.Background(), request(map[string]any{
		"SessionID":  testSessionID,
		"TransferID": testTransferID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to commit workspace upload")
}

// ---- CreateWorkspacePartUploadURLs ----

func TestCreateWorkspacePartUploadURLs(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transfer_id":"` + testTransferID + `","part_urls":[{"part_number":1,"upload_url":"https://s3.example.com/part1"},{"part_number":2,"upload_url":"https://s3.example.com/part2"}],"expires_at":"2026-08-01T13:00:00Z"}`))
	})

	res, err := tool.createWorkspacePartUploadURLs(context.Background(), request(map[string]any{
		"SessionID":   testSessionID,
		"TransferID":  testTransferID,
		"PartNumbers": []any{float64(1), float64(2)},
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/workspace/transfers/"+testTransferID+"/part-upload-urls", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, []any{float64(1), float64(2)}, gotBody["part_numbers"])

	text := callText(t, res)
	require.Contains(t, text, "s3.example.com/part1")
	require.Contains(t, text, "s3.example.com/part2")
}

func TestCreateWorkspacePartUploadURLsValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":   {"TransferID": testTransferID, "PartNumbers": []any{float64(1)}},
		"invalid session id":   {"SessionID": "not-a-uuid", "TransferID": testTransferID, "PartNumbers": []any{float64(1)}},
		"missing transfer id":  {"SessionID": testSessionID, "PartNumbers": []any{float64(1)}},
		"missing part numbers": {"SessionID": testSessionID, "TransferID": testTransferID},
		"empty part numbers":   {"SessionID": testSessionID, "TransferID": testTransferID, "PartNumbers": []any{}},
		"non-integer part":     {"SessionID": testSessionID, "TransferID": testTransferID, "PartNumbers": []any{"one"}},
		"zero part number":     {"SessionID": testSessionID, "TransferID": testTransferID, "PartNumbers": []any{float64(0)}},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.createWorkspacePartUploadURLs(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestCreateWorkspacePartUploadURLsAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"transfer not in progress"}`))
	})

	res, err := tool.createWorkspacePartUploadURLs(context.Background(), request(map[string]any{
		"SessionID":   testSessionID,
		"TransferID":  testTransferID,
		"PartNumbers": []any{float64(1)},
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to create part upload URLs")
}

func TestCreateWorkspacePartUploadURLsEmptyResponse(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transfer_id":"` + testTransferID + `"}`))
	})

	res, err := tool.createWorkspacePartUploadURLs(context.Background(), request(map[string]any{
		"SessionID":   testSessionID,
		"TransferID":  testTransferID,
		"PartNumbers": []any{float64(1)},
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	text := callText(t, res)
	require.Contains(t, text, "[]")
}
