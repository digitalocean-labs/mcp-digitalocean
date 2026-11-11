package insights

import "github.com/mark3labs/mcp-go/mcp"

const alertPolicyV1Notice = "DEPRECATED: alert-policy-* uses the legacy Monitoring API and will be removed. Use insights-alert-rule-list | insights-alert-rule-get | insights-alert-rule-create | insights-alert-rule-update | insights-alert-rule-delete."

const (
	descAlertPolicyGetV1    = "DEPRECATED — Get a legacy Monitoring alert policy. Use insights-alert-rule-get."
	descAlertPolicyListV1   = "DEPRECATED — List legacy Monitoring alert policies. Use insights-alert-rule-list."
	descAlertPolicyCreateV1 = "DEPRECATED — Create a legacy Monitoring alert policy (Type like v1/insights/droplet/cpu). Use insights-alert-rule-create."
	descAlertPolicyUpdateV1 = "DEPRECATED — Update a legacy Monitoring alert policy. Use insights-alert-rule-update."
	descAlertPolicyDeleteV1 = "DEPRECATED — Delete a legacy Monitoring alert policy. Use insights-alert-rule-delete."
)

// withAlertPolicyDeprecation keeps Content[0] as the JSON payload so existing
// e2e unmarshallers keep working, and adds a second text block models will see.
func withAlertPolicyDeprecation(res *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
	if err != nil || res == nil || res.IsError {
		return res, err
	}
	res.Content = append(res.Content, mcp.TextContent{Type: "text", Text: alertPolicyV1Notice})
	return res, nil
}
