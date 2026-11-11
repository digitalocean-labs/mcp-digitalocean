package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestAuthFromRequestExtractsAuth(t *testing.T) {
	r := httptest.NewRequest("POST", "/mcp", nil)
	r.Header.Set("Authorization", "Bearer doo_token")

	ctx := AuthFromRequest(context.Background(), r)

	if got, _ := ctx.Value(AuthKey{}).(string); got != "Bearer doo_token" {
		t.Fatalf("auth key = %q, want %q", got, "Bearer doo_token")
	}
}
