package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type AuthKey struct{}

// RouterResourceKey is the context key under which the encrypted resource
// identifier supplied by a trusted MCP router is stored.
type RouterResourceKey struct{}

// RouterResourceHeader is the request header a trusted MCP router uses to
// forward the AES-256-GCM ciphertext of its own MCP resource URL. When present,
// this server relays it to the DigitalOcean API as the encrypted resource
// identifier so the backend validates the OAuth audience against the router
// (the resource the client actually authenticated to) instead of this server.
const RouterResourceHeader = "X-Router-Encrypted-Resource-Identifier"

// UserAgentKey is the context key under which the caller's User-Agent is stored.
type UserAgentKey struct{}

// maxUserAgentLen caps the caller-controlled User-Agent before it is logged.
const maxUserAgentLen = 256

// AuthFromRequest extracts the auth token, any router-supplied encrypted
// resource identifier, and the caller's User-Agent from the request headers.
func AuthFromRequest(ctx context.Context, r *http.Request) context.Context {
	ctx = WithAuthKey(ctx, r.Header.Get("Authorization"))
	ctx = WithUserAgent(ctx, r.UserAgent())
	return WithRouterResource(ctx, r.Header.Get(RouterResourceHeader))
}

// WithUserAgent adds the caller's User-Agent to the context, trimmed, stripped
// of control characters, and capped at maxUserAgentLen bytes.
func WithUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, UserAgentKey{}, sanitizeUserAgent(userAgent))
}

// UserAgentFromContext returns the caller's User-Agent stored in ctx, or ""
// when absent.
func UserAgentFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(UserAgentKey{}).(string); ok {
		return v
	}
	return ""
}

func sanitizeUserAgent(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(s))
	if len(s) <= maxUserAgentLen {
		return s
	}
	s = s[:maxUserAgentLen]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// WithAuthKey adds an auth key to the context.
func WithAuthKey(ctx context.Context, auth string) context.Context {
	return context.WithValue(ctx, AuthKey{}, auth)
}

// WithRouterResource adds the router-supplied encrypted resource identifier to
// the context.
func WithRouterResource(ctx context.Context, encryptedResource string) context.Context {
	return context.WithValue(ctx, RouterResourceKey{}, encryptedResource)
}

// RouterResourceFromContext returns the router-supplied encrypted resource
// identifier stored in ctx, or "" when absent.
func RouterResourceFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(RouterResourceKey{}).(string); ok {
		return v
	}
	return ""
}

// ToolLoggingMiddleware is a middleware that logs tool errors.
type ToolLoggingMiddleware struct {
	Logger *slog.Logger
}

const (
	// ToolCallResultError is an error that is driven by a faulty llm or user input. These error are typically retryable upon self-correction.
	ToolCallResultError = "tool_call_result_error"

	// ToolCallError is an error that is out of the control of the client. For instance, the API server is down. In this case, no amount of self-correction will be helpful.
	ToolCallError = "tool_call_error"

	// ToolCallSuccess is when the call succeeds entirely.
	ToolCallSuccess = "tool_call_success"
)

// ToolMiddleware wraps a tool handler to log duration and success/error status.
func (m *ToolLoggingMiddleware) ToolMiddleware(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		result, err := next(ctx, req)
		userAgent := UserAgentFromContext(ctx)
		if err != nil {
			m.Logger.Error("tool call result",
				"tool", req.Params.Name,
				"duration_seconds", time.Since(start).Seconds(),
				"request.user_agent", userAgent,
				"error", err,
				"tool_call_outcome", ToolCallError,
			)
			return result, err
		}

		if result.IsError {
			var payload string
			if len(result.Content) > 0 {
				textContent, ok := result.Content[0].(mcp.TextContent)
				if ok {
					payload = textContent.Text
				}
			}
			m.Logger.Error("tool call result",
				"tool", req.Params.Name,
				"duration_seconds", time.Since(start).Seconds(),
				"request.user_agent", userAgent,
				"content", payload,
				"tool_call_outcome", ToolCallResultError,
			)
			return result, err
		}

		m.Logger.Info("tool call result",
			"tool", req.Params.Name,
			"duration_seconds", time.Since(start).Seconds(),
			"request.user_agent", userAgent,
			"tool_call_outcome", ToolCallSuccess,
		)

		return result, err
	}
}
