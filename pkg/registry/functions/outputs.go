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
// The OpenWhisk data-plane tools (actions, packages and activations) describe
// their responses with the models in openwhisk_models.go, but emit them with
// ResultRaw rather than Result. Those handlers deliberately never decode the
// response — they forward the bytes OpenWhisk returned — and decoding them
// into a Go type just to re-encode it would drop any field the model does not
// declare. ResultRaw keeps the passthrough intact while still declaring a
// schema, which holds because the generated schema requires nothing and
// permits additional properties.
//
// actionOut, packageOut and activationOut are each shared across that
// resource's tools, since the list view OpenWhisk returns is a subset of the
// detail view and validates against the same schema.
//
// functions-invoke-action is the one data-plane tool that stays text-only, and
// not for lack of a model: it answers with a full activation when blocking,
// with just an activation id when not, and with the invoked function's own
// return value when Result is set. That last shape is user-defined, so
// declaring any schema risks rejecting a perfectly good invocation whose
// result happens to use a key this package models with a different type.
var (
	namespaceListOut = common.NewOutput[[]godo.FunctionsNamespace]("namespaces")
	namespaceOut     = common.NewOutput[*godo.FunctionsNamespace]("namespace")
	accessKeyListOut = common.NewOutput[[]godo.FunctionsAccessKey]("access_keys")
	accessKeyOut     = common.NewOutput[*godo.FunctionsAccessKey]("access_key")
	triggerListOut   = common.NewOutput[[]godo.FunctionsTrigger]("triggers")
	triggerOut       = common.NewOutput[*godo.FunctionsTrigger]("trigger")

	actionListOut       = common.NewOutput[[]owAction]("actions")
	actionOut           = common.NewOutput[*owAction]("action")
	packageListOut      = common.NewOutput[[]owPackage]("packages")
	packageOut          = common.NewOutput[*owPackage]("package")
	activationListOut   = common.NewOutput[[]owActivation]("activations")
	activationOut       = common.NewOutput[*owActivation]("activation")
	activationLogsOut   = common.NewObjectOutput[owActivationLogs]()
	activationResultOut = common.NewObjectOutput[owActivationResult]()
)
