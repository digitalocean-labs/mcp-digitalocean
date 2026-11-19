package docs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"
)

// MinClientIDSecretBytes is the shortest secret NewClientTagger accepts.
const MinClientIDSecretBytes = 32

const (
	clientIDHexLen  = 16
	maxUserAgentLen = 128
)

// ClientTagger derives a pseudonymous client ID for search usage logs, so
// weekly distinct clients can be counted without storing IPs. The ID is an
// HMAC of the same IP key the rate limiter uses, under a key derived from
// the secret and the ISO week: one client keeps one ID for a week, and IDs
// cannot be joined across weeks or reversed without the secret.
type ClientTagger struct {
	secret        []byte
	trustedHeader string
	now           func() time.Time
}

// NewClientTagger returns a tagger keyed by secret. trustedHeader has the
// same meaning as RateLimitConfig.TrustedIPHeader.
func NewClientTagger(secret []byte, trustedHeader string) (*ClientTagger, error) {
	if len(secret) < MinClientIDSecretBytes {
		return nil, fmt.Errorf("client id secret must be at least %d bytes", MinClientIDSecretBytes)
	}
	if strings.TrimSpace(trustedHeader) == "" {
		return nil, errors.New("client ids require a trusted client IP header; without one every client behind the edge gets the same id")
	}
	return &ClientTagger{
		secret:        append([]byte(nil), secret...),
		trustedHeader: strings.TrimSpace(trustedHeader),
		now:           time.Now,
	}, nil
}

// ID returns the client's ID for the current week, or unidentifiedClient
// when the request carries no usable IP.
func (t *ClientTagger) ID(r *http.Request) string {
	key, ok := clientKey(r, t.trustedHeader)
	if !ok {
		return unidentifiedClient
	}
	year, week := t.now().UTC().ISOWeek()
	weekKey := hmacSHA256(t.secret, fmt.Sprintf("docs-search-client-id/%04d-W%02d", year, week))
	return hex.EncodeToString(hmacSHA256(weekKey, key))[:clientIDHexLen]
}

func hmacSHA256(key []byte, msg string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(msg))
	return m.Sum(nil)
}

// userAgentFamilies maps a lowercase User-Agent substring to a stable
// family name. Order matters: AI fetchers and crawlers send Mozilla-style
// strings, so they are checked before the generic browser match.
var userAgentFamilies = []struct{ token, family string }{
	{"chatgpt-user", "chatgpt-user"},
	{"oai-searchbot", "oai-searchbot"},
	{"gptbot", "gptbot"},
	{"claude-user", "claude-user"},
	{"claude-searchbot", "claude-searchbot"},
	{"claudebot", "claudebot"},
	{"claude-code", "claude-code"},
	{"perplexity-user", "perplexity-user"},
	{"perplexitybot", "perplexitybot"},
	{"cursor", "cursor"},
	{"googlebot", "googlebot"},
	{"bingbot", "bingbot"},
	{"curl/", "curl"},
	{"wget/", "wget"},
	{"httpie/", "httpie"},
	{"postmanruntime/", "postman"},
	{"python-requests/", "python-requests"},
	{"python-httpx/", "python-httpx"},
	{"aiohttp/", "python-aiohttp"},
	{"python-urllib/", "python-urllib"},
	{"go-http-client/", "go"},
	{"axios/", "node-axios"},
	{"node-fetch/", "node"},
	{"undici", "node"},
	{"okhttp/", "okhttp"},
	{"java/", "java"},
	{"deno/", "deno"},
	{"bun/", "bun"},
	{"mozilla/", "browser"},
}

// userAgentFamily buckets a User-Agent so usage can be grouped by caller
// type: "none" when absent, "other" when unrecognized.
func userAgentFamily(ua string) string {
	ua = strings.ToLower(strings.TrimSpace(ua))
	if ua == "" {
		return "none"
	}
	// Node's built-in fetch sends the bare string "node".
	if ua == "node" {
		return "node"
	}
	for _, f := range userAgentFamilies {
		if strings.Contains(ua, f.token) {
			return f.family
		}
	}
	return "other"
}

// cleanUserAgent drops control characters and bounds length so a hostile
// header cannot forge log lines or bloat them.
func cleanUserAgent(ua string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(ua) {
		if b.Len() >= maxUserAgentLen {
			break
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
