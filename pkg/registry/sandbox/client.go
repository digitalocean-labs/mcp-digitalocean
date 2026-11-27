package sandbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// sandboxesPath is the base of the public /v2/sandboxes API. It has no leading
// slash because godo resolves paths against the configured base URL.
const sandboxesPath = "v2/sandboxes"

// Tool builds the Sandboxes MCP tools. Handlers obtain a godo client from the
// injected factory, so stdio and HTTP transports share one code path.
type Tool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewTool creates a Sandboxes MCP tool set.
func NewTool(client func(ctx context.Context) (*godo.Client, error)) *Tool {
	return &Tool{client: client}
}

// Tools returns every Sandboxes MCP tool.
func (t *Tool) Tools() []server.ServerTool {
	var tools []server.ServerTool
	tools = append(tools, t.sandboxTools()...)
	tools = append(tools, t.snapshotTools()...)
	tools = append(tools, t.templateTools()...)
	tools = append(tools, t.commandTools()...)
	tools = append(tools, t.execTools()...)
	tools = append(tools, t.fileTools()...)
	return tools
}

// do sends one JSON API request. body is encoded as JSON when non-nil, and out
// receives the decoded response when non-nil.
func (t *Tool) do(ctx context.Context, method, path string, body, out any) error {
	client, err := t.client(ctx)
	if err != nil {
		return fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	req, err := client.NewRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	_, err = client.Do(ctx, req, out)
	return err
}

// sandboxPath joins path segments under /v2/sandboxes, escaping each one so a
// caller-supplied ID cannot change the route.
func sandboxPath(segments ...string) string {
	parts := []string{sandboxesPath}
	for _, s := range segments {
		parts = append(parts, url.PathEscape(s))
	}
	return strings.Join(parts, "/")
}

// withQuery appends q to path when q is non-empty.
func withQuery(path string, q url.Values) string {
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

// pageQuery builds page_size and page_token from the PageSize and PageToken
// arguments shared by the list tools.
func pageQuery(args map[string]any) url.Values {
	q := url.Values{}
	if n, ok := argInt(args, "PageSize"); ok && n > 0 {
		q.Set("page_size", strconv.FormatInt(n, 10))
	}
	if tok := argString(args, "PageToken"); tok != "" {
		q.Set("page_token", tok)
	}
	return q
}

// argString returns a trimmed string argument, or "" when absent.
func argString(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return strings.TrimSpace(v)
}

// requireString returns a trimmed required string argument, or a tool error.
func requireString(args map[string]any, key string) (string, *mcp.CallToolResult) {
	v := argString(args, key)
	if v == "" {
		return "", mcp.NewToolResultError(key + " is required")
	}
	return v, nil
}

// argInt returns a numeric argument. MCP delivers JSON numbers as float64.
func argInt(args map[string]any, key string) (int64, bool) {
	switch v := args[key].(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	}
	return 0, false
}

// argBool returns a boolean argument, or false when absent.
func argBool(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
}

// argStringSlice returns a string-array argument, skipping non-string items.
func argStringSlice(args map[string]any, key string) []string {
	raw, _ := args[key].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// argStringMap returns a string-to-string object argument, or nil when empty.
func argStringMap(args map[string]any, key string) map[string]string {
	raw, _ := args[key].(map[string]any)
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprint(v)
	}
	return out
}
