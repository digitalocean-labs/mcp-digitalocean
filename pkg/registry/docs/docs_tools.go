package docs

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"mcp-digitalocean/pkg/registry/common"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultSearchLimit = 10

	// maxEmbeddedPageBytes caps the page body docs-troubleshoot inlines for
	// its top result; the rest is one docs-get-page call away.
	maxEmbeddedPageBytes = 16 * 1024
)

// DocsTool provides MCP tool handlers for querying DigitalOcean documentation.
type DocsTool struct {
	client DocsService
}

// NewDocsTool creates a new DocsTool instance.
func NewDocsTool() *DocsTool {
	return &DocsTool{client: NewDocsClient()}
}

// writeEntry renders one result with the metadata an agent needs to cite it:
// canonical URL, product, page type, last-modified date, and an excerpt.
// The excerpt is the query-focused passage from the page body when one was
// retrieved, otherwise the page description.
func writeEntry(sb *strings.Builder, position int, entry DocsEntry, excerpt string) {
	url := canonicalDocURL(entry.URL)
	fmt.Fprintf(sb, "%d. %s\n   %s\n", position, entry.Title, url)

	var meta []string
	if entry.Section != "" {
		meta = append(meta, entry.Section)
	}
	if category := categorizeDocLink(entry.URL); category != "other" {
		meta = append(meta, category)
	}
	if entry.LastModified != "" {
		meta = append(meta, "updated "+entry.LastModified)
	}
	if len(meta) > 0 {
		fmt.Fprintf(sb, "   [%s]\n", strings.Join(meta, " · "))
	}
	switch {
	case excerpt != "":
		fmt.Fprintf(sb, "   %s\n", excerpt)
	case entry.Description != "":
		fmt.Fprintf(sb, "   %s\n", entry.Description)
	}
	sb.WriteByte('\n')
}

// writeEntries renders a ranked list, enriching each entry with a body
// excerpt where the corpus provides one.
func (d *DocsTool) writeEntries(ctx context.Context, sb *strings.Builder, first int, entries []DocsEntry, query string) {
	excerpts := d.client.ContentExcerpts(ctx, entries, query)
	for i, entry := range entries {
		writeEntry(sb, first+i, entry, excerpts[canonicalDocURL(entry.URL)])
	}
}

// paginate returns the slice of content starting at start with at most
// maxLength bytes, snapped to UTF-8 boundaries, plus the offset to continue
// from (len(content) when nothing remains). A budget smaller than the next
// rune returns an empty window; the tool reports that as an argument error.
func paginate(content string, start, maxLength int) (string, int) {
	if start < 0 {
		start = 0
	}
	for start < len(content) && !utf8.RuneStart(content[start]) {
		start++
	}
	if start >= len(content) {
		return "", len(content)
	}
	if maxLength <= 0 || maxLength >= len(content)-start {
		return content[start:], len(content)
	}
	end := start + maxLength
	for end > start && !utf8.RuneStart(content[end]) {
		end--
	}
	return content[start:end], end
}

// writeContinuation tells the agent exactly how to fetch the rest of a page
// that was cut at a budget.
func writeContinuation(sb *strings.Builder, url string, next, total int) {
	fmt.Fprintf(sb, "\n[Truncated: showing bytes up to %d of %d. Call docs-get-page with URL=%s and StartIndex=%d for the rest.]\n", next, total, url, next)
}

func pageOffset(args map[string]any, name string) (int, error) {
	raw, exists := args[name]
	if !exists {
		return 0, nil
	}
	value, ok := raw.(float64)
	if !ok || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || math.Trunc(value) != value || value > maxResponseSize {
		return 0, fmt.Errorf("%s must be an integer between 0 and %d bytes", name, maxResponseSize)
	}
	return int(value), nil
}

// canonicalDocURL strips the index.html.md suffix that llms.txt entries carry
// so results always cite the canonical page URL.
func canonicalDocURL(url string) string {
	return strings.TrimSuffix(url, "index.html.md")
}

