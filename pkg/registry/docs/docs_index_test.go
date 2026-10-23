package docs

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// regressionQuery is one judged query from testdata/regression_queries.json.
// Expected URLs are canonical; the first is the primary target and any later
// ones are acceptable alternates.
type regressionQuery struct {
	Query    string   `json:"query"`
	Tool     string   `json:"tool"`
	Expected []string `json:"expected"`
}

// Recall@5 on every judged query is the contract; MRR is gated loosely so
// a tie-break change or an upstream content edit does not fail the build.
// Measured: fixture 0.88, live 0.88 (2026-10-20). The live threshold lives
// in docs_live_test.go so this file stays unused-const-clean without the
// integration tag.
const minimumFixtureMRR = 0.80

func loadFixtureIndex(t *testing.T) *DocsIndex {
	t.Helper()
	data, err := os.ReadFile("testdata/llms-index.json")
	require.NoError(t, err)
	index, err := parseLlmsIndex(data)
	require.NoError(t, err)
	return index
}

func loadRegressionQueries(t *testing.T) []regressionQuery {
	t.Helper()
	data, err := os.ReadFile("testdata/regression_queries.json")
	require.NoError(t, err)
	var queries []regressionQuery
	require.NoError(t, json.Unmarshal(data, &queries))
	require.NotEmpty(t, queries)
	return queries
}

func rankForTool(index *DocsIndex, q regressionQuery) []DocsEntry {
	if q.Tool == "troubleshoot" {
		return rankTroubleshoot(index, parseQueryTerms(q.Query))
	}
	return SearchIndex(index, q.Query)
}

// rankOf returns the 1-based rank of the first expected URL, or 0.
func rankOf(results []DocsEntry, expected []string) int {
	for i, r := range results {
		url := canonicalDocURL(r.URL)
		for _, e := range expected {
			if url == e {
				return i + 1
			}
		}
	}
	return 0
}

func TestParseLlmsIndex(t *testing.T) {
	index := loadFixtureIndex(t)

	require.Equal(t, "llms-index.json", index.Source)
	require.Greater(t, len(index.Entries), 100)
	require.Contains(t, index.Sections, "Support")
	require.Contains(t, index.Sections, "Droplets")

	var smtp *DocsEntry
	for i := range index.Entries {
		if index.Entries[i].URL == "https://docs.digitalocean.com/support/why-is-smtp-blocked/" {
			smtp = &index.Entries[i]
		}
	}
	require.NotNil(t, smtp, "support article should be in the index")
	require.Equal(t, "Why is SMTP blocked?", smtp.Title)
	require.Equal(t, "Support", smtp.Section)
	require.Equal(t, "https://docs.digitalocean.com/support/why-is-smtp-blocked/index.html.md", smtp.MarkdownURL)
	require.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, smtp.LastModified, "lastmod should be trimmed to a date")
	require.NotEmpty(t, smtp.Description)
}

func TestParseLlmsIndex_Invalid(t *testing.T) {
	_, err := parseLlmsIndex([]byte(`{not json`))
	require.Error(t, err)

	_, err = parseLlmsIndex([]byte(`[]`))
	require.Error(t, err)

	// Entries without a title/URL or outside the docs host are skipped;
	// HTML entities in text fields are decoded.
	index, err := parseLlmsIndex([]byte(`[
		{"title":"Keep","url":"https://docs.digitalocean.com/products/droplets/","product":"Droplets","description":"They don&rsquo;t affect the app."},
		{"title":"","url":"https://docs.digitalocean.com/products/empty/"},
		{"title":"Elsewhere","url":"https://example.com/x/"},
		{"title":"No product","url":"https://docs.digitalocean.com/misc/"}
	]`))
	require.NoError(t, err)
	require.Len(t, index.Entries, 2)
	require.Equal(t, "They don’t affect the app.", index.Entries[0].Description)
	require.Equal(t, "General", index.Entries[1].Section)
}

