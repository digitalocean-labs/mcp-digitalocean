//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDocsSearchAPI starts a docs-only anonymous server, the shape of the
// hosted docs deployment, and exercises the plain-HTTP search endpoint
// against the live docs index.
func TestDocsSearchAPI(t *testing.T) {
	ctx := context.Background()

	container, err := startMcpServer(ctx, McpServerConfig{
		BindAddr:  "0.0.0.0:8080",
		LogLevel:  "debug",
		Transport: "http",
		Services:  "docs",
		ExtraEnv: map[string]string{
			"ALLOW_ANONYMOUS":              "true",
			"DOCS_SEARCH_API":              "true",
			"DOCS_SEARCH_RATE_LIMIT_MODE":  "enforce",
			"DOCS_SEARCH_CLIENT_IP_HEADER": "CF-Connecting-IP",
			"DOCS_SEARCH_RATE_LIMIT_RPS":   "1",
			"DOCS_SEARCH_RATE_LIMIT_BURST": "5",
		},
	})
	require.NoError(t, err, "failed to start MCP server")
	defer container.Terminate(ctx)

	port, err := container.MappedPort(ctx, "8080/tcp")
	require.NoError(t, err)
	base := fmt.Sprintf("http://127.0.0.1:%s/v1/search", port.Port())
	httpClient := &http.Client{Timeout: 30 * time.Second}

	get := func(t *testing.T, query, clientIP string) *http.Response {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+query, nil)
		require.NoError(t, err)
		req.Header.Set("CF-Connecting-IP", clientIP)
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		return resp
	}

	t.Run("returns ranked results", func(t *testing.T) {
		resp := get(t, "?q=how+to+create+a+droplet&limit=3", "198.51.100.1")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				MarkdownURL string `json:"markdown_url"`
			} `json:"results"`
			Meta struct {
				Total         int    `json:"total"`
				PagesSearched int    `json:"pages_searched"`
				Stability     string `json:"stability"`
			} `json:"meta"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.NotEmpty(t, body.Results)
		require.LessOrEqual(t, len(body.Results), 3)
		require.Contains(t, body.Results[0].URL, "https://docs.digitalocean.com/")
		require.Greater(t, body.Meta.PagesSearched, 1000)
		require.Equal(t, "stable", body.Meta.Stability)
	})

	t.Run("rejects unknown parameters", func(t *testing.T) {
		resp := get(t, "?q=droplet&cachebust=1", "198.51.100.2")
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("enforces the rate limit", func(t *testing.T) {
		var last int
		for i := 0; i < 8; i++ {
			resp := get(t, "?q=vpc", "198.51.100.3")
			resp.Body.Close()
			last = resp.StatusCode
		}
		require.Equal(t, http.StatusTooManyRequests, last)
	})
}
