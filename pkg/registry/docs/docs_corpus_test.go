package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	resizeURL   = docsBase + "/products/droplets/how-to/resize/"
	apiURL      = docsBase + "/reference/api/reference/droplets/"
	supportDir  = docsBase + "/support/"
	dropletsDir = docsBase + "/products/droplets/"
	apiDir      = docsBase + "/reference/api/reference/"
)

func shardLine(url, content string) string {
	rec, _ := json.Marshal(map[string]any{"kind": "doc", "canonical_url": url, "content": content})
	return string(rec) + "\n"
}

func manifestLine(dir string) string {
	return fmt.Sprintf(`{"doc_count":1,"kind":"shard","url":"%sdocs-corpus.jsonl"}`+"\n", dir)
}

func TestParseCorpusManifest(t *testing.T) {
	data := `{"kind":"corpus","shard_count":2}` + "\n" +
		`{"kind":"doc","canonical_url":"https://docs.digitalocean.com/","content":"home"}` + "\n" +
		manifestLine(supportDir) +
		manifestLine(dropletsDir) +
		`{"kind":"shard","url":"https://evil.example.com/docs-corpus.jsonl"}` + "\n" +
		"not json\n"

	m := parseCorpusManifest(data)
	require.Len(t, m.shardByDir, 2)
	require.Equal(t, supportDir+"docs-corpus.jsonl", m.shardByDir[supportDir])

	require.Equal(t, supportDir+"docs-corpus.jsonl", m.shardFor(smtpURL))
	require.Equal(t, dropletsDir+"docs-corpus.jsonl", m.shardFor(resizeURL), "walks up to the nearest ancestor shard")
	require.Equal(t, dropletsDir+"docs-corpus.jsonl", m.shardFor(resizeURL+"index.html.md"), "canonicalizes markdown URLs")
	require.Empty(t, m.shardFor(apiURL), "no ancestor shard")
	require.Empty(t, m.shardFor(docsBase+"/"))
}

func TestParseShard_TolerantOfTruncation(t *testing.T) {
	long := strings.Repeat("x", maxBodyBytes+100)
	data := shardLine(smtpURL, "SMTP ports are blocked.") +
		shardLine(resizeURL, long) +
		`{"kind":"doc","canonical_url":"` + apiURL + `","content":"cut mid-str`

	bodies := parseShard(data)
	require.Len(t, bodies, 2, "the truncated final line is skipped")
	require.Equal(t, "SMTP ports are blocked.", bodies[smtpURL])
	require.Len(t, bodies[resizeURL], maxBodyBytes, "bodies are capped")
}

func TestParseShardContentFormat(t *testing.T) {
	content := "# Heading\n![screenshot](https://example.com/image.png)\n[Docs](https://example.com/) **bold** _italic_ `code`"
	var data strings.Builder
	for i, format := range []string{"markdown", "text", ""} {
		record, err := json.Marshal(corpusDoc{
			CanonicalURL: fmt.Sprintf("%s/page%d/", docsBase, i), Content: content, ContentFormat: format,
		})
		require.NoError(t, err)
		data.Write(record)
		data.WriteByte('\n')
	}
	bodies := parseShard(data.String())
	require.Equal(t, "Heading\n\nDocs bold italic code", bodies[docsBase+"/page0/"])
	require.Equal(t, content, bodies[docsBase+"/page1/"])
	require.Equal(t, content, bodies[docsBase+"/page2/"])

	record, err := json.Marshal(corpusDoc{CanonicalURL: smtpURL, Content: strings.Repeat("*", maxBodyBytes) + "tail", ContentFormat: "markdown"})
	require.NoError(t, err)
	require.Empty(t, parseShard(string(record))[smtpURL], "truncate before stripping markup")
}

func TestCorpusSlotWaitDoesNotPoisonManifest(t *testing.T) {
	originalBudget := excerptBudget
	excerptBudget = 50 * time.Millisecond
	t.Cleanup(func() { excerptBudget = originalBudget })
	var hits atomic.Int32
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		fmt.Fprint(w, manifestLine(supportDir))
	}))
	for range shardConcurrency {
		client.slots() <- struct{}{}
	}
	for range maxConsecutiveTimeouts + 1 {
		require.Nil(t, client.getCorpusManifest(context.Background()))
		_, cached := client.cache.get("corpusManifest")
		require.False(t, cached, "waiting for capacity must not negative-cache the manifest")
		client.timeoutMu.Lock()
		timeouts := client.timeouts["corpusManifest"]
		client.timeoutMu.Unlock()
		require.Zero(t, timeouts, "local contention is not an origin timeout")
	}
	require.Zero(t, hits.Load())
	for range shardConcurrency {
		<-client.slots()
	}
	require.NotNil(t, client.getCorpusManifest(context.Background()), "load recovers as soon as a slot is available")
	require.EqualValues(t, 1, hits.Load())
}

