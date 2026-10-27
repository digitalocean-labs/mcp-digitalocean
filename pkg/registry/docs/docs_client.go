package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	neturl "net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"golang.org/x/sync/singleflight"
)

const (
	docsBase      = "https://docs.digitalocean.com"
	llmsTxtPath   = "/llms.txt"
	llmsIndexPath = "/llms-index.json"
	userAgent     = "mcp-digitalocean-docs/1.0"

	indexCacheTTL    = 1 * time.Hour
	pageCacheTTL     = 30 * time.Minute
	negativeCacheTTL = 10 * time.Minute

	maxResponseSize = 10 * 1024 * 1024 // 10 MB
	maxCacheEntries = 256
)

// DocsEntry represents a single documentation page known to the index.
type DocsEntry struct {
	Title        string
	URL          string // canonical page URL
	Description  string
	Section      string // product or top-level grouping, e.g. "Droplets", "Support"
	MarkdownURL  string // index.html.md variant when known, otherwise empty
	LastModified string // ISO-8601 date when known, otherwise empty
}

// DocsIndex is the searchable candidate universe of documentation pages.
type DocsIndex struct {
	Entries   []DocsEntry
	Sections  []string
	Source    string // "llms-index.json" or "llms.txt"
	FetchedAt time.Time

	// tokens is parallel to Entries. It is populated by newDocsIndex; an
	// index built as a literal (tests) is tokenized on the fly instead.
	tokens []entryTokens
}

// newDocsIndex builds an index and tokenizes its entries once.
func newDocsIndex(entries []DocsEntry, sections []string, source string) *DocsIndex {
	tokens := make([]entryTokens, len(entries))
	for i, entry := range entries {
		tokens[i] = tokenizeEntry(entry)
	}
	return &DocsIndex{
		Entries:   entries,
		Sections:  sections,
		Source:    source,
		FetchedAt: time.Now(),
		tokens:    tokens,
	}
}

// entryTokensAt returns the cached tokens for entry i, tokenizing on demand
// when the index was not built with newDocsIndex.
func (index *DocsIndex) entryTokensAt(i int) entryTokens {
	if i < len(index.tokens) {
		return index.tokens[i]
	}
	return tokenizeEntry(index.Entries[i])
}

// cacheEntry holds a cached value with expiry.
type cacheEntry struct {
	data      any
	expiresAt time.Time
}

// cache is a simple TTL-based in-memory cache.
type cache struct {
	mu    sync.RWMutex
	store map[string]cacheEntry
}

func newCache() *cache {
	return &cache{store: make(map[string]cacheEntry)}
}

func (c *cache) get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.store[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.store, key)
		return nil, false
	}
	return entry.data, true
}

func (c *cache) set(key string, data any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.store[key]; !exists && len(c.store) >= maxCacheEntries {
		c.evictLocked()
	}
	c.store[key] = cacheEntry{data: data, expiresAt: time.Now().Add(ttl)}
}

func (c *cache) evictLocked() {
	now := time.Now()
	var oldestKey string
	var oldestExpiry time.Time
	for key, entry := range c.store {
		if !entry.expiresAt.After(now) {
			delete(c.store, key)
			continue
		}
		if oldestExpiry.IsZero() || entry.expiresAt.Before(oldestExpiry) {
			oldestKey, oldestExpiry = key, entry.expiresAt
		}
	}
	if len(c.store) >= maxCacheEntries {
		delete(c.store, oldestKey)
	}
}

// RelatedLink represents a link found within a docs page.
type RelatedLink struct {
	URL      string
	Title    string
	Category string // "how-to", "reference", "support", "getting-started", "concept", "details", "other"
}

// DocsService defines the interface for fetching and searching DigitalOcean documentation.
type DocsService interface {
	GetDocsIndex() (*DocsIndex, error)
	GetServiceIndex(service string) (*DocsIndex, error)
	FetchDocPage(url string) (string, error)
	FindQuickstart(service string) (string, string, error)
	FindTroubleshootPage(symptom string) ([]DocsEntry, error)
	ExtractRelatedLinks(url string) ([]RelatedLink, error)
	// ContentExcerpts returns a query-focused excerpt of the page body for
	// each entry it can resolve, keyed by canonical URL. Best effort.
	ContentExcerpts(ctx context.Context, entries []DocsEntry, query string) map[string]string
}

// DocsClient fetches and searches DigitalOcean documentation.
type DocsClient struct {
	httpClient *http.Client
	cache      *cache

	indexGroup  singleflight.Group
	fetchGroup  singleflight.Group
	corpusGroup singleflight.Group

	// corpusSlots bounds concurrent corpus loads across the client; see slots().
	corpusSlots     chan struct{}
	corpusSlotsOnce sync.Once
	// timeouts counts consecutive deadline failures per corpus key.
	timeoutMu sync.Mutex
	timeouts  map[string]int

	// maxResponseSize is the accepted-body cap; tests lower it.
	maxResponseSize int64
	// staleIndex is the last successfully loaded index. It is served when a
	// refresh fails so transient docs-site errors do not break search.
	staleIndex atomic.Pointer[DocsIndex]
}

// NewDocsClient creates a new DocsClient.
func NewDocsClient() *DocsClient {
	return &DocsClient{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return validateDocsURL(req.URL)
			},
		},
		cache:           newCache(),
		corpusSlots:     make(chan struct{}, shardConcurrency),
		maxResponseSize: maxResponseSize,
	}
}

