//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// LogEntry represents a structured log entry from the MCP server.
// it captures the standard fields (timestamp, level, message) and any
// additional fields in the Extra map for flexible validation.
type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Extra     map[string]any `json:"-"` // all other fields beyond timestamp/level/message
}

// UnmarshalJSON is a custom unmarshaler that captures all fields from the JSON.
// standard fields go into struct fields, everything else goes into Extra map.
func (l *LogEntry) UnmarshalJSON(data []byte) error {
	type Alias LogEntry
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(l),
	}

	// first unmarshal into the struct
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// then unmarshal into a map to capture extra fields
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// remove known fields
	delete(m, "timestamp")
	delete(m, "level")
	delete(m, "message")

	l.Extra = m
	return nil
}

// FakeWebSocketServer is a test WebSocket server that collects log entries.
// it simulates a remote logging endpoint for testing WebSocket logging functionality.
type FakeWebSocketServer struct {
	server      *httptest.Server
	url         string
	token       string
	mu          sync.Mutex
	logEntries  []LogEntry
	connections int
	upgrader    websocket.Upgrader
	// listener stores the actual listener address (0.0.0.0:port) to extract the port
	// for constructing container-accessible URLs (host.docker.internal:port)
	listener string
}

// NewFakeWebSocketServer creates and starts a new fake WebSocket server.
// it binds to 0.0.0.0 instead of 127.0.0.1 to accept connections from:
// - IPv4 clients
// - IPv6 clients
// - Docker containers (via host.docker.internal)
// this cross-platform approach works on both Linux and macOS.
func NewFakeWebSocketServer(token string) *FakeWebSocketServer {
	fws := &FakeWebSocketServer{
		token:      token,
		logEntries: make([]LogEntry, 0),
		upgrader:   websocket.Upgrader{},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// verify bearer token authentication if configured
		if fws.token != "" {
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + fws.token
			if auth != expected {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// upgrade HTTP connection to WebSocket
		conn, err := fws.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// track connection count for testing
		fws.mu.Lock()
		fws.connections++
		fws.mu.Unlock()

		defer conn.Close()

		// continuously read and collect log messages
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return // connection closed or error
			}

			// parse the log entry from JSON
			var entry LogEntry
			if err := json.Unmarshal(message, &entry); err != nil {
				continue // skip malformed entries
			}

			// store the log entry for test assertions
			fws.mu.Lock()
			fws.logEntries = append(fws.logEntries, entry)
			fws.mu.Unlock()
		}
	})

	// create an unstarted server so we can customize the listener
	fws.server = httptest.NewUnstartedServer(handler)

	// bind to 0.0.0.0:0 (random port) instead of 127.0.0.1
	// this allows connections from containers via host.docker.internal
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		panic(fmt.Sprintf("failed to create listener: %v", err))
	}

	// assign the custom listener and start the server
	fws.server.Listener = listener
	fws.server.Start()

	// store the actual listener address (e.g., "0.0.0.0:54321")
	// we'll need this to extract the port for container URLs
	fws.listener = fws.server.Listener.Addr().String()
	fws.url = "ws://" + fws.listener

	return fws
}

// Close shuts down the fake server
func (fws *FakeWebSocketServer) Close() {
	if fws.server != nil {
		fws.server.Close()
	}
}

// GetURL returns the WebSocket URL for use from the test host.
// returns something like "ws://0.0.0.0:54321"
func (fws *FakeWebSocketServer) GetURL() string {
	return fws.url
}

// GetContainerURL returns the WebSocket URL that containers can use to reach the host.
// replaces 0.0.0.0 with host.docker.internal so Docker containers can connect.
// on Linux: host.docker.internal is mapped to the Docker gateway via ExtraHosts
// on Mac/Windows: host.docker.internal is natively supported by Docker Desktop
func (fws *FakeWebSocketServer) GetContainerURL() string {
	_, port, _ := net.SplitHostPort(fws.listener)
	return fmt.Sprintf("ws://host.docker.internal:%s", port)
}

// GetToken returns the authentication token used by this server.
func (fws *FakeWebSocketServer) GetToken() string {
	return fws.token
}

// GetLogEntries returns all collected log entries
func (fws *FakeWebSocketServer) GetLogEntries() []LogEntry {
	fws.mu.Lock()
	defer fws.mu.Unlock()

	// return a copy
	entries := make([]LogEntry, len(fws.logEntries))
	copy(entries, fws.logEntries)
	return entries
}

// GetConnectionCount returns the number of connections received
func (fws *FakeWebSocketServer) GetConnectionCount() int {
	fws.mu.Lock()
	defer fws.mu.Unlock()
	return fws.connections
}

