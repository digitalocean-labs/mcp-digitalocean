package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func TestParseLimit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  int
	}{
		{"absent", nil, 5},
		{"string", "25", 5},
		{"integer", 25, 5},
		{"fraction below one", 0.5, 5},
		{"zero", 0.0, 5},
		{"negative", -1.0, 5},
		{"nan", math.NaN(), 5},
		{"fraction", 2.9, 2},
		{"valid", 25.0, 25},
		{"large", 100000.0, 50},
		{"max float", math.MaxFloat64, 50},
		{"positive infinity", math.Inf(1), 50},
		{"negative infinity", math.Inf(-1), 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{}
			if tc.value != nil {
				args["Limit"] = tc.value
			}
			require.Equal(t, tc.want, parseLimit(args, 5))
		})
	}
	require.Equal(t, defaultSearchLimit, parseLimit(nil, defaultSearchLimit))
}

func TestTroubleshootFractionalLimit(t *testing.T) {
	mock := NewMockDocsService(gomock.NewController(t))
	mock.EXPECT().FindTroubleshootPage("SMTP blocked").Return([]DocsEntry{{Title: "SMTP", URL: smtpURL}}, nil)
	mock.EXPECT().FetchDocPage(smtpURL).Return("SMTP is blocked.", nil)
	tool := setupDocsToolWithMock(mock)
	var response *mcp.CallToolResult
	require.NotPanics(t, func() {
		var err error
		response, err = tool.troubleshoot(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{"Symptom": "SMTP blocked", "Limit": 0.5}},
		})
		require.NoError(t, err)
	})
	require.False(t, response.IsError)
	require.Contains(t, response.Content[0].(mcp.TextContent).Text, "SMTP is blocked.")
}

func TestNormalizePageURL(t *testing.T) {
	for _, raw := range []string{
		"/products/droplets/", "products/droplets/", " /products/droplets/ ",
		docsBase + "/products/droplets/?a=1#x", "/products/x/../droplets/",
		"/products//droplets/", "/products/%64roplets/",
	} {
		got, err := normalizePageURL(raw)
		require.NoError(t, err, "%q", raw)
		require.Equal(t, docsBase+"/products/droplets/", got)
	}
	got, err := normalizePageURL("//example.com/x")
	require.NoError(t, err)
	require.Equal(t, docsBase+"/example.com/x", got)
	got, err = normalizePageURL(docsBase)
	require.NoError(t, err)
	require.Equal(t, docsBase+"/", got)
	got, err = normalizePageURL("/a%20b")
	require.NoError(t, err)
	require.Equal(t, docsBase+"/a%20b", got)
	for _, raw := range []string{
		"https://example.com/x", "https://docs.digitalocean.com.evil.example/x",
		"http://docs.digitalocean.com/x", "https://user@docs.digitalocean.com/x",
		"https://docs.digitalocean.com@evil.example/x", "https://docs.digitalocean.com:443/x",
		"https://docs.digitalocean.com:8443/x", "ftp://docs.digitalocean.com/x",
		"/x\\y", "/x\ny", "/x\x00y", "/x\ry", "/x\ty", "/bad%escape", "/x/\n",
	} {
		_, err := normalizePageURL(raw)
		require.Error(t, err, "%q", raw)
	}
}

func TestPageCacheKeyCanonical(t *testing.T) {
	var hits int
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		require.Equal(t, "/products/droplets/index.html.md", r.URL.Path)
		require.Empty(t, r.URL.RawQuery)
		fmt.Fprint(w, "# Droplet docs")
	}))
	for _, raw := range []string{
		"/products/droplets/?a=1", "/products/droplets/?a=2", "/products/droplets/#x",
		"/products/x/../droplets/", "products/droplets/", docsBase + "/products/droplets/",
	} {
		body, err := client.FetchDocPage(raw)
		require.NoError(t, err)
		require.Equal(t, "# Droplet docs", body)
	}
	require.Equal(t, 1, hits)
	require.Len(t, client.cache.store, 2, "one canonical page entry and one fetch entry")
}

