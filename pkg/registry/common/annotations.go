package common

import (
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// Operation is the coarse-grained data operation a tool performs. It maps
// 1:1 to the tool registry's classification.operation field.
type Operation string

const (
	OpRead   Operation = "read"
	OpCreate Operation = "create"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
)

// RegistryMetaKey is the reverse-DNS-namespaced key under a tool's _meta
// that holds the registry metadata set by the hint profiles (permission,
// operation, parallelizable, streaming_safe). MCP reserves _meta for
// exactly this kind of non-standard, implementation-specific data.
const RegistryMetaKey = "mcp.digitalocean.com/registry"

// hints bundles the four standard MCP tool annotation hints — readOnly,
// destructive, idempotent, openWorld — together with the registry metadata
// that maps 1:1 into the tool registry (operation, parallelizable,
// streamingSafe). Tool registrations should reference one of the profile
// vars below (HintsRead, HintsCreate, HintsAction, HintsToggle, HintsDelete,
// HintsReplace) rather than constructing a hints literal at the call site;
// the profile names document the intent at the registration point and make
// it impossible to swap two bool arguments.
type hints struct {
	readOnly, destructive, idempotent, openWorld bool
	operation                                    Operation
	parallelizable, streamingSafe                bool
}

// apply sets the four MCP tool hint annotations on t (overriding the mcp-go
// library defaults, which assume the worst case: not read-only, destructive,
// not idempotent) and stashes the registry metadata under t.Meta. The
// permission is derived 1:1 from the tool name (droplet-get ->
// tools.digitalocean.droplet_get).
func (h hints) apply(t *mcp.Tool) {
	mcp.WithReadOnlyHintAnnotation(h.readOnly)(t)
	mcp.WithDestructiveHintAnnotation(h.destructive)(t)
	mcp.WithIdempotentHintAnnotation(h.idempotent)(t)
	mcp.WithOpenWorldHintAnnotation(h.openWorld)(t)

	reg := map[string]any{
		"permission":     "tools.digitalocean." + strings.ReplaceAll(t.Name, "-", "_"),
		"operation":      string(h.operation),
		"parallelizable": h.parallelizable,
		"streamingSafe":  h.streamingSafe,
	}
	if t.Meta == nil {
		t.Meta = &mcp.Meta{AdditionalFields: map[string]any{}}
	} else if t.Meta.AdditionalFields == nil {
		t.Meta.AdditionalFields = map[string]any{}
	}
	t.Meta.AdditionalFields[RegistryMetaKey] = reg
}

// WithHints returns a ToolOption that applies the given hint profile to a
// tool registration.
func WithHints(h hints) mcp.ToolOption { return h.apply }

// Hint profiles. Every tool in this package falls into one of six
// categories. openWorld is false on every profile.
//
// Delete is destructive AND idempotent: repeating a delete on an
// already-removed resource returns not-found with no additional side
// effect. Replace (rebuild, restore) is destructive but non-idempotent
// because each call kicks off a fresh action that replaces droplet state.
//
// parallelizable and streamingSafe are false on every profile today; they
// live on the struct so they are explicit and can diverge later.
var (
	// HintsRead — read-only list/get/inspect tools. Idempotent because
	// observing state again returns the same data.
	HintsRead = hints{readOnly: true, idempotent: true, operation: OpRead}

	// HintsCreate — non-destructive, non-idempotent tool that creates a new
	// resource (droplet create, image create, snapshot).
	HintsCreate = hints{operation: OpCreate}

	// HintsAction — non-destructive mutation that does not converge on a
	// target state; each call kicks off a fresh action (reboot, power-cycle,
	// password reset).
	HintsAction = hints{operation: OpUpdate}

	// HintsToggle — mutation that converges on a target state. Repeated
	// calls have no additional effect (power-on twice = still on; rename
	// to the same name = no-op).
	HintsToggle = hints{idempotent: true, operation: OpUpdate}

	// HintsDelete — destructive AND idempotent (see package note above).
	HintsDelete = hints{destructive: true, idempotent: true, operation: OpDelete}

	// HintsReplace — destructive AND non-idempotent (see package note above).
	HintsReplace = hints{destructive: true, operation: OpUpdate}
)
