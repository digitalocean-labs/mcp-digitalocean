package dbaas

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
// parallelizable, streamingSafe. See common/annotations.go for the rationale
// behind the six profile categories these rows map to. permission is not
// listed per row because it is derived 1:1 from the tool name and asserted
// generically. risk is set per tool (via common.WithRisk) because it varies
// within a single hint profile.
var expectedAnnotations = map[string]struct {
	readOnly, destructive, idempotent, openWorld bool
	operation                                    common.Operation
	risk                                         common.Risk
	parallelizable, streamingSafe                bool
}{
	// cluster.go
	"db-cluster-list":                   {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-get":                    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-get-ca":                 {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-create":                 {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"db-cluster-delete":                 {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
	"db-cluster-resize":                 {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"db-cluster-list-backups":           {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-list-options":           {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-upgrade-major-version":  {false, false, false, false, common.OpUpdate, common.RiskHigh, false, false},
	"db-cluster-start-online-migration": {false, false, false, false, common.OpUpdate, common.RiskMedium, false, false},
	"db-cluster-stop-online-migration":  {false, false, false, false, common.OpUpdate, common.RiskMedium, false, false},
	"db-cluster-get-migration":          {true, false, true, false, common.OpRead, common.RiskLow, false, false},

	// firewall.go
	"db-cluster-get-firewall-rules":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-update-firewall-rules": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// kafka.go
	"db-cluster-list-topics":         {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-create-topic":        {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"db-cluster-get-topic":           {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-delete-topic":        {false, true, true, false, common.OpDelete, common.RiskMedium, false, false},
	"db-cluster-update-topic":        {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"db-cluster-get-kafka-config":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-update-kafka-config": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// mongo.go
	"db-cluster-get-mongodb-config":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-update-mongodb-config": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// mysql.go
	"db-cluster-get-mysql-config":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-update-mysql-config": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"db-cluster-get-sql-mode":        {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-set-sql-mode":        {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// opensearch.go
	"db-cluster-get-opensearch-config": {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-update-os-config":      {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// postgres.go
	"db-cluster-get-postgresql-config": {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-update-psql-config":    {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// redis.go
	"db-cluster-get-redis-config":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-update-redis-config": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// user.go
	"db-cluster-get-user":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-list-users":  {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"db-cluster-create-user": {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"db-cluster-update-user": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"db-cluster-delete-user": {false, true, true, false, common.OpDelete, common.RiskMedium, false, false},
}

func TestToolAnnotations(t *testing.T) {
	clientFn := func(context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test-token"), nil
	}

	var all []server.ServerTool
	all = append(all, NewClusterTool(clientFn).Tools()...)
	all = append(all, NewFirewallTool(clientFn).Tools()...)
	all = append(all, NewKafkaTool(clientFn).Tools()...)
	all = append(all, NewMongoTool(clientFn).Tools()...)
	all = append(all, NewMysqlTool(clientFn).Tools()...)
	all = append(all, NewOpenSearchTool(clientFn).Tools()...)
	all = append(all, NewPostgreSQLTool(clientFn).Tools()...)
	all = append(all, NewRedisTool(clientFn).Tools()...)
	all = append(all, NewUserTool(clientFn).Tools()...)

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
