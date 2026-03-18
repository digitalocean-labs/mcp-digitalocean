package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ReadOnlyMiddleware is a middleware that blocks mutating tool calls.
type ReadOnlyMiddleware struct {
	Logger     *slog.Logger
	isMutating func(string) bool
}

var mutatingActionTokens = map[string]struct{}{
	"add":      {},
	"assign":   {},
	"change":   {},
	"convert":  {},
	"create":   {},
	"delete":   {},
	"disable":  {},
	"enable":   {},
	"flush":    {},
	"install":  {},
	"power":    {},
	"reboot":   {},
	"rebuild":  {},
	"recycle":  {},
	"release":  {},
	"remove":   {},
	"rename":   {},
	"reserve":  {},
	"reset":    {},
	"resize":   {},
	"restore":  {},
	"set":      {},
	"shutdown": {},
	"snapshot": {},
	"start":    {},
	"stop":     {},
	"transfer": {},
	"unassign": {},
	"update":   {},
	"upgrade":  {},
}

// defaultIsMutatingTool checks whether a tool name is mutating
// by checking if any hyphen-separated token matches a known mutating action.
// Tool names use varied conventions: "apps-delete" (action last), "power-on-droplet" (action first),
// "reset-droplet-password" (action first), so all tokens must be checked.
func defaultIsMutatingTool(toolName string) bool {
	tokens := strings.Split(strings.ToLower(toolName), "-")
	for _, token := range tokens {
		if _, ok := mutatingActionTokens[token]; ok {
			return true
		}
	}
	return false
}

// ToolMiddleware wraps a tool handler to block mutating tools in read-only mode.
func (m *ReadOnlyMiddleware) ToolMiddleware(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// default to the internal mutating classifier if no custom classifier is provided.
		classifier := m.isMutating
		if classifier == nil {
			classifier = defaultIsMutatingTool
		}

		toolName := req.Params.Name
		if classifier(toolName) {
			if m.Logger != nil {
				m.Logger.Warn("blocked mutating tool in read-only mode", "tool", toolName)
			}
			return mcp.NewToolResultError(fmt.Sprintf("tool %q is disabled in read-only mode", toolName)), nil
		}

		return next(ctx, req)
	}
}
