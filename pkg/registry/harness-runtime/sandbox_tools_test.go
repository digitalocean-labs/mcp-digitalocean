package harnessruntime

import (
	"context"
	"encoding/json"
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

func testPolicy() map[string]any {
	var p map[string]any
	_ = json.Unmarshal([]byte(`{"rules":[{"action":"allow","resource":"*"}]}`), &p)
	return p
}

func TestValidatePolicy(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]any

	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"violations":[]}`))
	})

	res, err := tool.validatePolicy(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Policy":    testPolicy(),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Equal(t, "/v2/agents/sessions/"+testSessionID+"/policy/validate", gotPath)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Contains(t, gotContentType, "application/json")

	policyInBody, ok := gotBody["policy"].(map[string]any)
	require.True(t, ok, "request body should have a 'policy' key")
	require.NotEmpty(t, policyInBody["rules"])

	text := callText(t, res)
	require.Contains(t, text, `"valid"`)
}

func TestValidatePolicyWithViolations(t *testing.T) {
	tool := newTestTool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":false,"violations":[{"rule":"no-wildcard","message":"wildcard resources are not permitted"}]}`))
	})

	res, err := tool.validatePolicy(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Policy":    testPolicy(),
	}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	text := callText(t, res)
	require.Contains(t, text, "no-wildcard")
	require.Contains(t, text, "wildcard resources are not permitted")
}

func TestValidatePolicyValidation(t *testing.T) {
	tool := newTestTool(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("API must not be called for invalid input")
	})

	for name, args := range map[string]map[string]any{
		"missing session id":   {"Policy": testPolicy()},
		"not uuid":             {"SessionID": "not-a-uuid", "Policy": testPolicy()},
		"missing policy":       {"SessionID": testSessionID},
		"empty policy":         {"SessionID": testSessionID, "Policy": map[string]any{}},
		"policy not an object": {"SessionID": testSessionID, "Policy": "not-an-object"},
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
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"id":"not_found","message":"session not found"}`))
	})

	res, err := tool.validatePolicy(context.Background(), request(map[string]any{
		"SessionID": testSessionID,
		"Policy":    testPolicy(),
	}))
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, callText(t, res), "Failed to validate policy")
}
