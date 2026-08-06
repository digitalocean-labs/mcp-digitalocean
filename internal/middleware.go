package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type AuthKey struct{}

// AuthFromRequest extracts the auth token from the request headers.
func AuthFromRequest(ctx context.Context, r *http.Request) context.Context {
	return WithAuthKey(ctx, r.Header.Get("Authorization"))
}

// WithAuthKey adds an auth key to the context.
func WithAuthKey(ctx context.Context, auth string) context.Context {
	return context.WithValue(ctx, AuthKey{}, auth)
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

// calculateResponseSize calculates the total size of response content in bytes.
func calculateResponseSize(result *mcp.CallToolResult) int {
	if result == nil {
		return 0
	}
	totalSize := 0
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			totalSize += len(textContent.Text)
		}
	}
	return totalSize
}

// ToolMiddleware wraps a tool handler to log duration and success/error status.
func (m *ToolLoggingMiddleware) ToolMiddleware(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		result, err := next(ctx, req)
		if err != nil {
			m.Logger.Error("tool call result",
				"tool", req.Params.Name,
				"duration_seconds", time.Since(start).Seconds(),
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
				"content", payload,
				"tool_call_outcome", ToolCallResultError,
			)
			return result, err
		}

		responseSize := calculateResponseSize(result)
		m.Logger.Info("tool call result",
			"tool", req.Params.Name,
			"duration_seconds", time.Since(start).Seconds(),
			"response_bytes", responseSize,
			"tool_call_outcome", ToolCallSuccess,
		)

		return result, err
	}
}
