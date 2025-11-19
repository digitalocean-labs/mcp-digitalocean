// wslogging provides a slog.Handler that can optionally send logs to a WebSocket endpoint.
// It is a drop-in replacement for slog.JSONHandler that maintains stderr logging by default,
// but can be configured to send logs to a WebSocket server for centralized log aggregation.
package wslogging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// bufferSize is the size of the log buffer channel
	bufferSize = 1000
	// handshakeTimeout is the timeout for WebSocket handshake
	handshakeTimeout = 10 * time.Second
	// batchInterval is the interval for flushing batched log messages to WebSocket
	batchInterval = 5 * time.Second
	// maxBatchSize is the maximum number of messages to batch before forcing a flush
	maxBatchSize = 50
)

// Handler implements slog.Handler interface with optional WebSocket logging support.
// By default, it logs to the provided io.Writer (typically stderr).
// When configured with a WebSocket URL, it sends logs to both stderr and the WebSocket endpoint.
type Handler struct {
	// standard handler for stderr logging (primary/baseline destination)
	stderrHandler slog.Handler

	// WebSocket configuration
	wsEnabled bool
	wsURL     string
	wsToken   string
	wsBuffer  chan []byte
	wsMu      *sync.Mutex

	// batch stores accumulated log messages for batched WebSocket transmission
	// protected by flushMu
	batch [][]byte

	// flushMu protects batch modifications
	flushMu *sync.Mutex

	// handler state for WithAttrs/WithGroup
	attrs  []slog.Attr
	groups []string

	// lifecycle management
	closeOnce *sync.Once
	closed    bool
}

// NewHandler creates a new Handler that logs to the provided io.Writer.
func NewHandler(out io.Writer, opts *slog.HandlerOptions) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	h := &Handler{
		stderrHandler: slog.NewJSONHandler(out, opts),
		wsMu:          &sync.Mutex{},
		batch:         make([][]byte, 0, maxBatchSize),
		flushMu:       &sync.Mutex{},
		closeOnce:     &sync.Once{},
	}

	return h
}

// Enabled reports whether the handler handles records at the given level.
// It delegates to the stderr handler's Enabled method.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.stderrHandler.Enabled(ctx, level)
}

// Handle processes a log record.
// Logs are always written to stderr (primary destination).
// If WebSocket logging is enabled, logs are also sent to the WebSocket endpoint asynchronously (complementary destination).
// Both destinations are independent - failure in one does not affect the other.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Check if handler is closed and capture wsEnabled state (thread-safe)
	h.wsMu.Lock()
	closed := h.closed
	wsEnabled := h.wsEnabled
	h.wsMu.Unlock()

	if closed {
		return fmt.Errorf("wslogging: handler has been closed and is no longer accepting log messages")
	}

	// always write to stderr (primary logging destination)
	stderrErr := h.stderrHandler.Handle(ctx, r)

	// if WebSocket is enabled, also send to WebSocket asynchronously
	if wsEnabled {
		go h.sendToWebSocket(r)
	}

	// return stderr error if it failed (primary logging destination)
	// we ignore WebSocket errors as it's a complementary logging destination
	return stderrErr
}

// sendToWebSocket sends a log record to the WebSocket asynchronously.
// This method is meant to be called as a goroutine.
func (h *Handler) sendToWebSocket(r slog.Record) {
	entry := h.buildLogEntry(r)
	data, err := json.Marshal(entry)
	if err != nil {
		logDiagnostic(os.Stderr, "failed to marshal log entry: %v\n", err)
		return
	}

	// atomically check if closed and send to buffer (hold lock briefly for non-blocking select)
	h.wsMu.Lock()
	defer h.wsMu.Unlock()

	if h.closed {
		logDiagnostic(os.Stderr, "dropping log message: handler is closed\n")
		return
	}

	select {
	case h.wsBuffer <- data:
		// successfully queued for WebSocket transmission
	default:
		// buffer is full, drop the WebSocket message
		logDiagnostic(os.Stderr, "dropping log message: buffer is full (%d messages)\n", bufferSize)
	}
}

