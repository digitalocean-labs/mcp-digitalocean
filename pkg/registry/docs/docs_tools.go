package docs

import (
	"context"
	"fmt"
	"strings"

	"mcp-digitalocean/pkg/registry/common"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultSearchLimit = 10
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
// canonical URL, product, page type, last-modified date, and the page
// description as an excerpt.
func writeEntry(sb *strings.Builder, position int, entry DocsEntry) {
	fmt.Fprintf(sb, "%d. %s\n   %s\n", position, entry.Title, canonicalDocURL(entry.URL))

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
	if entry.Description != "" {
		fmt.Fprintf(sb, "   %s\n", entry.Description)
	}
	sb.WriteByte('\n')
}

// canonicalDocURL strips the index.html.md suffix that llms.txt entries carry
// so results always cite the canonical page URL.
func canonicalDocURL(url string) string {
	return strings.TrimSuffix(url, "index.html.md")
}

// searchDocs ranks documentation pages by keyword match against page metadata.
func (d *DocsTool) searchDocs(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	query, ok := args["Query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("Query is required and must be a non-empty string"), nil
	}

	limit := defaultSearchLimit
	if limitFloat, ok := args["Limit"].(float64); ok && limitFloat > 0 {
		limit = int(limitFloat)
	}

	index, err := d.client.GetDocsIndex()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to load docs index", err), nil
	}

	results := SearchIndex(index, query)

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No results found for %q. Try different search terms or a more general query.", query)), nil
	}

	if len(results) > limit {
		results = results[:limit]
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d result(s) for %q (searched %d pages):\n\n", len(results), query, len(index.Entries))
	for i, r := range results {
		writeEntry(&sb, i+1, r)
	}
	sb.WriteString("Use docs-get-page with a result URL to read the full page.\n")

	return mcp.NewToolResultText(sb.String()), nil
}

// getDoc fetches the full markdown content of a specific docs page.
func (d *DocsTool) getDoc(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	url, ok := args["URL"].(string)
	if !ok || url == "" {
		return mcp.NewToolResultError("URL is required and must be a non-empty string"), nil
	}

	content, err := d.client.FetchDocPage(url)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to fetch doc page", err), nil
	}

	return mcp.NewToolResultText(content), nil
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
func (d *DocsTool) troubleshoot(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	content, err := d.client.FetchDocPage(topEntry.URL)
	if err == nil && len(content) > 0 {
		fmt.Fprintf(&sb, "## %s\n\nSource: %s\n", topEntry.Title, canonicalDocURL(topEntry.URL))
		if topEntry.LastModified != "" {
			fmt.Fprintf(&sb, "Last updated: %s\n", topEntry.LastModified)
		}
		fmt.Fprintf(&sb, "\n%s\n\n---\n\n", content)
	} else {
		writeEntry(&sb, 1, topEntry)
	}

	if len(entries) > 1 {
		sb.WriteString("### Other related pages:\n\n")
		for i, e := range entries[1:] {
			writeEntry(&sb, i+2, e)
		}
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
				mcp.WithDescription("Keyword search over the titles, descriptions, and URLs of every public DigitalOcean docs page, including individual support articles, API/CLI/SDK references, and how-to guides. Returns ranked results with canonical URL, product, page type, last-updated date, and a one-line excerpt. Does not search page bodies; use docs-get-page to read a result in full."),
				mcp.WithString("Query", mcp.Required(), mcp.Description("Search keywords (e.g., \"why is smtp blocked\", \"spaces cors\", \"doctl compute droplet create\")")),
				mcp.WithNumber("Limit", mcp.DefaultNumber(defaultSearchLimit), mcp.Description("Maximum number of results to return")),
			),
		},
		{
			Handler: d.getDoc,
			Tool: mcp.NewTool(
				"docs-get-page",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Fetch the full markdown content of a specific DigitalOcean docs page. Returns clean markdown suitable for LLM consumption."),
				mcp.WithString("URL", mcp.Required(), mcp.Description("Full URL or path of the docs page (e.g., https://docs.digitalocean.com/products/droplets/getting-started/quickstart/ or /products/droplets/getting-started/quickstart/)")),
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
				mcp.WithDescription("Find support and troubleshooting articles matching an error message or symptom by keyword, ranking support articles and troubleshooting-style titles first. Returns the full content of the best matching page, plus links to related pages. Use this when diagnosing errors or unexpected behavior with DigitalOcean services."),
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
