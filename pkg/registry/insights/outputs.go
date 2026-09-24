package insights

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// Every payload here is a bare resource or a bare array, so each one needs an
// envelope to satisfy MCP's object-root requirement. The field names follow the
// keys the DigitalOcean API itself uses: "check"/"checks" and "state" for
// uptime checks, "alert"/"alerts" for the alerts hanging off a check. Alert
// policies use "alert_policy"/"alert_policies" rather than the API's bare
// "policy", so that the two kinds of alert in this package stay distinct.
//
// uptimeCheckOut is shared by the get, create and update tools, which all
// resolve to the same check resource; uptimeAlertOut and alertPolicyOut are
// shared the same way.
//
// uptimecheck-delete, uptimecheck-alert-delete and alert-policy-delete stay
// text-only: each returns a fixed success message rather than a resource, so an
// output schema would describe nothing.
var (
	uptimeCheckOut      = common.NewOutput[*godo.UptimeCheck]("check")
	uptimeChecksOut     = common.NewOutput[[]godo.UptimeCheck]("checks")
	uptimeCheckStateOut = common.NewOutput[*godo.UptimeCheckState]("state")
	uptimeAlertOut      = common.NewOutput[*godo.UptimeAlert]("alert")
	uptimeAlertsOut     = common.NewOutput[[]godo.UptimeAlert]("alerts")
	alertPolicyOut      = common.NewOutput[*godo.AlertPolicy]("alert_policy")
	alertPoliciesOut    = common.NewOutput[[]godo.AlertPolicy]("alert_policies")
)