// WithAttrs returns a new Handler with the given attributes added.
// It creates a new handler that shares the WebSocket connection but has updated attributes.
//
// Note: We must return a new handler instance to maintain attribute isolation between loggers.
// When logger.With("key", "value") is called, slog creates a derived logger with its own attributes.
// Each derived logger needs its own handler that knows about its specific attributes, but they all
// share the same WebSocket connection, buffer, and mutex for efficient resource usage.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)

	newHandler := &Handler{
		stderrHandler: h.stderrHandler.WithAttrs(attrs),
		// WebSocket configuration must be copied so derived handlers maintain WS logging capability
		wsEnabled: h.wsEnabled,
		wsURL:     h.wsURL,
		wsToken:   h.wsToken,
		// these are shared across all derived handlers for efficiency
		wsBuffer: h.wsBuffer, // shared buffer
		wsMu:     h.wsMu,     // shared mutex
		batch:    h.batch,    // shared batch (managed by single logWriter goroutine)
		flushMu:  h.flushMu,  // shared flush mutex
		// each derived handler has its own attributes and groups
		attrs:     newAttrs,
		groups:    h.groups,
		closeOnce: h.closeOnce, // shared to ensure Close() runs only once
		closed:    h.closed,
	}

	return newHandler
}

// WithGroup returns a new Handler with the given group name added.
// It creates a new handler that shares the WebSocket connection but has updated groups.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, 0, len(h.groups)+1)
	newGroups = append(newGroups, h.groups...)
	newGroups = append(newGroups, name)

	newHandler := &Handler{
		stderrHandler: h.stderrHandler.WithGroup(name),
		// WebSocket configuration must be copied so derived handlers maintain WS logging capability
		wsEnabled: h.wsEnabled,
		wsURL:     h.wsURL,
		wsToken:   h.wsToken,
		// these are shared across all derived handlers for efficiency
		wsBuffer: h.wsBuffer, // shared buffer
		wsMu:     h.wsMu,     // shared mutex
		batch:    h.batch,    // shared batch (managed by single logWriter goroutine)
		flushMu:  h.flushMu,  // shared flush mutex
		// each derived handler has its own attributes and groups
		attrs:     h.attrs,
		groups:    newGroups,
		closeOnce: h.closeOnce, // shared to ensure Close() runs only once
		closed:    h.closed,
	}

	return newHandler
}

// ConfigureWebSocket configures the handler to send logs to a WebSocket endpoint.
// The wsURL should be in the format "ws://host:port/path" or "wss://host:port/path".
// The token is optional and will be sent in the Authorization header if provided.
func (h *Handler) ConfigureWebSocket(wsURL, token string) error {
	if wsURL == "" {
		return fmt.Errorf("WebSocket URL cannot be empty")
	}

	// validate URL format
	parsedURL, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	if parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss" {
		return fmt.Errorf("invalid WebSocket URL scheme: %s (must be ws or wss)", parsedURL.Scheme)
	}

	// warn if no token provided (security risk)
	if token == "" {
		logDiagnostic(os.Stderr, "WARNING: no authentication token provided - this is a security risk\n")
	}

	h.wsMu.Lock()
	defer h.wsMu.Unlock()

	// store configuration in handler fields so WithAttrs() and WithGroup() can copy them
	// when creating derived handlers (required for maintaining WS logging across logger.With() calls)
	h.wsURL = wsURL
	h.wsToken = token
	h.wsEnabled = true
	h.wsBuffer = make(chan []byte, bufferSize)

	// log startup diagnostic to stdout
	logDiagnostic(os.Stdout, "configuring WebSocket logging to %s\n", wsURL)
	return nil
}

// Start initiates the log writer goroutine.
// This method should be called after creating the handler and calling ConfigureWebSocket to enable remote logging.
// The provided context controls the lifecycle of the background goroutine - when the context is cancelled,
// the goroutine will gracefully shut down.
func (h *Handler) Start(ctx context.Context) {
	// start log writer goroutine
	go h.logWriter(ctx)
}

// Close gracefully shuts down the handler and flushes any remaining buffered messages.
func (h *Handler) Close(ctx context.Context) error {
	var err error
	h.closeOnce.Do(func() {
		h.wsMu.Lock()
		h.closed = true
		h.wsMu.Unlock()

		// close the buffer channel to signal logWriter to exit
		if h.wsBuffer != nil {
			close(h.wsBuffer)
		}
	})
	return err
}

// buildLogEntry constructs a map representing the log entry for WebSocket transmission.
func (h *Handler) buildLogEntry(r slog.Record) map[string]any {
	entry := make(map[string]any)

	if !r.Time.IsZero() {
		entry["timestamp"] = r.Time.Format(time.RFC3339Nano)
	}

	entry["level"] = r.Level.String()
	entry["message"] = r.Message

	// build attributes map from handler's persistent attributes and record attributes
	if len(h.attrs) > 0 || r.NumAttrs() > 0 {
		current := entry

		// navigate to the correct nested position based on groups
		for _, group := range h.groups {
			groupMap := make(map[string]any)
			current[group] = groupMap
			current = groupMap
		}

		// add handler's persistent attributes first
		for _, attr := range h.attrs {
			h.addAttrToMap(current, attr, nil)
		}

		// add record attributes
		r.Attrs(func(attr slog.Attr) bool {
			h.addAttrToMap(current, attr, nil)
			return true
		})
	}

	return entry
}