func TestTruncateUTF8(t *testing.T) {
	require.Equal(t, "héllo", truncateUTF8("héllo", 10))
	require.Equal(t, "h", truncateUTF8("héllo", 2), "does not split a two-byte character")
	require.Equal(t, "hé", truncateUTF8("héllo", 3))
}

func TestExcerpt(t *testing.T) {
	body := "Intro paragraph that says nothing relevant at all.\n\n" +
		strings.Repeat("filler sentence about unrelated topics. ", 20) +
		"SMTP ports 25, 465, and 587 are blocked on Droplets to prevent spam. This block applies to all Droplets by default.\n" +
		strings.Repeat("more filler text here. ", 20)

	t.Run("picks the densest window and marks the cut", func(t *testing.T) {
		text := excerpt(body, parseQueryTerms("why is smtp blocked"))
		require.Contains(t, text, "SMTP ports 25, 465, and 587 are blocked")
		require.True(t, strings.HasPrefix(text, "..."), "window starts mid-page")
		require.True(t, strings.HasSuffix(text, "..."), "window ends mid-page")
		require.NotContains(t, text, "\n", "whitespace is collapsed")
		require.LessOrEqual(t, len(text), excerptLength+2*len("...")+excerptLead)
	})

	t.Run("prefers the window covering more distinct terms", func(t *testing.T) {
		body := "droplet droplet droplet droplet. " + strings.Repeat("pad ", 100) + "resize a droplet from the control panel."
		text := excerpt(body, parseQueryTerms("resize droplet"))
		require.Contains(t, text, "resize a droplet")
	})

	t.Run("matches stems", func(t *testing.T) {
		text := excerpt("You can configure CORS on a Space.", parseQueryTerms("cors configuration"))
		require.Contains(t, text, "configure CORS")
	})

	t.Run("empty when nothing matches", func(t *testing.T) {
		require.Empty(t, excerpt(body, parseQueryTerms("kubernetes")))
		require.Empty(t, excerpt("", parseQueryTerms("smtp")))
	})

	t.Run("short bodies are returned whole", func(t *testing.T) {
		require.Equal(t, "SMTP is blocked.", excerpt("SMTP is blocked.", parseQueryTerms("smtp")))
	})
}

// corpusServer serves a manifest and shards entirely in memory. The client
// still uses http.Client, exercising redirects, status and body handling.
type corpusServer struct {
	client *DocsClient
	hits   map[string]*atomic.Int32
}

func newCorpusServer(t *testing.T, shards map[string]string) *corpusServer {
	t.Helper()
	cs := &corpusServer{hits: make(map[string]*atomic.Int32)}
	mux := http.NewServeMux()
	var manifest strings.Builder
	manifest.WriteString(`{"kind":"corpus"}` + "\n")
	for dir := range shards {
		manifest.WriteString(manifestLine(docsBase + dir))
	}
	cs.hits[corpusManifestPath] = new(atomic.Int32)
	mux.HandleFunc(corpusManifestPath, func(w http.ResponseWriter, _ *http.Request) {
		cs.hits[corpusManifestPath].Add(1)
		fmt.Fprint(w, manifest.String())
	})
	for dir, body := range shards {
		path := dir + "docs-corpus.jsonl"
		cs.hits[path] = new(atomic.Int32)
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			cs.hits[path].Add(1)
			if body == "oversized" {
				w.Header().Set("Content-Length", fmt.Sprint(maxResponseSize+1))
				w.WriteHeader(http.StatusOK)
				return
			}
			if body == "missing" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			fmt.Fprint(w, body)
		})
	}
	cs.client = newHandlerClient(mux)
	return cs
}

