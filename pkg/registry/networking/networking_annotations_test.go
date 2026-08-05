package networking

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
	// vpc_tools.go
	"vpc-get":          {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"vpc-list":         {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"vpc-list-members": {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"vpc-create":       {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"vpc-delete":       {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},

	// vpc_peering_tools.go
	"vpc-peering-get":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"vpc-peering-list":   {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"vpc-peering-create": {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"vpc-peering-delete": {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},

	// reserved_ips_tools.go
	"reserved-ip-get":      {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"reserved-ip-list":     {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"reserved-ip-reserve":  {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"reserved-ip-release":  {false, true, true, false, common.OpDelete, common.RiskMedium, false, false},
	"reserved-ip-assign":   {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"reserved-ip-unassign": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// partner_attachment_tools.go
	"partner-attachment-get":             {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"partner-attachment-list":            {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"partner-attachment-create":          {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"partner-attachment-delete":          {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
	"partner-attachment-get-service-key": {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"partner-attachment-get-bgp-config":  {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"partner-attachment-update":          {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// load_balancers_tools.go
	"lb-create":           {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"lb-delete":           {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
	"lb-delete-cache":     {false, true, true, false, common.OpDelete, common.RiskMedium, false, false},
	"lb-get":              {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"lb-list":             {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"lb-add-droplets":     {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"lb-remove-droplets":  {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"lb-update":           {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"lb-add-fwd-rules":    {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"lb-remove-fwd-rules": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// firewall_tools.go
	"firewall-get":             {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"firewall-list":            {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"firewall-create":          {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"firewall-delete":          {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
	"firewall-add-droplets":    {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"firewall-add-tags":        {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"firewall-remove-droplets": {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"firewall-remove-tags":     {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"firewall-add-rules":       {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},
	"firewall-remove-rules":    {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// domains_tools.go
	"domain-get":           {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"domain-list":          {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"domain-record-get":    {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"domain-record-list":   {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"domain-create":        {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"domain-delete":        {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
	"domain-record-create": {false, false, false, false, common.OpCreate, common.RiskLow, false, false},
	"domain-record-delete": {false, true, true, false, common.OpDelete, common.RiskMedium, false, false},
	"domain-record-edit":   {false, false, true, false, common.OpUpdate, common.RiskMedium, false, false},

	// certificate_tools.go
	"certificate-get":                 {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"certificate-list":                {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"custom-certificate-create":       {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"lets-encrypt-certificate-create": {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"certificate-delete":              {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},

	// byoip_prefix_tools.go
	"byoip-prefix-get":           {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"byoip-prefix-list":          {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"byoip-prefix-resources-get": {true, false, true, false, common.OpRead, common.RiskLow, false, false},
	"byoip-prefix-create":        {false, false, false, false, common.OpCreate, common.RiskMedium, false, false},
	"byoip-prefix-delete":        {false, true, true, false, common.OpDelete, common.RiskHigh, false, false},
}

func TestToolAnnotations(t *testing.T) {
	clientFn := func(context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test-token"), nil
	}

	var all []server.ServerTool
	all = append(all, NewVPCTool(clientFn).Tools()...)
	all = append(all, NewVPCPeeringTool(clientFn).Tools()...)
	all = append(all, NewReservedIPTool(clientFn).Tools()...)
	all = append(all, NewPartnerAttachmentTool(clientFn).Tools()...)
	all = append(all, NewLoadBalancersTool(clientFn).Tools()...)
	all = append(all, NewFirewallTool(clientFn).Tools()...)
	all = append(all, NewDomainsTool(clientFn).Tools()...)
	all = append(all, NewCertificateTool(clientFn).Tools()...)
	all = append(all, NewBYOIPPrefixTool(clientFn).Tools()...)

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