// validateDocsURL confines requests, including redirects, to the public HTTPS
// docs origin.
func validateDocsURL(u *neturl.URL) error {
	if u.Scheme != "https" || u.Host != "docs.digitalocean.com" || u.User != nil {
		return fmt.Errorf("refusing to fetch %s: only %s pages are supported", u, docsBase)
	}
	return nil
}

func normalizePageURL(raw string) (string, error) {
	if strings.ContainsAny(raw, "\\\x00\r\n\t") {
		return "", fmt.Errorf("invalid page URL: forbidden character")
	}
	raw = strings.TrimSpace(raw)
	u, err := neturl.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid page URL: %w", err)
	}
	if u.Scheme == "" {
		u, err = neturl.Parse(docsBase + "/" + strings.TrimLeft(raw, "/"))
		if err != nil {
			return "", fmt.Errorf("invalid page URL: %w", err)
		}
	}
	if err := validateDocsURL(u); err != nil {
		return "", err
	}
	cleaned := path.Clean("/" + u.Path)
	if strings.HasSuffix(u.Path, "/") && cleaned != "/" {
		cleaned += "/"
	}
	return (&neturl.URL{Scheme: u.Scheme, Host: u.Host, Path: cleaned}).String(), nil
}

func isDocsURL(raw string) bool {
	u, err := neturl.Parse(raw)
	return err == nil && validateDocsURL(u) == nil
}

func cleanInline(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
	return strings.Join(strings.Fields(text), " ")
}

func lastModDate(raw string) string {
	if len(raw) < len(time.DateOnly) {
		return ""
	}
	date := raw[:len(time.DateOnly)]
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		return ""
	}
	return date
}

func (d *DocsClient) fetch(url string) (string, error) {
	if cached, ok := d.cache.get("fetch:" + url); ok {
		return cached.(string), nil
	}

	result, err, _ := d.fetchGroup.Do(url, func() (any, error) {
		if cached, ok := d.cache.get("fetch:" + url); ok {
			return cached, nil
		}
		text, err := d.fetchUncached(url)
		if err != nil {
			return nil, err
		}
		d.cache.set("fetch:"+url, text, pageCacheTTL)
		return text, nil
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// fetchUncached performs a bounded GET without touching the response cache.
// Index loading uses it directly because the parsed index is cached
// separately and the raw body would only duplicate memory.
func (d *DocsClient) fetchUncached(url string) (string, error) {
	return d.fetchUncachedContext(context.Background(), url)
}

func (d *DocsClient) fetchUncachedContext(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	if err := validateDocsURL(req.URL); err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch %s: HTTP %d", url, resp.StatusCode)
	}
	limit := d.maxResponseSize
	if limit <= 0 {
		limit = maxResponseSize
	}
	if resp.ContentLength > limit {
		// Refuse up front rather than download 10 MB of a response that
		// would be cut mid-document anyway.
		return "", fmt.Errorf("failed to fetch %s: response is %d bytes, limit is %d", url, resp.ContentLength, limit)
	}

	// One extra byte distinguishes an exactly-at-limit body from an oversized
	// chunked or decompressed response without trusting Content-Length.
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return "", fmt.Errorf("failed to read response from %s: %w", url, err)
	}
	if int64(len(body)) > limit {
		return "", fmt.Errorf("failed to fetch %s: response exceeds limit of %d bytes", url, limit)
	}

	return string(body), nil
}

// parseLlmsTxt parses a llms.txt markdown index into structured entries.
func parseLlmsTxt(text string) *DocsIndex {
	entries := make([]DocsEntry, 0)
	sectionSet := make(map[string]struct{})
	var sections []string
	currentSection := "General"

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)

		// Track section headers (## or ###)
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			section := cleanInline(html.UnescapeString(strings.TrimLeft(trimmed, "# ")))
			currentSection = section
			if _, exists := sectionSet[section]; !exists {
				sectionSet[section] = struct{}{}
				sections = append(sections, section)
			}
			continue
		}

		// Parse link entries: - [Title](URL): Description
		if matches := entryRe.FindStringSubmatch(trimmed); matches != nil {
			if !isDocsURL(matches[2]) {
				continue
			}
			entries = append(entries, DocsEntry{
				Title:       cleanInline(html.UnescapeString(matches[1])),
				URL:         matches[2],
				Description: cleanInline(html.UnescapeString(matches[3])),
				Section:     currentSection,
			})
		}
	}

	return newDocsIndex(entries, sections, "llms.txt")
}

// llmsIndexRecord mirrors one element of llms-index.json.
type llmsIndexRecord struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	MarkdownURL string `json:"markdown_url"`
	Description string `json:"description"`
	Product     string `json:"product"`
	Section     string `json:"section"`
	LastMod     string `json:"lastmod"`
}

