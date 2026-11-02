package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckAnonymousServices(t *testing.T) {
	tests := []struct {
		name     string
		services []string
		wantErr  bool
	}{
		{name: "docs only", services: []string{"docs"}},
		{name: "all services", services: nil, wantErr: true},
		{name: "account scoped", services: []string{"droplets"}, wantErr: true},
		{name: "docs plus account scoped", services: []string{"docs", "droplets"}, wantErr: true},
		// The registry resolves service names case-sensitively and does not trim,
		// so these never start. Reject them here rather than reporting that
		// anonymous access is enabled and failing later.
		{name: "wrong case", services: []string{"Docs"}, wantErr: true},
		{name: "untrimmed", services: []string{" docs"}, wantErr: true},
		{name: "trailing comma leaves an empty name", services: []string{"docs", ""}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAnonymousServices(tt.services)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkAnonymousServices(%v) error = %v, wantErr %v", tt.services, err, tt.wantErr)
			}
		})
	}
}

func TestCheckAnonymousServicesErrorNamesEmptyEntry(t *testing.T) {
	err := checkAnonymousServices([]string{"docs", ""})
	if err == nil {
		t.Fatal("expected an error for an empty service name")
	}
	if !strings.Contains(err.Error(), `""`) {
		t.Fatalf("error should quote the empty name so it is visible, got %q", err)
	}
}

func TestInvalidBoolEnv(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  string
	}{
		{name: "unset", want: ""},
		{name: "valid", value: "true", set: true, want: ""},
		{name: "valid numeric", value: "1", set: true, want: ""},
		{name: "unparseable is reported verbatim", value: "yes", set: true, want: "yes"},
		{name: "on is not a bool", value: "on", set: true, want: "on"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv("ALLOW_ANONYMOUS", tt.value)
			}
			if got := invalidBoolEnv("ALLOW_ANONYMOUS"); got != tt.want {
				t.Fatalf("invalidBoolEnv(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  bool
	}{
		{name: "unset keeps fallback", want: false},
		{name: "true", value: "true", set: true, want: true},
		{name: "uppercase", value: "TRUE", set: true, want: true},
		{name: "numeric", value: "1", set: true, want: true},
		{name: "padded", value: " true ", set: true, want: true},
		{name: "false", value: "false", set: true, want: false},
		{name: "unparseable falls back", value: "yes please", set: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv("ALLOW_ANONYMOUS", tt.value)
			}
			if got := getEnvBool("ALLOW_ANONYMOUS", false); got != tt.want {
				t.Fatalf("getEnvBool(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestRegisterRoutes(t *testing.T) {
	ok := func(name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Route", name)
		})
	}
	guard := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	serve := func(mux *http.ServeMux, method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec
	}

	t.Run("mcp stays guarded and extra routes are not", func(t *testing.T) {
		mux := http.NewServeMux()
		registerRoutes(mux, ok("mcp"), ok("wk").ServeHTTP, nil, guard, map[string]http.Handler{"/v1/search": ok("search")})

		if rec := serve(mux, http.MethodPost, "/mcp"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("/mcp without auth = %d, want 401", rec.Code)
		}
		if rec := serve(mux, http.MethodGet, "/v1/search"); rec.Header().Get("X-Route") != "search" {
			t.Fatalf("/v1/search not routed to search handler: %d", rec.Code)
		}
		if rec := serve(mux, http.MethodGet, "/.well-known/oauth-protected-resource"); rec.Header().Get("X-Route") != "wk" {
			t.Fatalf("well-known route missing: %d", rec.Code)
		}
	})

	t.Run("anonymous deployment serves mcp and search only", func(t *testing.T) {
		mux := http.NewServeMux()
		registerRoutes(mux, ok("mcp"), nil, nil, nil, map[string]http.Handler{"/v1/search": ok("search")})

		if rec := serve(mux, http.MethodPost, "/mcp"); rec.Header().Get("X-Route") != "mcp" {
			t.Fatalf("/mcp not routed: %d", rec.Code)
		}
		for _, p := range []string{"/v1/search/", "/v1/search/x", "/", "/.well-known/oauth-protected-resource"} {
			if rec := serve(mux, http.MethodGet, p); rec.Code != http.StatusNotFound {
				t.Fatalf("%s = %d, want 404", p, rec.Code)
			}
		}
	})
}
