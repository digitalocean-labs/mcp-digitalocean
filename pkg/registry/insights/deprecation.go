package insights

import "github.com/mark3labs/mcp-go/mcp"

const alertPolicyV1Notice = "DEPRECATED: alert-policy-* uses the legacy Monitoring API and will be removed. Use insights-alert-rule-list | insights-alert-rule-get | insights-alert-rule-create | insights-alert-rule-update | insights-alert-rule-delete."

const (
	descAlertPolicyGetV1    = "DEPRECATED — Get a legacy Monitoring alert policy. Use insights-alert-rule-get."
	descAlertPolicyListV1   = "DEPRECATED — List legacy Monitoring alert policies. Use insights-alert-rule-list."
	descAlertPolicyCreateV1 = "DEPRECATED — Create a legacy Monitoring alert policy (Type like v1/insights/droplet/cpu). Use insights-alert-rule-create."
	descAlertPolicyUpdateV1 = "DEPRECATED — Update a legacy Monitoring alert policy. Use insights-alert-rule-update."
	descAlertPolicyDeleteV1 = "DEPRECATED — Delete a legacy Monitoring alert policy. Use insights-alert-rule-delete."

	titleAlertPolicyGetV1    = "Deprecated — Get alert policy (v1)"
	titleAlertPolicyListV1   = "Deprecated — List alert policies (v1)"
	titleAlertPolicyCreateV1 = "Deprecated — Create alert policy (v1)"
	titleAlertPolicyUpdateV1 = "Deprecated — Update alert policy (v1)"
	titleAlertPolicyDeleteV1 = "Deprecated — Delete alert policy (v1)"

	titleAlertRuleGetV2    = "Insights v2 — Get alert rule"
	titleAlertRuleListV2   = "Insights v2 — List alert rules"
	titleAlertRuleCreateV2 = "Insights v2 — Create alert rule"
	titleAlertRuleUpdateV2 = "Insights v2 — Update alert rule"
	titleAlertRuleDeleteV2 = "Insights v2 — Delete alert rule"
)

func withDisplayTitle(title string) mcp.ToolOption {
	return func(t *mcp.Tool) {
		mcp.WithToolTitle(title)(t)
		mcp.WithTitleAnnotation(title)(t)
	}
}

// withAlertPolicyDeprecation keeps Content[0] as the JSON payload so existing
// e2e unmarshallers keep working, and adds a second text block models will see.
func withAlertPolicyDeprecation(res *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
	if err != nil || res == nil || res.IsError {
		return res, err
	}
	res.Content = append(res.Content, mcp.TextContent{Type: "text", Text: alertPolicyV1Notice})
	return res, nil
}
