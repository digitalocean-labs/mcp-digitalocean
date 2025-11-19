# WebSocket Logging

This package provides a drop-in replacement for Go's standard `slog.Handler` that writes logs to both stderr and a WebSocket endpoint.

## Why?

When running services in production, you often want logs in two places:
- **Stderr** - for local debugging and container logs
- **WebSocket** - for real-time log streaming to external systems

This handler does both automatically.

## Basic Usage

```go
import "mcp-digitalocean/internal/wslogging"

// create the handler (works just like slog.NewJSONHandler)
handler := wslogging.NewHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})

// optionally configure WebSocket logging
if wsURL != "" {
    handler.ConfigureWebSocket(wsURL, authToken)
    handler.Start(ctx)
    defer handler.Close(context.Background())
}

// use it like any other slog handler
logger := slog.New(handler)
logger.Info("hello world")
```

## Adding Persistent Fields

You can add fields that appear in every log entry using `WithAttrs()`. This is useful for context like service configuration:

```go
// add persistent fields (like enabled services)
handler = handler.WithAttrs([]slog.Attr{
    slog.String("enabled_services", "apps,networking,databases"),
}).(*wslogging.Handler)

logger := slog.New(handler)
logger.Info("server started")
// output: {"level":"INFO","message":"server started","enabled_services":"apps,networking,databases",...}
```

This is particularly helpful for:
- **Metrics and filtering** - Query logs by service configuration
- **Debugging** - Understand which services were enabled when issues occurred
- **Observability** - Track behavior across different service combinations

> **Note**: Use comma-separated strings for list values (like `"apps,networking"`) instead of arrays. This keeps cardinality low in metrics systems and makes queries simpler.

## Features

- **Dual output** - Logs go to both stderr and WebSocket simultaneously
- **Non-blocking** - Uses buffered channels so logging never blocks your application
- **Time-based batching** - Messages are batched for 5 seconds or up to 50 messages to reduce WebSocket write overhead
- **Thread-safe** - All operations are protected with mutexes for safe concurrent access
- **Standard interface** - Implements `slog.Handler`, so it works with the standard library
- **Persistent attributes** - Add context fields that appear in all logs
- **Graceful shutdown** - `Close()` flushes remaining messages before shutdown

## Configuration

### Batching Behavior

To reduce WebSocket write overhead, logs are batched before being sent:
- **Batch interval**: 5 seconds - messages accumulate for up to 5 seconds before flushing
- **Max batch size**: 50 messages - batch flushes immediately when 50 messages are queued
- **Buffer size**: 1000 messages - the channel buffer between Handle() and logWriter()
- **Graceful shutdown**: During `Close()`, any pending messages are flushed immediately

This batching significantly improves performance under high log volume while maintaining reasonable latency (max 5 seconds delay).

### Thread Safety

The handler uses two mutexes for thread-safe operation:
- **wsMu**: Protects WebSocket configuration (wsEnabled, wsURL, wsToken, closed state)
- **flushMu**: Protects the batch slice during accumulation and flushing

The design ensures:
- Non-blocking log writes via buffered channel
- Safe concurrent access from multiple goroutines
- Single writer to batch (logWriter goroutine)
- Copy semantics during flush to avoid holding locks during network I/O

## Testing

Run the tests with:
```bash
go test -v ./internal/wslogging -timeout 60s
```

Note: Some tests (like ping/pong) take up to 35 seconds, so use a 60-second timeout.