// parseLlmsIndex parses llms-index.json, the compact per-page metadata index
// published alongside llms.txt. Unlike llms.txt (a curated ~300-entry
// overview), it lists every public page, including individual support
// articles, so it is the preferred candidate universe for search.
func parseLlmsIndex(data []byte) (*DocsIndex, error) {
	var records []llmsIndexRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("failed to parse llms-index.json: %w", err)
	}

	entries := make([]DocsEntry, 0, len(records))
	sectionSet := make(map[string]struct{})
	var sections []string

	for _, r := range records {
		if r.URL == "" || r.Title == "" {
			continue
		}
		if !isDocsURL(r.URL) {
			continue
		}
		section := cleanInline(html.UnescapeString(r.Product))
		if section == "" {
			section = cleanInline(html.UnescapeString(r.Section))
		}
		if section == "" {
			section = "General"
		}
		if _, exists := sectionSet[section]; !exists {
			sectionSet[section] = struct{}{}
			sections = append(sections, section)
		}
		if !isDocsURL(r.MarkdownURL) {
			r.MarkdownURL = ""
		}
		entries = append(entries, DocsEntry{
			Title:        cleanInline(html.UnescapeString(r.Title)),
			URL:          r.URL,
			Description:  cleanInline(html.UnescapeString(r.Description)),
			Section:      section,
			MarkdownURL:  r.MarkdownURL,
			LastModified: lastModDate(r.LastMod),
		})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("llms-index.json contained no usable entries")
	}

	sort.Strings(sections)
	return newDocsIndex(entries, sections, "llms-index.json"), nil
}