func TestLightStem(t *testing.T) {
	tests := map[string]string{
		"configuration": "configur",
		"configure":     "configur",
		"codes":         "code",
		"droplets":      "droplet",
		"recovery":      "recover",
		"failing":       "fail",
		"blocked":       "block",
		"gpu":           "gpu",
		"smtp":          "smtp",
		"iso":           "iso",
		"keys":          "keys", // too short to strip
	}
	for in, want := range tests {
		require.Equal(t, want, lightStem(in), in)
	}
}

func TestParseQueryTerms(t *testing.T) {
	texts := func(terms []queryTerm) []string {
		out := make([]string, len(terms))
		for i, term := range terms {
			out[i] = term.text
		}
		return out
	}

	require.Equal(t, []string{"smtp", "blocked"}, texts(parseQueryTerms("Why is SMTP blocked?")))
	require.Equal(t, []string{"droplet", "100", "disk", "usage"}, texts(parseQueryTerms("Droplet 100% disk usage")))
	require.Equal(t, []string{"digitalocean", "droplet"}, texts(parseQueryTerms("digitalocean_droplet")))
	// Non-ASCII letters stay inside their token instead of splitting into fragments.
	require.Equal(t, []string{"dröplet", "resize"}, texts(parseQueryTerms("Dröplet resize")))
	// Single-character fragments are noise when other terms exist.
	require.Equal(t, []string{"droplet", "s3"}, texts(parseQueryTerms("droplet & s3 x")))
	// All stop words: keep them rather than return nothing.
	require.Equal(t, []string{"how", "do", "i"}, texts(parseQueryTerms("how do I")))
	require.Empty(t, parseQueryTerms("  ?!  "))
	require.Empty(t, parseQueryTerms(""))
	require.Len(t, parseQueryTerms(strings.Repeat("word ", maxQueryTerms+10)), maxQueryTerms)

	term := parseQueryTerms("configuration")[0]
	require.True(t, term.matches("how to configure cors"))
	require.False(t, term.matches("how to manage domains"))
}

func TestQueryTerm_MatchesExactTokens(t *testing.T) {
	term := parseQueryTerms("droplet")[0]
	for _, title := range []string{"droplet", "resize a droplet today", "droplet-based (droplet)", "my_droplet", "droplets and one droplet"} {
		require.True(t, term.matchesExactTokens(fieldTerms(title)), title)
	}
	for _, title := range []string{"droplets", ""} {
		require.False(t, term.matchesExactTokens(fieldTerms(title)), title)
	}
}

func TestTitleBoostUsesExactTokens(t *testing.T) {
	for _, tc := range []struct {
		query, title string
		titleMatch   bool
		exactMatch   bool
	}{
		{"droplet", "MY_DROPLET", true, true},
		{"configuration", "configure", true, false},
		{"postgres", "postgresql", true, false},
		{"spaces", "namespace", false, false},
		{"caf", "caf\u00e9", false, false},
		{"clair", "\u00e9clair", false, false},
	} {
		index := newDocsIndex([]DocsEntry{{Title: tc.title, URL: docsBase + "/x/"}}, nil, "llms-index.json")
		matches, _ := matchTerms(index, parseQueryTerms(tc.query))
		require.Equal(t, tc.titleMatch, matches[0][0].title, tc.query)
		require.Equal(t, tc.exactMatch, matches[0][0].titleWord, tc.query)
	}
	index := newDocsIndex([]DocsEntry{{Title: "caf\u00e9", URL: docsBase + "/x/"}}, nil, "llms-index.json")
	require.Empty(t, SearchIndex(index, "caf"), "the boost must not introduce a result with no matching token")
}

