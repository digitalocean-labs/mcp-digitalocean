package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"mcp-digitalocean/internal/calltrace"

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

// AuthFromRequest extracts the auth token and any router-supplied encrypted
// resource identifier from the request headers.
func AuthFromRequest(ctx context.Context, r *http.Request) context.Context {
	ctx = WithAuthKey(ctx, r.Header.Get("Authorization"))
	ctx = WithRouterResource(ctx, r.Header.Get(RouterResourceHeader))
	return WithRequestMeta(ctx, RequestMetaFromRequest(r))
}

// RequestMetaKey is the context key for RequestMeta.
type RequestMetaKey struct{}

// RequestMeta is the non-secret request context the tool logging middleware
// adds to each call record. It never holds credentials.
type RequestMeta struct {
	// ViaRouter is true when a trusted MCP router forwarded the request.
	ViaRouter bool
	// SessionRef is a truncated SHA-256 of the Mcp-Session-Id header, so calls
	// in one client session can be grouped without logging the session handle.
	SessionRef string
	// UserAgent is the caller's User-Agent, cleaned and truncated. Behind the
	// router this is the router, not the end client.
	UserAgent string
}

const (
	sessionIDHeader = "Mcp-Session-Id"
	maxUserAgentLen = 128
	maxLogValueLen  = 256
)

// RequestMetaFromRequest extracts RequestMeta from the request headers.
func RequestMetaFromRequest(r *http.Request) RequestMeta {
	meta := RequestMeta{
		ViaRouter: strings.TrimSpace(r.Header.Get(RouterResourceHeader)) != "",
		UserAgent: cleanLogValue(r.Header.Get("User-Agent"), maxUserAgentLen),
	}
	if sid := strings.TrimSpace(r.Header.Get(sessionIDHeader)); sid != "" {
		sum := sha256.Sum256([]byte(sid))
		meta.SessionRef = hex.EncodeToString(sum[:16])
	}
	return meta
}

// WithRequestMeta adds request metadata to the context.
func WithRequestMeta(ctx context.Context, meta RequestMeta) context.Context {
	return context.WithValue(ctx, RequestMetaKey{}, meta)
}

// RequestMetaFromContext returns the RequestMeta stored in ctx, or the zero
// value when absent (stdio transport).
func RequestMetaFromContext(ctx context.Context) RequestMeta {
	meta, _ := ctx.Value(RequestMetaKey{}).(RequestMeta)
	return meta
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

// ToolLoggingMiddleware is a middleware that logs one record per tool call.
type ToolLoggingMiddleware struct {
	Logger *slog.Logger
	// LogDocsQueries adds the free-text Query (docs-search) and Symptom
	// (docs-troubleshoot) arguments to docs tool records, redacted and
	// truncated. Off by default: people paste error output that can carry
	// secrets, so turning it on needs a privacy review.
	LogDocsQueries bool
}

const (
	// ToolCallResultError is an error that is driven by a faulty llm or user input. These error are typically retryable upon self-correction.
	ToolCallResultError = "tool_call_result_error"

	// ToolCallError is an error that is out of the control of the client. For instance, the API server is down. In this case, no amount of self-correction will be helpful.
	ToolCallError = "tool_call_error"

	// ToolCallSuccess is when the call succeeds entirely.
	ToolCallSuccess = "tool_call_success"
)

// ToolMiddleware wraps a tool handler to log duration, outcome, and request
// context. Docs tools also log their structured arguments and whatever the
// handler recorded through calltrace (result count, top URL). Credentials and
// arguments of authenticated tools are never logged.
func (m *ToolLoggingMiddleware) ToolMiddleware(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		ctx, trace := calltrace.New(ctx)
		result, err := next(ctx, req)

		attrs := []slog.Attr{
			slog.String("tool", req.Params.Name),
			slog.Float64("duration_seconds", time.Since(start).Seconds()),
		}
		attrs = append(attrs, requestAttrs(RequestMetaFromContext(ctx))...)
		if isDocsTool(req.Params.Name) {
			attrs = append(attrs, docsArgAttrs(req.Params.Name, req.GetArguments(), m.LogDocsQueries)...)
			attrs = append(attrs, trace.Attrs()...)
		}

		if err != nil {
			attrs = append(attrs,
				slog.Any("error", err),
				slog.String("tool_call_outcome", ToolCallError),
			)
			m.Logger.LogAttrs(ctx, slog.LevelError, "tool call result", attrs...)
			return result, err
		}

		if result != nil && result.IsError {
			var payload string
			if len(result.Content) > 0 {
				textContent, ok := result.Content[0].(mcp.TextContent)
				if ok {
					payload = textContent.Text
				}
			}
			attrs = append(attrs,
				slog.String("content", payload),
				slog.String("tool_call_outcome", ToolCallResultError),
			)
			m.Logger.LogAttrs(ctx, slog.LevelError, "tool call result", attrs...)
			return result, err
		}

		attrs = append(attrs, slog.String("tool_call_outcome", ToolCallSuccess))
		m.Logger.LogAttrs(ctx, slog.LevelInfo, "tool call result", attrs...)
		return result, err
	}
}