func TestCacheBounded(t *testing.T) {
	c := newCache()
	c.set("soonest", "keep until capacity", time.Minute)
	for i := 1; i < maxCacheEntries; i++ {
		c.set(fmt.Sprint(i), i, time.Hour)
	}
	c.set("1", "replacement", time.Hour)
	require.Len(t, c.store, maxCacheEntries)
	_, ok := c.get("soonest")
	require.True(t, ok, "replacing a key must not evict another entry")
	c.set("new", "new", time.Hour)
	require.Len(t, c.store, maxCacheEntries)
	_, ok = c.get("soonest")
	require.False(t, ok, "evict the entry closest to expiry")
	for i := range 2 * maxCacheEntries {
		c.set(fmt.Sprintf("extra:%d", i), i, time.Hour)
		require.LessOrEqual(t, len(c.store), maxCacheEntries)
	}
}

func TestCacheEvictsExpiredFirst(t *testing.T) {
	c := newCache()
	c.set("soonest", "live", time.Minute)
	for i := 1; i < maxCacheEntries-2; i++ {
		c.set(fmt.Sprint(i), i, time.Hour)
	}
	c.set("expired1", nil, -time.Second)
	c.set("expired2", nil, -time.Second)
	require.Len(t, c.store, maxCacheEntries)
	c.set("new", "new", time.Hour)
	_, ok := c.store["expired1"]
	require.False(t, ok)
	_, ok = c.store["expired2"]
	require.False(t, ok)
	_, ok = c.get("soonest")
	require.True(t, ok)
	require.Len(t, c.store, maxCacheEntries-1)
}

func TestConcurrentColdPageFetches(t *testing.T) {
	var hits atomic.Int32
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		time.Sleep(20 * time.Millisecond)
		fmt.Fprint(w, "# Shared page")
	}))
	start := make(chan struct{})
	errs := make(chan error, 8)
	var ready, done sync.WaitGroup
	ready.Add(8)
	done.Add(8)
	for range 8 {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			body, err := client.FetchDocPage("/shared/")
			if err == nil && body != "# Shared page" {
				err = fmt.Errorf("unexpected page %q", body)
			}
			errs <- err
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, hits.Load())
}

func TestRelatedLinksOriginPolicy(t *testing.T) {
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[suffix](https://docs.digitalocean.com.evil.example/x/)
[userinfo destination](https://docs.digitalocean.com@evil.example/x/)
[userinfo](https://user@docs.digitalocean.com/x/)
[port](https://docs.digitalocean.com:443/x/)
[http](http://docs.digitalocean.com/x/)
[good](https://docs.digitalocean.com/products/droplets/)`)
	}))
	links, err := client.ExtractRelatedLinks("/source/")
	require.NoError(t, err)
	require.Equal(t, []RelatedLink{{URL: docsBase + "/products/droplets/", Title: "good", Category: "other"}}, links)
}

func TestServiceSlugValidation(t *testing.T) {
	for _, slug := range []string{"droplets", "networking/vpc", "one/two/three", "a0-b", strings.Repeat("a", 64)} {
		require.True(t, validServiceSlug(slug), "%q", slug)
	}
	var hits atomic.Int32
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		http.Error(w, "unexpected request", http.StatusNotFound)
	}))
	for _, slug := range []string{"", "../..", "a?b", "a#b", "a/b/c/d", "/a", "a/", "a//b", "-a", "A", strings.Repeat("a", 65)} {
		require.False(t, validServiceSlug(slug), "%q", slug)
		if slug == "A" {
			continue // Service names are lowercased before validation.
		}
		index, err := client.GetServiceIndex(slug)
		require.NoError(t, err)
		require.Nil(t, index)
		_, _, err = client.FindQuickstart(slug)
		require.Error(t, err)
	}
	require.Zero(t, hits.Load())
}