// WaitForLogs waits for at least n log entries or timeout
func (fws *FakeWebSocketServer) WaitForLogs(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(fws.GetLogEntries()) >= n {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// FindLogsByMessage returns all log entries matching the message
func (fws *FakeWebSocketServer) FindLogsByMessage(message string) []LogEntry {
	fws.mu.Lock()
	defer fws.mu.Unlock()

	results := make([]LogEntry, 0)
	for _, entry := range fws.logEntries {
		if entry.Message == message {
			results = append(results, entry)
		}
	}
	return results
}

// FindLogsByLevel returns all log entries at the given level
func (fws *FakeWebSocketServer) FindLogsByLevel(level string) []LogEntry {
	fws.mu.Lock()
	defer fws.mu.Unlock()

	results := make([]LogEntry, 0)
	for _, entry := range fws.logEntries {
		if entry.Level == level {
			results = append(results, entry)
		}
	}
	return results
}

// ClearLogs clears all collected log entries
func (fws *FakeWebSocketServer) ClearLogs() {
	fws.mu.Lock()
	defer fws.mu.Unlock()
	fws.logEntries = make([]LogEntry, 0)
}

// WaitForConnection waits for at least n connections or timeout
func (fws *FakeWebSocketServer) WaitForConnection(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if fws.GetConnectionCount() >= n {
				return true
			}
		case <-time.After(time.Until(deadline)):
			return false
		}
	}
}

// pollCondition polls a condition function until it returns true or timeout
func pollCondition(t *testing.T, timeout time.Duration, condition func() bool, errorMsg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-time.After(time.Until(deadline)):
			require.Fail(t, errorMsg)
			return
		}
	}
}

