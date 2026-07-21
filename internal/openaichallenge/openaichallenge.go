// Package openaichallenge serves the OpenAI ChatGPT app domain verification
// token at /.well-known/openai-apps-challenge.
package openaichallenge

import (
	"net/http"
	"strings"
)

// WellKnownPath is the path OpenAI uses to verify domain ownership for ChatGPT
// app / MCP connector submissions.
const WellKnownPath = "/.well-known/openai-apps-challenge"

// Handler returns an http.HandlerFunc that serves token as plain text for GET
// and HEAD. When token is empty, the handler responds with 404.
func Handler(token string) http.HandlerFunc {
	token = strings.TrimSpace(token)
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write([]byte(token))
	}
}
