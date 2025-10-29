// Package edgelogging provides a slog.Handler that can optionally send logs to a WebSocket endpoint.
// It is a drop-in replacement for slog.JSONHandler that maintains stderr logging by default,
// but can be configured to send logs to a WebSocket server for centralized log aggregation.
package edgelogging

import (
	"context"
	"fmt"
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
	closeOnce *sync.Once
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
		closeOnce:        &sync.Once{},
	}

	return h
}

// Enabled reports whether the handler handles records at the given level.
// It delegates to the fallback handler's Enabled method.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.fallbackHandler.Enabled(ctx, level)
}

// Handle processes a log record.
// If WebSocket logging is not enabled, it delegates to the fallback handler (stderr).
// If WebSocket logging is enabled, it sends the log to WebSocket asynchronously.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if h.closed {
		return nil
	}

	// If WebSocket is not enabled, use fallback handler (stderr)
	if !h.wsEnabled {
		return h.fallbackHandler.Handle(ctx, r)
	}

	// TODO: implement websocket logging logic here
	return nil
}

// WithAttrs returns a new Handler with the given attributes added.
// It creates a new handler that shares the WebSocket connection but has updated attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	newHandler := &Handler{
		fallbackHandler:  h.fallbackHandler.WithAttrs(attrs),
		wsEnabled:        h.wsEnabled,
		wsURL:            h.wsURL,
		wsToken:          h.wsToken,
		wsConn:           h.wsConn,
		wsBuffer:         h.wsBuffer,
		wsMu:             h.wsMu, // Share the mutex
		wsReconnectDelay: h.wsReconnectDelay,
		wsMaxReconnects:  h.wsMaxReconnects,
		attrs:            newAttrs,
		groups:           h.groups,
		hostname:         h.hostname,
		processID:        h.processID,
		closeOnce:        h.closeOnce, // Share the closeOnce
		closed:           h.closed,
	}

	return newHandler
}

// WithGroup returns a new Handler with the given group name added.
// Subsequent keys will be qualified by the group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(newGroups)-1] = name

	newHandler := &Handler{
		fallbackHandler:  h.fallbackHandler.WithGroup(name),
		wsEnabled:        h.wsEnabled,
		wsURL:            h.wsURL,
		wsToken:          h.wsToken,
		wsConn:           h.wsConn,
		wsBuffer:         h.wsBuffer,
		wsMu:             h.wsMu, // Share the mutex
		wsReconnectDelay: h.wsReconnectDelay,
		wsMaxReconnects:  h.wsMaxReconnects,
		attrs:            h.attrs,
		groups:           newGroups,
		hostname:         h.hostname,
		processID:        h.processID,
		closeOnce:        h.closeOnce, // Share the closeOnce
		closed:           h.closed,
	}

	return newHandler
}

// ConfigureWebSocket enables WebSocket logging with the given URL and token.
// This method should be called after creating the handler to enable remote logging.
// If url is empty, it returns an error.
func (h *Handler) ConfigureWebSocket(url, token string) error {
	if url == "" {
		return fmt.Errorf("WebSocket URL cannot be empty")
	}

	h.wsMu.Lock()
	defer h.wsMu.Unlock()

	h.wsURL = url
	h.wsToken = token
	h.wsEnabled = true
	h.wsBuffer = make(chan []byte, 1000) // Buffer for 1000 messages

	// TODO: Start connection manager and log writer goroutines in next commits
	return nil
}
