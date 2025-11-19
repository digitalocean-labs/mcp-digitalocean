package wslogging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestNewHandler tests the handler constructor
func TestNewHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}

	if handler.stderrHandler == nil {
		t.Error("stderrHandler is nil")
	}

	if handler.wsEnabled {
		t.Error("wsEnabled should be false by default")
	}
}

// TestHandler_Enabled tests the Enabled method
func TestHandler_Enabled(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		testWith slog.Level
		want     bool
	}{
		{
			name:     "debug level allows debug",
			level:    slog.LevelDebug,
			testWith: slog.LevelDebug,
			want:     true,
		},
		{
			name:     "info level blocks debug",
			level:    slog.LevelInfo,
			testWith: slog.LevelDebug,
			want:     false,
		},
		{
			name:     "info level allows info",
			level:    slog.LevelInfo,
			testWith: slog.LevelInfo,
			want:     true,
		},
		{
			name:     "error level blocks info",
			level:    slog.LevelError,
			testWith: slog.LevelInfo,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := NewHandler(&buf, &slog.HandlerOptions{
				Level: tt.level,
			})

			got := handler.Enabled(context.Background(), tt.testWith)
			if got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHandler_Handle_Stderr tests logging to stderr (default mode)
func TestHandler_Handle_Stderr(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("output missing log message: %s", output)
	}

	if !strings.Contains(output, "key") {
		t.Errorf("output missing attribute key: %s", output)
	}

	if !strings.Contains(output, "value") {
		t.Errorf("output missing attribute value: %s", output)
	}
}

// TestHandler_WithAttrs tests attribute addition
func TestHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// add attributes
	handler2 := handler.WithAttrs([]slog.Attr{
		slog.String("request_id", "123"),
		slog.Int("user_id", 456),
	})

	logger := slog.New(handler2)
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "request_id") {
		t.Errorf("output missing request_id: %s", output)
	}

	if !strings.Contains(output, "123") {
		t.Errorf("output missing request_id value: %s", output)
	}

	if !strings.Contains(output, "user_id") {
		t.Errorf("output missing user_id: %s", output)
	}
}

// TestHandler_WithAttrs_EnabledServices tests that enabled_services attribute persists across logs
func TestHandler_WithAttrs_EnabledServices(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// add enabled_services attribute (simulates what main.go does)
	handler2 := handler.WithAttrs([]slog.Attr{
		slog.String("enabled_services", "apps,networking,databases"),
	})

	logger := slog.New(handler2)

	// send multiple log messages
	logger.Info("first message")
	logger.Info("second message")
	logger.Warn("warning message")

	output := buf.String()

	// verify enabled_services appears in all log entries
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		if !strings.Contains(line, "enabled_services") {
			t.Errorf("line %d missing enabled_services: %s", i, line)
		}
		if !strings.Contains(line, "apps,networking,databases") {
			t.Errorf("line %d missing enabled_services value: %s", i, line)
		}
	}
}

// TestHandler_WithGroup tests group nesting
func TestHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// add a group
	handler2 := handler.WithGroup("request")
	logger := slog.New(handler2)
	logger.Info("test message", "method", "GET", "path", "/api")

	output := buf.String()
	if !strings.Contains(output, "request") {
		t.Errorf("output missing group name: %s", output)
	}

	if !strings.Contains(output, "method") {
		t.Errorf("output missing attribute in group: %s", output)
	}
}

// TestHandler_Close tests the Close method
func TestHandler_Close(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// close should not error even without WebSocket
	ctx := context.Background()
	err := handler.Close(ctx)
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// multiple closes should be safe
	err = handler.Close(ctx)
	if err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}

	// handler should be marked as closed
	if !handler.closed {
		t.Error("handler not marked as closed")
	}

	// attempting to log after close should return an error
	logger := slog.New(handler)
	ctx = context.Background()
	logger.Info("this should fail")

	// manually call Handle to check the error
	err = handler.Handle(ctx, slog.Record{})
	if err == nil {
		t.Error("Handle() should return error after Close(), but got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "handler has been closed") {
		t.Errorf("Expected error about handler being closed, got: %v", err)
	}
}