// searchDocs ranks documentation pages by keyword match against page metadata.
func (d *DocsTool) searchDocs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	query, ok := args["Query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("Query is required and must be a non-empty string"), nil
	}

	limit := defaultSearchLimit
	if limitFloat, ok := args["Limit"].(float64); ok && limitFloat > 0 {
		limit = int(limitFloat)
	}
	product, _ := args["Product"].(string)
	product = strings.TrimSpace(product)

	index, err := d.client.GetDocsIndex()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to load docs index", err), nil
	}

	scope := "pages"
	if product != "" {
		scope = fmt.Sprintf("pages in %q", product)
		index = filterByProduct(index, product)
		if len(index.Entries) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No docs pages match Product %q. Omit Product to search everything.", product)), nil
		}
	}

	results := SearchIndex(index, query)

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No results found for %q (searched %d %s). Try different search terms or a more general query.", query, len(index.Entries), scope)), nil
	}

	if len(results) > limit {
		results = results[:limit]
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d result(s) for %q (searched %d %s):\n\n", len(results), query, len(index.Entries), scope)
	d.writeEntries(ctx, &sb, 1, results, query)
	sb.WriteString("Use docs-get-page with a result URL to read the full page.\n")

	return mcp.NewToolResultText(sb.String()), nil
}

// getDoc fetches the markdown content of a specific docs page, optionally a
// window of it.
func (d *DocsTool) getDoc(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	url, ok := args["URL"].(string)
	if !ok || url == "" {
		return mcp.NewToolResultError("URL is required and must be a non-empty string"), nil
	}
	start, err := pageOffset(args, "StartIndex")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	maxLength, err := pageOffset(args, "MaxLength")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	content, err := d.client.FetchDocPage(url)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to fetch doc page", err), nil
	}

	if start >= len(content) && start > 0 {
		return mcp.NewToolResultError(fmt.Sprintf("StartIndex %d is past the end of the page (%d bytes)", start, len(content))), nil
	}
	page, next := paginate(content, start, maxLength)
	if next >= len(content) {
		return mcp.NewToolResultText(page), nil
	}
	if page == "" {
		return mcp.NewToolResultError("MaxLength is too small to fit the next UTF-8 character; increase the byte budget"), nil
	}

	var sb strings.Builder
	sb.WriteString(page)
	sb.WriteByte('\n')
	writeContinuation(&sb, url, next, len(content))
	return mcp.NewToolResultText(sb.String()), nil
}

// findDocsForService returns documentation pages for a specific DigitalOcean service.
func (d *DocsTool) findDocsForService(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	service, ok := args["Service"].(string)
	if !ok || service == "" {
		return mcp.NewToolResultError("Service is required and must be a non-empty string"), nil
	}

	slug := resolveServiceSlug(service)

	// Try service-specific llms.txt first
	serviceIndex, err := d.client.GetServiceIndex(slug)
	if err == nil && serviceIndex != nil && len(serviceIndex.Entries) > 0 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Documentation for %q (%d pages):\n\n", service, len(serviceIndex.Entries)))

		// Group by section
		sectionEntries := make(map[string][]DocsEntry)
		var sectionOrder []string
		for _, entry := range serviceIndex.Entries {
			if _, exists := sectionEntries[entry.Section]; !exists {
				sectionOrder = append(sectionOrder, entry.Section)
			}
			sectionEntries[entry.Section] = append(sectionEntries[entry.Section], entry)
		}

		for _, section := range sectionOrder {
			sb.WriteString(fmt.Sprintf("### %s\n", section))
			for _, e := range sectionEntries[section] {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", e.Title, e.URL))
				if e.Description != "" {
					sb.WriteString(fmt.Sprintf("  %s\n", e.Description))
				}
			}
			sb.WriteByte('\n')
		}

		return mcp.NewToolResultText(sb.String()), nil
	}

	// Fall back to searching the main index
	mainIndex, err := d.client.GetDocsIndex()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to load docs index", err), nil
	}

	results := searchIndex(mainIndex, service, true)

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No documentation found for service %q.\n\nAvailable sections: %s",
			service, strings.Join(mainIndex.Sections, ", "))), nil
	}

	maxResults := 20
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Documentation related to %q (%d results):\n\n", service, len(results)))
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", r.Title, r.URL))
		if r.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", r.Description))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// getQuickstart returns the quickstart guide for a DigitalOcean service.
