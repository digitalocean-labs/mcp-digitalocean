package harnessruntime

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func imageMessageResourceJSON(kind, checkpointID string, vulnerable bool, messages string) string {
	v := "false"
	if vulnerable {
		v = "true"
	}
	return `{"kind":"` + kind + `","session_id":"` + testSessionID + `","session_name":"support-agent","checkpoint_id":"` + checkpointID +
		`","checkpoint_name":"","image_id":"img_1","resolution_complete":true,"vulnerable":` + v + `,"messages":` + messages + `}`
}

const testImageMessagesJSON = `[{"type":"vulnerability","title":"Image update available","message":"This sandbox image has a published fix.","more_info_url":"https://docs.digitalocean.com/"}]`

// ---- ListImageMessages ----

func TestListImageMessages(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resources":[` + imageMessageResourceJSON("SESSION", "", true, testImageMessagesJSON) + `],"next_page_token":"tok2"}`))
	})

	res, err := tool.listImageMessages(context.Background(), request(map[string]any{
		"Type":      "vulnerability",
		"PageSize":  float64(10),
		"PageToken": "tok1",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/image-messages", gotPath)
	require.Equal(t, "vulnerability", gotQuery.Get("type"))
	require.Equal(t, "10", gotQuery.Get("page_size"))
	require.Equal(t, "tok1", gotQuery.Get("page_token"))

	text := callText(t, res)
	require.Contains(t, text, testSessionID)
	require.Contains(t, text, "Image update available")
	require.Contains(t, text, "tok2")
	require.Regexp(t, `"vulnerable":\s*true`, text)
}

func TestListImageMessagesByName(t *testing.T) {
	var gotQuery url.Values
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resources":[` + imageMessageResourceJSON("CHECKPOINT", testCheckpointID, false, `[]`) + `],"next_page_token":""}`))
	})

	res, err := tool.listImageMessages(context.Background(), request(map[string]any{
		"CheckpointName": "before-upgrade",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, url.Values{"checkpoint_name": {"before-upgrade"}}, gotQuery)
	require.Contains(t, callText(t, res), testCheckpointID)
}

func TestListImageMessagesEmpty(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resources":[{"kind":"SESSION","session_id":"` + testSessionID + `"}]}`))
	})

	res, err := tool.listImageMessages(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"messages":\s*\[\]`, callText(t, res))
}

func TestListImageMessagesValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"unknown type":          {"Type": "license"},
		"both names":            {"SessionName": "a", "CheckpointName": "b"},
		"page size not integer": {"PageSize": 1.5},
		"page size zero":        {"PageSize": float64(0)},
		"page size too large":   {"PageSize": float64(maxPageSize + 1)},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.listImageMessages(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestListImageMessagesAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(`{"id":"not_implemented","message":"image messages are not available"}`))
	})

	res, err := tool.listImageMessages(context.Background(), request(nil))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list image messages")
}

// ---- GetSessionImageMessages ----

func TestGetSessionImageMessages(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resource":` + imageMessageResourceJSON("SESSION", "", true, testImageMessagesJSON) + `}`))
	})

	res, err := tool.getSessionImageMessages(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Type":      "vulnerability",
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/sessions/"+testSessionID+"/image-messages", gotPath)
	require.Equal(t, "vulnerability", gotQuery.Get("type"))
	text := callText(t, res)
	require.Contains(t, text, "Image update available")
	require.Regexp(t, `"vulnerable":\s*true`, text)
}

func TestGetSessionImageMessagesNoNotice(t *testing.T) {
	var gotRawQuery string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resource":{"kind":"SESSION","session_id":"` + testSessionID + `","vulnerable":false}}`))
	})

	res, err := tool.getSessionImageMessages(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Empty(t, gotRawQuery)
	require.Regexp(t, `"messages":\s*\[\]`, callText(t, res))
}

func TestGetSessionImageMessagesValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":   {},
		"invalid session uuid": {"SessionID": "not-a-uuid"},
		"unknown type":         {"SessionID": testSessionID, "Type": "license"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getSessionImageMessages(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetSessionImageMessagesAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"id":"forbidden","message":"session belongs to another team"}`))
	})

	res, err := tool.getSessionImageMessages(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get session image messages")
}

// ---- GetCheckpointImageMessages ----

func TestGetCheckpointImageMessages(t *testing.T) {
	var gotPath string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resource":` + imageMessageResourceJSON("CHECKPOINT", testCheckpointID, false, `[]`) + `}`))
	})

	res, err := tool.getCheckpointImageMessages(context.Background(), request(map[string]any{
		"SessionID":    testSessionID,
		"CheckpointID": testCheckpointID,
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/sessions/"+testSessionID+"/checkpoints/"+testCheckpointID+"/image-messages", gotPath)
	text := callText(t, res)
	require.Contains(t, text, testCheckpointID)
	require.Contains(t, text, "CHECKPOINT")
}

func TestGetCheckpointImageMessagesValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":    {"CheckpointID": testCheckpointID},
		"invalid session uuid":  {"SessionID": "not-a-uuid", "CheckpointID": testCheckpointID},
		"missing checkpoint id": {"SessionID": testSessionID},
		"unknown type":          {"SessionID": testSessionID, "CheckpointID": testCheckpointID, "Type": "license"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.getCheckpointImageMessages(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestGetCheckpointImageMessagesAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"checkpoint not found"}`))
	})

	res, err := tool.getCheckpointImageMessages(context.Background(), request(map[string]any{
		"SessionID":    testSessionID,
		"CheckpointID": testCheckpointID,
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to get checkpoint image messages")
}