// mockWebSocketServer creates a test WebSocket server
func mockWebSocketServer(t *testing.T, token string) (*httptest.Server, chan []byte) {
	t.Helper()

	messages := make(chan []byte, 100)
	upgrader := websocket.Upgrader{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check authorization if token provided
		if token != "" {
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + token
			if auth != expected {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// read messages and send to channel
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			messages <- message
		}
	})

	server := httptest.NewServer(handler)
	return server, messages
}

// TestHandler_WebSocket_SendsLogs tests that logs are successfully sent over WebSocket
func TestHandler_WebSocket_SendsLogs(t *testing.T) {
	server, messages := mockWebSocketServer(t, "test-token")
	defer server.Close()

	// convert http:// to ws://
	wsURL := httpToWebSocketURL(server.URL)

	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// add enabled_services attribute (simulates main.go)
	handlerWithServices := handler.WithAttrs([]slog.Attr{
		slog.String("enabled_services", "apps,networking"),
	}).(*Handler)

	// configure WebSocket
	err := handlerWithServices.ConfigureWebSocket(wsURL, "test-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}

	// start WebSocket with background context
	ctx := context.Background()
	handlerWithServices.Start(ctx)
	defer handlerWithServices.Close(context.Background())

	// wait for connection
	time.Sleep(100 * time.Millisecond)

	// send a log message
	logger := slog.New(handlerWithServices)
	logger.Info("websocket test", "key", "value")

	// wait for message to arrive
	select {
	case msg := <-messages:
		var logEntry map[string]any
		if err := json.Unmarshal(msg, &logEntry); err != nil {
			t.Fatalf("failed to unmarshal log entry: %v", err)
		}

		if logEntry["message"] != "websocket test" {
			t.Errorf("message = %v, want 'websocket test'", logEntry["message"])
		}

		if logEntry["level"] != "INFO" {
			t.Errorf("level = %v, want 'INFO'", logEntry["level"])
		}

		if logEntry["key"] != "value" {
			t.Errorf("key = %v, want 'value'", logEntry["key"])
		}

		// check for standard fields
		if _, ok := logEntry["timestamp"]; !ok {
			t.Error("missing timestamp field")
		}

		// check for enabled_services field
		if enabledServices, ok := logEntry["enabled_services"]; !ok {
			t.Error("missing enabled_services field")
		} else if enabledServices != "apps,networking" {
			t.Errorf("enabled_services = %v, want 'apps,networking'", enabledServices)
		}

	case <-time.After(6 * time.Second):
		t.Fatal("timeout waiting for WebSocket message")
	}
}

// TestHandler_DualLogging tests that logs appear in BOTH stderr and WebSocket when WebSocket is enabled
func TestHandler_DualLogging(t *testing.T) {
	server, messages := mockWebSocketServer(t, "test-token")
	defer server.Close()

	wsURL := httpToWebSocketURL(server.URL)

	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// configure WebSocket
	err := handler.ConfigureWebSocket(wsURL, "test-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}

	// start WebSocket with background context
	ctx := context.Background()
	handler.Start(ctx)
	defer handler.Close(context.Background())

	// wait for connection
	time.Sleep(100 * time.Millisecond)

	// send a log message
	logger := slog.New(handler)
	logger.Info("dual logging test", "destination", "both")

	// verify the log appears in stderr (buf)
	stderrOutput := buf.String()
	if !strings.Contains(stderrOutput, "dual logging test") {
		t.Errorf("stderr missing log message: %s", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "destination") {
		t.Errorf("stderr missing attribute key: %s", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "both") {
		t.Errorf("stderr missing attribute value: %s", stderrOutput)
	}

	// verify the log also appears in WebSocket
	select {
	case msg := <-messages:
		var logEntry map[string]any
		if err := json.Unmarshal(msg, &logEntry); err != nil {
			t.Fatalf("failed to unmarshal WebSocket log entry: %v", err)
		}

		if logEntry["message"] != "dual logging test" {
			t.Errorf("WebSocket message = %v, want 'dual logging test'", logEntry["message"])
		}

		if logEntry["level"] != "INFO" {
			t.Errorf("WebSocket level = %v, want 'INFO'", logEntry["level"])
		}

		if logEntry["destination"] != "both" {
			t.Errorf("WebSocket destination = %v, want 'both'", logEntry["destination"])
		}

	case <-time.After(6 * time.Second):
		t.Fatal("timeout waiting for WebSocket message")
	}
}

// TestHandler_WebSocket_WithAuth tests WebSocket with authentication
func TestHandler_WebSocket_WithAuth(t *testing.T) {
	server, messages := mockWebSocketServer(t, "secret-token")
	defer server.Close()

	wsURL := httpToWebSocketURL(server.URL)

	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// configure with correct token
	err := handler.ConfigureWebSocket(wsURL, "secret-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}

	// start WebSocket with background context
	ctx := context.Background()
	handler.Start(ctx)
	defer handler.Close(context.Background())

	// wait for connection
	time.Sleep(100 * time.Millisecond)

	// send a log message
	logger := slog.New(handler)
	logger.Info("auth test")

	// should receive message
	select {
	case <-messages:
		// success
	case <-time.After(6 * time.Second):
		t.Fatal("timeout waiting for authenticated WebSocket message")
	}
}

// TestHandler_ConfigureWebSocket_EmptyURL tests error on empty URL
func TestHandler_ConfigureWebSocket_EmptyURL(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	err := handler.ConfigureWebSocket("", "token")
	if err == nil {
		t.Error("ConfigureWebSocket() with empty URL should return error")
	}
}

// httpToWebSocketURL converts an HTTP test server URL to a WebSocket URL
func httpToWebSocketURL(httpURL string) string {
	// httptest.NewServer returns http:// URLs, but we need ws:// for WebSocket
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

// waitForCondition polls a condition function until it returns true or timeout
func waitForCondition(t *testing.T, timeout time.Duration, checkInterval time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(checkInterval)
	}
	t.Fatal(message)
}

// TestHandler_WebSocket_BufferFull tests that stderr (primary destination) receives all logs
// even when WebSocket buffer overflows and drops messages
func TestHandler_WebSocket_BufferFull(t *testing.T) {
	// Note: This test intentionally generates ~1000 "buffer is full" diagnostic messages
	// to stderr. These are expected and demonstrate proper backpressure handling.

	// create handler with stderr going to our buffer
	var stderrBuf bytes.Buffer
	handler := NewHandler(&stderrBuf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// configure with invalid URL so connection fails (ensures buffer fills up)
	_ = handler.ConfigureWebSocket("ws://localhost:1/invalid", "token")

	// start WebSocket with a timeout context to prevent infinite retries
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	handler.Start(ctx)
	defer handler.Close(context.Background())

	// send many messages rapidly (more than buffer size of 1000)
	const totalMessages = 2000
	logger := slog.New(handler)
	for i := 0; i < totalMessages; i++ {
		logger.Info("flood test", "i", i)
	}

	// give time for async logging to complete
	time.Sleep(100 * time.Millisecond)

	// verify ALL messages made it to stderr (primary destination)
	// even though WebSocket buffer filled up and dropped messages
	stderrLines := bytes.Split(stderrBuf.Bytes(), []byte("\n"))
	actualLogCount := 0
	for _, line := range stderrLines {
		if bytes.Contains(line, []byte("flood test")) {
			actualLogCount++
		}
	}
	require.Equal(t, totalMessages, actualLogCount,
		"All %d messages should be in stderr even though WebSocket dropped some", totalMessages)

	t.Logf("Verified: all %d logs made it to stderr (primary destination)", totalMessages)
	t.Logf("WebSocket dropped ~%d messages due to buffer overflow", totalMessages-bufferSize)
}

// TestBuildLogEntry tests the log entry construction
func TestBuildLogEntry(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// create a record
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	record.AddAttrs(
		slog.String("key1", "value1"),
		slog.Int("key2", 123),
	)

	entry := handler.buildLogEntry(record)

	if entry["message"] != "test message" {
		t.Errorf("message = %v, want 'test message'", entry["message"])
	}

	if entry["level"] != "INFO" {
		t.Errorf("level = %v, want 'INFO'", entry["level"])
	}

	if entry["key1"] != "value1" {
		t.Errorf("key1 = %v, want 'value1'", entry["key1"])
	}

	if v, ok := entry["key2"].(int64); !ok || v != 123 {
		t.Errorf("key2 = %v (type %T), want 123", entry["key2"], entry["key2"])
	}
}

// TestConfigureWebSocket_URLValidation tests URL validation in ConfigureWebSocket
func TestConfigureWebSocket_URLValidation(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		token     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid ws URL",
			url:       "ws://localhost:8080",
			token:     "test-token",
			wantError: false,
		},
		{
			name:      "valid wss URL",
			url:       "wss://logs.example.com/stream",
			token:     "test-token",
			wantError: false,
		},
		{
			name:      "valid wss URL with path",
			url:       "wss://logs.example.com:9000/path/to/logs",
			token:     "test-token",
			wantError: false,
		},
		{
			name:      "empty URL",
			url:       "",
			token:     "test-token",
			wantError: true,
			errorMsg:  "WebSocket URL cannot be empty",
		},
		{
			name:      "invalid scheme - http",
			url:       "http://example.com",
			token:     "test-token",
			wantError: true,
			errorMsg:  "invalid WebSocket URL scheme",
		},
		{
			name:      "invalid scheme - https",
			url:       "https://example.com",
			token:     "test-token",
			wantError: true,
			errorMsg:  "invalid WebSocket URL scheme",
		},
		{
			name:      "invalid URL format",
			url:       "not-a-url",
			token:     "test-token",
			wantError: true,
			errorMsg:  "invalid WebSocket URL scheme",
		},
		{
			name:      "valid URL with empty token",
			url:       "ws://localhost:8080",
			token:     "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := NewHandler(&buf, nil)

			err := handler.ConfigureWebSocket(tt.url, tt.token)

			if tt.wantError {
				if err == nil {
					t.Errorf("ConfigureWebSocket() error = nil, want error containing '%s'", tt.errorMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ConfigureWebSocket() error = %v, want error containing '%s'", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ConfigureWebSocket() unexpected error = %v", err)
					return
				}
				// clean up
				handler.Close(context.Background())
			}
		})
	}
}

// TestHandler_ThreadSafeBatch tests that the batch is thread-safe under concurrent access
func TestHandler_ThreadSafeBatch(t *testing.T) {
	server, messages := mockWebSocketServer(t, "test-token")
	defer server.Close()

	wsURL := httpToWebSocketURL(server.URL)

	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// configure WebSocket
	err := handler.ConfigureWebSocket(wsURL, "test-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}

	ctx := context.Background()
	handler.Start(ctx)
	defer handler.Close(context.Background())

	// wait for connection
	time.Sleep(100 * time.Millisecond)

	// spawn multiple goroutines writing logs concurrently
	const numGoroutines = 10
	const logsPerGoroutine = 20
	const totalExpected = numGoroutines * logsPerGoroutine

	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			logger := slog.New(handler)
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info("concurrent test", "goroutine", id, "msg", j)
			}
			done <- true
		}(i)
	}

	// wait for all goroutines to finish
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// collect all messages (with timeout)
	receivedCount := 0
	timeout := time.After(10 * time.Second)
	for receivedCount < totalExpected {
		select {
		case <-messages:
			receivedCount++
		case <-timeout:
			t.Fatalf("timeout: received %d/%d messages", receivedCount, totalExpected)
		}
	}

	if receivedCount != totalExpected {
		t.Errorf("received %d messages, want %d", receivedCount, totalExpected)
	}

	t.Logf("Successfully received all %d messages from %d concurrent goroutines", receivedCount, numGoroutines)
}

// TestHandler_WebSocket_PingPong tests that the handler sends pings and handles pongs
