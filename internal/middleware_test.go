package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
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
