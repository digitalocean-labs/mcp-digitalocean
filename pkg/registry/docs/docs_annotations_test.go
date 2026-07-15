package docs

import (
	"testing"
)

// expectedAnnotations is the per-tool annotation contract enforced by
// TestToolAnnotations. Adding a tool to this package WITHOUT a row here, or
// changing an annotation in code without updating the row, fails the test.
//
// Order: readOnly, destructive, idempotent, openWorld. All docs tools are
// read-only lookups against published documentation, so every row maps to
// the common.HintsRead profile.
var expectedAnnotations = map[string]struct {
	readOnly, destructive, idempotent, openWorld bool
}{
	"docs-search":           {true, false, true, false},
	"docs-get-page":         {true, false, true, false},
	"docs-find-for-service": {true, false, true, false},
	"docs-get-quickstart":   {true, false, true, false},
}

func TestToolAnnotations(t *testing.T) {
	all := NewDocsTool().Tools()

	if len(all) != len(expectedAnnotations) {
		t.Fatalf("tool count mismatch: registered=%d, expected=%d (add new tools to expectedAnnotations)", len(all), len(expectedAnnotations))
	}

	seen := make(map[string]bool, len(all))
	for _, st := range all {
		name := st.Tool.Name
		if seen[name] {
			t.Errorf("duplicate tool registration: %q", name)
			continue
		}
		seen[name] = true

		want, ok := expectedAnnotations[name]
		if !ok {
			t.Errorf("tool %q has no expected annotation row - add one to expectedAnnotations", name)
			continue
		}

		a := st.Tool.Annotations
		if a.ReadOnlyHint == nil || a.DestructiveHint == nil || a.IdempotentHint == nil || a.OpenWorldHint == nil {
			t.Errorf("tool %q is missing one or more annotation hints (got %+v)", name, a)
			continue
		}
		if got := *a.ReadOnlyHint; got != want.readOnly {
			t.Errorf("tool %q readOnlyHint = %v, want %v", name, got, want.readOnly)
		}
		if got := *a.DestructiveHint; got != want.destructive {
			t.Errorf("tool %q destructiveHint = %v, want %v", name, got, want.destructive)
		}
		if got := *a.IdempotentHint; got != want.idempotent {
			t.Errorf("tool %q idempotentHint = %v, want %v", name, got, want.idempotent)
		}
		if got := *a.OpenWorldHint; got != want.openWorld {
			t.Errorf("tool %q openWorldHint = %v, want %v", name, got, want.openWorld)
		}
	}
}
