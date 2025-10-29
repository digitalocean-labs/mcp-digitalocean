// Package edgelogging provides a slog.Handler that can optionally send logs to a WebSocket endpoint.
// It is a drop-in replacement for slog.JSONHandler that maintains stderr logging by default,
// but can be configured to send logs to a WebSocket server for centralized log aggregation.
package edgelogging

import (
	"context"
	"encoding/json"
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

	// Build log entry as JSON for WebSocket
	entry := h.buildLogEntry(r)

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Try to send to buffer (non-blocking)
	select {
	case h.wsBuffer <- data:
		return nil
	default:
		// Buffer is full, drop the message
		return fmt.Errorf("log buffer full, message dropped")
	}
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

// Close gracefully shuts down the handler and closes the WebSocket connection if open.
// It should be called when the application is shutting down.
// This method is safe to call multiple times.
func (h *Handler) Close() error {
	var err error
	h.closeOnce.Do(func() {
		h.closed = true

		// Close the buffer channel if it exists
		if h.wsBuffer != nil {
			close(h.wsBuffer)
		}

		// Close WebSocket connection if open
		h.wsMu.Lock()
		defer h.wsMu.Unlock()

		if h.wsConn != nil {
			// Send close message
			_ = h.wsConn.WriteMessage(websocket.CloseMessage, []byte{})
			err = h.wsConn.Close()
			h.wsConn = nil
		}
	})
	return err
}

// buildLogEntry constructs a map representing the log entry for WebSocket transmission.
func (h *Handler) buildLogEntry(r slog.Record) map[string]interface{} {
	entry := make(map[string]interface{})

	// Add timestamp
	if !r.Time.IsZero() {
		entry["timestamp"] = r.Time.Format(time.RFC3339Nano)
	}

	// Add level
	entry["level"] = r.Level.String()

	// Add message
	entry["message"] = r.Message

	// Add standard context fields
	entry["hostname"] = h.hostname
	entry["process_id"] = h.processID

	// Add attributes from WithAttrs
	for _, attr := range h.attrs {
		h.addAttrToMap(entry, attr, h.groups)
	}

	// Add attributes from the record
	r.Attrs(func(attr slog.Attr) bool {
		h.addAttrToMap(entry, attr, h.groups)
		return true
	})

	return entry
}

// addAttrToMap adds an attribute to the map, handling groups and nested attributes.
func (h *Handler) addAttrToMap(entry map[string]interface{}, attr slog.Attr, groups []string) {
	attr.Value = attr.Value.Resolve()

	// Ignore empty attributes
	if attr.Equal(slog.Attr{}) {
		return
	}

	key := attr.Key

	// Navigate to the correct nested map for groups
	current := entry
	for _, group := range groups {
		if _, exists := current[group]; !exists {
			current[group] = make(map[string]interface{})
		}
		if nested, ok := current[group].(map[string]interface{}); ok {
			current = nested
		}
	}

	// Handle different value kinds
	switch attr.Value.Kind() {
	case slog.KindGroup:
		groupAttrs := attr.Value.Group()
		if len(groupAttrs) == 0 {
			return
		}
		if attr.Key != "" {
			current[key] = make(map[string]interface{})
			if nested, ok := current[key].(map[string]interface{}); ok {
				for _, ga := range groupAttrs {
					h.addAttrToMap(nested, ga, nil)
				}
			}
		} else {
			// Inline group
			for _, ga := range groupAttrs {
				h.addAttrToMap(current, ga, nil)
			}
		}
	default:
		current[key] = attr.Value.Any()
	}
}