func TestIndexTextAndURLPolicy(t *testing.T) {
	records := []llmsIndexRecord{
		{Title: "Title\nwith&#10;controls\x00", Description: "One\r\n two\t&amp; three", Product: "App&#10;Platform", URL: smtpURL, MarkdownURL: "https://evil.example/x", LastMod: "2026-02-30T00:00:00Z"},
		{Title: "Section", Section: "Getting\nstarted", URL: docsBase + "/valid/", MarkdownURL: docsBase + "/valid/index.html.md", LastMod: "2026-10-21T12:00:00Z"},
	}
	for _, url := range []string{"https://evil.example/x", docsBase + ".evil.example/x", "https://user@docs.digitalocean.com/x", docsBase + ":443/x", "http://docs.digitalocean.com/x"} {
		records = append(records, llmsIndexRecord{Title: "Bad", URL: url})
	}
	data, err := json.Marshal(records)
	require.NoError(t, err)
	index, err := parseLlmsIndex(data)
	require.NoError(t, err)
	require.Len(t, index.Entries, 2)
	require.Equal(t, "Title with controls", index.Entries[0].Title)
	require.Equal(t, "One two & three", index.Entries[0].Description)
	require.Equal(t, "App Platform", index.Entries[0].Section)
	require.Empty(t, index.Entries[0].MarkdownURL)
	require.Empty(t, index.Entries[0].LastModified)
	require.Equal(t, "Getting started", index.Entries[1].Section)
	require.Equal(t, docsBase+"/valid/index.html.md", index.Entries[1].MarkdownURL)
	require.Equal(t, "2026-10-21", index.Entries[1].LastModified)

	txt := "## App&#10;Platform\n- [Title&#10;with&#9;controls](" + smtpURL + "): One&#13;&#10;two &amp; three\n"
	for _, record := range records[2:] {
		txt += "- [Bad](" + record.URL + "): skip\n"
	}
	index = parseLlmsTxt(txt)
	require.Len(t, index.Entries, 1)
	require.Equal(t, "Title with controls", index.Entries[0].Title)
	require.Equal(t, "One two & three", index.Entries[0].Description)
	require.Equal(t, "App Platform", index.Entries[0].Section)
}

func TestLastModDate(t *testing.T) {
	for _, raw := range []string{"", "garbage", "2026-02-30", "2026-13-01", "2026-1-01"} {
		require.Empty(t, lastModDate(raw), "%q", raw)
	}
	require.Equal(t, "2024-02-29", lastModDate("2024-02-29"))
	require.Equal(t, "2026-10-21", lastModDate("2026-10-21T00:00:00Z"))
}

func TestOversizedShardIsNotCachedAsPartialSuccess(t *testing.T) {
	var hits int
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		fmt.Fprint(w, shardLine(smtpURL, "SMTP body"), strings.Repeat(" ", testResponseCap))
	}))
	client.maxResponseSize = testResponseCap
	for range 2 {
		require.Nil(t, client.loadShard(context.Background(), supportDir+"docs-corpus.jsonl"))
	}
	require.Equal(t, 1, hits, "oversized feed is negative-cached, not stored as a partial map")
}

