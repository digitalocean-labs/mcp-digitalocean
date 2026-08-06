package registry

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"mcp-digitalocean/pkg/registry/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
)

// annotationTestClient is a stand-in godo client. Register only needs a client
// factory to build the tools; no tool handler is invoked here.
func annotationTestClient(context.Context) (*godo.Client, error) {
	return godo.NewFromToken("test-token"), nil
}

// validOperations and validRisks bound the registry _meta values every tool
// must carry.
var (
	validOperations = map[string]bool{
		string(common.OpRead):   true,
		string(common.OpCreate): true,
		string(common.OpUpdate): true,
		string(common.OpDelete): true,
	}
	validRisks = map[string]bool{
		string(common.RiskLow):    true,
		string(common.RiskMedium): true,
		string(common.RiskHigh):   true,
	}
)

// TestEveryRegisteredToolAnnotated is the backstop guard: it registers every
// supported service through the real Register path (exactly what the binary
// does) and asserts that every resulting tool carries complete MCP hint
// annotations and a well-formed registry _meta (permission derived from the
// name, a valid operation, and a valid risk).
//
// Because it reads the tools back from the live server via ListTools rather
// than from a hand-maintained enumeration, a newly added tool is covered
// automatically: if it ships without annotations, this test fails and — since
// CI runs `go test ./...` — the pipeline goes red. No per-service test needs to
// be touched for the guard to catch it.
func TestEveryRegisteredToolAnnotated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svr := server.NewMCPServer("test", "test")

	// No services specified => Register loads all supported services.
	if err := Register(logger, svr, annotationTestClient); err != nil {
		t.Fatalf("Register: %v", err)
	}

	tools := svr.ListTools()
	if len(tools) == 0 {
		t.Fatal("no tools registered — Register wired up nothing")
	}

	for name, st := range tools {
		a := st.Tool.Annotations
		if a.ReadOnlyHint == nil || a.DestructiveHint == nil || a.IdempotentHint == nil || a.OpenWorldHint == nil {
			t.Errorf("tool %q is missing MCP hint annotations — register it with common.WithHints(...)", name)
		}

		if st.Tool.Meta == nil || st.Tool.Meta.AdditionalFields == nil {
			t.Errorf("tool %q is missing _meta — register it with common.WithHints(...)", name)
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
		if op, _ := reg["operation"].(string); !validOperations[op] {
			t.Errorf("tool %q has invalid/missing operation %q (set via common.WithHints)", name, op)
		}
		if risk, _ := reg["risk"].(string); !validRisks[risk] {
			t.Errorf("tool %q has invalid/missing risk %q — set it with common.WithRisk(...)", name, risk)
		}
	}
}
