package harnessruntime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- ListSandboxSizes ----

func TestListSandboxSizes(t *testing.T) {
	var gotPath, gotMethod string
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sizes":[{"slug":"mars-1vcpu-1gb","vcpus":1,"memory_mb":1024,"price_per_hour_usd":0.0535,"price_per_month_usd":39.804},{"slug":"mars-2vcpu-4gb","vcpus":2,"memory_mb":4096,"price_per_hour_usd":0.126,"price_per_month_usd":93.744}]}`))
	})

	res, err := tool.listSandboxSizes(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/sandbox/sizes", gotPath)
	require.Equal(t, http.MethodGet, gotMethod)

	text := callText(t, res)
	require.Contains(t, text, "mars-1vcpu-1gb")
	require.Contains(t, text, "mars-2vcpu-4gb")
	require.Contains(t, text, "0.0535")
	require.Contains(t, text, "93.744")
}

func TestListSandboxSizesEmpty(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	res, err := tool.listSandboxSizes(context.Background(), request(nil))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Regexp(t, `"sizes":\s*\[\]`, callText(t, res))
}

func TestListSandboxSizesAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"id":"unauthorized","message":"Unable to authenticate you"}`))
	})

	res, err := tool.listSandboxSizes(context.Background(), request(nil))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to list sandbox sizes")
}

// ---- ValidatePolicy ----

func testPolicyManifest() map[string]any {
	var m map[string]any
	_ = json.Unmarshal([]byte(`{"agent":"codex","size":"mars-2vcpu-4gb","permissions":{"default":"allow"}}`), &m)
	return m
}

func TestValidatePolicy(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody string

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		bodyBytes, _ := io.ReadAll(r.Body)
		gotBody = string(bodyBytes)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"agent":"codex","version":"latest","ok":true,"defaultAction":"allow","defaultActionVerdict":"exact","defaultActionRendered":"allow","verdicts":[]}`))
	})

	res, err := tool.validatePolicy(context.Background(), request(map[string]any{
		"Manifest": testPolicyManifest(),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/policy/validate", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, gotContentType, "application/x-yaml")
	require.Contains(t, gotBody, "codex")
	require.Contains(t, gotBody, "mars-2vcpu-4gb")

	text := callText(t, res)
	require.Contains(t, text, `"ok"`)
	require.Contains(t, text, "codex")
}

func TestValidatePolicyDegraded(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"agent":"codex","version":"latest","ok":false,"defaultAction":"allow","defaultActionVerdict":"degraded","defaultActionRendered":"ask","defaultActionReason":"codex cannot enforce allow for file_edit","verdicts":[{"rule":"file_edit","verdict":"unsupported","reason":"not supported by this agent version"}]}`))
	})

	res, err := tool.validatePolicy(context.Background(), request(map[string]any{
		"Manifest": testPolicyManifest(),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	text := callText(t, res)
	require.Contains(t, text, "unsupported")
	require.Contains(t, text, "file_edit")
	require.Contains(t, text, `"ok": false`)
}

func TestValidatePolicyValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing manifest":       {},
		"empty manifest object":  {"Manifest": map[string]any{}},
		"empty manifest string":  {"Manifest": ""},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := tool.validatePolicy(context.Background(), request(args))
			require.NoError(t, err)
			require.True(t, res.IsError)
		})
	}
}

func TestValidatePolicyAPIError(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"id":"bad_request","message":"invalid manifest"}`))
	})

	res, err := tool.validatePolicy(context.Background(), request(map[string]any{
		"Manifest": testPolicyManifest(),
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to validate policy")
}