// GetDocsIndex returns the searchable page index. It prefers llms-index.json
// (complete page universe) and falls back to llms.txt when that is
// unavailable. Concurrent cold loads are collapsed into a single fetch, and
// a previously loaded index is served if a refresh fails.
func (d *DocsClient) GetDocsIndex() (*DocsIndex, error) {
	if cached, ok := d.cache.get("docsIndex"); ok {
		return cached.(*DocsIndex), nil
	}

	result, err, _ := d.indexGroup.Do("docsIndex", func() (any, error) {
		if cached, ok := d.cache.get("docsIndex"); ok {
			return cached, nil
		}
		index, err := d.loadIndex()
		if err != nil {
			if stale := d.staleIndex.Load(); stale != nil {
				// Back off before retrying so an origin outage does not add
				// a fetch timeout to every query.
				d.cache.set("docsIndex", stale, negativeCacheTTL)
				return stale, nil
			}
			return nil, err
		}
		d.cache.set("docsIndex", index, indexCacheTTL)
		d.staleIndex.Store(index)
		return index, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*DocsIndex), nil
}

func (d *DocsClient) loadIndex() (*DocsIndex, error) {
	data, indexErr := d.fetchUncached(docsBase + llmsIndexPath)
	if indexErr == nil {
		index, parseErr := parseLlmsIndex([]byte(data))
		if parseErr == nil {
			return index, nil
		}
		indexErr = parseErr
	}
	if stale := d.staleIndex.Load(); stale != nil && stale.Source == "llms-index.json" {
		// Let GetDocsIndex serve the richer stale index with a short retry TTL
		// instead of replacing it with the much smaller llms.txt fallback.
		return nil, indexErr
	}

	text, txtErr := d.fetchUncached(docsBase + llmsTxtPath)
	if txtErr != nil {
		return nil, fmt.Errorf("failed to fetch docs index: %w (llms-index.json: %v)", txtErr, indexErr)
	}
	index := parseLlmsTxt(text)
	if len(index.Entries) == 0 {
		return nil, fmt.Errorf("docs index is empty (llms-index.json: %v)", indexErr)
	}
	return index, nil
}

// GetServiceIndex fetches the llms.txt for a specific service.
func (d *DocsClient) GetServiceIndex(service string) (*DocsIndex, error) {
	slug := resolveServiceSlug(service)
	if !validServiceSlug(slug) {
		return nil, nil
	}
	cacheKey := "serviceIndex:" + slug

	if cached, ok := d.cache.get(cacheKey); ok {
		if cached == nil {
			return nil, nil
		}
		return cached.(*DocsIndex), nil
	}

	// Try products path first, then platform, then reference
	paths := []string{
		fmt.Sprintf("%s/products/%s/llms.txt", docsBase, slug),
		fmt.Sprintf("%s/platform/%s/llms.txt", docsBase, slug),
		fmt.Sprintf("%s/reference/%s/llms.txt", docsBase, slug),
	}

	for _, url := range paths {
		text, err := d.fetch(url)
		if err != nil {
			continue
		}
		index := parseLlmsTxt(text)
		if len(index.Entries) > 0 {
			d.cache.set(cacheKey, index, indexCacheTTL)
			return index, nil
		}
	}

	// Cache the miss
	d.cache.set(cacheKey, nil, negativeCacheTTL)
	return nil, nil
}

// FetchDocPage fetches a doc page as clean markdown.
func (d *DocsClient) FetchDocPage(url string) (string, error) {
	pageURL, err := normalizePageURL(url)
	if err != nil {
		return "", err
	}
	cacheKey := "page:" + pageURL

	if cached, ok := d.cache.get(cacheKey); ok {
		return cached.(string), nil
	}

	// Try index.html.md for raw markdown
	mdURL := pageURL
	if strings.HasSuffix(mdURL, "index.html.md") {
		// Already correct
	} else if strings.HasSuffix(mdURL, "/") {
		mdURL += "index.html.md"
	} else {
		mdURL += "/index.html.md"
	}

	seen := make(map[string]bool)
	for _, target := range []string{d.cachedMarkdownURL(pageURL), mdURL, pageURL} {
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		var content string
		content, err = d.fetch(target)
		if err == nil {
			cleaned := cleanMarkdown(content)
			d.cache.set(cacheKey, cleaned, pageCacheTTL)
			return cleaned, nil
		}
	}
	return "", fmt.Errorf("failed to fetch doc page %s: %w", url, err)
}

func (d *DocsClient) cachedMarkdownURL(pageURL string) string {
	// A page read must not download the whole index just to discover a URL.
	index := d.staleIndex.Load()
	if cached, ok := d.cache.get("docsIndex"); ok {
		index = cached.(*DocsIndex)
	}
	if index == nil {
		return ""
	}
	key := strings.TrimSuffix(canonicalDocURL(pageURL), "/")
	for _, entry := range index.Entries {
		if !isDocsURL(entry.MarkdownURL) {
			continue
		}
		mdURL, err := normalizePageURL(entry.MarkdownURL)
		if err != nil {
			continue
		}
		if mdURL == pageURL {
			return mdURL
		}
		canonical, err := normalizePageURL(entry.URL)
		if err == nil && strings.TrimSuffix(canonicalDocURL(canonical), "/") == key {
			return mdURL
		}
	}
	return ""
}

// queryTerm is one normalized query token plus a light stem so that, for
// example, "configuration" also matches "configure" and "codes" matches
// "code". Terms match complete field tokens, never arbitrary substrings.
type queryTerm struct {
	text string
	stem string
}

// minPrefixLen is the shortest query term allowed to match a longer token by
// prefix, so "postgres" finds "postgresql" and "kube" finds "kubernetes"
// while "spaces" still does not find "namespace".
const minPrefixLen = 4

// matchesToken reports whether the query term matches one field token:
// exact text, equal stem, or a prefix of at least minPrefixLen characters.
func (q queryTerm) matchesToken(token queryTerm) bool {
	if token.text == q.text || token.stem == q.stem {
		return true
	}
	return len(q.text) >= minPrefixLen && strings.HasPrefix(token.text, q.text)
}

func (q queryTerm) matches(field string) bool {
	return q.matchesTokens(fieldTerms(field))
}

func (q queryTerm) matchesTokens(tokens []queryTerm) bool {
	for _, token := range tokens {
		if q.matchesToken(token) {
			return true
		}
	}
	return false
}

func (q queryTerm) matchesExactTokens(tokens []queryTerm) bool {
	for _, token := range tokens {
		if token.text == q.text {
			return true
		}
	}
	return false
}

// entryTokens holds the tokenized metadata fields of one index entry. They
// are built once when the index is parsed so queries do not re-tokenize
// 4,600 entries each time.
type entryTokens struct {
	title, description, section, url []queryTerm
}

func tokenizeEntry(entry DocsEntry) entryTokens {
	return entryTokens{
		title:       fieldTerms(strings.ToLower(entry.Title)),
		description: fieldTerms(strings.ToLower(entry.Description)),
		section:     fieldTerms(strings.ToLower(entry.Section)),
		url:         fieldTerms(urlPathLower(entry.URL)),
	}
}

// fieldTerms keeps all metadata tokens, including stop words.
func fieldTerms(field string) []queryTerm {
	raw := tokenSplitRe.Split(field, -1)
	terms := make([]queryTerm, 0, len(raw))
	for _, token := range raw {
		if token != "" {
			terms = append(terms, queryTerm{text: token, stem: lightStem(token)})
		}
	}
	return terms
}

// stopWords are dropped from queries when other terms remain. They carry no
// ranking signal and, with a 4,600-page universe, "how", "my" or "is" would
// otherwise match hundreds of unrelated titles.
var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "to": {}, "of": {}, "in": {}, "on": {}, "for": {}, "with": {},
	"and": {}, "or": {}, "is": {}, "are": {}, "was": {}, "be": {}, "my": {}, "i": {}, "how": {},
	"do": {}, "does": {}, "why": {}, "what": {}, "when": {}, "can": {}, "cant": {}, "not": {},
	"from": {}, "at": {}, "as": {}, "by": {}, "it": {}, "its": {}, "this": {}, "that": {},
}

// stemSuffixes are stripped (longest first) when at least four characters
// remain. This is deliberately conservative: it only needs to reconcile the
// common inflections seen in titles and URL slugs, not be a real stemmer.
var stemSuffixes = []string{"ization", "ations", "ically", "ation", "ings", "edly", "ing", "ies", "ers", "ors", "ed", "es", "ly", "al", "er", "or", "s", "e", "y"}

func lightStem(token string) string {
	if len(token) <= 4 {
		return token
	}
	for _, suffix := range stemSuffixes {
		if strings.HasSuffix(token, suffix) && len(token)-len(suffix) >= 4 {
			return token[:len(token)-len(suffix)]
		}
	}
	return token
}

var wordTokenRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

var tokenSplitRe = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// maxQueryTerms bounds per-query work (each term is matched against every
// index entry); terms beyond the cap are ignored.
const maxQueryTerms = 16

