package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	docsBase         = "https://docs.digitalocean.com"
	llmsTxtURL       = docsBase + "/llms.txt"
	llmsIndexJSONURL = docsBase + "/llms-index.json"
	userAgent        = "mcp-digitalocean-docs/1.0"

	indexCacheTTL    = 1 * time.Hour
	pageCacheTTL     = 30 * time.Minute
	negativeCacheTTL = 10 * time.Minute

	maxResponseSize = 10 * 1024 * 1024 // 10 MB
)

// DocsEntry represents a single entry from the llms.txt index.
type DocsEntry struct {
	Title       string
	URL         string
	Description string
	Section     string
}

// DocsIndex represents a parsed llms.txt file.
type DocsIndex struct {
	Entries   []DocsEntry
	Sections  []string
	FetchedAt time.Time
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

// LLMSIndexRecord is one row from https://docs.digitalocean.com/llms-index.json
type LLMSIndexRecord struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	MarkdownURL string `json:"markdown_url"`
	Product     string `json:"product"`
	Section     string `json:"section"`
	Lastmod     string `json:"lastmod"`
}

// SemanticSearchHit is one ranked result from SemanticSearch.
type SemanticSearchHit struct {
	Title   string
	URL     string
	Score   float64
	Snippet string
}

// DocsService defines the interface for fetching and searching DigitalOcean documentation.
type DocsService interface {
	GetDocsIndex() (*DocsIndex, error)
	GetServiceIndex(service string) (*DocsIndex, error)
	FetchDocPage(url string) (string, error)
	FindQuickstart(service string) (string, string, error)
	FindTroubleshootPage(symptom string) ([]DocsEntry, error)
	ExtractRelatedLinks(url string) ([]RelatedLink, error)
	GetLLMSIndexRecords() ([]LLMSIndexRecord, error)
	SemanticSearch(ctx context.Context, query string, resultLimit int) ([]SemanticSearchHit, error)
}

// DocsClient fetches and searches DigitalOcean documentation.
type DocsClient struct {
	httpClient *http.Client
	cache      *cache
}

// NewDocsClient creates a new DocsClient.
func NewDocsClient() *DocsClient {
	return &DocsClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		cache:      newCache(),
	}
}

func (d *DocsClient) fetch(url string) (string, error) {
	if cached, ok := d.cache.get("fetch:" + url); ok {
		return cached.(string), nil
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", fmt.Errorf("failed to read response from %s: %w", url, err)
	}

	text := string(body)
	d.cache.set("fetch:"+url, text, pageCacheTTL)
	return text, nil
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

	return &DocsIndex{
		Entries:   entries,
		Sections:  sections,
		FetchedAt: time.Now(),
	}
}

// GetDocsIndex fetches and parses the main llms.txt index.
func (d *DocsClient) GetDocsIndex() (*DocsIndex, error) {
	if cached, ok := d.cache.get("docsIndex"); ok {
		return cached.(*DocsIndex), nil
	}

	text, err := d.fetch(llmsTxtURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch docs index: %w", err)
	}

	index := parseLlmsTxt(text)
	d.cache.set("docsIndex", index, indexCacheTTL)
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

// SearchIndex searches the docs index using text matching with ranked scoring.
// Scoring considers title, description, section, and URL path segments.
// Entries matching all query terms are boosted above partial matches.
func SearchIndex(index *DocsIndex, query string) []DocsEntry {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil
	}

	type scored struct {
		entry DocsEntry
		score int
	}

	wordRegexes := make([]*regexp.Regexp, len(terms))
	for i, term := range terms {
		wordRegexes[i] = regexp.MustCompile(`\b` + regexp.QuoteMeta(term) + `\b`)
	}

	var results []scored

	for _, entry := range index.Entries {
		titleLower := strings.ToLower(entry.Title)
		descLower := strings.ToLower(entry.Description)
		sectionLower := strings.ToLower(entry.Section)
		urlLower := strings.ToLower(entry.URL)
		score := 0
		matchedTerms := 0

		for i, term := range terms {
			termMatched := false

			if strings.Contains(titleLower, term) {
				score += 10
				termMatched = true
			}
			if wordRegexes[i].MatchString(titleLower) {
				score += 5
			}
			if strings.Contains(descLower, term) {
				score += 3
				termMatched = true
			}
			if strings.Contains(sectionLower, term) {
				score += 2
				termMatched = true
			}
			// Match against URL path segments (e.g., "manage-domains" matches "domain")
			if strings.Contains(urlLower, term) {
				score += 4
				termMatched = true
			}

			if termMatched {
				matchedTerms++
			}
		}

		// Boost entries that match all query terms
		if len(terms) > 1 && matchedTerms == len(terms) {
			score += 15
		}

		if score > 0 {
			results = append(results, scored{entry: entry, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	entries := make([]DocsEntry, len(results))
	for i, r := range results {
		entries[i] = r.entry
	}
	return entries
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

	terms := strings.Fields(strings.ToLower(symptom))
	if len(terms) == 0 {
		return nil, fmt.Errorf("symptom query must not be empty")
	}

	type scored struct {
		entry DocsEntry
		score int
	}

	var results []scored

	for _, entry := range index.Entries {
		titleLower := strings.ToLower(entry.Title)
		descLower := strings.ToLower(entry.Description)
		urlLower := strings.ToLower(entry.URL)
		sectionLower := strings.ToLower(entry.Section)
		score := 0
		matchedTerms := 0

		for _, term := range terms {
			termMatched := false
			if strings.Contains(titleLower, term) {
				score += 10
				termMatched = true
			}
			if strings.Contains(descLower, term) {
				score += 3
				termMatched = true
			}
			if strings.Contains(urlLower, term) {
				score += 2
				termMatched = true
			}
			if termMatched {
				matchedTerms++
			}
		}

		if score == 0 {
			continue
		}

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
			score += 15
		}

		results = append(results, scored{entry: entry, score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	entries := make([]DocsEntry, len(results))
	for i, r := range results {
		entries[i] = r.entry
	}
	return entries, nil
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

// GetLLMSIndexRecords fetches and parses llms-index.json (cached).
func (d *DocsClient) GetLLMSIndexRecords() ([]LLMSIndexRecord, error) {
	const cacheKey = "llmsIndexRecords"
	if v, ok := d.cache.get(cacheKey); ok {
		return v.([]LLMSIndexRecord), nil
	}

	body, err := d.fetch(llmsIndexJSONURL)
	if err != nil {
		return nil, fmt.Errorf("fetch llms-index.json: %w", err)
	}

	var records []LLMSIndexRecord
	if err := json.Unmarshal([]byte(body), &records); err != nil {
		return nil, fmt.Errorf("parse llms-index.json: %w", err)
	}

	d.cache.set(cacheKey, records, indexCacheTTL)
	return records, nil
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
