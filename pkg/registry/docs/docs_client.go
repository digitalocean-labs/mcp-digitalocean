package docs

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	neturl "net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	c.store[key] = cacheEntry{data: data, expiresAt: time.Now().Add(ttl)}
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
}

// DocsClient fetches and searches DigitalOcean documentation.
type DocsClient struct {
	httpClient *http.Client
	cache      *cache

	indexGroup singleflight.Group

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

func (d *DocsClient) fetch(url string) (string, error) {
	if cached, ok := d.cache.get("fetch:" + url); ok {
		return cached.(string), nil
	}

	text, err := d.fetchUncached(url)
	if err != nil {
		return "", err
	}
	d.cache.set("fetch:"+url, text, pageCacheTTL)
	return text, nil
}

// fetchUncached performs a bounded GET without touching the response cache.
// Index loading uses it directly because the parsed index is cached
// separately and the raw body would only duplicate memory.
func (d *DocsClient) fetchUncached(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
			section := strings.TrimLeft(trimmed, "# ")
			currentSection = section
			if _, exists := sectionSet[section]; !exists {
				sectionSet[section] = struct{}{}
				sections = append(sections, section)
			}
			continue
		}

		// Parse link entries: - [Title](URL): Description
		if matches := entryRe.FindStringSubmatch(trimmed); matches != nil {
			entries = append(entries, DocsEntry{
				Title:       matches[1],
				URL:         matches[2],
				Description: matches[3],
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
		if !strings.HasPrefix(r.URL, docsBase+"/") && r.URL != docsBase+"/" {
			continue
		}
		section := r.Product
		if section == "" {
			section = "General"
		}
		if _, exists := sectionSet[section]; !exists {
			sectionSet[section] = struct{}{}
			sections = append(sections, section)
		}
		entries = append(entries, DocsEntry{
			Title:        html.UnescapeString(r.Title),
			URL:          r.URL,
			Description:  html.UnescapeString(r.Description),
			Section:      section,
			MarkdownURL:  r.MarkdownURL,
			LastModified: strings.SplitN(r.LastMod, "T", 2)[0],
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
	cacheKey := "page:" + url

	if cached, ok := d.cache.get(cacheKey); ok {
		return cached.(string), nil
	}

	// Normalize URL
	pageURL := url
	if !strings.HasPrefix(pageURL, "http") {
		if !strings.HasPrefix(pageURL, "/") {
			pageURL = "/" + pageURL
		}
		pageURL = docsBase + pageURL
	} else if pageURL != docsBase && !strings.HasPrefix(pageURL, docsBase+"/") {
		// Keep the docs tools closed-world: absolute URLs outside the docs
		// site are rejected rather than proxied.
		return "", fmt.Errorf("refusing to fetch %s: only %s pages are supported", url, docsBase)
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

	content, err := d.fetch(mdURL)
	if err == nil {
		cleaned := cleanMarkdown(content)
		d.cache.set(cacheKey, cleaned, pageCacheTTL)
		return cleaned, nil
	}

	// Fall back to the original URL
	content, err = d.fetch(pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch doc page %s: %w", url, err)
	}

	cleaned := cleanMarkdown(content)
	d.cache.set(cacheKey, cleaned, pageCacheTTL)
	return cleaned, nil
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

// matchesWord reports whether the term appears in field as a whole word,
// i.e. not flanked by letters or digits.
func (q queryTerm) matchesWord(field string) bool {
	for start := 0; ; {
		idx := strings.Index(field[start:], q.text)
		if idx < 0 {
			return false
		}
		from := start + idx
		to := from + len(q.text)
		if (from == 0 || !isWordByte(field[from-1])) && (to == len(field) || !isWordByte(field[to])) {
			return true
		}
		start = from + 1
	}
}

func isWordByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
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

	for i, entry := range index.Entries {
		titleLower := strings.ToLower(entry.Title)
		tokens := index.entryTokensAt(i)

		row := backing[i*len(terms) : (i+1)*len(terms)]
		for t, term := range terms {
			row[t] = fieldMatch{
				title:       term.matchesTokens(tokens.title),
				titleWord:   term.matchesWord(titleLower),
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
		if !strings.Contains(linkURL, "docs.digitalocean.com") {
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
var htmlTags = regexp.MustCompile(`<[^>]+>`)

func cleanMarkdown(md string) string {
	md = excessiveNewlines.ReplaceAllString(md, "\n\n")
	md = htmlTags.ReplaceAllString(md, "")
	return strings.TrimSpace(md)
}

// serviceAliases maps common service name aliases to URL slugs.
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
	"snapshots":            "images/snapshots",
	"backups":              "images/backups",
	"marketplace":          "marketplace",
	"gradient":             "gradient",
	"gradient ai":          "gradient",
	"dedicated inference":  "ai-platform",
	"dedicated-inference":  "ai-platform",
	"gpu":                  "bare-metal-gpus",
	"bare metal":           "bare-metal-gpus",
	"gpu droplets":         "bare-metal-gpus",
	"inference":            "inference-hub",
	"inference hub":        "inference-hub",
	"cspm":                 "cspm",
	"cloud security":       "cspm",
	"nfs":                  "network-file-storage",
	"network file storage": "network-file-storage",
}

func resolveServiceSlug(service string) string {
	lower := strings.ToLower(strings.TrimSpace(service))
	if slug, ok := serviceAliases[lower]; ok {
		return slug
	}
	return strings.ReplaceAll(lower, " ", "-")
}