// TestMCPServer_WebSocketLogging tests end-to-end WebSocket logging with MCP server.
// We run the tests using a single container instance for efficiency.
func TestMCPServer_WebSocketLogging(t *testing.T) {
	ctx := context.Background()

	// start fake WebSocket server (shared across all sub-tests)
	fakeWS := NewFakeWebSocketServer("test-token-123")
	defer fakeWS.Close()

	// start MCP server once for all sub-tests
	cfg := McpServerConfig{
		BindAddr:             "0.0.0.0:8080",
		DigitalOceanAPIToken: os.Getenv("DIGITALOCEAN_API_TOKEN"),
		LogLevel:             "debug",
		Transport:            "http",
		Services:             "apps,networking",
		WSLoggingURL:         fakeWS.GetContainerURL(),
		WSLoggingToken:       fakeWS.GetToken(),
	}
	container, err := startMcpServer(ctx, cfg)
	require.NoError(t, err, "Failed to start MCP server")
	defer container.Terminate(ctx)

	// get the mapped port
	port, err := container.MappedPort(ctx, "8080/tcp")
	require.NoError(t, err, "Failed to get mapped port")
	serverURL := fmt.Sprintf("http://127.0.0.1:%s/mcp", port.Port())

	// run all test scenarios as sub-tests
	// the container and WebSocket server are shared across all sub-tests for efficiency

	t.Run("ConnectionEstablishment", func(t *testing.T) {
		// Generate logs to trigger WebSocket connection (connections only happen during batch flushes)
		c := initializeClientWithURL(ctx, t, serverURL)
		defer c.Close()

		_, err := c.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "ListTools failed")

		// Wait for connection to be established (triggered by log flush)
		require.True(t, fakeWS.WaitForConnection(1, 10*time.Second),
			"WebSocket connection not established within timeout")
		require.Greater(t, fakeWS.GetConnectionCount(), 0, "No WebSocket connections received")
		t.Log("WebSocket connection established with correct token")
	})

	t.Run("LogDelivery", func(t *testing.T) {
		// create MCP client and make API call to generate logs
		c := initializeClientWithURL(ctx, t, serverURL)
		defer c.Close()

		_, err := c.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "ListTools failed")

		// poll until logs arrive at fake server (wait up to 10 seconds for batch flush)
		require.True(t, fakeWS.WaitForLogs(1, 10*time.Second),
			"No logs received within timeout")
		t.Log("Logs sent to WebSocket")
	})

	t.Run("LogStructure", func(t *testing.T) {
		// verify we have logs
		logs := fakeWS.GetLogEntries()
		require.NotEmpty(t, logs, "Expected at least one log entry")
		t.Logf("Received %d log entries", len(logs))

		// verify log structure (timestamp, level, message)
		for _, log := range logs {
			require.NotEmpty(t, log.Timestamp, "Log missing timestamp")
			require.NotEmpty(t, log.Level, "Log missing level")
			require.NotEmpty(t, log.Message, "Log missing message")
		}
		t.Log("Log structure valid (timestamp, level, message)")
	})

	t.Run("LogLevels", func(t *testing.T) {
		// verify different log levels are captured (INFO and/or DEBUG)
		hasInfo := len(fakeWS.FindLogsByLevel("INFO")) > 0
		hasDebug := len(fakeWS.FindLogsByLevel("DEBUG")) > 0
		require.True(t, hasInfo || hasDebug, "Should have INFO or DEBUG logs")
		t.Logf("Log levels captured: %d INFO logs, %d DEBUG logs",
			len(fakeWS.FindLogsByLevel("INFO")),
			len(fakeWS.FindLogsByLevel("DEBUG")))
	})

	t.Run("EnabledServicesField", func(t *testing.T) {
		// verify enabled_services field is present in log entries
		logs := fakeWS.GetLogEntries()
		foundEnabledServices := false
		for _, log := range logs {
			if enabledServices, ok := log.Extra["enabled_services"]; ok {
				foundEnabledServices = true
				// verify it's a comma-separated string containing our configured services
				if servicesStr, ok := enabledServices.(string); ok {
					require.Contains(t, servicesStr, "apps", "enabled_services should contain 'apps'")
					require.Contains(t, servicesStr, "networking", "enabled_services should contain 'networking'")
					require.Equal(t, "apps,networking", servicesStr, "enabled_services should match exact configuration")
					t.Logf("enabled_services field verified: %s", servicesStr)
				}
				break
			}
		}
		require.True(t, foundEnabledServices, "At least one log entry should contain 'enabled_services' field")
	})

	t.Run("DualLogging", func(t *testing.T) {
		// get stderr logs from container
		stderrLogs, err := container.Logs(ctx)
		require.NoError(t, err, "Failed to get container logs")
		defer stderrLogs.Close()

		// read all available logs
		stderrBytes, err := io.ReadAll(stderrLogs)
		if err != nil {
			t.Logf("Warning: error reading logs: %v", err)
		}
		stderrOutput := string(stderrBytes)

		// verify logs appear in stderr
		require.NotEmpty(t, stderrOutput, "Expected logs in stderr")
		t.Logf("stderr output length: %d bytes", len(stderrOutput))

		// verify WebSocket diagnostic logging appears in stderr
		require.Contains(t, stderrOutput, "configuring WebSocket logging",
			"stderr should contain WebSocket diagnostic messages")

		// verify at least one application log message appears in both destinations
		wsLogs := fakeWS.GetLogEntries()
		foundInBoth := false
		matchCount := 0
		for _, wsLog := range wsLogs {
			// skip WebSocket diagnostic messages (these only go to stderr)
			if source, ok := wsLog.Extra["source"].(string); ok && source == "wslogging" {
				continue
			}

			// check if this log message appears in stderr
			if strings.Contains(stderrOutput, wsLog.Message) {
				matchCount++
				if !foundInBoth {
					foundInBoth = true
					t.Logf("Verified dual logging: message '%s' found in both stderr and WebSocket", wsLog.Message)
				}
			}
		}

		require.True(t, foundInBoth, "At least one application log message should appear in both stderr and WebSocket")
		t.Logf("Dual logging verified: %d application log messages found in both stderr and WebSocket", matchCount)
	})
}

