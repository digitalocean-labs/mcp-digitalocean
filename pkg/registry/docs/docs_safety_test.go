package docs

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const smtpURL = docsBase + "/support/why-is-smtp-blocked/"
const testResponseCap = 4096

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newHandlerClient(handler http.Handler) *DocsClient {
	client := NewDocsClient()
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		response := recorder.Result()
		response.Request = req
		return response, nil
	})
	return client
}

func TestFetchOriginPolicy(t *testing.T) {
	for _, target := range []string{
		"https://example.com/", "https://docs.digitalocean.com.evil.example/",
		"https://docs.digitalocean.com@127.0.0.1/", "http://docs.digitalocean.com/",
		"https://docs.digitalocean.com:8443/", "https://user@docs.digitalocean.com/",
	} {
		t.Run(target, func(t *testing.T) {
			client := newHandlerClient(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("disallowed origin reached the transport")
			}))
			_, err := client.fetchUncached(target)
			require.ErrorContains(t, err, "refusing to fetch")
		})
	}
}

func TestFetchRedirectPolicy(t *testing.T) {
	for _, target := range []string{
		"https://example.com/private", "http://127.0.0.1/private",
		"http://docs.digitalocean.com/private", "https://docs.digitalocean.com:8443/private",
		"https://user@docs.digitalocean.com/private", "//example.com/private",
	} {
		t.Run(target, func(t *testing.T) {
			var hits int
			client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				http.Redirect(w, r, target, http.StatusFound)
			}))
			_, err := client.fetchUncached(docsBase + "/start/")
			require.ErrorContains(t, err, "refusing to fetch")
			require.Equal(t, 1, hits, "redirect must be refused before its destination is fetched")
		})
	}

	t.Run("same origin is allowed", func(t *testing.T) {
		client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/start/" {
				http.Redirect(w, r, "/safe/", http.StatusFound)
				return
			}
			fmt.Fprint(w, "safe docs")
		}))
		body, err := client.fetchUncached(docsBase + "/start/")
		require.NoError(t, err)
		require.Equal(t, "safe docs", body)
	})

	t.Run("redirect count stays bounded", func(t *testing.T) {
		var hits int
		client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			http.Redirect(w, r, "/loop/", http.StatusFound)
		}))
		_, err := client.fetchUncached(docsBase + "/loop/")
		require.ErrorContains(t, err, "10 redirects")
		require.Equal(t, 10, hits)
	})
}

func TestFetchResponseSizeWithoutTrustedLength(t *testing.T) {
	body := strings.Repeat("x", testResponseCap+1)
	for _, tc := range []struct {
		name      string
		length    int64
		bytes     int
		wantError bool
	}{
		{"exact limit without length", -1, testResponseCap, false},
		{"chunked oversized", -1, testResponseCap + 1, true},
		{"incorrect small length", 1, testResponseCap + 1, true},
		{"declared oversized", testResponseCap + 1, testResponseCap + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewDocsClient()
			client.maxResponseSize = testResponseCap
			client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header),
					ContentLength: tc.length, Body: io.NopCloser(strings.NewReader(body[:tc.bytes])), Request: r}, nil
			})
			text, err := client.fetchUncached(docsBase + "/large/")
			if tc.wantError {
				require.ErrorContains(t, err, "limit")
				require.Empty(t, text, "never return a successful prefix")
			} else {
				require.NoError(t, err)
				require.Len(t, text, testResponseCap)
			}
		})
	}
}

func TestGetDocsIndex_PreservesJSONWhenOnlyPrimaryFails(t *testing.T) {
	var primaryDown atomic.Bool
	var txtHits atomic.Int32
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == llmsTxtPath {
			txtHits.Add(1)
			fmt.Fprint(w, fallbackLlmsTxt)
			return
		}
		if primaryDown.Load() {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		fmt.Fprintf(w, `[{"title":"SMTP","url":%q,"product":"Support"}]`, smtpURL)
	}))
	first, err := client.GetDocsIndex()
	require.NoError(t, err)
	client.cache.set("docsIndex", first, -time.Second)
	primaryDown.Store(true)
	again, err := client.GetDocsIndex()
	require.NoError(t, err)
	require.Same(t, first, again)
	require.Same(t, first, client.staleIndex.Load())
	require.Equal(t, "llms-index.json", again.Source)
	require.Zero(t, txtHits.Load(), "a small fallback must not replace a richer stale index")
}

func TestQueryStemsDoNotMatchUnrelatedSubstrings(t *testing.T) {
	for _, tc := range []struct {
		query, field string
		match        bool
	}{
		{"spaces", "namespace access", false}, {"worker", "networking", false},
		{"redis", "redirect", false}, {"spaces", "a space", true},
		{"configuration", "configure cors", true}, {"codes", "status code", true},
		// Prefixes of at least minPrefixLen characters match longer tokens.
		{"postgres", "postgresql pools", true}, {"mongo", "mongodb backups", true},
		{"kube", "kubernetes", true}, {"config", "configure cors", true},
		{"db", "mongodb", false}, {"pg", "postgresql", false},
	} {
		t.Run(tc.query+"/"+tc.field, func(t *testing.T) {
			require.Equal(t, tc.match, parseQueryTerms(tc.query)[0].matches(tc.field))
		})
	}
	index := loadFixtureIndex(t)
	results := SearchIndex(index, "spaces access keys")
	require.NotEmpty(t, results)
	require.Contains(t, []string{docsBase + "/products/spaces/how-to/manage-access/", docsBase + "/platform/teams/roles/permissions/spaces_key/"}, results[0].URL)
	rank := rankOf(results, []string{docsBase + "/products/spaces/how-to/manage-access/"})
	require.NotZero(t, rank)
	require.LessOrEqual(t, rank, 2)
	require.Empty(t, searchIndex(&DocsIndex{Entries: []DocsEntry{{Title: "Namespace access"}}}, "spaces access", true))
}

func TestRareTermsAtFullUniverseScale(t *testing.T) {
	index := &DocsIndex{Entries: make([]DocsEntry, 4600)}
	for i := range index.Entries {
		index.Entries[i].Title = "App Platform"
		if i < 23 {
			index.Entries[i].Description = "520 status code"
		}
	}
	terms := parseQueryTerms("app 520")
	_, weights := matchTerms(index, terms)
	require.Equal(t, []int{1, rareTermWeight}, weights)
	index.Entries[23].Description = "520 status code"
	_, weights = matchTerms(index, terms)
	require.Equal(t, []int{1, 1}, weights)
}
