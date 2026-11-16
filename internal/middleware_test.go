package middleware

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAuthFromRequestExtractsAuthAndRouterResource(t *testing.T) {
	r := httptest.NewRequest("POST", "/mcp", nil)
	r.Header.Set("Authorization", "Bearer doo_token")
	r.Header.Set(RouterResourceHeader, "cipher-from-router")

	ctx := AuthFromRequest(context.Background(), r)

	if got, _ := ctx.Value(AuthKey{}).(string); got != "Bearer doo_token" {
		t.Fatalf("auth key = %q, want %q", got, "Bearer doo_token")
	}
	if got := RouterResourceFromContext(ctx); got != "cipher-from-router" {
		t.Fatalf("router resource = %q, want %q", got, "cipher-from-router")
	}
}

func TestAuthFromRequestWithoutRouterHeader(t *testing.T) {
	r := httptest.NewRequest("POST", "/mcp", nil)
	r.Header.Set("Authorization", "Bearer doo_token")

	ctx := AuthFromRequest(context.Background(), r)

	if got := RouterResourceFromContext(ctx); got != "" {
		t.Fatalf("router resource = %q, want empty", got)
	}
}

func TestAuthFromRequestExtractsUserAgent(t *testing.T) {
	r := httptest.NewRequest("POST", "/mcp", nil)
	r.Header.Set("User-Agent", "mcp-router/1db28da")

	ctx := AuthFromRequest(context.Background(), r)

	if got := UserAgentFromContext(ctx); got != "mcp-router/1db28da" {
		t.Fatalf("user agent = %q, want %q", got, "mcp-router/1db28da")
	}
}

func TestUserAgentFromContextAbsent(t *testing.T) {
	if got := UserAgentFromContext(context.Background()); got != "" {
		t.Fatalf("user agent = %q, want empty", got)
	}
}

func TestWithUserAgentSanitizes(t *testing.T) {
	ctx := WithUserAgent(context.Background(), "  Cursor/0.48\r\n injected  ")
	if got := UserAgentFromContext(ctx); got != "Cursor/0.48 injected" {
		t.Fatalf("user agent = %q, want %q", got, "Cursor/0.48 injected")
	}

	long := strings.Repeat("a", maxUserAgentLen-1) + "é"
	got := UserAgentFromContext(WithUserAgent(context.Background(), long))
	if len(got) > maxUserAgentLen || !utf8.ValidString(got) {
		t.Fatalf("user agent not capped on a rune boundary: len=%d valid=%v", len(got), utf8.ValidString(got))
	}
}

func TestRouterResourceFromContextAbsent(t *testing.T) {
	if got := RouterResourceFromContext(context.Background()); got != "" {
		t.Fatalf("router resource = %q, want empty", got)
	}
}

func TestWithRouterResourceRoundTrip(t *testing.T) {
	ctx := WithRouterResource(context.Background(), "cipher")
	if got := RouterResourceFromContext(ctx); got != "cipher" {
		t.Fatalf("router resource = %q, want %q", got, "cipher")
	}
}