func TestContentExcerpts(t *testing.T) {
	server := newCorpusServer(t, map[string]string{
		"/support/":           shardLine(smtpURL, "Intro. SMTP ports 25, 465, and 587 are blocked on Droplets to prevent spam."),
		"/products/droplets/": shardLine(resizeURL, "Resize a Droplet to change its CPU, RAM, and disk."),
		"/reference/api/":     "oversized",
		"/products/spaces/":   "missing",
	})
	client := server.client

	entries := []DocsEntry{
		{URL: smtpURL},
		{URL: resizeURL + "index.html.md"},
		{URL: apiURL},
		{URL: docsBase + "/products/spaces/how-to/configure-cors/"},
		{URL: docsBase + "/products/kubernetes/"}, // no shard at all
	}

	excerpts := client.ContentExcerpts(context.Background(), entries, "smtp blocked droplet")
	require.Len(t, excerpts, 2)
	require.Contains(t, excerpts[smtpURL], "SMTP ports 25, 465, and 587 are blocked")
	require.Contains(t, excerpts[resizeURL], "Resize a Droplet", "markdown URL entries are keyed by canonical URL")

	// Second query: everything, including the oversized and missing shards,
	// is answered from cache.
	client.ContentExcerpts(context.Background(), entries, "resize")
	for path, hits := range server.hits {
		require.EqualValues(t, 1, hits.Load(), "unexpected re-fetch of %s", path)
	}

	t.Run("manifest failure degrades to no excerpts", func(t *testing.T) {
		client := newHandlerClient(http.NotFoundHandler())
		require.Nil(t, client.ContentExcerpts(context.Background(), entries, "smtp"))
		require.Nil(t, client.ContentExcerpts(context.Background(), entries, "smtp"))
	})

	t.Run("no terms or entries", func(t *testing.T) {
		require.Nil(t, client.ContentExcerpts(context.Background(), nil, "smtp"))
		require.Nil(t, client.ContentExcerpts(context.Background(), entries, "???"))
	})
}

func TestGroupByShard_CapsFanOut(t *testing.T) {
	m := &corpusManifest{shardByDir: make(map[string]string)}
	var entries []DocsEntry
	for i := 0; i < maxShardsPerQuery+3; i++ {
		dir := fmt.Sprintf("%s/products/p%d/", docsBase, i)
		m.shardByDir[dir] = dir + "docs-corpus.jsonl"
		entries = append(entries, DocsEntry{URL: dir + "how-to/"}, DocsEntry{URL: dir + "details/"})
	}

	shards := groupByShard(m, entries)
	require.Len(t, shards, maxShardsPerQuery)
	for i, shard := range shards {
		require.Equal(t, fmt.Sprintf("%s/products/p%d/docs-corpus.jsonl", docsBase, i), shard.url, "rank order is preserved")
		require.Len(t, shard.pages, 2, "pages sharing a shard are batched")
	}
}

func TestFetchUncached_RejectsOversizedByContentLength(t *testing.T) {
	client := newHandlerClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(maxResponseSize+1))
		w.WriteHeader(http.StatusOK)
	}))

	_, err := client.fetchUncached(docsBase + "/oversized/")
	require.ErrorContains(t, err, "limit is")
}

func TestFilterByProduct(t *testing.T) {
	index := loadFixtureIndex(t)

	cases := []struct {
		product string
		wantURL string
		reject  string
	}{
		{"kubernetes", "/products/kubernetes/", "/products/droplets/"},
		{"k8s", "/products/kubernetes/", "/products/droplets/"},
		{"App-Platform", "/products/app-platform/", "/support/why-is-smtp-blocked/"},
		{"vpc", "/products/networking/vpc/", "/products/droplets/"},
		{"support", "/support/", "/products/droplets/how-to/"},
		{"api", "/reference/api/", "/products/droplets/how-to/"},
	}
	for _, tc := range cases {
		t.Run(tc.product, func(t *testing.T) {
			filtered := filterByProduct(index, tc.product)
			require.NotEmpty(t, filtered.Entries)
			require.Equal(t, "llms-index.json", filtered.Source)
			var sawWant bool
			for _, e := range filtered.Entries {
				path := urlPathLower(e.URL)
				require.False(t, strings.HasPrefix(path, tc.reject), "unexpected %s", e.URL)
				sawWant = sawWant || strings.HasPrefix(path, tc.wantURL)
			}
			require.True(t, sawWant, "no %s pages", tc.wantURL)
		})
	}

	require.Same(t, index, filterByProduct(index, " "), "blank product is a no-op")
	require.Empty(t, filterByProduct(index, "nonexistent-product").Entries)
	require.Equal(t, "kubernetes", normalizeFacet("  Kubernetes "))
	require.Equal(t, "app platform", normalizeFacet("App-Platform"))
}

