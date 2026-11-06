// Package calltrace lets a tool handler hand log fields back to the tool
// logging middleware without putting them in the result the client sees.
package calltrace

import (
	"context"
	"log/slog"
	"sync"
)

type traceKey struct{}

// Trace collects log attributes for one tool call. A nil *Trace is valid and
// discards everything, so handlers can record unconditionally.
type Trace struct {
	mu    sync.Mutex
	attrs []slog.Attr
}

// New returns a context carrying a fresh Trace, and the Trace.
func New(ctx context.Context) (context.Context, *Trace) {
	t := &Trace{}
	return context.WithValue(ctx, traceKey{}, t), t
}

// FromContext returns the Trace stored in ctx, or nil when there is none
// (logging disabled, stdio without middleware, tests).
func FromContext(ctx context.Context) *Trace {
	t, _ := ctx.Value(traceKey{}).(*Trace)
	return t
}

// Add records attributes for the current call.
func (t *Trace) Add(attrs ...slog.Attr) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.attrs = append(t.attrs, attrs...)
}

// Attrs returns a copy of the recorded attributes.
func (t *Trace) Attrs() []slog.Attr {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]slog.Attr(nil), t.attrs...)
}