func TestPageWindowsRejectInvalidNumbersAndMakeProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockDocsService(ctrl)
	mock.EXPECT().FetchDocPage("/x/").Return("éabc", nil).AnyTimes()
	tool := setupDocsToolWithMock(mock)
	call := func(args map[string]any) *mcp.CallToolResult {
		args["URL"] = "/x/"
		response, err := tool.getDoc(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
		require.NoError(t, err)
		return response
	}
	for _, name := range []string{"StartIndex", "MaxLength"} {
		for _, value := range []any{-1.0, 0.5, math.Inf(1), math.NaN(), math.Nextafter(math.Exp2(63), 0), "10"} {
			require.True(t, call(map[string]any{name: value}).IsError, "%s=%v", name, value)
		}
	}
	response := call(map[string]any{"MaxLength": 1.0})
	require.True(t, response.IsError)
	require.Contains(t, response.Content[0].(mcp.TextContent).Text, "too small")
	response = call(map[string]any{"MaxLength": 2.0})
	require.False(t, response.IsError)
	text := response.Content[0].(mcp.TextContent).Text
	require.True(t, strings.HasPrefix(text, "é\n"))
	require.Contains(t, text, "StartIndex=2")
	response = call(map[string]any{"StartIndex": 2.0, "MaxLength": 1.0})
	require.False(t, response.IsError)
	require.True(t, strings.HasPrefix(response.Content[0].(mcp.TextContent).Text, "a\n"))
	page, next := paginate("hello", 1, int(^uint(0)>>1))
	require.Equal(t, "ello", page)
	require.Equal(t, 5, next, "utility arithmetic also handles max-int without overflow")
}

func TestExcerptsPreserveUnicodeAndMatches(t *testing.T) {
	for _, body := range []string{
		"İ: Export API_KEY=VALUE before using SMTP.",
		"İ" + strings.Repeat("a ", 20) + "SMTP " + strings.Repeat("x ", 200) + "Ⱥ",
		strings.Repeat("界", 40) + " SMTP " + strings.Repeat("界", 130),
		strings.Repeat("a ", 20) + "SMTP-" + strings.Repeat("x", 400),
	} {
		text := excerpt(body, parseQueryTerms("smtp"))
		require.NotEmpty(t, text, "body=%q", body)
		require.True(t, utf8.ValidString(text), "%q", text)
		require.Contains(t, text, "SMTP", "snapping must preserve a complete query hit and source case")
		require.LessOrEqual(t, len(text), excerptLength+2*len("..."))
		if strings.Contains(body, "API_KEY") {
			require.Contains(t, text, "API_KEY=VALUE")
		}
	}
	require.Empty(t, excerpt("namespace access", parseQueryTerms("spaces")))
	// A very long token is not a complete keyword match.
	require.Empty(t, excerpt(strings.Repeat("a ", 20)+"SMTP"+strings.Repeat("x", 400), parseQueryTerms("smtp")))
}

func waitFor[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for test synchronization")
		var zero T
		return zero
	}
}

func TestContentExcerpts_CancellationDoesNotPoisonSharedLoads(t *testing.T) {
	for _, blockPath := range []string{corpusManifestPath, "/support/docs-corpus.jsonl"} {
		t.Run(blockPath, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
			var manifestHits, shardHits atomic.Int32
			client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == corpusManifestPath {
					manifestHits.Add(1)
				} else {
					shardHits.Add(1)
				}
				if r.URL.Path == blockPath {
					close(entered)
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
				}
				if r.URL.Path == corpusManifestPath {
					fmt.Fprint(w, manifestLine(supportDir))
				} else {
					fmt.Fprint(w, shardLine(smtpURL, "SMTP body"))
				}
			}))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan map[string]string, 1)
			entries := []DocsEntry{{URL: smtpURL}}
			go func() { done <- client.ContentExcerpts(ctx, entries, "smtp") }()
			waitFor(t, entered)
			cancel()
			require.Empty(t, waitFor(t, done), "caller stops waiting without killing the shared load")
			healthy := make(chan map[string]string, 1)
			go func() { healthy <- client.ContentExcerpts(context.Background(), entries, "smtp") }()
			releaseOnce.Do(func() { close(release) })
			require.Contains(t, waitFor(t, healthy), smtpURL)
			require.EqualValues(t, 1, manifestHits.Load())
			require.EqualValues(t, 1, shardHits.Load())
		})
	}
}