func TestPaginate(t *testing.T) {
	content := "héllo wörld, this is a page"

	page, next := paginate(content, 0, 0)
	require.Equal(t, content, page)
	require.Equal(t, len(content), next)

	page, next = paginate(content, 0, 2)
	require.Equal(t, "h", page, "does not split a two-byte character")
	require.Equal(t, 1, next)

	page, next = paginate(content, next, 10)
	require.Equal(t, "éllo wör", page, "10 bytes, two of them multi-byte runes")
	require.Equal(t, 11, next)

	page, next = paginate(content, 2, 100)
	require.Equal(t, "llo wörld, this is a page", page, "a start inside a character snaps forward to the next rune")
	require.Equal(t, len(content), next)

	page, next = paginate(content, len(content)+5, 10)
	require.Empty(t, page)
	require.Equal(t, len(content), next)
}

func TestSearchDocs_ExcerptsAndProductFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	index := loadFixtureIndex(t)
	mock := NewMockDocsService(ctrl)
	mock.EXPECT().GetDocsIndex().Return(index, nil).AnyTimes()
	mock.EXPECT().ContentExcerpts(gomock.Any(), gomock.Any(), "resize").DoAndReturn(
		func(_ context.Context, entries []DocsEntry, _ string) map[string]string {
			require.LessOrEqual(t, len(entries), 3, "only the returned results are enriched")
			return map[string]string{canonicalDocURL(entries[0].URL): "...Resize a Droplet to add CPU and RAM..."}
		})
	tool := setupDocsToolWithMock(mock)

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"Query": "resize", "Product": "droplets", "Limit": float64(3)}}}
	resp, err := tool.searchDocs(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	text := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, text, `pages in "droplets")`)
	require.Contains(t, text, "...Resize a Droplet to add CPU and RAM...", "body excerpt replaces the description")
	require.NotContains(t, text, "/products/kubernetes/")

	t.Run("unknown product", func(t *testing.T) {
		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"Query": "resize", "Product": "nonexistent"}}}
		resp, err := tool.searchDocs(context.Background(), req)
		require.NoError(t, err)
		require.Contains(t, resp.Content[0].(mcp.TextContent).Text, `No docs pages match Product "nonexistent"`)
	})
}

func TestGetDoc_Pagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const url = "https://docs.digitalocean.com/products/droplets/"
	content := strings.Repeat("0123456789", 10)
	mock := NewMockDocsService(ctrl)
	mock.EXPECT().FetchDocPage(url).Return(content, nil).AnyTimes()
	tool := setupDocsToolWithMock(mock)

	call := func(args map[string]any) *mcp.CallToolResult {
		args["URL"] = url
		resp, err := tool.getDoc(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
		require.NoError(t, err)
		return resp
	}

	text := call(map[string]any{}).Content[0].(mcp.TextContent).Text
	require.Equal(t, content, text, "default is the whole page, no footer")

	text = call(map[string]any{"MaxLength": float64(30)}).Content[0].(mcp.TextContent).Text
	require.True(t, strings.HasPrefix(text, content[:30]))
	require.Contains(t, text, "[Truncated: showing bytes up to 30 of 100. Call docs-get-page with URL="+url+" and StartIndex=30 for the rest.]")

	text = call(map[string]any{"MaxLength": float64(30), "StartIndex": float64(90)}).Content[0].(mcp.TextContent).Text
	require.Equal(t, content[90:], text, "final window has no footer")

	resp := call(map[string]any{"StartIndex": float64(100)})
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content[0].(mcp.TextContent).Text, "past the end")
}

func TestTroubleshoot_BudgetsEmbeddedPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	entries := []DocsEntry{
		{Title: "Why is SMTP blocked?", URL: smtpURL, Section: "Support"},
		{Title: "Resize", URL: resizeURL, Description: "Resize description.", Section: "Droplets"},
	}
	long := strings.Repeat("word ", maxEmbeddedPageBytes/5+200)
	mock := NewMockDocsService(ctrl)
	mock.EXPECT().FindTroubleshootPage("smtp").Return(entries, nil)
	mock.EXPECT().FetchDocPage(smtpURL).Return(long, nil)
	mock.EXPECT().ContentExcerpts(gomock.Any(), entries[1:], "smtp").Return(map[string]string{resizeURL: "...body excerpt..."})
	tool := setupDocsToolWithMock(mock)

	resp, err := tool.troubleshoot(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"Symptom": "smtp"}}})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	text := resp.Content[0].(mcp.TextContent).Text
	require.Less(t, len(text), len(long), "embedded page is cut")
	require.Contains(t, text, fmt.Sprintf("Call docs-get-page with URL=%s and StartIndex=%d", smtpURL, maxEmbeddedPageBytes))
	require.Contains(t, text, "2. Resize\n")
	require.Contains(t, text, "...body excerpt...")
	require.NotContains(t, text, "Resize description.")
}
