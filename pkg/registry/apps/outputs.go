package apps

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// appOut, appSpecOut, appsOut and logsOut publish a bare resource or a bare
// array, so each one needs an envelope to satisfy MCP's object-root
// requirement. deploymentStatusOut publishes DeploymentStatus, which already
// pairs a deployment with its health, so an envelope would add a redundant
// second key. Its tool still answers "no deployments found for app X" as plain
// text when the app has never deployed; that branch carries no structured
// content, which the spec permits and the server's validator skips.
//
// appUpdateOut covers a tool with two return shapes: apps-update answers with
// the updated app when given a spec and with a new deployment when it only
// forces a rebuild. A tool may declare just one outputSchema, so the payload
// is an envelope holding both as optional fields with exactly one set, which
// describes either branch without misdescribing the other. Its text half stays
// the bare app or deployment it has always been, so it uses ResultWithText.
//
// apps-delete stays text-only: its result is a fixed success message, not a
// resource, so an output schema would describe nothing.
var (
	appOut              = common.NewOutput[*godo.App]("app")
	appSpecOut          = common.NewOutput[*godo.AppSpec]("spec")
	appsOut             = common.NewOutput[[]*AppSummary]("apps")
	logsOut             = common.NewOutput[*godo.AppLogs]("logs")
	deploymentStatusOut = common.NewObjectOutput[DeploymentStatus]()
	appUpdateOut        = common.NewObjectOutput[appUpdateResult]()
)