func TestCorpusCancelledLoadsAreRetried(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(cause.Error(), func(t *testing.T) {
			client := NewDocsClient()
			var hits atomic.Int32
			client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if hits.Add(1) == 1 {
					return nil, cause
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), ContentLength: -1,
					Body: io.NopCloser(strings.NewReader(manifestLine(supportDir))), Request: r}, nil
			})
			require.Nil(t, client.getCorpusManifest(context.Background()))
			require.NotNil(t, client.getCorpusManifest(context.Background()))
			require.EqualValues(t, 2, hits.Load())
		})
	}
}

func TestContentExcerpts_CoalescesConcurrentColdQueries(t *testing.T) {
	server := newCorpusServer(t, map[string]string{"/support/": shardLine(smtpURL, "SMTP body")})
	var wg sync.WaitGroup
	results := make(chan map[string]string, 20)
	start := make(chan struct{})
	for range 20 {
		wg.Go(func() {
			<-start
			results <- server.client.ContentExcerpts(context.Background(), []DocsEntry{{URL: smtpURL}}, "smtp")
		})
	}
	close(start)
	wg.Wait()
	close(results)
	for result := range results {
		require.Contains(t, result, smtpURL)
	}
	for path, hits := range server.hits {
		require.EqualValues(t, 1, hits.Load(), path)
	}
}

func TestContentExcerpts_OrdersFirstWorkerBatchAndBoundsAllQueries(t *testing.T) {
	entered := make(chan string, 12)
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var active, maximum atomic.Int32
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		for old := maximum.Load(); current > old && !maximum.CompareAndSwap(old, current); old = maximum.Load() {
		}
		entered <- r.URL.Path
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		page := strings.TrimSuffix(docsBase+r.URL.Path, "docs-corpus.jsonl") + "page/"
		fmt.Fprint(w, shardLine(page, "SMTP body"))
	}))
	m := &corpusManifest{shardByDir: make(map[string]string)}
	var entries []DocsEntry
	for i := range 12 {
		dir := fmt.Sprintf("%s/p%d/", docsBase, i)
		m.shardByDir[dir] = dir + "docs-corpus.jsonl"
		entries = append(entries, DocsEntry{URL: dir + "page/"})
	}
	client.cache.set("corpusManifest", m, indexCacheTTL)
	first := make(chan map[string]string, 1)
	go func() { first <- client.ContentExcerpts(context.Background(), entries[:8], "smtp") }()
	batch := make(map[string]bool)
	for range shardConcurrency {
		batch[waitFor(t, entered)] = true
	}
	for i := range shardConcurrency {
		require.True(t, batch[fmt.Sprintf("/p%d/docs-corpus.jsonl", i)], "top-ranked shard must start in the first batch")
	}
	var wg sync.WaitGroup
	for _, entry := range entries[8:] {
		wg.Go(func() { client.ContentExcerpts(context.Background(), []DocsEntry{entry}, "smtp") })
	}
	releaseOnce.Do(func() { close(release) })
	require.Len(t, waitFor(t, first), 8)
	wg.Wait()
	require.EqualValues(t, shardConcurrency, maximum.Load(), "bound is shared across simultaneous queries")
}

func TestCorpusRepeatedTimeoutsAreNegativeCached(t *testing.T) {
	client := NewDocsClient()
	var hits atomic.Int32
	client.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		hits.Add(1)
		return nil, context.DeadlineExceeded
	})
	for range maxConsecutiveTimeouts {
		require.Nil(t, client.getCorpusManifest(context.Background()))
	}
	require.EqualValues(t, maxConsecutiveTimeouts, hits.Load())
	require.Nil(t, client.getCorpusManifest(context.Background()))
	require.EqualValues(t, maxConsecutiveTimeouts, hits.Load(), "a persistently slow origin is negative-cached")

	_, cached := client.cache.get("corpusManifest")
	require.True(t, cached)
}

