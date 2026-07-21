// Package oauthmeta serves the OAuth 2.0 Protected Resource Metadata document
// (RFC 9728) that lets MCP clients discover which authorization server issues
// tokens for this resource server.
//
// The document is served at WellKnownPath and is used by MCP clients during the
// OAuth flow to locate the authorization server and supported bearer methods.
package oauthmeta

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// WellKnownPath is the standard discovery path for OAuth Protected Resource
// Metadata, as defined by RFC 9728 and required by the MCP authorization spec.
const WellKnownPath = "/.well-known/oauth-protected-resource"

// ProdAuthorizationServer is the authorization server issuer URL advertised in
// the OAuth protected resource metadata.
const ProdAuthorizationServer = "https://cloud.digitalocean.com"

// Config controls the metadata document served by Handler.
type Config struct {
	// Resource, when non-empty, is advertised verbatim as the resource
	// identifier. When empty, the resource is derived from each incoming
	// request's scheme and host, so the document works without explicit
	// configuration (e.g. https://droplets-mcp-hswwk.ondigitalocean.app).
	Resource string

	// AuthorizationServers lists the authorization server issuer URLs that can
	// issue tokens for this resource.
	AuthorizationServers []string

	// BearerMethodsSupported lists how the bearer token may be presented.
	// Valid values per RFC 9728 are "header", "body", and "query".
	BearerMethodsSupported []string
}

// metadata is the RFC 9728 OAuth 2.0 Protected Resource Metadata document.
// Optional fields use omitempty so they stay out of the response when unset.
type metadata struct {
	Resource               string   `json:"resource"`
	ResourceName           string   `json:"resource_name,omitempty"`
	AuthorizationServers   []string `json:"authorization_servers,omitempty"`
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`
}

// Handler returns an http.HandlerFunc that serves the protected resource
// metadata as JSON. The resource identifier is taken from cfg.Resource when set,
// otherwise derived from the request; resource_name is always the leading
// subdomain label of the resource host.
func Handler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		resource := cfg.Resource
		if strings.TrimSpace(resource) == "" {
			resource = resourceFromRequest(r)
		}

		body, err := json.MarshalIndent(metadata{
			Resource:               resource,
			ResourceName:           subdomain(resource),
			AuthorizationServers:   cfg.AuthorizationServers,
			BearerMethodsSupported: cfg.BearerMethodsSupported,
		}, "", "  ")
		if err != nil {
			http.Error(w, "failed to encode metadata", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(body)
	}
}

// resourceFromRequest reconstructs this server's public URL (scheme://host) from
// the incoming request, honoring the X-Forwarded-* headers set by upstream
// proxies and load balancers (as used by DigitalOcean App Platform).
func resourceFromRequest(r *http.Request) string {
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		if i := strings.IndexByte(proto, ','); i >= 0 {
			proto = proto[:i]
		}
		scheme = strings.TrimSpace(proto)
	} else if r.TLS != nil {
		scheme = "https"
	}

	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host
}

// ChallengeConfig configures the WWW-Authenticate challenge returned to clients
// that reach a protected endpoint without a bearer token.
type ChallengeConfig struct {
	// Resource, when set, fixes the resource identifier used to build the
	// resource_metadata URL. When empty, it is derived from each request.
	Resource string

	// Scopes are advertised in the challenge's scope parameter (e.g. read write).
	Scopes []string
}

// RequireBearer wraps next so that requests without a bearer token receive a
// 401 with an RFC 9728 / RFC 6750 WWW-Authenticate challenge pointing at this
// server's protected resource metadata. Requests that carry a bearer token are
// passed through unchanged; token validation itself is left to the upstream API.
func RequireBearer(next http.Handler, cfg ChallengeConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if BearerToken(r) == "" {
			w.Header().Set("WWW-Authenticate", challenge(r, cfg))
			http.Error(w, "missing or invalid bearer token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// BearerToken returns the token from an "Authorization: Bearer <token>" header,
// or "" when the header is absent, malformed, or carries an empty token.
func BearerToken(r *http.Request) string {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if len(auth) <= len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}

// challenge builds the WWW-Authenticate header value, advertising the resource
// metadata URL and the requested scopes.
func challenge(r *http.Request, cfg ChallengeConfig) string {
	resource := cfg.Resource
	if strings.TrimSpace(resource) == "" {
		resource = resourceFromRequest(r)
	}
	metadataURL := strings.TrimRight(resource, "/") + WellKnownPath

	parts := []string{fmt.Sprintf("resource_metadata=%q", metadataURL)}
	if len(cfg.Scopes) > 0 {
		parts = append(parts, fmt.Sprintf("scope=%q", strings.Join(cfg.Scopes, " ")))
	}
	return "Bearer " + strings.Join(parts, ", ")
}

// subdomain returns the leading label of the resource host, e.g.
// "https://droplets-mcp-hswwk.ondigitalocean.app" -> "droplets-mcp-hswwk".
// It returns the whole host when there is no dot-separated subdomain.
func subdomain(resource string) string {
	host := resource
	if u, err := url.Parse(resource); err == nil && u.Host != "" {
		host = u.Host
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if i := strings.IndexByte(host, '.'); i > 0 {
		return host[:i]
	}
	return host
}