func (d *DocsTool) getQuickstart(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	service, ok := args["Service"].(string)
	if !ok || service == "" {
		return mcp.NewToolResultError("Service is required and must be a non-empty string"), nil
	}

	url, content, err := d.client.FindQuickstart(service)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("No quickstart guide found for %q. Try using docs-find-for-service to browse available documentation.", service)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("# Quickstart: %s\n\nSource: %s\n\n---\n\n%s", service, url, content)), nil
}

// troubleshoot searches for troubleshooting pages matching an error message or symptom.
func (d *DocsTool) troubleshoot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	symptom, ok := args["Symptom"].(string)
	if !ok || symptom == "" {
		return mcp.NewToolResultError("Symptom is required and must be a non-empty string"), nil
	}

	limit := 5
	if limitFloat, ok := args["Limit"].(float64); ok && limitFloat > 0 {
		limit = int(limitFloat)
	}

	entries, err := d.client.FindTroubleshootPage(symptom)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to search troubleshooting pages", err), nil
	}

	if len(entries) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No troubleshooting pages found for %q. Try docs-search for a broader search.", symptom)), nil
	}

	if len(entries) > limit {
		entries = entries[:limit]
	}

	// Fetch content for the top result
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d troubleshooting result(s) for %q:\n\n", len(entries), symptom))

	topEntry := entries[0]
	topURL := canonicalDocURL(topEntry.URL)
	content, err := d.client.FetchDocPage(topEntry.URL)
	if err == nil && len(content) > 0 {
		fmt.Fprintf(&sb, "## %s\n\nSource: %s\n", topEntry.Title, topURL)
		if topEntry.LastModified != "" {
			fmt.Fprintf(&sb, "Last updated: %s\n", topEntry.LastModified)
		}
		page, next := paginate(content, 0, maxEmbeddedPageBytes)
		fmt.Fprintf(&sb, "\n%s\n", page)
		if next < len(content) {
			writeContinuation(&sb, topURL, next, len(content))
		}
		sb.WriteString("\n---\n\n")
	} else {
		d.writeEntries(ctx, &sb, 1, entries[:1], symptom)
	}

	if len(entries) > 1 {
		sb.WriteString("### Other related pages:\n\n")
		d.writeEntries(ctx, &sb, 2, entries[1:], symptom)
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// getRelatedLinks extracts and categorizes outbound documentation links from a docs page.
func (d *DocsTool) getRelatedLinks(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	url, ok := args["URL"].(string)
	if !ok || url == "" {
		return mcp.NewToolResultError("URL is required and must be a non-empty string"), nil
	}

	links, err := d.client.ExtractRelatedLinks(url)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to extract related links", err), nil
	}

	if len(links) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No related documentation links found on %s.", url)), nil
	}

	// Group by category
	categories := make(map[string][]RelatedLink)
	var categoryOrder []string
	for _, link := range links {
		if _, exists := categories[link.Category]; !exists {
			categoryOrder = append(categoryOrder, link.Category)
		}
		categories[link.Category] = append(categories[link.Category], link)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Related links from %s (%d links):\n\n", url, len(links)))

	for _, cat := range categoryOrder {
		sb.WriteString(fmt.Sprintf("### %s\n", cat))
		for _, link := range categories[cat] {
			sb.WriteString(fmt.Sprintf("- [%s](%s)\n", link.Title, link.URL))
		}
		sb.WriteByte('\n')
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// Tools returns the list of MCP server tools for documentation.
func (d *DocsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.searchDocs,
			Tool: mcp.NewTool(
				"docs-search",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Keyword search over the titles, descriptions, and URLs of every public DigitalOcean docs page, including individual support articles, API/CLI/SDK references, and how-to guides. Returns ranked results with canonical URL, product, page type, last-updated date, and an excerpt: the passage of the page body that mentions the query terms when available, otherwise the page description. Ranking does not search page bodies; use docs-get-page to read a result in full."),
				mcp.WithString("Query", mcp.Required(), mcp.Description("Search keywords (e.g., \"why is smtp blocked\", \"spaces cors\", \"doctl compute droplet create\")")),
				mcp.WithNumber("Limit", mcp.DefaultNumber(defaultSearchLimit), mcp.Description("Maximum number of results to return")),
				mcp.WithString("Product", mcp.Description("Optional product or area to restrict results to, e.g. \"droplets\", \"app platform\", \"kubernetes\", \"vpc\", \"support\", \"api\". Omit to search all pages.")),
			),
		},
		{
			Handler: d.getDoc,
			Tool: mcp.NewTool(
				"docs-get-page",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Fetch the markdown content of a specific DigitalOcean docs page. Returns the whole page by default; set MaxLength to read it in windows, and follow the StartIndex given in the truncation note to continue."),
				mcp.WithString("URL", mcp.Required(), mcp.Description("Full URL or path of the docs page (e.g., https://docs.digitalocean.com/products/droplets/getting-started/quickstart/ or /products/droplets/getting-started/quickstart/)")),
				mcp.WithNumber("MaxLength", mcp.Description("Maximum UTF-8 bytes to return (integer, 0 to 10485760). Omit or 0 for the whole page. Must fit the next character.")),
				mcp.WithNumber("StartIndex", mcp.Description("UTF-8 byte offset to start from (integer, 0 to 10485760), as given by a previous truncation note. An offset inside a character advances to the next character. Defaults to 0.")),
			),
		},
		{
			Handler: d.findDocsForService,
			Tool: mcp.NewTool(
				"docs-find-for-service",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Given a DigitalOcean service name (e.g., \"droplets\", \"managed kubernetes\", \"app platform\"), return a list of relevant documentation pages with titles and URLs."),
				mcp.WithString("Service", mcp.Required(), mcp.Description("DigitalOcean service name (e.g., \"droplets\", \"kubernetes\", \"app platform\", \"databases\")")),
			),
		},
		{
			Handler: d.getQuickstart,
			Tool: mcp.NewTool(
				"docs-get-quickstart",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get the quickstart or getting-started guide for a DigitalOcean service. Returns the full content as clean markdown."),
				mcp.WithString("Service", mcp.Required(), mcp.Description("DigitalOcean service name (e.g., \"droplets\", \"kubernetes\", \"app platform\")")),
			),
		},
		{
			Handler: d.troubleshoot,
			Tool: mcp.NewTool(
				"docs-troubleshoot",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Find support and troubleshooting articles matching an error message or symptom by keyword, ranking support articles and troubleshooting-style titles first. Returns the content of the best matching page (up to 16 KB, with a pointer to the rest), plus related pages with excerpts. Use this when diagnosing errors or unexpected behavior with DigitalOcean services."),
				mcp.WithString("Symptom", mcp.Required(), mcp.Description("Error message, symptom description, or keywords describing the problem (e.g., \"SSL certificate not working\", \"520 status code\", \"deployment failed health check\")")),
				mcp.WithNumber("Limit", mcp.DefaultNumber(5), mcp.Description("Maximum number of results to return")),
			),
		},
		{
			Handler: d.getRelatedLinks,
			Tool: mcp.NewTool(
				"docs-get-related",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Extract and categorize all outbound documentation links from a specific docs page. Returns links grouped by type (how-to, reference, support, getting-started, concept, details). Use this to discover related pages and navigate the docs graph."),
				mcp.WithString("URL", mcp.Required(), mcp.Description("Full URL or path of the docs page to extract links from (e.g., https://docs.digitalocean.com/products/app-platform/how-to/manage-domains/ or /products/app-platform/how-to/manage-domains/)")),
			),
		},
	}
}