// parseQueryTerms lowercases, splits on punctuation, drops stop words and
// single-character fragments (unless nothing else remains) and attaches stems.
func parseQueryTerms(query string) []queryTerm {
	raw := tokenSplitRe.Split(strings.ToLower(query), -1)
	var tokens, kept []string
	for _, tok := range raw {
		if tok == "" {
			continue
		}
		tokens = append(tokens, tok)
		if _, stop := stopWords[tok]; stop || len(tok) < 2 {
			continue
		}
		kept = append(kept, tok)
	}
	if len(kept) == 0 {
		kept = tokens
	}
	if len(kept) > maxQueryTerms {
		kept = kept[:maxQueryTerms]
	}

	terms := make([]queryTerm, len(kept))
	for i, tok := range kept {
		terms[i] = queryTerm{text: tok, stem: lightStem(tok)}
	}
	return terms
}

// urlPathLower returns the lowercased URL path for matching. The host is
// excluded so a term like "digitalocean" does not match every entry.
func urlPathLower(url string) string {
	return strings.ToLower(strings.TrimPrefix(url, docsBase))
}

type scoredEntry struct {
	entry DocsEntry
	score int
}

// rankEntries orders by descending score, breaking ties by URL so results
// are deterministic across runs.
func rankEntries(results []scoredEntry) []DocsEntry {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].entry.URL < results[j].entry.URL
	})

	entries := make([]DocsEntry, len(results))
	for i, r := range results {
		entries[i] = r.entry
	}
	return entries
}

// filterByProduct prefers exact product names, aliases, and URL slugs.
// Token matches are used only when no exact identity matches.
func filterByProduct(index *DocsIndex, product string) *DocsIndex {
	want := normalizeFacet(product)
	if want == "" {
		return index
	}
	service := resolveServiceSlug(product)
	alias := normalizeFacet(service)
	slug := "/" + service + "/"
	exact := make([]bool, len(index.Entries))
	hasExact := false
	for i, entry := range index.Entries {
		section := normalizeFacet(entry.Section)
		exact[i] = section == want || section == alias || strings.Contains(urlPathLower(entry.URL), slug)
		hasExact = hasExact || exact[i]
	}
	terms := fieldTerms(want)

	filtered := &DocsIndex{Source: index.Source, FetchedAt: index.FetchedAt}
	sectionSet := make(map[string]struct{})
	for i, entry := range index.Entries {
		matches := exact[i]
		if !hasExact {
			matches = true
			for _, term := range terms {
				if !term.matchesTokens(index.entryTokensAt(i).section) {
					matches = false
					break
				}
			}
		}
		if !matches {
			continue
		}
		filtered.Entries = append(filtered.Entries, entry)
		filtered.tokens = append(filtered.tokens, index.entryTokensAt(i))
		if _, seen := sectionSet[entry.Section]; !seen {
			sectionSet[entry.Section] = struct{}{}
			filtered.Sections = append(filtered.Sections, entry.Section)
		}
	}
	sort.Strings(filtered.Sections)
	return filtered
}

// normalizeFacet lowercases and collapses punctuation so "App-Platform",
// "app platform" and "App Platform" compare equal.
func normalizeFacet(s string) string {
	return strings.Join(strings.Fields(tokenSplitRe.ReplaceAllString(strings.ToLower(s), " ")), " ")
}

// SearchIndex ranks index entries against a query by keyword matching over
// title, description, section, and URL path segments. Entries matching all
// query terms receive an additive boost. It does not search page
// bodies.
func SearchIndex(index *DocsIndex, query string) []DocsEntry {
	return searchIndex(index, query, false)
}

// searchIndex implements SearchIndex. When requireAllTerms is set, entries
// that match only some query terms are dropped instead of ranked lower; the
// service-name fallback uses this so a generic token like "service" in an
// unknown name does not return unrelated pages.
func searchIndex(index *DocsIndex, query string, requireAllTerms bool) []DocsEntry {
	terms := parseQueryTerms(query)
	if len(terms) == 0 {
		return nil
	}

	matches, weights := matchTerms(index, terms)
	var results []scoredEntry

	for i, entry := range index.Entries {
		score := 0
		matchedTerms := 0

		for t, m := range matches[i] {
			termScore := 0
			if m.title {
				termScore += 10
			}
			if m.titleWord {
				termScore += 5
			}
			if m.description {
				termScore += 3
			}
			if m.section {
				termScore += 2
			}
			if m.url {
				termScore += 4
			}
			score += termScore * weights[t]
			if m.any() {
				matchedTerms++
			}
		}

		if requireAllTerms && matchedTerms < len(terms) {
			continue
		}

		// Boost entries that match all query terms
		if len(terms) > 1 && matchedTerms == len(terms) {
			score += allTermsBoost
		}

		if score > 0 {
			results = append(results, scoredEntry{entry: entry, score: score})
		}
	}

	return rankEntries(results)
}

// fieldMatch records which fields of one entry a query term matched.
type fieldMatch struct {
	title, titleWord, description, section, url bool
}

func (m fieldMatch) any() bool {
	return m.title || m.description || m.section || m.url
}

const (
	// allTermsBoost rewards complete coverage; field scores still affect order.
	allTermsBoost = 15

	// rareTermFraction and rareTermWeight double the contribution of terms
	// that match at most this fraction of the index. In a query like
	// "app platform 520", the two common words would otherwise outweigh the
	// one term that identifies the page. A full IDF weighting was measured
	// to hurt the regression set; this bounded form does not.
	rareTermFraction = 0.005
	rareTermWeight   = 2
)

