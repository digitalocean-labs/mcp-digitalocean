//go:build integration

package testing

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestDocsSearch(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	t.Run("valid query", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-search",
				Arguments: map[string]interface{}{
					"Query": "how to create a droplet",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "result(s) for")
		require.Contains(t, text, "https://docs.digitalocean.com")
		t.Logf("Search returned results: %.200s...", text)
	})

	t.Run("with limit", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-search",
				Arguments: map[string]interface{}{
					"Query": "kubernetes",
					"Limit": 3,
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "result(s) for")
		t.Logf("Search with limit returned: %.200s...", text)
	})

	t.Run("missing query", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "docs-search",
				Arguments: map[string]interface{}{},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for missing query")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "Query is required")
	})
}

func TestDocsGetPage(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	t.Run("valid URL", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-get-page",
				Arguments: map[string]interface{}{
					"URL": "https://docs.digitalocean.com/products/droplets/getting-started/quickstart/",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "Droplet")
		require.NotEmpty(t, text)
		t.Logf("Fetched page content: %.200s...", text)
	})

	t.Run("relative path", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-get-page",
				Arguments: map[string]interface{}{
					"URL": "/products/droplets/getting-started/quickstart/",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "Droplet")
	})

	t.Run("invalid URL", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-get-page",
				Arguments: map[string]interface{}{
					"URL": "https://docs.digitalocean.com/nonexistent-page-that-does-not-exist/",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for invalid URL")
	})

	t.Run("missing URL", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "docs-get-page",
				Arguments: map[string]interface{}{},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for missing URL")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "URL is required")
	})
}

func TestDocsFindForService(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	t.Run("valid service", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-find-for-service",
				Arguments: map[string]interface{}{
					"Service": "droplets",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "droplets")
		require.Contains(t, text, "https://docs.digitalocean.com")
		t.Logf("Found docs for droplets: %.200s...", text)
	})

	t.Run("service alias", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-find-for-service",
				Arguments: map[string]interface{}{
					"Service": "k8s",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "https://docs.digitalocean.com")
		t.Logf("Found docs for k8s alias: %.200s...", text)
	})

	t.Run("invalid service", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-find-for-service",
				Arguments: map[string]interface{}{
					"Service": "nonexistent-service-xyz",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		// Should not error — returns a helpful fallback message
		require.False(t, resp.IsError)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "No documentation found")
	})

	t.Run("missing service", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "docs-find-for-service",
				Arguments: map[string]interface{}{},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for missing service")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "Service is required")
	})
}

func TestDocsGetQuickstart(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	t.Run("valid service", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-get-quickstart",
				Arguments: map[string]interface{}{
					"Service": "droplets",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "Quickstart")
		require.Contains(t, text, "droplets")
		t.Logf("Got quickstart: %.200s...", text)
	})

	t.Run("service with no quickstart", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-get-quickstart",
				Arguments: map[string]interface{}{
					"Service": "nonexistent-service-xyz",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		// Should not error — returns a helpful fallback message
		require.False(t, resp.IsError)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "No quickstart guide found")
	})

	t.Run("missing service", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "docs-get-quickstart",
				Arguments: map[string]interface{}{},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for missing service")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "Service is required")
	})
}

func TestDocsTroubleshoot(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	t.Run("valid symptom", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-troubleshoot",
				Arguments: map[string]interface{}{
					"Symptom": "520 status code app platform",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "troubleshooting result(s)")
		require.Contains(t, text, "https://docs.digitalocean.com")
		t.Logf("Troubleshoot returned: %.300s...", text)
	})

	t.Run("with limit", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-troubleshoot",
				Arguments: map[string]interface{}{
					"Symptom": "deployment failed",
					"Limit":   2,
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "troubleshooting result(s)")
		t.Logf("Troubleshoot with limit returned: %.300s...", text)
	})

	t.Run("missing symptom", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "docs-troubleshoot",
				Arguments: map[string]interface{}{},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for missing symptom")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "Symptom is required")
	})
}

func TestDocsGetRelated(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	t.Run("valid URL", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-get-related",
				Arguments: map[string]interface{}{
					"URL": "https://docs.digitalocean.com/products/app-platform/how-to/manage-domains/",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "Related links from")
		require.Contains(t, text, "https://docs.digitalocean.com")
		t.Logf("Related links returned: %.300s...", text)
	})

	t.Run("relative path", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-get-related",
				Arguments: map[string]interface{}{
					"URL": "/products/droplets/getting-started/quickstart/",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

		text := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, text, "https://docs.digitalocean.com")
	})

	t.Run("invalid URL", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "docs-get-related",
				Arguments: map[string]interface{}{
					"URL": "https://docs.digitalocean.com/nonexistent-page-that-does-not-exist/",
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for invalid URL")
	})

	t.Run("missing URL", func(t *testing.T) {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "docs-get-related",
				Arguments: map[string]interface{}{},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.IsError, "Expected error for missing URL")

		errorText := resp.Content[0].(mcp.TextContent).Text
		require.Contains(t, errorText, "URL is required")
	})
}
