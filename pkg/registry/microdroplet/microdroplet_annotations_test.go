package microdroplet

import (
	"context"
	"strings"
	"testing"

	"mcp-digitalocean/pkg/registry/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
)

var expectedAnnotations = map[string]struct {
	readOnly, destructive, idempotent, openWorld bool
	operation                                    common.Operation
	risk                                         common.Risk
	parallelizable, streamingSafe                bool
}{
	"microdroplet-create":             {false, false, false, false, common.OpCreate, common.RiskHigh, false, false},
	"microdroplet-list":               {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"microdroplet-get":                {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"microdroplet-delete":             {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
	"microdroplet-pause":              {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"microdroplet-resume":             {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"microdroplet-checkpoint-create":  {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"microdroplet-checkpoint-list":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"microdroplet-checkpoint-get":     {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"microdroplet-checkpoint-delete":  {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
}

func TestToolAnnotations(t *testing.T) {
	client := func(ctx context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test-token"), nil
	}

	tools := NewMicroDropletTool(client).Tools()
	byName := make(map[string]server.ServerTool, len(tools))
	for _, st := range tools {
		byName[st.Tool.Name] = st
	}

	for name, want := range expectedAnnotations {
		st, ok := byName[name]
		if !ok {
			t.Errorf("tool %q missing from Tools()", name)
			continue
		}

		a := st.Tool.Annotations
		if a.ReadOnlyHint == nil || *a.ReadOnlyHint != want.readOnly {
			t.Errorf("tool %q readOnlyHint = %v, want %v", name, a.ReadOnlyHint, want.readOnly)
		}
		if a.DestructiveHint == nil || *a.DestructiveHint != want.destructive {
			t.Errorf("tool %q destructiveHint = %v, want %v", name, a.DestructiveHint, want.destructive)
		}
		if a.IdempotentHint == nil || *a.IdempotentHint != want.idempotent {
			t.Errorf("tool %q idempotentHint = %v, want %v", name, a.IdempotentHint, want.idempotent)
		}
		if a.OpenWorldHint == nil || *a.OpenWorldHint != want.openWorld {
			t.Errorf("tool %q openWorldHint = %v, want %v", name, a.OpenWorldHint, want.openWorld)
		}

		reg := st.Tool.Meta.AdditionalFields[common.RegistryMetaKey].(map[string]any)
		if reg["operation"] != string(want.operation) {
			t.Errorf("tool %q operation = %v, want %v", name, reg["operation"], want.operation)
		}
		if reg["risk"] != string(want.risk) {
			t.Errorf("tool %q risk = %v, want %v", name, reg["risk"], want.risk)
		}

		wantPerm := "tools.digitalocean." + strings.ReplaceAll(name, "-", "_")
		if reg["permission"] != wantPerm {
			t.Errorf("tool %q permission = %v, want %v", name, reg["permission"], wantPerm)
		}
	}

	if len(byName) != len(expectedAnnotations) {
		t.Errorf("registered %d tools, expected %d — update expectedAnnotations when adding tools", len(byName), len(expectedAnnotations))
	}
}