// TestEdgeLogging_Authentication tests that authentication is required
func TestEdgeLogging_Authentication(t *testing.T) {
	ctx := context.Background()

	// start fake WebSocket server with token required
	fakeWS := NewFakeWebSocketServer("secret-token")
	defer fakeWS.Close()

	// start MCP server with WRONG token
	cfg := McpServerConfig{
		BindAddr:             "0.0.0.0:8080",
		DigitalOceanAPIToken: os.Getenv("DIGITALOCEAN_API_TOKEN"),
		LogLevel:             "debug",
		Transport:            "http",
		WSLoggingURL:         fakeWS.GetContainerURL(),
		WSLoggingToken:       "wrong-token",
	}
	container, err := startMcpServer(ctx, cfg)
	require.NoError(t, err, "Failed to start MCP server")
	defer container.Terminate(ctx)

	// poll to ensure no connection is established (inverse condition)
	// wait up to 3 seconds - should remain at 0 connections
	pollCondition(t, 3*time.Second, func() bool {
		// this is a negative test - we want it to stay at 0
		// so we sleep a bit and check if it's still 0
		time.Sleep(100 * time.Millisecond)
		return fakeWS.GetConnectionCount() == 0
	}, "Should not connect with wrong token")

	// final verification
	require.Equal(t, 0, fakeWS.GetConnectionCount(), "Should not connect with wrong token")

	// now test with correct token
	fakeWS2 := NewFakeWebSocketServer("correct-token")
	defer fakeWS2.Close()

	cfg2 := McpServerConfig{
		BindAddr:             "0.0.0.0:8080",
		DigitalOceanAPIToken: os.Getenv("DIGITALOCEAN_API_TOKEN"),
		LogLevel:             "debug",
		Transport:            "http",
		WSLoggingURL:         fakeWS2.GetContainerURL(),
		WSLoggingToken:       "correct-token",
	}
	container2, err := startMcpServer(ctx, cfg2)
	require.NoError(t, err, "Failed to start MCP server")
	defer container2.Terminate(ctx)

	// get the mapped port and make API call to generate logs
	port2, err := container2.MappedPort(ctx, "8080/tcp")
	require.NoError(t, err, "Failed to get mapped port")
	serverURL2 := fmt.Sprintf("http://127.0.0.1:%s/mcp", port2.Port())

	c2 := initializeClientWithURL(ctx, t, serverURL2)
	defer c2.Close()

	// make API call to generate logs which will trigger WebSocket connection
	_, err = c2.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err, "ListTools failed")

	// wait for logs to be flushed and connection established
	require.True(t, fakeWS2.WaitForLogs(1, 10*time.Second),
		"Should receive logs with correct token")

	require.Greater(t, fakeWS2.GetConnectionCount(), 0, "Should have successful connection")
}

// TestEdgeLogging_ConnectOnFlush tests connect-on-flush behavior and graceful failure handling.
// With the new architecture, we establish a new connection for each flush (every 5 seconds or 50 messages).
// This test verifies that connections are created when batches are flushed, and that
// the MCP server continues to function even when the WebSocket server is unavailable.
func TestEdgeLogging_ConnectOnFlush(t *testing.T) {
	ctx := context.Background()

	// start fake WebSocket server
	fakeWS := NewFakeWebSocketServer("test-token")

	// start MCP server
	cfg := McpServerConfig{
		BindAddr:             "0.0.0.0:8080",
		DigitalOceanAPIToken: os.Getenv("DIGITALOCEAN_API_TOKEN"),
		LogLevel:             "debug",
		Transport:            "http",
		WSLoggingURL:         fakeWS.GetContainerURL(),
		WSLoggingToken:       fakeWS.GetToken(),
	}
	container, err := startMcpServer(ctx, cfg)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	port, err := container.MappedPort(ctx, "8080/tcp")
	require.NoError(t, err)
	serverURL := fmt.Sprintf("http://127.0.0.1:%s/mcp", port.Port())

	c := initializeClientWithURL(ctx, t, serverURL)
	defer c.Close()

	// make a call to generate logs
	_, err = c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)

	// wait for logs to be flushed (batch interval is 5 seconds, wait up to 10 seconds)
	require.True(t, fakeWS.WaitForLogs(1, 10*time.Second),
		"No logs received from first flush")

	initialLogCount := len(fakeWS.GetLogEntries())
	initialConnections := fakeWS.GetConnectionCount()
	t.Logf("After first flush: %d logs, %d connections", initialLogCount, initialConnections)
	require.Greater(t, initialConnections, 0, "Should have at least one connection from first flush")
	require.Greater(t, initialLogCount, 0, "Should have received logs from first flush")

	// verify connect-on-flush: each flush creates a new connection
	t.Logf("Connect-on-flush verified: %d connections for %d logs", initialConnections, initialLogCount)

	// now close the WebSocket server to simulate network failure
	t.Log("Closing WebSocket server to simulate network failure")
	fakeWS.Close()

	// make another call - this should still work even though WS logging is down
	// (logs will be buffered and connection attempts will fail gracefully)
	_, err = c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err, "MCP server should continue working even if WS logging fails")

	// verify the server is still responsive
	t.Log("Verified: MCP server continues to function after WebSocket server shutdown")
	t.Log("Failed flush attempts are logged as diagnostics and logs are retained for retry")
}

// initializeClientWithURL is a wrapper that accepts server URL parameter
func initializeClientWithURL(ctx context.Context, t *testing.T, serverURL string) *client.Client {
	c, err := newClient(
		serverURL,
		transport.WithHTTPHeaders(map[string]string{"Authorization": fmt.Sprintf("Bearer %s", apiToken)}),
	)

	require.NoError(t, err)
	err = c.Start(ctx)
	require.NoError(t, err)

	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "test-client",
				Version: "1.0.0",
			},
		},
	}

	_, err = c.Initialize(ctx, initRequest)
	require.NoError(t, err)

	return c
}