// addAttrToMap adds an attribute to the map, handling groups recursively.
func (h *Handler) addAttrToMap(current map[string]any, attr slog.Attr, groups []string) {
	key := attr.Key
	switch attr.Value.Kind() {
	case slog.KindGroup:
		groupMap := make(map[string]any)
		current[key] = groupMap
		for _, ga := range attr.Value.Group() {
			h.addAttrToMap(groupMap, ga, nil)
		}
	default:
		current[key] = attr.Value.Any()
	}
}

// logWriter reads from the buffer and writes to WebSocket with time-based batching.
// Messages are accumulated for up to batchInterval (5s) or until maxBatchSize is reached,
// then a connection is established, messages are sent, and connection is closed.
// This should be called as a goroutine.
func (h *Handler) logWriter(ctx context.Context) {
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// context cancelled - flush remaining messages before shutdown
			h.flushBatch()
			return

		case data, ok := <-h.wsBuffer:
			if !ok {
				// channel closed - flush remaining messages before shutdown
				h.flushBatch()
				return
			}

			// add message to batch (protected by flushMu)
			h.flushMu.Lock()
			h.batch = append(h.batch, data)
			batchLen := len(h.batch)
			h.flushMu.Unlock()

			// flush if batch is full
			if batchLen >= maxBatchSize {
				h.flushBatch()
			}

		case <-ticker.C:
			// periodic flush
			h.flushBatch()
		}
	}
}

// flushBatch sends accumulated messages to the websocket.
// It establishes a connection, sends all messages, and closes the connection.
// The batch is cleared only if all messages are sent successfully.
// Uses copy semantics to avoid holding locks during network I/O, with atomic write-back.
func (h *Handler) flushBatch() {
	// make a local copy of h.batch to work with
	// this allows us to do network I/O without holding any locks
	h.flushMu.Lock()
	defer h.flushMu.Unlock()

	if len(h.batch) == 0 {
		return
	}

	localBatch := make([][]byte, len(h.batch))
	copy(localBatch, h.batch)

	// read WebSocket configuration (protected by wsMu)
	h.wsMu.Lock()
	closed := h.closed
	wsURL := h.wsURL
	wsToken := h.wsToken
	h.wsMu.Unlock()

	if closed {
		return
	}

	// establish connection (no locks held during network I/O)
	conn, err := h.connect(wsURL, wsToken)
	if err != nil {
		logDiagnostic(os.Stderr, "failed to connect to WebSocket: %v\n", err)
		// don't clear batch - we'll retry on next flush
		return
	}
	defer conn.Close()

	// send all messages in the local batch copy (no locks held during network I/O)
	sentCount := 0
	for _, data := range localBatch {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			logDiagnostic(os.Stderr, "failed to write message to WebSocket: %v\n", err)
			break
		}
		sentCount++
	}

	// atomic write-back - update h.batch based on what was sent
	batchLen := len(localBatch)
	remainingCount := batchLen - sentCount

	if remainingCount > 0 {
		// some messages failed to send - keep only the unsent messages in the batch.
		// copy remaining messages to a new slice to release memory from the old underlying array.
		newBatch := make([][]byte, remainingCount)
		copy(newBatch, h.batch[sentCount:])
		h.batch = newBatch
	} else {
		// all messages sent successfully (remainingCount == 0) - clear the batch completely
		h.batch = h.batch[:0]
	}
}

// connect establishes a WebSocket connection to the configured endpoint.
func (h *Handler) connect(wsURL, token string) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
	}

	// set up headers for authentication
	headers := make(map[string][]string)
	if token != "" {
		headers["Authorization"] = []string{fmt.Sprintf("Bearer %s", token)}
	}

	// connect to WebSocket
	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// logDiagnostic writes a diagnostic message to the specified writer.
// This is used for logging infrastructure issues (not application logs).
// Messages are written as JSON to maintain consistency with application logs.
func logDiagnostic(w io.Writer, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	// trim trailing newline if present (we'll add it back after JSON)
	message = strings.TrimSuffix(message, "\n")

	entry := map[string]any{
		"level":  "ERROR",
		"msg":    message,
		"source": "wslogging",
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to plain text if JSON marshaling fails
		fmt.Fprintf(w, "[wslogging] %s\n", message)
		return
	}

	fmt.Fprintf(w, "%s\n", data)
}