// matchTerms evaluates every term against every entry once and returns the
// per-entry matches plus a per-term weight (1, or rareTermWeight for terms
// matching few entries). Both rankers build their scores from this.
func matchTerms(index *DocsIndex, terms []queryTerm) ([][]fieldMatch, []int) {
	matches := make([][]fieldMatch, len(index.Entries))
	backing := make([]fieldMatch, len(index.Entries)*len(terms))
	docFreq := make([]int, len(terms))

	for i := range index.Entries {
		tokens := index.entryTokensAt(i)

		row := backing[i*len(terms) : (i+1)*len(terms)]
		for t, term := range terms {
			row[t] = fieldMatch{
				title:       term.matchesTokens(tokens.title),
				titleWord:   term.matchesExactTokens(tokens.title),
				description: term.matchesTokens(tokens.description),
				section:     term.matchesTokens(tokens.section),
				// URL path segments, e.g. "manage-domains" matches "domain"
				url: term.matchesTokens(tokens.url),
			}
			if row[t].any() {
				docFreq[t]++
			}
		}
		matches[i] = row
	}

	rareLimit := int(float64(len(index.Entries)) * rareTermFraction)
	weights := make([]int, len(terms))
	for t, df := range docFreq {
		weights[t] = 1
		if df > 0 && df <= rareLimit {
			weights[t] = rareTermWeight
		}
	}
	return matches, weights
}