func TestCorpusSuccessResetsTimeoutCount(t *testing.T) {
	client := NewDocsClient()
	var hits atomic.Int32
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if hits.Add(1)%2 == 1 {
			return nil, context.DeadlineExceeded
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), ContentLength: -1,
			Body: io.NopCloser(strings.NewReader(manifestLine(supportDir))), Request: r}, nil
	})
	for range maxConsecutiveTimeouts + 1 {
		require.Nil(t, client.getCorpusManifest(context.Background()))
		require.NotNil(t, client.getCorpusManifest(context.Background()))
		client.cache.set("corpusManifest", (*corpusManifest)(nil), -time.Second)
	}
	require.EqualValues(t, 2*(maxConsecutiveTimeouts+1), hits.Load(), "alternating stalls never trip the negative cache")
}

func TestZeroValueClientDoesNotDeadlockOnSlots(t *testing.T) {
	client := &DocsClient{cache: newCache()}
	client.httpClient = NewDocsClient().httpClient
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), ContentLength: -1,
			Body: io.NopCloser(strings.NewReader(manifestLine(supportDir))), Request: r}, nil
	})
	done := make(chan *corpusManifest, 1)
	go func() { done <- client.getCorpusManifest(context.Background()) }()
	select {
	case m := <-done:
		require.NotNil(t, m)
	case <-time.After(time.Second):
		t.Fatal("corpus load blocked on an uninitialised slot channel")
	}
}

func TestFetchDocPagePrefersMarkdownURL(t *testing.T) {
	for _, tc := range []struct {
		name       string
		markdown   string
		request    string
		stale      bool
		failCustom bool
		failFixed  bool
		wantPaths  []string
	}{
		{"metadata", docsBase + "/alternate/page.md", "/page/", false, false, false, []string{"/alternate/page.md"}},
		{"stale metadata", docsBase + "/alternate/page.md", "/page/", true, false, false, []string{"/alternate/page.md"}},
		{"canonical request", docsBase + "/alternate/page.md", "/x/../page/?a=1#x", false, false, false, []string{"/alternate/page.md"}},
		{"markdown request", docsBase + "/alternate/page.md", "/alternate/page.md", false, false, false, []string{"/alternate/page.md"}},
		{"metadata missing", "", "/page/", false, false, false, []string{"/page/index.html.md"}},
		{"metadata unavailable", docsBase + "/alternate/page.md", "/page/", false, true, false, []string{"/alternate/page.md", "/page/index.html.md"}},
		{"original fallback", docsBase + "/alternate/page.md", "/page/", false, true, true, []string{"/alternate/page.md", "/page/index.html.md", "/page/"}},
		{"duplicate fallback", docsBase + "/page/index.html.md", "/page/", false, false, true, []string{"/page/index.html.md", "/page/"}},
		{"off origin", "https://evil.example/page.md", "/page/", false, false, false, []string{"/page/index.html.md"}},
		{"off origin redirect", docsBase + "/alternate/page.md", "/page/", false, true, false, []string{"/alternate/page.md", "/page/index.html.md"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				require.Equal(t, "docs.digitalocean.com", r.URL.Host)
				if tc.name == "off origin redirect" && r.URL.Path == "/alternate/page.md" {
					http.Redirect(w, r, "https://evil.example/page.md", http.StatusFound)
					return
				}
				if (tc.failCustom && r.URL.Path == "/alternate/page.md") || (tc.failFixed && r.URL.Path == "/page/index.html.md") {
					http.NotFound(w, r)
					return
				}
				fmt.Fprint(w, "# Page")
			}))
			index := &DocsIndex{Entries: []DocsEntry{{Title: "Page", URL: docsBase + "/page/", MarkdownURL: tc.markdown}}}
			if tc.stale {
				client.staleIndex.Store(index)
			} else {
				client.cache.set("docsIndex", index, indexCacheTTL)
			}
			for range 2 {
				content, err := client.FetchDocPage(tc.request)
				require.NoError(t, err)
				require.Equal(t, "# Page", content)
			}
			require.Equal(t, tc.wantPaths, paths, "page reads must not download the index")
		})
	}
}
