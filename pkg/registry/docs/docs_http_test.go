package docs

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var quietLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func searchTestIndex() *DocsIndex {
	return &DocsIndex{
		Entries: []DocsEntry{
			{
				Title:        "How to Create a Droplet",
				URL:          "https://docs.digitalocean.com/products/droplets/how-to/create/",
				MarkdownURL:  "https://docs.digitalocean.com/products/droplets/how-to/create/index.html.md",
				Description:  "Create Droplets from the control panel.",
				Section:      "Droplets",
				LastModified: "2026-10-01",
			},
			{
				Title:       "Droplet Quickstart",
				URL:         "https://docs.digitalocean.com/products/droplets/getting-started/quickstart/",
				Description: "Get started with Droplets.",
				Section:     "Droplets",
			},
			{
				Title:       "How to Create a VPC",
				URL:         "https://docs.digitalocean.com/products/networking/vpc/how-to/create/",
				Description: "Create a VPC network.",
				Section:     "VPC",
			},
		},
	}
}

func doSearch(t *testing.T, h http.Handler, method, target string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body map[string]any
	if rec.Body.Len() > 0 {
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	}
	return rec, body
}

func TestSearchHandler_Results(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockDocsService(ctrl)
	m.EXPECT().GetDocsIndex().Return(searchTestIndex(), nil)
	m.EXPECT().ContentExcerpts(gomock.Any(), gomock.Any(), "create droplet").Return(map[string]string{
		"https://docs.digitalocean.com/products/droplets/how-to/create/": "...to create a Droplet, open the control panel...",
	})

	rec, body := doSearch(t, newSearchHandler(m, quietLogger), http.MethodGet, "/v1/search?q=create+droplet&limit=2")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, rec.Header().Get("Cache-Control"), "public")

	results := body["results"].([]any)
	require.NotEmpty(t, results)
	require.LessOrEqual(t, len(results), 2)
	first := results[0].(map[string]any)
	require.Equal(t, "How to Create a Droplet", first["title"])
	require.Equal(t, "https://docs.digitalocean.com/products/droplets/how-to/create/", first["url"])
	require.Equal(t, "https://docs.digitalocean.com/products/droplets/how-to/create/index.html.md", first["markdown_url"])
	require.Equal(t, "Droplets", first["product"])
	require.Equal(t, "how-to", first["page_type"])
	require.Equal(t, "2026-10-01", first["last_modified"])
	require.Equal(t, "...to create a Droplet, open the control panel...", first["excerpt"])

	meta := body["meta"].(map[string]any)
	require.Equal(t, "create droplet", meta["query"])
	require.Equal(t, "stable", meta["stability"])
	require.EqualValues(t, 3, meta["pages_searched"])
	require.EqualValues(t, len(results), meta["total"])
}

