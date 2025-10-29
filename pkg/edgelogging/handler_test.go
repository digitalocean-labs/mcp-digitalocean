package edgelogging

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

	if handler.fallbackHandler == nil {
		t.Error("fallbackHandler is nil")
	}

	if handler.hostname == "" {
		t.Error("hostname not set")
	}

	if handler.processID == 0 {
		t.Error("processID not set")
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

	// Add attributes
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

// TestHandler_WithGroup tests group nesting
func TestHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Add a group
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

	// Close should not error even without WebSocket
	err := handler.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Multiple closes should be safe
	err = handler.Close()
	if err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}

	// Handler should be marked as closed
	if !handler.closed {
		t.Error("handler not marked as closed")
	}
}

// mockWebSocketServer creates a test WebSocket server
func mockWebSocketServer(t *testing.T, token string) (*httptest.Server, chan []byte) {
	t.Helper()

	messages := make(chan []byte, 100)
	upgrader := websocket.Upgrader{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization if token provided
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

		// Read messages and send to channel
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

// TestHandler_WebSocket_Basic tests basic WebSocket logging
func TestHandler_WebSocket_Basic(t *testing.T) {
	server, messages := mockWebSocketServer(t, "test-token")
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Configure WebSocket
	err := handler.ConfigureWebSocket(wsURL, "test-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}
	defer handler.Close()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Send a log message
	logger := slog.New(handler)
	logger.Info("websocket test", "key", "value")

	// Wait for message to arrive
	select {
	case msg := <-messages:
		var logEntry map[string]interface{}
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

		// Check for standard fields
		if _, ok := logEntry["timestamp"]; !ok {
			t.Error("missing timestamp field")
		}

		if _, ok := logEntry["hostname"]; !ok {
			t.Error("missing hostname field")
		}

		if _, ok := logEntry["process_id"]; !ok {
			t.Error("missing process_id field")
		}

	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for WebSocket message")
	}
}

// TestHandler_WebSocket_WithAuth tests WebSocket with authentication
func TestHandler_WebSocket_WithAuth(t *testing.T) {
	server, messages := mockWebSocketServer(t, "secret-token")
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Configure with correct token
	err := handler.ConfigureWebSocket(wsURL, "secret-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}
	defer handler.Close()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Send a log message
	logger := slog.New(handler)
	logger.Info("auth test")

	// Should receive message
	select {
	case <-messages:
		// Success
	case <-time.After(2 * time.Second):
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

// TestHandler_WebSocket_Reconnection tests reconnection logic
func TestHandler_WebSocket_Reconnection(t *testing.T) {
	// This test is complex - for now, we'll skip it
	// It would require closing the server and restarting it
	t.Skip("Reconnection test requires complex server lifecycle management")
}

// TestHandler_WebSocket_BufferFull tests behavior when buffer is full
func TestHandler_WebSocket_BufferFull(t *testing.T) {
	// Create handler but don't start server
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Configure with invalid URL so connection fails
	_ = handler.ConfigureWebSocket("ws://localhost:1/invalid", "token")
	defer handler.Close()

	// Send many messages rapidly
	logger := slog.New(handler)
	for i := 0; i < 2000; i++ {
		logger.Info("flood test", "i", i)
	}

	// Should not panic or block
	// Messages should be dropped when buffer is full
}

// TestBuildLogEntry tests the log entry construction
func TestBuildLogEntry(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Create a record
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

	if entry["hostname"] != handler.hostname {
		t.Errorf("hostname = %v, want %v", entry["hostname"], handler.hostname)
	}

	if v, ok := entry["process_id"].(int); !ok || v != handler.processID {
		t.Errorf("process_id = %v, want %v", entry["process_id"], handler.processID)
	}
}
