package droplet

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
// parallelizable, streamingSafe. See annotations.go for the rationale behind
// the six profile categories these rows map to. permission is not listed
// per row because it is derived 1:1 from the tool name and asserted
// generically. risk is set per tool (via common.WithRisk) because it varies
// within a single hint profile.
var expectedAnnotations = map[string]struct {
	readOnly, destructive, idempotent, openWorld bool
	operation                                    common.Operation
	risk                                         common.Risk
	parallelizable, streamingSafe                bool
}{
	// droplet_tools.go
	"droplet-create":             {false, false, false, false, common.OpCreate, common.RiskHigh, false, false},
	"droplet-delete":             {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
	"droplet-enable-private-net": {false, false, true, false, common.OpUpdate, common.RiskHigh, false, false},
	"droplet-kernels":            {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"droplet-get":                {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"droplet-backup-policy":      {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"droplet-action":             {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"droplet-list":               {true, false, true, false, common.OpRead, common.RiskLow, false, false},

	// droplet_actions_tools.go
	"reboot-droplet":                  {false, false, false, false, common.OpUpdate, common.RiskMedium, false, false},
	"reset-droplet-password":          {false, false, false, false, common.OpUpdate, common.RiskMedium, false, false},
	"rebuild-droplet-by-slug":         {false, true, false, false, common.OpUpdate, common.RiskHigh, false, false},
	"power-cycle-droplets-tag":        {false, false, false, false, common.OpUpdate, common.RiskHigh, false, false},
	"power-on-droplets-tag":           {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"power-off-droplets-tag":          {false, false, true, false, common.OpUpdate, common.RiskHigh, false, false},
	"shutdown-droplets-tag":           {false, false, true, false, common.OpUpdate, common.RiskHigh, false, false},
	"enable-backups-droplets-tag":     {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"disable-backups-droplets-tag":    {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"snapshot-droplets-tag":           {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"enable-ipv6-droplets-tag":        {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"enable-private-net-droplets-tag": {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"power-cycle-droplet":             {false, false, false, false, common.OpUpdate, common.RiskMedium, false, false},
	"power-on-droplet":                {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"power-off-droplet":               {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"shutdown-droplet":                {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"restore-droplet":                 {false, true, false, false, common.OpUpdate, common.RiskHigh, false, false},
	"resize-droplet":                  {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"rebuild-droplet":                 {false, true, false, false, common.OpUpdate, common.RiskHigh, false, false},
	"rename-droplet":                  {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"change-kernel-droplet":           {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"enable-ipv6-droplet":             {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"enable-backups-droplet":          {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"disable-backups-droplet":         {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"snapshot-droplet":                {false, false, false, false, common.OpCreate, common.RiskLow, false, false},

	// image_actions_tools.go
	"image-action-transfer": {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"image-action-convert":  {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"image-action-get":      {true, false, true, false, common.OpRead, common.RiskLow, false, false},

	// images_tools.go
	"image-list":   {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"image-get":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"image-create": {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"image-update": {false, false, true, false, common.OpUpdate, common.RiskLow, false, false},
	"image-delete": {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},

	// sizes_tools.go
	"size-list": {true, false, true, false, common.OpRead, common.RiskLow, false, false},
}

func TestToolAnnotations(t *testing.T) {
	clientFn := func(context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test-token"), nil
	}

	var all []server.ServerTool
	all = append(all, NewDropletTool(clientFn).Tools()...)
	all = append(all, NewDropletActionsTool(clientFn).Tools()...)
	all = append(all, NewImageActionsTool(clientFn).Tools()...)
	all = append(all, NewImageTool(clientFn).Tools()...)
	all = append(all, NewSizesTool(clientFn).Tools()...)

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
