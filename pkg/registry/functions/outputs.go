package functions

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// Every payload here is either a bare godo resource or an array of them, so
// each one carries an envelope to satisfy MCP's object-root requirement.
// namespaceOut is shared by the get and create tools, and triggerOut by the
// get, create and update tools, because each group returns the same resource
// shape.
//
// The delete tools (namespace, access key, trigger, action, package) stay
// text-only: each returns a fixed success message rather than a resource, so an
// output schema would describe nothing. functions-deployment-guide likewise
// stays text-only — it returns the embedded DEPLOY_SPEC.md markdown.
//
// The OpenWhisk data-plane tools (actions, packages and activations) also stay
// text-only. Those handlers never parse the response: they take the raw bytes
// the OpenWhisk API returned and re-indent them, and this package has no typed
// OpenWhisk models to reflect. Decoding them into a Go type to gain a schema
// would silently drop any field the type does not declare, changing what the
// tools return. Activation results compound this — a result is whatever JSON
// the user's own function returned, so it has no describable shape at all.
var (
	namespaceListOut = common.NewOutput[[]godo.FunctionsNamespace]("namespaces")
	namespaceOut     = common.NewOutput[*godo.FunctionsNamespace]("namespace")
	accessKeyListOut = common.NewOutput[[]godo.FunctionsAccessKey]("access_keys")
	accessKeyOut     = common.NewOutput[*godo.FunctionsAccessKey]("access_key")
	triggerListOut   = common.NewOutput[[]godo.FunctionsTrigger]("triggers")
	triggerOut       = common.NewOutput[*godo.FunctionsTrigger]("trigger")
)