// TestRegressionSet_Fixture is the judged ranking regression set. Every
// query must place an expected page in the top 5 against the fixture
// universe (a 218-page sample of llms-index.json including each target and
// selected competitors; it does not represent the full live universe).
func TestRegressionSet_Fixture(t *testing.T) {
	index := loadFixtureIndex(t)
	queries := loadRegressionQueries(t)

	hits := 0
	reciprocal := 0.0
	for _, q := range queries {
		t.Run(q.Query, func(t *testing.T) {
			results := rankForTool(index, q)
			require.NotEmpty(t, results, "zero results")
			rank := rankOf(results, q.Expected)
			require.NotZero(t, rank, "expected %v not ranked; top-3: %v", q.Expected, topURLs(results, 3))
			require.LessOrEqual(t, rank, 5, "expected %v at rank %d; top-5: %v", q.Expected, rank, topURLs(results, 5))
			hits++
			reciprocal += 1 / float64(rank)
		})
	}
	t.Logf("fixture Recall@5=%d/%d MRR=%.3f", hits, len(queries), reciprocal/float64(len(queries)))
	require.GreaterOrEqual(t, reciprocal/float64(len(queries)), minimumFixtureMRR, "fixture MRR regressed")
}

func topURLs(results []DocsEntry, n int) []string {
	if len(results) < n {
		n = len(results)
	}
	urls := make([]string, n)
	for i := 0; i < n; i++ {
		urls[i] = canonicalDocURL(results[i].URL)
	}
	return urls
}

func TestSearchIndex_Deterministic(t *testing.T) {
	index := loadFixtureIndex(t)
	first := topURLs(SearchIndex(index, "droplet"), 10)
	for i := 0; i < 5; i++ {
		require.Equal(t, first, topURLs(SearchIndex(index, "droplet"), 10))
	}
}

func TestSearchIndex_RareTermOutweighsCommonOnes(t *testing.T) {
	index := loadFixtureIndex(t)

	// "app" and "platform" match dozens of titles; "520" identifies one page.
	results := SearchIndex(index, "App Platform 520")
	require.NotEmpty(t, results)
	require.Equal(t, "https://docs.digitalocean.com/support/why-am-i-receiving-520-status-codes-from-my-app/", results[0].URL)

	// The weight only applies to genuinely rare terms.
	_, weights := matchTerms(index, parseQueryTerms("droplet 520"))
	require.Equal(t, []int{1, rareTermWeight}, weights)
}

func TestSearchIndex_RequireAllTerms(t *testing.T) {
	index := loadFixtureIndex(t)

	// A generic token inside an unknown service name must not surface
	// unrelated pages when every term is required.
	require.NotEmpty(t, searchIndex(index, "nonexistent-service-xyz", false))
	require.Empty(t, searchIndex(index, "nonexistent-service-xyz", true))

	// Known names still resolve in strict mode.
	require.NotEmpty(t, searchIndex(index, "kubernetes", true))
	strict := searchIndex(index, "droplet pricing", true)
	require.NotEmpty(t, strict)
	require.Equal(t, "https://docs.digitalocean.com/products/droplets/details/pricing/", strict[0].URL)
}

func TestSearchIndex_PunctuationAndStemming(t *testing.T) {
	index := &DocsIndex{Entries: []DocsEntry{
		{Title: "How to Configure Cross-Origin Resource Sharing", URL: "https://docs.digitalocean.com/products/spaces/how-to/configure-cors/", Section: "Spaces"},
		{Title: "Why is SMTP blocked?", URL: "https://docs.digitalocean.com/support/why-is-smtp-blocked/", Section: "Support"},
		{Title: "Droplet Pricing", URL: "https://docs.digitalocean.com/products/droplets/details/pricing/", Section: "Droplets"},
	}}

	require.Equal(t, "Why is SMTP blocked?", SearchIndex(index, "SMTP blocked?")[0].Title)
	require.Equal(t, "How to Configure Cross-Origin Resource Sharing", SearchIndex(index, "cors configuration")[0].Title)
	require.Empty(t, SearchIndex(index, "!!!"))

	// The host part of the URL is not a match target.
	require.Empty(t, SearchIndex(index, "digitalocean"))
	require.Len(t, SearchIndex(index, "docs"), 0)
}

