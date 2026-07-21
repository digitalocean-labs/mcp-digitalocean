package oauthmeta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_ConfiguredResource(t *testing.T) {
	h := Handler(Config{
		Resource:               "https://droplets-mcp-hswwk.ondigitalocean.app",
		AuthorizationServers:   []string{"https://cloud.digitalocean.com"},
		BearerMethodsSupported: []string{"header"},
	})

	req := httptest.NewRequest(http.MethodGet, WellKnownPath, nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var got struct {
		Resource               string   `json:"resource"`
		ResourceName           string   `json:"resource_name"`
		AuthorizationServers   []string `json:"authorization_servers"`
		BearerMethodsSupported []string `json:"bearer_methods_supported"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if got.Resource != "https://droplets-mcp-hswwk.ondigitalocean.app" {
		t.Fatalf("resource = %q", got.Resource)
	}
	if got.ResourceName != "droplets-mcp-hswwk" {
		t.Fatalf("resource_name = %q, want subdomain droplets-mcp-hswwk", got.ResourceName)
	}
	if len(got.AuthorizationServers) != 1 || got.AuthorizationServers[0] != "https://cloud.digitalocean.com" {
		t.Fatalf("authorization_servers = %v", got.AuthorizationServers)
	}
	if len(got.BearerMethodsSupported) != 1 || got.BearerMethodsSupported[0] != "header" {
		t.Fatalf("bearer_methods_supported = %v", got.BearerMethodsSupported)
	}
}

func TestHandler_ResourceDerivedFromRequest(t *testing.T) {
	h := Handler(Config{
		AuthorizationServers:   []string{"https://cloud.digitalocean.com"},
		BearerMethodsSupported: []string{"header"},
	})

	req := httptest.NewRequest(http.MethodGet, WellKnownPath, nil)
	req.Host = "droplets-mcp-hswwk.ondigitalocean.app"
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	h(rec, req)

	var got struct {
		Resource     string `json:"resource"`
		ResourceName string `json:"resource_name"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got.Resource != "https://droplets-mcp-hswwk.ondigitalocean.app" {
		t.Fatalf("resource = %q, want derived https URL", got.Resource)
	}
	if got.ResourceName != "droplets-mcp-hswwk" {
		t.Fatalf("resource_name = %q", got.ResourceName)
	}
}

func TestHandler_XForwardedHostAndProtoList(t *testing.T) {
	h := Handler(Config{})

	req := httptest.NewRequest(http.MethodGet, WellKnownPath, nil)
	req.Host = "internal:8080"
	req.Header.Set("X-Forwarded-Proto", "https, http")
	req.Header.Set("X-Forwarded-Host", "stage-mcp.s2r1.internal.digitalocean.com")
	rec := httptest.NewRecorder()
	h(rec, req)

	var got struct {
		Resource     string `json:"resource"`
		ResourceName string `json:"resource_name"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got.Resource != "https://stage-mcp.s2r1.internal.digitalocean.com" {
		t.Fatalf("resource = %q", got.Resource)
	}
	if got.ResourceName != "stage-mcp" {
		t.Fatalf("resource_name = %q, want stage-mcp", got.ResourceName)
	}
}

func TestHandler_OmitsOptionalFields(t *testing.T) {
	h := Handler(Config{Resource: "https://x.example.com"})

	req := httptest.NewRequest(http.MethodGet, WellKnownPath, nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := raw["authorization_servers"]; ok {
		t.Fatalf("authorization_servers should be omitted when empty")
	}
	if _, ok := raw["bearer_methods_supported"]; ok {
		t.Fatalf("bearer_methods_supported should be omitted when empty")
	}
}

func TestHandler_RejectsNonGet(t *testing.T) {
	h := Handler(Config{Resource: "https://x.example.com"})

	req := httptest.NewRequest(http.MethodPost, WellKnownPath, nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Fatalf("Allow = %q, want %q", allow, "GET, HEAD")
	}
}

func TestHandler_HeadHasNoBody(t *testing.T) {
	h := Handler(Config{Resource: "https://x.example.com"})

	req := httptest.NewRequest(http.MethodHead, WellKnownPath, nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD response should have empty body, got %d bytes", rec.Body.Len())
	}
}

func TestRequireBearer_MissingTokenChallenges(t *testing.T) {
	var nextCalled bool
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })

	h := RequireBearer(next, ChallengeConfig{
		Resource: "https://droplets-mcp-hswwk.ondigitalocean.app",
		Scopes:   []string{"read", "write"},
	})

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("next handler should not be called when bearer token is missing")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	got := rec.Header().Get("WWW-Authenticate")
	want := `Bearer resource_metadata="https://droplets-mcp-hswwk.ondigitalocean.app/.well-known/oauth-protected-resource", scope="read write"`
	if got != want {
		t.Fatalf("WWW-Authenticate =\n  %q\nwant\n  %q", got, want)
	}
}

func TestRequireBearer_DerivesResourceMetadataFromRequest(t *testing.T) {
	h := RequireBearer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), ChallengeConfig{
		Scopes: []string{"read", "write"},
	})

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Host = "stage-mcp.s2r1.internal.digitalocean.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	got := rec.Header().Get("WWW-Authenticate")
	want := `Bearer resource_metadata="https://stage-mcp.s2r1.internal.digitalocean.com/.well-known/oauth-protected-resource", scope="read write"`
	if got != want {
		t.Fatalf("WWW-Authenticate =\n  %q\nwant\n  %q", got, want)
	}
}

func TestRequireBearer_PassesThroughWithToken(t *testing.T) {
	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	h := RequireBearer(next, ChallengeConfig{Resource: "https://x.example.com"})

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("next handler should be called when a bearer token is present")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("WWW-Authenticate should not be set when authenticated")
	}
}

func TestBearerToken(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"", ""},
		{"Bearer", ""},
		{"Bearer ", ""},
		{"Basic abc", ""},
		{"Bearer abc123", "abc123"},
		{"bearer abc123", "abc123"},
		{"Bearer   spaced  ", "spaced"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		if c.header != "" {
			req.Header.Set("Authorization", c.header)
		}
		if got := BearerToken(req); got != c.want {
			t.Fatalf("BearerToken(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}

func TestSubdomain(t *testing.T) {
	cases := map[string]string{
		"https://droplets-mcp-hswwk.ondigitalocean.app": "droplets-mcp-hswwk",
		"https://cloud.digitalocean.com":                "cloud",
		"https://localhost:8080":                        "localhost",
		"https://singlelabel":                           "singlelabel",
	}
	for in, want := range cases {
		if got := subdomain(in); got != want {
			t.Fatalf("subdomain(%q) = %q, want %q", in, got, want)
		}
	}
}
