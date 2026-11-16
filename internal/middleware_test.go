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

func TestAuthFromRequestExtractsCallerUserAgent(t *testing.T) {
	r := httptest.NewRequest("POST", "/mcp", nil)
	r.Header.Set("User-Agent", " Cursor/0.48.2 mcp-router/1db28da ")

	ctx := AuthFromRequest(context.Background(), r)

	if got := CallerUserAgentFromContext(ctx); got != "Cursor/0.48.2 mcp-router/1db28da" {
		t.Fatalf("caller user agent = %q, want %q", got, "Cursor/0.48.2 mcp-router/1db28da")
	}
}

func TestWithCallerUserAgentIgnoresEmpty(t *testing.T) {
	ctx := context.Background()
	if got := WithCallerUserAgent(ctx, "   "); got != ctx {
		t.Fatalf("empty user agent should leave ctx unchanged")
	}
	if got := CallerUserAgentFromContext(ctx); got != "" {
		t.Fatalf("caller user agent = %q, want empty", got)
	}
}

func TestWithCallerUserAgentCapsOnRuneBoundary(t *testing.T) {
	exact := strings.Repeat("a", maxCallerUserAgentLen)
	if got := CallerUserAgentFromContext(WithCallerUserAgent(context.Background(), exact)); got != exact {
		t.Fatalf("value at the cap should be kept whole, got len %d", len(got))
	}

	long := strings.Repeat("a", maxCallerUserAgentLen-1) + "é" + strings.Repeat("b", 1000)
	got := CallerUserAgentFromContext(WithCallerUserAgent(context.Background(), long))
	if want := strings.Repeat("a", maxCallerUserAgentLen-1); got != want || !utf8.ValidString(got) {
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