const fallbackLlmsTxt = "## Droplets\n\n- [Droplet Quickstart](https://docs.digitalocean.com/products/droplets/getting-started/quickstart/index.html.md): Start here.\n"

func TestGetDocsIndex_PrefersLlmsIndexJSON(t *testing.T) {
	fixture, err := os.ReadFile("testdata/llms-index.json")
	require.NoError(t, err)

	var txtHits atomic.Int32
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case llmsIndexPath:
			_, _ = w.Write(fixture)
		case llmsTxtPath:
			txtHits.Add(1)
			_, _ = w.Write([]byte(fallbackLlmsTxt))
		default:
			http.NotFound(w, r)
		}
	}))

	index, err := client.GetDocsIndex()
	require.NoError(t, err)
	require.Equal(t, "llms-index.json", index.Source)
	require.Greater(t, len(index.Entries), 100)
	require.Zero(t, txtHits.Load(), "llms.txt should not be fetched when llms-index.json works")
}

func TestGetDocsIndex_FallsBackToLlmsTxt(t *testing.T) {
	for name, indexHandler := range map[string]http.HandlerFunc{
		"missing": func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
		"invalid": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"oops":`)) },
	} {
		t.Run(name, func(t *testing.T) {
			client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case llmsIndexPath:
					indexHandler(w, r)
				case llmsTxtPath:
					_, _ = w.Write([]byte(fallbackLlmsTxt))
				default:
					http.NotFound(w, r)
				}
			}))

			index, err := client.GetDocsIndex()
			require.NoError(t, err)
			require.Equal(t, "llms.txt", index.Source)
			require.Len(t, index.Entries, 1)
		})
	}
}

func TestGetDocsIndex_ErrorWhenBothSourcesFail(t *testing.T) {
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))

	_, err := client.GetDocsIndex()
	require.Error(t, err)
	require.Contains(t, err.Error(), "llms-index.json")
}

func TestGetDocsIndex_ServesStaleOnRefreshFailure(t *testing.T) {
	var healthy atomic.Bool
	healthy.Store(true)
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !healthy.Load() {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		if r.URL.Path == llmsTxtPath {
			_, _ = w.Write([]byte(fallbackLlmsTxt))
			return
		}
		http.NotFound(w, r)
	}))

	first, err := client.GetDocsIndex()
	require.NoError(t, err)

	// Expire the cached copy and take the origin down: the stale index is served.
	client.cache.set("docsIndex", first, -time.Second)
	healthy.Store(false)

	again, err := client.GetDocsIndex()
	require.NoError(t, err)
	require.Same(t, first, again)

	// The stale copy is re-cached so the failing origin is not hit on every call.
	cached, ok := client.cache.get("docsIndex")
	require.True(t, ok)
	require.Same(t, first, cached)
}

func TestGetDocsIndex_CollapsesConcurrentColdLoads(t *testing.T) {
	var hits atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != llmsTxtPath {
			http.NotFound(w, r)
			return
		}
		if hits.Add(1) == 1 {
			close(entered)
		}
		<-release
		_, _ = w.Write([]byte(fallbackLlmsTxt))
	}))

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.GetDocsIndex()
			require.NoError(t, err)
		}()
	}
	// Wait for the shared origin load before releasing it.
	<-entered
	close(release)
	wg.Wait()

	// One request for the (404) JSON index and one for llms.txt counts as a single load.
	require.Equal(t, int32(1), hits.Load(), "concurrent cold loads should share one fetch")
}

func TestCanonicalDocURL(t *testing.T) {
	require.Equal(t, "https://docs.digitalocean.com/products/droplets/", canonicalDocURL("https://docs.digitalocean.com/products/droplets/index.html.md"))
	require.Equal(t, "https://docs.digitalocean.com/products/droplets/", canonicalDocURL("https://docs.digitalocean.com/products/droplets/"))
	require.True(t, strings.HasSuffix(canonicalDocURL("/support/x/"), "/"))
}
