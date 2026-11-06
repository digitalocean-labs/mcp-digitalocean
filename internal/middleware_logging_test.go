package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"mcp-digitalocean/internal/calltrace"

	"github.com/mark3labs/mcp-go/mcp"
)

func runLogged(t *testing.T, m *ToolLoggingMiddleware, ctx context.Context, name string, args map[string]any, h func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	m.Logger = slog.New(slog.NewJSONHandler(&buf, nil))
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	_, _ = m.ToolMiddleware(h)(ctx, req)
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log line not JSON: %v: %s", err, buf.String())
	}
	return rec
}

func okHandler(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	calltrace.FromContext(ctx).Add(slog.Int("result_count", 2), slog.String("top_url", "https://docs.digitalocean.com/products/droplets/"))
	return mcp.NewToolResultText("ok"), nil
}

func requestCtx(headers map[string]string) context.Context {
	r := httptest.NewRequest("POST", "/mcp", nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return AuthFromRequest(context.Background(), r)
}

func TestDocsSearchRecordOmitsQueryByDefault(t *testing.T) {
	ctx := requestCtx(map[string]string{"User-Agent": "claude-code/2.1", "Mcp-Session-Id": "abc123"})
	rec := runLogged(t, &ToolLoggingMiddleware{}, ctx, "docs-search",
		map[string]any{"Query": "spaces cors", "Product": "spaces", "Limit": float64(5)}, okHandler)

	if _, ok := rec["query"]; ok {
		t.Fatalf("query logged without LogDocsQueries: %v", rec)
	}
	want := map[string]any{
		"tool": "docs-search", "product": "spaces", "limit": float64(5),
		"result_count": float64(2), "top_url": "https://docs.digitalocean.com/products/droplets/",
		"via_router": false, "client_user_agent": "claude-code/2.1", "tool_call_outcome": ToolCallSuccess,
	}
	for k, v := range want {
		if rec[k] != v {
			t.Errorf("%s = %v, want %v", k, rec[k], v)
		}
	}
	ref, _ := rec["session_ref"].(string)
	if len(ref) != 32 || strings.Contains(ref, "abc123") {
		t.Errorf("session_ref = %q, want 32 hex chars that do not contain the session id", ref)
	}
}

func TestDocsSearchQueryRedactedWhenEnabled(t *testing.T) {
	q := "auth fails with dop_v1_0123456789abcdef0123456789abcdef and Bearer eyJhbGciOi.payload token=hunter2 for kubernetes-cluster-autoscaler-configuration-reference"
	rec := runLogged(t, &ToolLoggingMiddleware{LogDocsQueries: true}, requestCtx(nil), "docs-search",
		map[string]any{"Query": q}, okHandler)
	got, _ := rec["query"].(string)
	for _, secret := range []string{"dop_v1_", "eyJhbGciOi", "hunter2"} {
		if strings.Contains(got, secret) {
			t.Errorf("query %q leaks %q", got, secret)
		}
	}
	if !strings.HasPrefix(got, "auth fails with [REDACTED]") {
		t.Errorf("query = %q, want the non-secret text kept", got)
	}
	if !strings.Contains(got, "kubernetes-cluster-autoscaler-configuration-reference") && !strings.HasSuffix(got, "...") {
		t.Errorf("long slug without digits was redacted: %q", got)
	}
}

func TestOpaqueRunRedaction(t *testing.T) {
	in := "key 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08 and kubernetes-cluster-autoscaler-configuration"
	got := redactSecrets(in)
	if strings.Contains(got, "9f86d081") {
		t.Errorf("hex run not redacted: %q", got)
	}
	if !strings.Contains(got, "kubernetes-cluster-autoscaler-configuration") {
		t.Errorf("slug redacted: %q", got)
	}
}

func TestQueryTruncated(t *testing.T) {
	rec := runLogged(t, &ToolLoggingMiddleware{LogDocsQueries: true}, requestCtx(nil), "docs-search",
		map[string]any{"Query": strings.Repeat("droplet ", 100)}, okHandler)
	got, _ := rec["query"].(string)
	if len([]rune(got)) != maxLogValueLen+3 || !strings.HasSuffix(got, "...") {
		t.Errorf("query length = %d, want %d with ellipsis", len([]rune(got)), maxLogValueLen+3)
	}
}

func TestAuthenticatedToolLogsNoArgsOrCredentials(t *testing.T) {
	ctx := requestCtx(map[string]string{
		"Authorization":      "Bearer doo_v1_secrettokenvalue",
		RouterResourceHeader: "cipher-from-router",
	})
	handler := func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		calltrace.FromContext(ctx).Add(slog.String("should_not", "appear"))
		return mcp.NewToolResultText("ok"), nil
	}
	var buf bytes.Buffer
	m := &ToolLoggingMiddleware{LogDocsQueries: true}
	m.Logger = slog.New(slog.NewJSONHandler(&buf, nil))
	req := mcp.CallToolRequest{}
	req.Params.Name = "droplet-create"
	req.Params.Arguments = map[string]any{"Name": "web-1", "UserData": "#!/bin/sh\nexport SECRET=x"}
	_, _ = m.ToolMiddleware(handler)(ctx, req)

	line := buf.String()
	for _, leak := range []string{"doo_v1_", "secrettokenvalue", "cipher-from-router", "web-1", "SECRET", "should_not", "Authorization"} {
		if strings.Contains(line, leak) {
			t.Errorf("authenticated tool record leaks %q: %s", leak, line)
		}
	}
	if !strings.Contains(line, `"via_router":true`) {
		t.Errorf("via_router missing: %s", line)
	}
}

func TestErrorPathsKeepOutcomes(t *testing.T) {
	rec := runLogged(t, &ToolLoggingMiddleware{}, requestCtx(nil), "docs-get-page",
		map[string]any{"URL": "/products/droplets/"},
		func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return nil, errors.New("boom")
		})
	if rec["tool_call_outcome"] != ToolCallError || rec["url"] != "/products/droplets/" || rec["level"] != "ERROR" {
		t.Errorf("error record = %v", rec)
	}

	rec = runLogged(t, &ToolLoggingMiddleware{}, requestCtx(nil), "docs-get-page",
		map[string]any{"URL": "/x/"},
		func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("not found"), nil
		})
	if rec["tool_call_outcome"] != ToolCallResultError || rec["content"] != "not found" {
		t.Errorf("result error record = %v", rec)
	}
}

func TestStdioContextHasNoRequestMeta(t *testing.T) {
	rec := runLogged(t, &ToolLoggingMiddleware{}, context.Background(), "docs-search",
		map[string]any{"Query": "x"}, okHandler)
	if _, ok := rec["session_ref"]; ok {
		t.Errorf("session_ref without a request: %v", rec)
	}
	if rec["via_router"] != false {
		t.Errorf("via_router = %v, want false", rec["via_router"])
	}
}

func TestCleanLogValueDropsControlChars(t *testing.T) {
	if got := cleanLogValue("a\nb\x1bc", 10); got != "abc" {
		t.Errorf("cleanLogValue = %q, want %q", got, "abc")
	}
}
