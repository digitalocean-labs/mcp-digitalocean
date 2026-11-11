package insights

import (
	"time"

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
//
// Dashboards: godo.Dashboard carries its Perses document in Spec, tagged
// `json:"-"` and emitted by a custom MarshalJSON as the JSON-encoded string the
// API uses on the wire. Reflecting godo.Dashboard directly would therefore
// advertise a schema with no "spec" even though every response carries one,
// so the dashboard tools publish the local dashboard type below, which mirrors
// godo's wire shape field for field and declares spec as a string.
// dashboardSchemaOut is map[string]any rather than json.RawMessage: RawMessage
// is a []byte and would reflect to a base64 string, while the handler passes
// the upstream JSON Schema through verbatim with ResultRaw.
var (
	uptimeCheckOut          = common.NewOutput[*godo.UptimeCheck]("check")
	uptimeChecksOut         = common.NewOutput[[]godo.UptimeCheck]("checks")
	uptimeCheckStateOut     = common.NewOutput[*godo.UptimeCheckState]("state")
	uptimeAlertOut          = common.NewOutput[*godo.UptimeAlert]("alert")
	uptimeAlertsOut         = common.NewOutput[[]godo.UptimeAlert]("alerts")
	alertPolicyOut          = common.NewOutput[*godo.AlertPolicy]("alert_policy")
	alertPoliciesOut        = common.NewOutput[[]godo.AlertPolicy]("alert_policies")
	alertRuleOut            = common.NewOutput[*godo.AlertRule]("alert_rule")
	alertRulesOut           = common.NewOutput[[]godo.AlertRule]("alert_rules")
	notificationChannelOut  = common.NewOutput[*godo.NotificationChannel]("notification_channel")
	notificationChannelsOut = common.NewOutput[[]godo.NotificationChannel]("notification_channels")
	alertInstanceOut        = common.NewOutput[*godo.AlertInstance]("alert_instance")
	alertInstancesOut       = common.NewOutput[[]godo.AlertInstance]("alert_instances")
	promQueryOut            = common.NewOutput[*godo.PromQueryResponse]("query")
	promQueryRangeOut       = common.NewOutput[*godo.PromQueryRangeResponse]("query_range")
	promSeriesOut           = common.NewOutput[*godo.PromSeriesResponse]("series")
	promLabelsOut           = common.NewOutput[*godo.PromLabelsResponse]("labels")
	logsSearchOut           = common.NewOutput[*godo.LogsSearchResponse]("logs")
	spansSearchOut          = common.NewOutput[*godo.SpansSearchResponse]("spans")
	tracesSearchOut         = common.NewOutput[*godo.TracesSearchResponse]("traces")
	traceOut                = common.NewOutput[*godo.Trace]("trace")
	resourceOut             = common.NewOutput[*godo.InsightsResource]("resource")
	resourcesOut            = common.NewOutput[*godo.InsightsResourceList]("resources")
	dashboardOut            = common.NewOutput[*dashboard]("dashboard")
	dashboardsOut           = common.NewObjectOutput[dashboardList]()
	dashboardSchemaOut      = common.NewOutput[map[string]any]("schema")
)

// dashboard is godo.Dashboard as it appears on the wire: the same fields in
// the same order, with spec visible to the schema reflector as the JSON-encoded
// string the API returns.
type dashboard struct {
	UUID        string    `json:"uuid"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Spec        string    `json:"spec" jsonschema:"description=Perses dashboard document as a JSON-encoded string"`
}

// dashboardList is godo.DashboardList with its dashboards in the wire shape.
type dashboardList struct {
	Dashboards []dashboard                `json:"dashboards,omitempty"`
	Pagination *godo.DashboardsPagination `json:"pagination,omitempty"`
}

func dashboardFromGodo(d *godo.Dashboard) *dashboard {
	if d == nil {
		return nil
	}
	return &dashboard{
		UUID:        d.UUID,
		Name:        d.Name,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
		Spec:        string(d.Spec),
	}
}

func dashboardListFromGodo(l *godo.DashboardList) dashboardList {
	if l == nil {
		return dashboardList{}
	}
	out := dashboardList{Pagination: l.Pagination}
	if len(l.Dashboards) > 0 {
		out.Dashboards = make([]dashboard, 0, len(l.Dashboards))
		for i := range l.Dashboards {
			out.Dashboards = append(out.Dashboards, *dashboardFromGodo(&l.Dashboards[i]))
		}
	}
	return out
}
