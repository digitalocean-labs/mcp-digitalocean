package docr

import (
	"context"
	"strings"
	"testing"

	"mcp-digitalocean/pkg/registry/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
)

// expectedAnnotations is the per-tool annotation contract enforced by
// TestToolAnnotations. Adding a tool to this package WITHOUT a row here, or
// changing an annotation in code without updating the row, fails the test.
//
// Order: readOnly, destructive, idempotent, openWorld, operation, risk,
// parallelizable, streamingSafe. See pkg/registry/common/annotations.go for
// the rationale behind the six profile categories these rows map to.
// permission is not listed per row because it is derived 1:1 from the tool
// name and asserted generically. risk is set per tool (via common.WithRisk)
// because it varies within a single hint profile.
var expectedAnnotations = map[string]struct {
	readOnly, destructive, idempotent, openWorld bool
	operation                                    common.Operation
	risk                                         common.Risk
	parallelizable, streamingSafe                bool
}{
	// registry_tools.go
	"docr-get":                {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"docr-list":               {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"docr-create":             {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"docr-delete":             {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
	"docr-docker-credentials": {true, false, true, false, common.OpRead, common.RiskMedium, false, false},
	"docr-options":            {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"docr-validate-name":      {true, false, true, false, common.OpRead, common.RiskLow, false, false},

	// repository_tools.go
	"docr-repository-list":            {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"docr-repository-tag-list":        {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"docr-repository-tag-delete":      {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
	"docr-repository-manifest-list":   {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"docr-repository-manifest-delete": {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},

	// garbage_collection_tools.go
	"docr-garbage-collection-start":  {false, true, false, false, common.OpUpdate, common.RiskHigh, false, false},
	"docr-garbage-collection-get":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"docr-garbage-collection-list":   {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"docr-garbage-collection-update": {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},

	// subscription_tools.go
	"docr-subscription-get":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"docr-subscription-update": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
}

func TestToolAnnotations(t *testing.T) {
	clientFn := func(context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test-token"), nil
	}

	var all []server.ServerTool
	all = append(all, NewRegistryTool(clientFn).Tools()...)
	all = append(all, NewRepositoryTool(clientFn).Tools()...)
	all = append(all, NewGarbageCollectionTool(clientFn).Tools()...)
	all = append(all, NewSubscriptionTool(clientFn).Tools()...)

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
			t.Errorf("tool %q has no expected annotation row — add one to expectedAnnotations", name)
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

		// Registry metadata (_meta), set by the hint profile.
		if st.Tool.Meta == nil || st.Tool.Meta.AdditionalFields == nil {
			t.Errorf("tool %q is missing _meta", name)
			continue
		}
		reg, ok := st.Tool.Meta.AdditionalFields[common.RegistryMetaKey].(map[string]any)
		if !ok {
			t.Errorf("tool %q is missing %q registry metadata", name, common.RegistryMetaKey)
			continue
		}
		wantPerm := "tools.digitalocean." + strings.ReplaceAll(name, "-", "_")
		if got, _ := reg["permission"].(string); got != wantPerm {
			t.Errorf("tool %q permission = %q, want %q", name, got, wantPerm)
		}
		if got, _ := reg["operation"].(string); got != string(want.operation) {
			t.Errorf("tool %q operation = %q, want %q", name, got, want.operation)
		}
		if got, _ := reg["risk"].(string); got != string(want.risk) {
			t.Errorf("tool %q risk = %q, want %q", name, got, want.risk)
		}
		if got, _ := reg["parallelizable"].(bool); got != want.parallelizable {
			t.Errorf("tool %q parallelizable = %v, want %v", name, got, want.parallelizable)
		}
		if got, _ := reg["streamingSafe"].(bool); got != want.streamingSafe {
			t.Errorf("tool %q streamingSafe = %v, want %v", name, got, want.streamingSafe)
		}
	}
}
