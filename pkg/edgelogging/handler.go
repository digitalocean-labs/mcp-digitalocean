// Package edgelogging provides a slog.Handler that can optionally send logs to a WebSocket endpoint.
// It is a drop-in replacement for slog.JSONHandler that maintains stderr logging by default,
// but can be configured to send logs to a WebSocket server for centralized log aggregation.
package edgelogging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Handler implements slog.Handler interface with optional WebSocket logging support.
// By default, it logs to the provided io.Writer (typically stderr).
// When configured with a WebSocket URL, it sends logs to the WebSocket endpoint instead.
type Handler struct {
	// Fallback handler for local logging (stderr)
	fallbackHandler slog.Handler

	// WebSocket configuration
	wsEnabled        bool
	wsURL            string
	wsToken          string
	wsConn           *websocket.Conn
	wsBuffer         chan []byte
	wsMu             *sync.Mutex
	wsReconnectDelay time.Duration
	wsMaxReconnects  int

	// Handler state for WithAttrs/WithGroup
	attrs  []slog.Attr
	groups []string

	// Context fields (added to WebSocket logs)
	hostname  string
	processID int

	// Lifecycle management
	closeOnce sync.Once
	closed    bool
}

// NewHandler creates a new Handler that logs to the provided io.Writer.
// If EDGE_LOGGING_URL environment variable is set, it will be configured to send logs
// to the WebSocket endpoint instead of the writer.
func NewHandler(out io.Writer, opts *slog.HandlerOptions) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	h := &Handler{
		fallbackHandler:  slog.NewJSONHandler(out, opts),
		wsMu:             &sync.Mutex{},
		wsReconnectDelay: 5 * time.Second,
		wsMaxReconnects:  -1, // unlimited
		hostname:         hostname,
		processID:        os.Getpid(),
	}

	return h
}
