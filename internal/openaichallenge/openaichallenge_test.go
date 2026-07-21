package openaichallenge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_servesToken(t *testing.T) {
	const token = "openai-challenge-token-abc123"
	handler := Handler(token)

	req := httptest.NewRequest(http.MethodGet, WellKnownPath, nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain; charset=utf-8", ct)
	}
	if got := rec.Body.String(); got != token {
		t.Fatalf("body = %q, want %q", got, token)
	}
}

func TestHandler_headOmitsBody(t *testing.T) {
	handler := Handler("token")

	req := httptest.NewRequest(http.MethodHead, WellKnownPath, nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body length = %d, want 0", rec.Body.Len())
	}
}

func TestHandler_emptyTokenNotFound(t *testing.T) {
	handler := Handler("")

	req := httptest.NewRequest(http.MethodGet, WellKnownPath, nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_methodNotAllowed(t *testing.T) {
	handler := Handler("token")

	req := httptest.NewRequest(http.MethodPost, WellKnownPath, nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Fatalf("Allow = %q, want GET, HEAD", allow)
	}
}
