package harnessruntime

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- ListRegions ----

func TestListRegions(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"regions":[{"slug":"mem1","is_default":false},{"slug":"ric1","is_default":true}]}`))
	})

	res, err := tool.listRegions(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/harness-runtime/regions", gotPath)
	require.Equal(t, http.MethodGet, gotMethod)

	text := callText(t, res)
	require.Contains(t, text, "mem1")
	require.Contains(t, text, "ric1")
	require.Regexp(t, `"slug":\s*"mem1",\s*"is_default":\s*false`, text)
	require.Regexp(t, `"slug":\s*"ric1",\s*"is_default":\s*true`, text)
}

func TestListRegionsEmpty(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listRegions(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Regexp(t, `"regions":\s*\[\]`, callText(t, res))
}

func TestListRegionsAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"id":"unauthorized","message":"Unable to authenticate you"}`))
	})

	res, err := tool.listRegions(context.Background(), request(nil))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list regions")
}