func TestSearchHandler_ExcerptFallsBackToDescription(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockDocsService(ctrl)
	m.EXPECT().GetDocsIndex().Return(searchTestIndex(), nil)
	m.EXPECT().ContentExcerpts(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	_, body := doSearch(t, newSearchHandler(m, quietLogger), http.MethodGet, "/v1/search?q=vpc")
	first := body["results"].([]any)[0].(map[string]any)
	require.Equal(t, "Create a VPC network.", first["excerpt"])
}

func TestSearchHandler_ProductFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockDocsService(ctrl)
	m.EXPECT().GetDocsIndex().Return(searchTestIndex(), nil)
	m.EXPECT().ContentExcerpts(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	_, body := doSearch(t, newSearchHandler(m, quietLogger), http.MethodGet, "/v1/search?q=create&product=vpc")
	results := body["results"].([]any)
	require.Len(t, results, 1)
	require.Equal(t, "How to Create a VPC", results[0].(map[string]any)["title"])
	require.EqualValues(t, 1, body["meta"].(map[string]any)["pages_searched"])
}

func TestSearchHandler_NoResultsIsEmptyArray(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockDocsService(ctrl)
	m.EXPECT().GetDocsIndex().Return(searchTestIndex(), nil)

	rec, _ := doSearch(t, newSearchHandler(m, quietLogger), http.MethodGet, "/v1/search?q=zzzqqq")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"results":[]`)
}

func TestSearchHandler_BadRequests(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		status int
		id     string
	}{
		{"missing q", http.MethodGet, "/v1/search", http.StatusBadRequest, "bad_request"},
		{"blank q", http.MethodGet, "/v1/search?q=++", http.StatusBadRequest, "bad_request"},
		{"q too long", http.MethodGet, "/v1/search?q=" + strings.Repeat("a", maxQueryBytes+1), http.StatusBadRequest, "bad_request"},
		{"invalid utf8", http.MethodGet, "/v1/search?q=%ff%fe", http.StatusBadRequest, "bad_request"},
		{"product too long", http.MethodGet, "/v1/search?q=x&product=" + strings.Repeat("p", maxProductBytes+1), http.StatusBadRequest, "bad_request"},
		{"limit zero", http.MethodGet, "/v1/search?q=x&limit=0", http.StatusBadRequest, "bad_request"},
		{"limit too big", http.MethodGet, "/v1/search?q=x&limit=51", http.StatusBadRequest, "bad_request"},
		{"limit not a number", http.MethodGet, "/v1/search?q=x&limit=ten", http.StatusBadRequest, "bad_request"},
		{"post rejected", http.MethodPost, "/v1/search?q=x", http.StatusMethodNotAllowed, "method_not_allowed"},
		{"head rejected", http.MethodHead, "/v1/search?q=x", http.StatusMethodNotAllowed, "method_not_allowed"},
		{"unknown parameter", http.MethodGet, "/v1/search?q=x&cachebust=123", http.StatusBadRequest, "bad_request"},
		{"repeated parameter", http.MethodGet, "/v1/search?q=x&q=y", http.StatusBadRequest, "bad_request"},
		{"malformed query string", http.MethodGet, "/v1/search?q=%zz", http.StatusBadRequest, "bad_request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := NewMockDocsService(ctrl) // no calls expected: validation runs first
			rec, body := doSearch(t, newSearchHandler(m, quietLogger), tt.method, tt.target)
			require.Equal(t, tt.status, rec.Code)
			require.Equal(t, tt.id, body["id"])
			require.NotEmpty(t, body["message"])
			require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
		})
	}
}

func TestSearchHandler_Preflight(t *testing.T) {
	ctrl := gomock.NewController(t)
	rec, _ := doSearch(t, newSearchHandler(NewMockDocsService(ctrl), quietLogger), http.MethodOptions, "/v1/search")
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestSearchHandler_IndexUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockDocsService(ctrl)
	m.EXPECT().GetDocsIndex().Return(nil, errors.New("origin down"))

	rec, body := doSearch(t, newSearchHandler(m, quietLogger), http.MethodGet, "/v1/search?q=droplet")
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, "service_unavailable", body["id"])
	require.NotContains(t, rec.Body.String(), "origin down", "internal errors must not leak")
}

const cfIP = "CF-Connecting-IP"

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func mustRateLimit(t *testing.T, cfg RateLimitConfig) http.Handler {
	t.Helper()
	h, err := RateLimit(okHandler(), cfg, quietLogger)
	require.NoError(t, err)
	return h
}

func hit(h http.Handler, remoteAddr string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=x", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func from(ip string) map[string]string { return map[string]string{cfIP: ip} }

func TestRateLimitConfig_Validate(t *testing.T) {
	tests := []struct {
		name string
		cfg  RateLimitConfig
		ok   bool
	}{
		{"off ignores other fields", RateLimitConfig{Mode: RateLimitOff}, true},
		{"shadow without header", RateLimitConfig{Mode: RateLimitShadow, RPS: 10, Burst: 30}, true},
		{"enforce with header", RateLimitConfig{Mode: RateLimitEnforce, RPS: 10, Burst: 30, TrustedIPHeader: cfIP}, true},
		{"fractional rps", RateLimitConfig{Mode: RateLimitEnforce, RPS: 0.5, Burst: 1, TrustedIPHeader: cfIP}, true},
		{"enforce without header", RateLimitConfig{Mode: RateLimitEnforce, RPS: 10, Burst: 30}, false},
		{"zero rps", RateLimitConfig{Mode: RateLimitShadow, RPS: 0, Burst: 1}, false},
		{"NaN rps", RateLimitConfig{Mode: RateLimitShadow, RPS: math.NaN(), Burst: 1}, false},
		{"Inf rps", RateLimitConfig{Mode: RateLimitShadow, RPS: math.Inf(1), Burst: 1}, false},
		{"zero burst", RateLimitConfig{Mode: RateLimitShadow, RPS: 1, Burst: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			_, rlErr := RateLimit(okHandler(), tt.cfg, quietLogger)
			if tt.ok {
				require.NoError(t, err)
				require.NoError(t, rlErr)
			} else {
				require.Error(t, err)
				require.Error(t, rlErr, "an invalid config must not yield a disabled limiter")
			}
		})
	}
}

func TestRateLimit_EnforceReturns429AfterBurst(t *testing.T) {
	h := mustRateLimit(t, RateLimitConfig{Mode: RateLimitEnforce, RPS: 1, Burst: 3, TrustedIPHeader: cfIP})
	for i := 0; i < 3; i++ {
		rec := hit(h, "10.0.0.1:1", from("203.0.113.7"))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "1;w=1;burst=3", rec.Header().Get("RateLimit-Policy"))
		require.Empty(t, rec.Header().Get("RateLimit-Remaining"), "per-client counters must not ride on cacheable responses")
	}
	rec := hit(h, "10.0.0.1:1", from("203.0.113.7"))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "1", rec.Header().Get("Retry-After"))
	require.Equal(t, "0", rec.Header().Get("RateLimit-Remaining"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	require.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, rec.Body.String(), "too_many_requests")
	require.Contains(t, rec.Body.String(), "docs-corpus.jsonl")

	require.Equal(t, http.StatusOK, hit(h, "10.0.0.1:1", from("198.51.100.9")).Code, "a different client has its own budget")
}

func TestRateLimit_RetryAfterScalesWithRate(t *testing.T) {
	h := mustRateLimit(t, RateLimitConfig{Mode: RateLimitEnforce, RPS: 0.1, Burst: 1, TrustedIPHeader: cfIP})
	require.Equal(t, http.StatusOK, hit(h, "10.0.0.1:1", from("203.0.113.7")).Code)
	rec := hit(h, "10.0.0.1:1", from("203.0.113.7"))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	retry, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, retry, 9, "one token at 0.1 rps takes about 10 s")
	require.LessOrEqual(t, retry, 10)
	require.Equal(t, "0.1;w=1;burst=1", rec.Header().Get("RateLimit-Policy"))
}

func TestRateLimit_ShadowNeverBlocks(t *testing.T) {
	h := mustRateLimit(t, RateLimitConfig{Mode: RateLimitShadow, RPS: 1, Burst: 1})
	for i := 0; i < 5; i++ {
		require.Equal(t, http.StatusOK, hit(h, "203.0.113.7:1", nil).Code)
	}
}

func TestRateLimit_OffIsPassthrough(t *testing.T) {
	rec := hit(mustRateLimit(t, RateLimitConfig{Mode: RateLimitOff}), "203.0.113.7:1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Header().Get("RateLimit-Policy"))
}

func TestRateLimit_IPv6GroupedBy64(t *testing.T) {
	h := mustRateLimit(t, RateLimitConfig{Mode: RateLimitEnforce, RPS: 1, Burst: 1, TrustedIPHeader: cfIP})
	require.Equal(t, http.StatusOK, hit(h, "10.0.0.1:1", from("2001:db8:1:2::1")).Code)
	require.Equal(t, http.StatusTooManyRequests, hit(h, "10.0.0.1:1", from("2001:db8:1:2:ffff::9")).Code, "same /64 shares a bucket")
	require.Equal(t, http.StatusOK, hit(h, "10.0.0.1:1", from("2001:db8:1:3::1")).Code, "different /64 is a new client")
}

func TestRateLimit_TrustedHeaderUsesRightmostEntry(t *testing.T) {
	h := mustRateLimit(t, RateLimitConfig{Mode: RateLimitEnforce, RPS: 1, Burst: 1, TrustedIPHeader: cfIP})
	// A client-supplied left entry must not change the identity the edge appended.
	require.Equal(t, http.StatusOK, hit(h, "10.0.0.1:1", from("1.1.1.1, 203.0.113.7")).Code)
	require.Equal(t, http.StatusTooManyRequests, hit(h, "10.0.0.1:1", from("8.8.8.8, 203.0.113.7")).Code)
}

func TestRateLimit_EdgeBypassSharesOneBucket(t *testing.T) {
	h := mustRateLimit(t, RateLimitConfig{Mode: RateLimitEnforce, RPS: 1, Burst: 2, TrustedIPHeader: cfIP})
	// Requests without a usable trusted header did not come through the
	// edge; they share one budget instead of going unlimited.
	require.Equal(t, http.StatusOK, hit(h, "198.51.100.1:1", nil).Code)
	require.Equal(t, http.StatusOK, hit(h, "198.51.100.2:1", from("not-an-ip")).Code)
	require.Equal(t, http.StatusTooManyRequests, hit(h, "198.51.100.3:1", nil).Code)
	// Edge traffic is unaffected.
	require.Equal(t, http.StatusOK, hit(h, "10.0.0.1:1", from("203.0.113.7")).Code)
}

func TestRateLimit_ConnectionAddressWithoutTrustedHeader(t *testing.T) {
	rl := newRateLimiter(RateLimitConfig{Mode: RateLimitShadow, RPS: 1, Burst: 1}, quietLogger)
	h := rl.wrap(okHandler())
	// The header is ignored when not trusted, so rotating it does not mint
	// new identities.
	hit(h, "203.0.113.7:1", from("192.0.2.50"))
	hit(h, "203.0.113.7:2", from("192.0.2.51"))
	require.Equal(t, 1, rl.size())
}

func TestRateLimit_OptionsBypassesLimiter(t *testing.T) {
	h := mustRateLimit(t, RateLimitConfig{Mode: RateLimitEnforce, RPS: 1, Burst: 1, TrustedIPHeader: cfIP})
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodOptions, "/v1/search", nil)
		req.Header.Set(cfIP, "203.0.113.7")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimit_MemoryBoundedUnderChurn(t *testing.T) {
	rl := newRateLimiter(RateLimitConfig{Mode: RateLimitShadow, RPS: 1, Burst: 1, MaxClients: limiterShards * 4, IdleTTL: time.Hour}, quietLogger)
	start := time.Now()
	for i := 0; i < 50_000; i++ {
		rl.take("client-" + strconv.Itoa(i))
	}
	require.LessOrEqual(t, rl.size(), limiterShards*4)
	require.Less(t, time.Since(start), 2*time.Second, "eviction must stay O(shard), not O(total)")
}

func TestRateLimit_LogsOncePerLimitingEpisode(t *testing.T) {
	rl := newRateLimiter(RateLimitConfig{Mode: RateLimitEnforce, RPS: 1, Burst: 1, TrustedIPHeader: cfIP}, quietLogger)
	allowed, _, started := rl.take("a")
	require.True(t, allowed)
	require.False(t, started)
	_, _, started = rl.take("a")
	require.True(t, started, "first denial starts an episode")
	_, _, started = rl.take("a")
	require.False(t, started, "later denials in the same episode stay quiet")
}

func TestParseRateLimitMode(t *testing.T) {
	for _, s := range []string{"off", "shadow", "enforce", " Enforce "} {
		_, err := ParseRateLimitMode(s)
		require.NoError(t, err, s)
	}
	_, err := ParseRateLimitMode("block")
	require.Error(t, err)
}
