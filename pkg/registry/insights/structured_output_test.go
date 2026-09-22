package insights

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestStructuredOutputSatisfiesDeclaredSchema drives a migrated tool through a
// real MCPServer with WithOutputSchemaValidation enabled, so the server itself
// checks the emitted structuredContent against the outputSchema the tool
// advertises. Unit tests on the handler cannot catch a schema that the payload
// does not satisfy; only the server (or a spec-compliant client) validates.
//
// Validation is enabled only here, not in cmd/mcp-digitalocean: in production
// a schema mismatch should degrade to a slightly-wrong schema, not a failed
// tool call. This test is what keeps the mismatch from reaching production.
//
// alert-policy-list is the representative tool: it is the package's richest
// payload, an array of policies each nesting the alert notification settings.
func TestStructuredOutputSatisfiesDeclaredSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMonitoring := NewMockMonitoringService(ctrl)
	mockMonitoring.EXPECT().
		ListAlertPolicies(gomock.Any(), &godo.ListOptions{Page: defaultAlertPoliciesPage, PerPage: defaultAlertPoliciesPageSize}).
		Return([]godo.AlertPolicy{{
			UUID:        "policy-1",
			Type:        "v1/insights/droplet/cpu",
			Description: "high cpu",
			Compare:     godo.GreaterThan,
			Value:       80,
			Window:      "5m",
			Entities:    []string{"12345678"},
			Tags:        []string{"production"},
			Alerts: godo.Alerts{
				Email: []string{"ops@example.com"},
				Slack: []godo.SlackDetails{{URL: "https://hooks.slack.test/x", Channel: "#alerts"}},
			},
			Enabled: true,
		}}, nil, nil).
		Times(1)

	svr := server.NewMCPServer("test", "test", server.WithOutputSchemaValidation())
	svr.AddTools(setupAlertPolicyToolWithMock(mockMonitoring).Tools()...)

	ctx := context.Background()
	svr.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))

	raw, err := json.Marshal(svr.HandleMessage(ctx,
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"alert-policy-list","arguments":{}}}`)))
	require.NoError(t, err)

	var resp struct {
		Error  *struct{ Message string } `json:"error"`
		Result struct {
			IsError           bool              `json:"isError"`
			Content           []mcp.TextContent `json:"content"`
			StructuredContent struct {
				AlertPolicies []godo.AlertPolicy `json:"alert_policies"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	require.Nil(t, resp.Error, "server rejected the call")
	require.False(t, resp.Result.IsError, "tool returned an error result: %s", raw)

	// The payload is a bare array, so it is published under an envelope to
	// satisfy MCP's object-root requirement for structuredContent.
	require.Len(t, resp.Result.StructuredContent.AlertPolicies, 1)
	policy := resp.Result.StructuredContent.AlertPolicies[0]
	require.Equal(t, "policy-1", policy.UUID)
	require.Equal(t, []string{"ops@example.com"}, policy.Alerts.Email)
	require.Len(t, policy.Alerts.Slack, 1)

	// The text block is retained for backwards compatibility and carries the
	// same payload, unenveloped as it has always been.
	require.Len(t, resp.Result.Content, 1)
	var fromText []godo.AlertPolicy
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &fromText))
	require.Equal(t, resp.Result.StructuredContent.AlertPolicies, fromText)
}
