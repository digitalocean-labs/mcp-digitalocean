package openapi

import (
	"net/http"
	"strings"
)

// SearchToolResult is the structured payload for openapi-search.
type SearchToolResult struct {
	Hits []OperationSummary `json:"hits"`
}

// GetOperationToolResult is the structured payload for openapi-get-operation.
type GetOperationToolResult struct {
	Operation Operation `json:"operation"`
}

// ExecuteToolResult is the structured payload for openapi-execute and openapi-execute-delete.
type ExecuteToolResult struct {
	Status           int               `json:"status"`
	ResponseHeaders  map[string]string `json:"responseHeaders,omitempty"`
	Body             any               `json:"body,omitempty"`
	Truncated        bool              `json:"truncated"`
	SelectNotApplied bool              `json:"selectNotApplied,omitempty"`
}

func forwardedHeadersMap(h http.Header) map[string]string {
	if h == nil {
		return nil
	}
	out := make(map[string]string)
	for _, key := range forwardResponseHeaderKeys {
		if v := h.Get(key); v != "" {
			out[strings.ToLower(key)] = v
		}
	}
	return out
}
