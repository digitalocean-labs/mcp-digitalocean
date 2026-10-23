package docs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testLlmsTxt = `# DigitalOcean Documentation
> Comprehensive tutorials, references, example code, and more for DigitalOcean products.

## Platform

### Accounts

Manage your team membership and your name, sign-in method, and email subscriptions.

- [How to Manage Account Settings](https://docs.digitalocean.com/platform/accounts/settings/index.html.md): The My Account page lets you view and edit your login method.
- [How to Manage Two-Factor Authentication](https://docs.digitalocean.com/platform/accounts/2fa/index.html.md): Use 2FA to add security.

## Products

### Droplets

DigitalOcean Droplets are Linux-based virtual machines.

- [Droplet Quickstart](https://docs.digitalocean.com/products/droplets/getting-started/quickstart/index.html.md): Get started with Droplets quickly.
- [How to Create a Droplet](https://docs.digitalocean.com/products/droplets/how-to/create/index.html.md): Create a new Droplet from the control panel or API.
- [How to Resize Droplets](https://docs.digitalocean.com/products/droplets/how-to/resize/index.html.md): Resize your Droplet to a different plan.

### Kubernetes

DigitalOcean Kubernetes is a managed Kubernetes service.

- [Kubernetes Quickstart](https://docs.digitalocean.com/products/kubernetes/getting-started/quickstart/index.html.md): Get started with DOKS.
- [How to Create Clusters](https://docs.digitalocean.com/products/kubernetes/how-to/create-clusters/index.html.md): Create a new Kubernetes cluster.

### App Platform

Build, deploy, and scale apps quickly.

- [App Platform Quickstart](https://docs.digitalocean.com/products/app-platform/getting-started/quickstart/index.html.md): Deploy your first app.

## Reference

### API

- [API Reference](https://docs.digitalocean.com/reference/api/index.html.md): DigitalOcean API v2 reference.
`

func TestParseLlmsTxt(t *testing.T) {
	index := parseLlmsTxt(testLlmsTxt)

	require.NotNil(t, index)
	require.Equal(t, 9, len(index.Entries))

	// Check sections
	require.Contains(t, index.Sections, "Accounts")
	require.Contains(t, index.Sections, "Droplets")
	require.Contains(t, index.Sections, "Kubernetes")
	require.Contains(t, index.Sections, "App Platform")
	require.Contains(t, index.Sections, "API")

	// Check first entry
	require.Equal(t, "How to Manage Account Settings", index.Entries[0].Title)
	require.Equal(t, "https://docs.digitalocean.com/platform/accounts/settings/index.html.md", index.Entries[0].URL)
	require.Equal(t, "The My Account page lets you view and edit your login method.", index.Entries[0].Description)
	require.Equal(t, "Accounts", index.Entries[0].Section)
}

func TestParseLlmsTxt_Empty(t *testing.T) {
	index := parseLlmsTxt("")
	require.NotNil(t, index)
	require.Empty(t, index.Entries)
	require.Empty(t, index.Sections)
}

func TestSearchIndex(t *testing.T) {
	index := parseLlmsTxt(testLlmsTxt)

	tests := []struct {
		name          string
		query         string
		expectMinimum int
		expectFirst   string
	}{
		{
			name:          "search by service name",
			query:         "droplets",
			expectMinimum: 3,
		},
		{
			name:          "search for quickstart",
			query:         "quickstart kubernetes",
			expectMinimum: 1,
			expectFirst:   "Kubernetes Quickstart",
		},
		{
			name:          "search for create",
			query:         "create",
			expectMinimum: 2,
		},
		{
			name:          "search for resize",
			query:         "resize",
			expectMinimum: 1,
			expectFirst:   "How to Resize Droplets",
		},
		{
			name:          "no results",
			query:         "xyznonexistent12345",
			expectMinimum: 0,
		},
		{
			name:          "empty query",
			query:         "",
			expectMinimum: 0,
		},
		{
			name:          "matches URL path segments",
			query:         "quickstart",
			expectMinimum: 3,
		},
		{
			name:          "all terms boost ranks multi-term match higher",
			query:         "create droplet",
			expectMinimum: 1,
			expectFirst:   "How to Create a Droplet",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := SearchIndex(index, tc.query)
			if tc.expectMinimum == 0 {
				require.Empty(t, results)
				return
			}
			require.GreaterOrEqual(t, len(results), tc.expectMinimum)
			if tc.expectFirst != "" && len(results) > 0 {
				require.Equal(t, tc.expectFirst, results[0].Title)
			}
		})
	}
}

func TestResolveServiceSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"k8s", "kubernetes"},
		{"doks", "kubernetes"},
		{"managed kubernetes", "kubernetes"},
		{"KUBERNETES", "kubernetes"},
		{"app platform", "app-platform"},
		{"apps", "app-platform"},
		{"gpu", "bare-metal-gpus"},
		{"vms", "droplets"},
		{"dbaas", "databases"},
		{"object storage", "spaces"},
		{"docr", "container-registry"},
		{"serverless", "functions"},
		{"unknown service", "unknown-service"},
		{"  droplets  ", "droplets"},
		{"dedicated inference", "ai-platform"},
		{"dedicated-inference", "ai-platform"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := resolveServiceSlug(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFetchDocPage_RejectsExternalHosts(t *testing.T) {
	client := NewDocsClient()

	tests := []string{
		"https://example.com/products/droplets/",
		"https://docs.digitalocean.com.evil.example/products/droplets/",
		"http://attacker.example/",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			_, err := client.FetchDocPage(url)
			require.Error(t, err)
			require.Contains(t, err.Error(), "refusing to fetch")
		})
	}
}

func TestNormalizeDocURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"https://docs.digitalocean.com/products/droplets/how-to/create/",
			"https://docs.digitalocean.com/products/droplets/how-to/create/",
		},
		{
			"https://docs.digitalocean.com/products/droplets/how-to/create/#step-1",
			"https://docs.digitalocean.com/products/droplets/how-to/create/",
		},
		{
			"https://docs.digitalocean.com/products/droplets/how-to/create/index.html.md",
			"https://docs.digitalocean.com/products/droplets/how-to/create/",
		},
		{
			"/products/droplets/how-to/create/",
			"https://docs.digitalocean.com/products/droplets/how-to/create/",
		},
		{
			"products/droplets/",
			"https://docs.digitalocean.com/products/droplets/",
		},
		{
			"https://docs.digitalocean.com/products/droplets",
			"https://docs.digitalocean.com/products/droplets/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeDocURL(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestCategorizeDocLink(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://docs.digitalocean.com/products/droplets/how-to/create/", "how-to"},
		{"https://docs.digitalocean.com/reference/api/", "reference"},
		{"https://docs.digitalocean.com/support/520-error/", "support"},
		{"https://docs.digitalocean.com/products/droplets/getting-started/quickstart/", "getting-started"},
		{"https://docs.digitalocean.com/products/droplets/concepts/choosing-a-plan/", "concept"},
		{"https://docs.digitalocean.com/products/droplets/details/pricing/", "details"},
		{"https://docs.digitalocean.com/products/droplets/", "other"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := categorizeDocLink(tc.url)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFindTroubleshootPage(t *testing.T) {
	index := parseLlmsTxt(testLlmsTxt + `
### App Platform Support

- [Why am I receiving 520 status codes from my app?](https://docs.digitalocean.com/support/520-status-codes/index.html.md): Your app may have crashed.
- [My app deployment failed because of a health check](https://docs.digitalocean.com/support/health-check-failed/index.html.md): Your app is unavailable on the port.
`)

	// Verify support entries were parsed
	var supportCount int
	for _, e := range index.Entries {
		if e.Section == "App Platform Support" {
			supportCount++
		}
	}
	require.Equal(t, 2, supportCount, "should have parsed 2 support entries")
	client := NewDocsClient()
	client.cache.set("docsIndex", index, indexCacheTTL)
	results, err := client.FindTroubleshootPage("520 status code")
	require.NoError(t, err)
	require.NotEmpty(t, results)
	require.Equal(t, "Why am I receiving 520 status codes from my app?", results[0].Title)
	_, err = client.FindTroubleshootPage("!!!")
	require.ErrorContains(t, err, "no searchable terms")
}

func TestCleanMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes excessive newlines",
			input:    "Hello\n\n\n\n\nWorld",
			expected: "Hello\n\nWorld",
		},
		{
			name:     "removes HTML tags",
			input:    "Hello <div>World</div>",
			expected: "Hello World",
		},
		{
			name:     "trims whitespace",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "preserves normal markdown",
			input:    "# Title\n\n- Item 1\n- Item 2",
			expected: "# Title\n\n- Item 1\n- Item 2",
		},
		{
			name:     "keeps placeholders in a fenced code block",
			input:    "Run it:\n\n```shell\ndoctl compute ssh <droplet-id|name> [flags]\n```",
			expected: "Run it:\n\n```shell\ndoctl compute ssh <droplet-id|name> [flags]\n```",
		},
		{
			name:     "keeps HTML inside a fenced code block",
			input:    "Example:\n\n```html\n<div class=\"app\">hello</div>\n```",
			expected: "Example:\n\n```html\n<div class=\"app\">hello</div>\n```",
		},
		{
			name:     "keeps tilde fences and nested backticks",
			input:    "~~~html\n<span>a</span>\n```\n~~~",
			expected: "~~~html\n<span>a</span>\n```\n~~~",
		},
		{
			name:     "keeps HTML inside an inline code span",
			input:    "Use `<your-token>` here, not <b>bold</b>.",
			expected: "Use `<your-token>` here, not bold.",
		},
		{
			name:     "keeps a double-backtick span containing a backtick",
			input:    "``a `<i>` b`` and <i>x</i>",
			expected: "``a `<i>` b`` and x",
		},
		{
			name:     "keeps comparisons and autolinks in prose",
			input:    "Use when a < b && c > d, see <https://example.com>.",
			expected: "Use when a < b && c > d, see <https://example.com>.",
		},
		{
			name:     "still strips tags with attributes and self-closing tags",
			input:    `Hi <a href="/x" target="_blank">there</a><br/> you`,
			expected: "Hi there you",
		},
		{
			name:     "strips HTML comments in prose only",
			input:    "a <!-- hidden --> b\n\n```\n<!-- kept -->\n```",
			expected: "a  b\n\n```\n<!-- kept -->\n```",
		},
		{
			name:     "an unclosed code fence protects the rest of the document",
			input:    "intro\n\n```\n<div>raw</div>",
			expected: "intro\n\n```\n<div>raw</div>",
		},
		{
			name:     "a fence with an info string does not close the block",
			input:    "```\n<x>\n```python\n<div>kept</div>\n```",
			expected: "```\n<x>\n```python\n<div>kept</div>\n```",
		},
		{
			name:     "leaves blank lines inside a code block alone",
			input:    "```python\ndef a():\n    pass\n\n\ndef b():\n    pass\n```",
			expected: "```python\ndef a():\n    pass\n\n\ndef b():\n    pass\n```",
		},
		{
			name:     "collapses blank lines across a code block boundary",
			input:    "intro\n\n\n\n```\ncode\n```\n\n\n\noutro",
			expected: "intro\n\n```\ncode\n```\n\noutro",
		},
		{
			name:     "strips a doctype declaration",
			input:    "<!doctype html>\n<div>hi</div>",
			expected: "hi",
		},
		{
			name:     "strips a comment that spans lines",
			input:    "a <!-- one\ntwo --> b",
			expected: "a  b",
		},
		{
			name:     "strips a tag whose attributes wrap onto the next line",
			input:    "<div\n  class=\"x\">hi</div>",
			expected: "hi",
		},
		{
			// A bare placeholder is indistinguishable from a tag, so prose
			// still loses it. Pages put placeholders in backticks, which the
			// case above covers.
			name:     "a prose placeholder with no backticks is still stripped",
			input:    "Replace <token> with yours",
			expected: "Replace  with yours",
		},
		{
			name:     "an unspaced comparison in prose is still stripped",
			input:    "true when a<b && c>d",
			expected: "true when ad",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cleanMarkdown(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
