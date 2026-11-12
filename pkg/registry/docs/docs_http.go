package docs

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// SearchAPIPath is the plain-HTTP twin of the docs-search tool for clients
// that do not speak MCP. It shares the ranker, index, and excerpt layer with
// the tool so the two never disagree.
const SearchAPIPath = "/v1/search"

const (
	maxQueryBytes   = 256
	maxProductBytes = 64

	// searchCacheMaxAge matches the docs-corpus cache lifetime; results only
	// change when the published index does.
	searchCacheMaxAge = 300
)

var searchParams = map[string]struct{}{"q": {}, "limit": {}, "product": {}}

// SearchAPIResult is one ranked page in a /v1/search response.
type SearchAPIResult struct {
	Title        string `json:"title"`
	URL          string `json:"url"`
	MarkdownURL  string `json:"markdown_url,omitempty"`
	Product      string `json:"product,omitempty"`
	PageType     string `json:"page_type,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	Excerpt      string `json:"excerpt,omitempty"`
}

// SearchAPIResponse follows the public API's list shape: a named collection
// plus meta. The shape is preview and may change; meta.stability says so.
type SearchAPIResponse struct {
	Results []SearchAPIResult `json:"results"`
	Meta    SearchAPIMeta     `json:"meta"`
}

// SearchAPIMeta describes how a search was answered.
type SearchAPIMeta struct {
	Query         string `json:"query"`
	Product       string `json:"product,omitempty"`
	Total         int    `json:"total"`
	PagesSearched int    `json:"pages_searched"`
	Stability     string `json:"stability"`
}

// apiError mirrors the public API error body ({"id", "message"}).
type apiError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// SearchHandler serves SearchAPIPath. It is read-only, credential-free, and
// only ever fetches from the docs origin through DocsService.
type SearchHandler struct {
	client DocsService
	logger *slog.Logger
	// clients tags usage logs with a weekly pseudonymous client ID; nil
	// leaves the field out.
	clients *ClientTagger
}

// NewSearchHandler returns a handler backed by the same DocsClient as the
// MCP tools, so the two share one index, page cache, and corpus load budget.
// clients may be nil.
func NewSearchHandler(logger *slog.Logger, clients *ClientTagger) *SearchHandler {
	h := newSearchHandler(sharedDocsClient(), logger)
	h.clients = clients
	return h
}

func newSearchHandler(client DocsService, logger *slog.Logger) *SearchHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &SearchHandler{client: client, logger: logger}
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Public read-only content with no cookies or credentials, so any origin
	// may read it.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	start := time.Now()
	switch r.Method {
	case http.MethodGet:
	case http.MethodOptions:
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		w.Header().Set("Allow", "GET, OPTIONS")
		h.reject(w, r, start, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		h.reject(w, r, start, http.StatusBadRequest, "bad_request", "malformed query string")
		return
	}
	// Unknown or repeated parameters are rejected so every distinct URL is a
	// distinct search: otherwise ?q=x&junk=<random> defeats the CDN cache
	// and turns each request into an origin hit.
	for name, values := range params {
		if _, ok := searchParams[name]; !ok {
			h.reject(w, r, start, http.StatusBadRequest, "bad_request", "unknown parameter "+strconv.Quote(name)+"; supported: q, limit, product")
			return
		}
		if len(values) > 1 {
			h.reject(w, r, start, http.StatusBadRequest, "bad_request", "parameter "+strconv.Quote(name)+" given more than once")
			return
		}
	}

	query := strings.TrimSpace(params.Get("q"))
	if query == "" {
		h.reject(w, r, start, http.StatusBadRequest, "bad_request", "q is required")
		return
	}
	if len(query) > maxQueryBytes || !utf8.ValidString(query) {
		h.reject(w, r, start, http.StatusBadRequest, "bad_request", "q must be valid UTF-8 of at most 256 bytes")
		return
	}

	product := strings.TrimSpace(params.Get("product"))
	if len(product) > maxProductBytes || !utf8.ValidString(product) {
		h.reject(w, r, start, http.StatusBadRequest, "bad_request", "product must be valid UTF-8 of at most 64 bytes")
		return
	}

	limit := defaultSearchLimit
	if raw := params.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxResultLimit {
			h.reject(w, r, start, http.StatusBadRequest, "bad_request", "limit must be an integer from 1 to 50")
			return
		}
		limit = n
	}

	index, err := h.client.GetDocsIndex()
	if err != nil {
		h.logger.Error("docs search api: index unavailable", "error", err)
		h.reject(w, r, start, http.StatusServiceUnavailable, "service_unavailable", "docs index is temporarily unavailable")
		return
	}
	if product != "" {
		index = filterByProduct(index, product)
	}

	results := SearchIndex(index, query)
	if len(results) > limit {
		results = results[:limit]
	}

	resp := SearchAPIResponse{
		Results: h.toResults(r.Context(), results, query),
		Meta: SearchAPIMeta{
			Query:         query,
			Product:       product,
			Total:         len(results),
			PagesSearched: len(index.Entries),
			Stability:     "preview",
		},
	}

	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(searchCacheMaxAge))
	writeJSON(w, http.StatusOK, resp)

	// Query text is not logged: people paste error output that can carry
	// secrets. Term count and zero-result rate are enough to spot gaps.
	h.logger.Info("docs search api", append(h.usageAttrs(r, start, http.StatusOK),
		"terms", len(parseQueryTerms(query)),
		"product_filter", product != "",
		"results", len(results),
		"zero_results", len(results) == 0,
	)...)
}

// reject writes an API error and logs it as a usage event, so malformed
// calls count toward traffic and show which clients get the shape wrong.
func (h *SearchHandler) reject(w http.ResponseWriter, r *http.Request, start time.Time, status int, id, msg string) {
	writeAPIError(w, status, id, msg)
	h.logger.Info("docs search api", append(h.usageAttrs(r, start, status), "error_id", id)...)
}

// usageAttrs are the fields every search usage log carries.
func (h *SearchHandler) usageAttrs(r *http.Request, start time.Time, status int) []any {
	ua := r.Header.Get("User-Agent")
	attrs := []any{
		"status", status,
		"client_type", userAgentFamily(ua),
		"user_agent", cleanUserAgent(ua),
		"duration_ms", time.Since(start).Milliseconds(),
	}
	if h.clients != nil {
		attrs = append(attrs, "client_id", h.clients.ID(r))
	}
	return attrs
}

func (h *SearchHandler) toResults(ctx context.Context, entries []DocsEntry, query string) []SearchAPIResult {
	out := make([]SearchAPIResult, 0, len(entries))
	if len(entries) == 0 {
		return out
	}
	excerpts := h.client.ContentExcerpts(ctx, entries, query)
	for _, e := range entries {
		pageURL := canonicalDocURL(e.URL)
		excerpt := excerpts[pageURL]
		if excerpt == "" {
			excerpt = e.Description
		}
		pageType := categorizeDocLink(e.URL)
		if pageType == "other" {
			pageType = ""
		}
		out = append(out, SearchAPIResult{
			Title:        e.Title,
			URL:          pageURL,
			MarkdownURL:  e.MarkdownURL,
			Product:      e.Section,
			PageType:     pageType,
			LastModified: e.LastModified,
			Excerpt:      excerpt,
		})
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, id, msg string) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, apiError{ID: id, Message: msg})
}