func requestAttrs(meta RequestMeta) []slog.Attr {
	attrs := []slog.Attr{slog.Bool("via_router", meta.ViaRouter)}
	if meta.SessionRef != "" {
		attrs = append(attrs, slog.String("session_ref", meta.SessionRef))
	}
	if meta.UserAgent != "" {
		attrs = append(attrs, slog.String("client_user_agent", meta.UserAgent))
	}
	return attrs
}

func isDocsTool(name string) bool {
	return strings.HasPrefix(name, "docs-")
}

// docsArgAttrs returns the arguments worth logging for a docs tool. Docs tools
// read public content, so URL, Service, Product, and Limit are safe to log.
// Free text goes through redactSecrets and only when logFreeText is set.
func docsArgAttrs(tool string, args map[string]any, logFreeText bool) []slog.Attr {
	var attrs []slog.Attr
	str := func(argName, attrName string) {
		if v, ok := args[argName].(string); ok && strings.TrimSpace(v) != "" {
			attrs = append(attrs, slog.String(attrName, cleanLogValue(v, maxLogValueLen)))
		}
	}
	free := func(argName, attrName string) {
		if !logFreeText {
			return
		}
		if v, ok := args[argName].(string); ok && strings.TrimSpace(v) != "" {
			attrs = append(attrs, slog.String(attrName, cleanLogValue(redactSecrets(v), maxLogValueLen)))
		}
	}
	limit := func() {
		if v, ok := args["Limit"].(float64); ok {
			attrs = append(attrs, slog.Int("limit", int(v)))
		}
	}

	switch tool {
	case "docs-search":
		free("Query", "query")
		str("Product", "product")
		limit()
	case "docs-troubleshoot":
		free("Symptom", "symptom")
		limit()
	case "docs-get-page", "docs-get-related":
		str("URL", "url")
	case "docs-find-for-service", "docs-get-quickstart":
		str("Service", "service")
	}
	return attrs
}

// secretPatterns match strings that look like credentials: DigitalOcean token
// prefixes, bearer and basic auth values, key=value secrets, and long opaque
// runs of hex or base64. Over-redaction is fine; this text is for search
// quality analysis, not replay.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bdo[a-z]_v1_[a-z0-9]+`),
	regexp.MustCompile(`(?i)\b(bearer|basic)\s+[a-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)\b(token|secret|password|passwd|pwd|api[_-]?key|access[_-]?key)\s*[:=]\s*\S+`),
}

// opaqueRun matches long hex/base64-like runs. Only runs that mix letters and
// digits are redacted, so long hyphenated slugs and words survive.
var opaqueRun = regexp.MustCompile(`[A-Za-z0-9+/_=-]{32,}`)

func redactSecrets(s string) string {
	for _, re := range secretPatterns {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	return opaqueRun.ReplaceAllStringFunc(s, func(run string) string {
		if strings.ContainsAny(run, "0123456789") && strings.IndexFunc(run, unicode.IsLetter) >= 0 {
			return "[REDACTED]"
		}
		return run
	})
}

// cleanLogValue drops control characters and truncates to max runes.
func cleanLogValue(s string, max int) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	n := 0
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		if n == max {
			b.WriteString("...")
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}
