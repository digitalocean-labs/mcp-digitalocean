package calltrace

import (
	"context"
	"log/slog"
	"testing"
)

func TestNilTraceIsNoOp(t *testing.T) {
	tr := FromContext(context.Background())
	if tr != nil {
		t.Fatalf("FromContext without trace = %v, want nil", tr)
	}
	tr.Add(slog.Int("n", 1))
	if got := tr.Attrs(); got != nil {
		t.Fatalf("nil trace attrs = %v, want nil", got)
	}
}

func TestTraceRoundTrip(t *testing.T) {
	ctx, tr := New(context.Background())
	FromContext(ctx).Add(slog.Int("result_count", 3), slog.String("top_url", "u"))
	got := tr.Attrs()
	if len(got) != 2 || got[0].Key != "result_count" || got[1].Value.String() != "u" {
		t.Fatalf("attrs = %v", got)
	}
	got[0] = slog.Int("mutated", 0)
	if tr.Attrs()[0].Key != "result_count" {
		t.Fatal("Attrs must return a copy")
	}
}