// FindQuickstart finds the quickstart page for a service.
func (d *DocsClient) FindQuickstart(service string) (string, string, error) {
	slug := resolveServiceSlug(service)
	if !validServiceSlug(slug) {
		return "", "", fmt.Errorf("invalid service slug %q", slug)
	}

	// Try common quickstart URL patterns
	patterns := []string{
		fmt.Sprintf("%s/products/%s/getting-started/quickstart/", docsBase, slug),
		fmt.Sprintf("%s/products/%s/getting-started/", docsBase, slug),
		fmt.Sprintf("%s/products/%s/quickstart/", docsBase, slug),
	}

	for _, url := range patterns {
		content, err := d.FetchDocPage(url)
		if err == nil && len(content) > 100 {
			return url, content, nil
		}
	}

	// Fall back: search the service index for quickstart/getting-started entries
	serviceIndex, err := d.GetServiceIndex(slug)
	if err == nil && serviceIndex != nil {
		for _, entry := range serviceIndex.Entries {
			titleLower := strings.ToLower(entry.Title)
			if strings.Contains(titleLower, "quickstart") || strings.Contains(titleLower, "getting started") {
				content, err := d.FetchDocPage(entry.URL)
				if err == nil {
					return entry.URL, content, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("no quickstart found for service %q", service)
}

// FindTroubleshootPage searches for troubleshooting/support pages matching a symptom or error message.
// It prioritizes entries from support sections and pages with troubleshooting-style titles.
func (d *DocsClient) FindTroubleshootPage(symptom string) ([]DocsEntry, error) {
	index, err := d.GetDocsIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load docs index: %w", err)
	}

	terms := parseQueryTerms(symptom)
	if len(terms) == 0 {
		return nil, fmt.Errorf("symptom %q contains no searchable terms", symptom)
	}

	return rankTroubleshoot(index, terms), nil
}

// rankTroubleshoot scores index entries against symptom terms, boosting
// support articles and troubleshooting-style titles.
func rankTroubleshoot(index *DocsIndex, terms []queryTerm) []DocsEntry {
	matches, weights := matchTerms(index, terms)
	var results []scoredEntry

	for i, entry := range index.Entries {
		score := 0
		matchedTerms := 0

		// The section field is not scored here: support pages are handled
		// by the explicit boost below instead.
		for t, m := range matches[i] {
			termScore := 0
			if m.title {
				termScore += 10
			}
			if m.description {
				termScore += 3
			}
			if m.url {
				termScore += 2
			}
			score += termScore * weights[t]
			if m.title || m.description || m.url {
				matchedTerms++
			}
		}

		if score == 0 {
			continue
		}

		titleLower := strings.ToLower(entry.Title)
		sectionLower := strings.ToLower(entry.Section)
		urlLower := urlPathLower(entry.URL)

		// Boost support/troubleshooting pages
		if strings.Contains(sectionLower, "support") || strings.Contains(urlLower, "/support/") {
			score += 20
		}

		// Boost troubleshooting-style titles
		for _, prefix := range troubleshootPrefixes {
			if strings.HasPrefix(titleLower, prefix) {
				score += 10
				break
			}
		}

		// Boost entries matching all terms
		if len(terms) > 1 && matchedTerms == len(terms) {
			score += allTermsBoost
		}

		results = append(results, scoredEntry{entry: entry, score: score})
	}

	return rankEntries(results)
}

var troubleshootPrefixes = []string{
	"why do i",
	"why does",
	"why is",
	"why am i",
	"why can't",
	"why are",
	"how do i fix",
	"how do i troubleshoot",
	"how to troubleshoot",
	"my app",
	"my container",
	"my database",
}

// ExtractRelatedLinks fetches a docs page and extracts outbound docs.digitalocean.com links.
func (d *DocsClient) ExtractRelatedLinks(url string) ([]RelatedLink, error) {
	content, err := d.FetchDocPage(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page %s: %w", url, err)
	}

	// Normalize source URL for self-link filtering
	sourceBase := normalizeDocURL(url)

	seen := make(map[string]bool)
	var links []RelatedLink

	for _, match := range mdLinkRe.FindAllStringSubmatch(content, -1) {
		title := match[1]
		linkURL := match[2]

		// Only include docs.digitalocean.com links
		u, err := neturl.Parse(linkURL)
		if err != nil || validateDocsURL(u) != nil {
			continue
		}

		// Skip images, screenshots, and non-documentation assets
		lowerURL := strings.ToLower(linkURL)
		if strings.Contains(lowerURL, "/screenshots/") ||
			strings.HasSuffix(lowerURL, ".png") ||
			strings.HasSuffix(lowerURL, ".jpg") ||
			strings.HasSuffix(lowerURL, ".svg") ||
			strings.Contains(lowerURL, "/llms.txt") {
			continue
		}

		// Skip self-referencing links (same page, possibly different anchor)
		if normalizeDocURL(linkURL) == sourceBase {
			continue
		}

		// Deduplicate
		if seen[linkURL] {
			continue
		}
		seen[linkURL] = true

		links = append(links, RelatedLink{
			URL:      linkURL,
			Title:    title,
			Category: categorizeDocLink(linkURL),
		})
	}

	return links, nil
}

// normalizeDocURL strips anchors and index.html.md suffixes for comparison.
func normalizeDocURL(url string) string {
	// Strip anchor
	if idx := strings.Index(url, "#"); idx != -1 {
		url = url[:idx]
	}
	// Strip index.html.md suffix
	url = strings.TrimSuffix(url, "index.html.md")
	// Ensure trailing slash for consistent comparison
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	// Normalize relative to absolute
	if !strings.HasPrefix(url, "http") {
		if !strings.HasPrefix(url, "/") {
			url = "/" + url
		}
		url = docsBase + url
	}
	return strings.ToLower(url)
}

// categorizeDocLink categorizes a docs URL by its path pattern.
func categorizeDocLink(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "/how-to/"):
		return "how-to"
	case strings.Contains(lower, "/reference/"):
		return "reference"
	case strings.Contains(lower, "/support/") || strings.HasSuffix(lower, "/support"):
		return "support"
	case strings.Contains(lower, "/getting-started/") || strings.Contains(lower, "/quickstart"):
		return "getting-started"
	case strings.Contains(lower, "/concepts/"):
		return "concept"
	case strings.Contains(lower, "/details/"):
		return "details"
	default:
		return "other"
	}
}

var entryRe = regexp.MustCompile(`^-\s+\[([^\]]+)\]\(([^)]+)\)(?::\s*(.+))?$`)
var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\((https://docs\.digitalocean\.com[^)]+)\)`)
var excessiveNewlines = regexp.MustCompile(`\n{3,}`)

// htmlTags matches an HTML tag: a name, optional attributes, optional self
// closing slash. Requiring a letter after the angle bracket and a name made
// only of letters and digits keeps command placeholders (<droplet-id|name>),
// comparisons (a < b && c > d) and autolinks (<https://example.com>) intact,
// since none of those are HTML and all of them carry meaning. The attribute
// class excludes angle brackets but not newlines, so a tag whose attributes
// wrap onto the next line still matches.
var htmlTags = regexp.MustCompile(`</?[A-Za-z][A-Za-z0-9]*(\s[^<>]*)?/?>`)

// htmlComments and htmlDoctype cover the two non-tag constructs that reach this
// function when FetchDocPage falls back to an HTML page. Comments may span
// lines, hence the s flag.
var htmlComments = regexp.MustCompile(`(?s)<!--.*?-->`)
var htmlDoctype = regexp.MustCompile(`(?i)<!doctype[^<>]*>`)

// codeFenceOpen matches a fence opening, allowing the three leading spaces
// CommonMark permits and an info string. codeFenceClose is stricter: CommonMark
// allows no info string on a closing fence, so a line like ```python inside a
// block opens nothing and closes nothing.
var codeFenceOpen = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
var codeFenceClose = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})[ \t]*$")

// mdSegment is a run of lines that is either a fenced code block or the prose
// between blocks.
type mdSegment struct {
	text string
	code bool
}

// cleanMarkdown collapses runs of blank lines and strips HTML from prose,
// leaving fenced code blocks exactly as served. Markdown served to an agent is
// often copied and run, so a tag-shaped fragment inside a code block is
// content, not markup: stripping it silently turns a working example into a
// broken one, and collapsing its blank lines reflows it.
func cleanMarkdown(md string) string {
	segments := splitFencedCode(md)
	parts := make([]string, len(segments))
	for i, seg := range segments {
		if seg.code {
			parts[i] = seg.text
			continue
		}
		text := excessiveNewlines.ReplaceAllString(seg.text, "\n\n")
		text = stripHTMLOutsideCodeSpans(text)
		parts[i] = capBoundaryBlanks(text, i > 0, i < len(segments)-1)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// splitFencedCode divides a document into alternating prose and fenced code
// segments. Joining their text with a newline reproduces the input. An unclosed
// fence runs to the end of the document: treating the remainder as code keeps
// content rather than stripping it.
func splitFencedCode(md string) []mdSegment {
	var segments []mdSegment
	var cur []string
	fence := ""

	flush := func(code bool) {
		if len(cur) == 0 {
			return
		}
		segments = append(segments, mdSegment{text: strings.Join(cur, "\n"), code: code})
		cur = nil
	}

	for _, line := range strings.Split(md, "\n") {
		if fence == "" {
			if m := codeFenceOpen.FindStringSubmatch(line); m != nil {
				flush(false)
				fence = m[1]
			}
			cur = append(cur, line)
			continue
		}
		cur = append(cur, line)
		if m := codeFenceClose.FindStringSubmatch(line); m != nil &&
			m[1][0] == fence[0] && len(m[1]) >= len(fence) {
			flush(true)
			fence = ""
		}
	}
	flush(fence != "")
	return segments
}

// capBoundaryBlanks limits the blank lines a prose segment contributes at a
// seam with a neighbouring code block, since joining the segments adds a
// newline of its own. Without this a blank line next to a fence survives the
// collapse that the rest of the prose gets.
func capBoundaryBlanks(s string, leading, trailing bool) string {
	if leading {
		if t := strings.TrimLeft(s, "\n"); len(t) < len(s) {
			s = "\n" + t
		}
	}
	if trailing {
		if t := strings.TrimRight(s, "\n"); len(t) < len(s) {
			s = t + "\n"
		}
	}
	return s
}

// stripHTMLOutsideCodeSpans strips HTML, skipping inline code spans. A backtick
// run opens a span that only a run of the same length closes, so the backticks
// in a span such as `<div>` protect what they wrap. A span may wrap onto the
// next line but not across a blank line, which ends the paragraph.
func stripHTMLOutsideCodeSpans(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '`' {
			next := strings.IndexByte(s[i:], '`')
			if next < 0 {
				out.WriteString(stripHTMLFragment(s[i:]))
				break
			}
			out.WriteString(stripHTMLFragment(s[i : i+next]))
			i += next
			continue
		}
		run := 1
		for i+run < len(s) && s[i+run] == '`' {
			run++
		}
		marker := s[i : i+run]
		rest := s[i+run:]
		if end := strings.Index(rest, marker); end >= 0 && !strings.Contains(rest[:end], "\n\n") {
			out.WriteString(s[i : i+run+end+run])
			i += run + end + run
			continue
		}
		// An unclosed run is literal backticks, not a span.
		out.WriteString(marker)
		i += run
	}
	return out.String()
}

func stripHTMLFragment(s string) string {
	s = htmlComments.ReplaceAllString(s, "")
	s = htmlDoctype.ReplaceAllString(s, "")
	return htmlTags.ReplaceAllString(s, "")
}

// serviceAliases maps common service name aliases to URL slugs. A slug has to
// be one the site serves markdown for. Superseded product paths such as
// inference-hub, ai-platform, images/snapshots and network-file-storage are
// HTML-only 302 aliases: the redirect is not applied to the index.html.md
// variant, so it 404s, FetchDocPage falls back to the rendered page and
// FindQuickstart finds nothing.
var serviceAliases = map[string]string{
	"kubernetes":           "kubernetes",
	"k8s":                  "kubernetes",
	"doks":                 "kubernetes",
	"managed kubernetes":   "kubernetes",
	"droplets":             "droplets",
	"droplet":              "droplets",
	"vms":                  "droplets",
	"virtual machines":     "droplets",
	"apps":                 "app-platform",
	"app platform":         "app-platform",
	"databases":            "databases",
	"database":             "databases",
	"dbaas":                "databases",
	"spaces":               "spaces",
	"object storage":       "spaces",
	"functions":            "functions",
	"serverless":           "functions",
	"vpc":                  "networking/vpc",
	"networking":           "networking",
	"load balancers":       "networking/load-balancers",
	"load balancer":        "networking/load-balancers",
	"dns":                  "networking/dns",
	"domains":              "networking/dns",
	"firewall":             "networking/firewalls",
	"firewalls":            "networking/firewalls",
	"monitoring":           "monitoring",
	"registry":             "container-registry",
	"container registry":   "container-registry",
	"docr":                 "container-registry",
	"volumes":              "volumes",
	"block storage":        "volumes",
	"snapshots":            "snapshots",
	"backups":              "backups",
	"marketplace":          "marketplace",
	"serverless inference": "inference",
	"genai":                "inference",
	"gradient":             "inference",
	"gradient ai":          "inference",
	"dedicated inference":  "inference",
	"dedicated-inference":  "inference",
	"gpu":                  "bare-metal-gpus",
	"bare metal":           "bare-metal-gpus",
	"gpu droplets":         "bare-metal-gpus",
	"inference":            "inference",
	"inference hub":        "inference",
	"cspm":                 "cspm",
	"cloud security":       "cspm",
	"nfs":                  "nfs",
	"network file storage": "nfs",
}

func resolveServiceSlug(service string) string {
	lower := strings.ToLower(strings.TrimSpace(service))
	if slug, ok := serviceAliases[lower]; ok {
		return slug
	}
	return strings.ReplaceAll(lower, " ", "-")
}

var serviceSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*(/[a-z0-9][a-z0-9-]*){0,2}$`)

func validServiceSlug(slug string) bool {
	return len(slug) <= 64 && serviceSlugRe.MatchString(slug)
}
