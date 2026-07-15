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

// searchDocs performs full-text search across all DigitalOcean documentation.
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
	sb.WriteString(fmt.Sprintf("Found %d result(s) for %q:\n\n", len(results), query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n", i+1, r.Title, r.URL))
		if r.Description != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Description))
		}
		sb.WriteByte('\n')
	}

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

	results := SearchIndex(mainIndex, service)

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

// Tools returns the list of MCP server tools for documentation.
func (d *DocsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.searchDocs,
			Tool: mcp.NewTool(
				"docs-search",
				common.WithHints(common.HintsRead),
				mcp.WithDescription("Full-text search across DigitalOcean documentation. Returns ranked results with title, URL, and content snippet."),
				mcp.WithString("Query", mcp.Required(), mcp.Description("Search query string")),
				mcp.WithNumber("Limit", mcp.DefaultNumber(defaultSearchLimit), mcp.Description("Maximum number of results to return")),
			),
		},
		{
			Handler: d.getDoc,
			Tool: mcp.NewTool(
				"docs-get-page",
				common.WithHints(common.HintsRead),
				mcp.WithDescription("Fetch the full markdown content of a specific DigitalOcean docs page. Returns clean markdown suitable for LLM consumption."),
				mcp.WithString("URL", mcp.Required(), mcp.Description("Full URL or path of the docs page (e.g., https://docs.digitalocean.com/products/droplets/getting-started/quickstart/ or /products/droplets/getting-started/quickstart/)")),
			),
		},
		{
			Handler: d.findDocsForService,
			Tool: mcp.NewTool(
				"docs-find-for-service",
				common.WithHints(common.HintsRead),
				mcp.WithDescription("Given a DigitalOcean service name (e.g., \"droplets\", \"managed kubernetes\", \"app platform\"), return a list of relevant documentation pages with titles and URLs."),
				mcp.WithString("Service", mcp.Required(), mcp.Description("DigitalOcean service name (e.g., \"droplets\", \"kubernetes\", \"app platform\", \"databases\")")),
			),
		},
		{
			Handler: d.getQuickstart,
			Tool: mcp.NewTool(
				"docs-get-quickstart",
				common.WithHints(common.HintsRead),
				mcp.WithDescription("Get the quickstart or getting-started guide for a DigitalOcean service. Returns the full content as clean markdown."),
				mcp.WithString("Service", mcp.Required(), mcp.Description("DigitalOcean service name (e.g., \"droplets\", \"kubernetes\", \"app platform\")")),
			),
		},
	}
}
