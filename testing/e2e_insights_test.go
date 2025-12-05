//go:build integration

package testing

import (
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// deleteUptimeCheck is a cleanup helper that logs errors but doesn't fail the test.
func deleteUptimeCheck(t *testing.T, tc testContext, checkID string) {
	t.Logf("deleting uptime check %s...", checkID)
	resp, err := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "uptimecheck-delete",
			Arguments: map[string]interface{}{"ID": checkID},
		},
	})
	if err != nil {
		t.Logf("failed to delete uptime check: %v", err)
		return
	}
	if resp.IsError {
		t.Logf("uptimecheck-delete returned error: %v", resp.Content)
		return
	}
	t.Logf("deleted uptime check %s", checkID)
}

// deleteUptimeCheckAlert is a cleanup helper for uptime check alerts.
func deleteUptimeCheckAlert(t *testing.T, tc testContext, checkID, alertID string) {
	t.Logf("deleting uptime check alert %s...", alertID)
	_, _ = tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "uptimecheck-alert-delete",
			Arguments: map[string]interface{}{
				"CheckID": checkID,
				"AlertID": alertID,
			},
		},
	})
}

// deleteAlertPolicy is a cleanup helper for alert policies.
func deleteAlertPolicy(t *testing.T, tc testContext, uuid string) {
	t.Logf("deleting alert policy %s...", uuid)
	resp, err := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "alert-policy-delete",
			Arguments: map[string]interface{}{"UUID": uuid},
		},
	})
	if err != nil {
		t.Logf("failed to delete alert policy: %v", err)
		return
	}
	if resp.IsError {
		t.Logf("alert-policy-delete returned error: %v", resp.Content)
		return
	}
	t.Logf("deleted alert policy %s", uuid)
}

// TestUptimeCheckLifecycle tests the full lifecycle of an uptime check:
func TestUptimeCheckLifecycle(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	checkName := fmt.Sprintf("test-uptime-check-%d", time.Now().Unix())
	target := "https://www.digitalocean.com"

	// create uptime check
	t.Log("creating uptime check...")
	uptimeCheck := callTool[godo.UptimeCheck](t, "uptimecheck-create", map[string]interface{}{
		"Name":    checkName,
		"Type":    "https",
		"Target":  target,
		"Regions": []string{"us_east", "us_west"},
		"Enabled": true,
	})
	require.NotEmpty(t, uptimeCheck.ID, "uptime check ID should not be empty")
	t.Logf("created uptime check: %s (ID: %s)", uptimeCheck.Name, uptimeCheck.ID)

	defer func() { deleteUptimeCheck(t, tc, uptimeCheck.ID) }()

	// get uptime check
	t.Log("getting uptime check...")
	fetchedCheck := callTool[godo.UptimeCheck](t, "uptimecheck-get", map[string]interface{}{
		"ID": uptimeCheck.ID,
	})
	require.Equal(t, uptimeCheck.ID, fetchedCheck.ID)
	require.Equal(t, checkName, fetchedCheck.Name)
	t.Logf("fetched uptime check: %s", fetchedCheck.Name)

	// list uptime checks
	t.Log("listing uptime checks...")
	checks := callTool[[]godo.UptimeCheck](t, "uptimecheck-list", map[string]interface{}{
		"Page":    1,
		"PerPage": 50,
	})
	requireFoundInList(t, checks, func(c godo.UptimeCheck) bool { return c.ID == uptimeCheck.ID }, "uptime check")
	t.Logf("found uptime check in list (total: %d)", len(checks))

	// get uptime check state
	t.Log("getting uptime check state...")
	checkState := callTool[godo.UptimeCheckState](t, "uptimecheck-get-state", map[string]interface{}{
		"ID": uptimeCheck.ID,
	})
	t.Logf("uptime check state: %+v", checkState)

	// update uptime check
	updatedName := checkName + "-updated"
	t.Log("updating uptime check...")
	updatedCheck := callTool[godo.UptimeCheck](t, "uptimecheck-update", map[string]interface{}{
		"ID":      uptimeCheck.ID,
		"Name":    updatedName,
		"Type":    "https",
		"Target":  target,
		"Regions": []string{"us_east", "eu_west"},
		"Enabled": true,
	})
	require.Equal(t, updatedName, updatedCheck.Name)
	t.Logf("updated uptime check name to: %s", updatedCheck.Name)
}

// TestUptimeCheckAlertLifecycle tests the full lifecycle of an uptime check alert:
func TestUptimeCheckAlertLifecycle(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	// Get account email (required for alert notifications - must be verified)
	t.Log("fetching account information...")
	account := callTool[godo.Account](t, "account-get-information", map[string]interface{}{})
	require.NotEmpty(t, account.Email, "account email should not be empty")
	t.Logf("using account email: %s", account.Email)

	checkName := fmt.Sprintf("test-uptime-alert-check-%d", time.Now().Unix())
	target := "https://www.digitalocean.com"

	// create uptime check first (required for alerts)
	t.Log("creating uptime check for alert testing...")
	uptimeCheck := callTool[godo.UptimeCheck](t, "uptimecheck-create", map[string]interface{}{
		"Name":    checkName,
		"Type":    "https",
		"Target":  target,
		"Regions": []string{"us_east"},
		"Enabled": true,
	})
	t.Logf("created uptime check: %s (ID: %s)", uptimeCheck.Name, uptimeCheck.ID)

	defer func() { deleteUptimeCheck(t, tc, uptimeCheck.ID) }()

	// create uptime check alert
	// Note: The DigitalOcean API requires at least one notification (email or slack)
	alertName := fmt.Sprintf("test-alert-%d", time.Now().Unix())
	t.Log("creating uptime check alert...")
	alert := callTool[godo.UptimeAlert](t, "uptimecheck-alert-create", map[string]interface{}{
		"CheckID":      uptimeCheck.ID,
		"Name":         alertName,
		"Type":         "down",
		"Period":       "2m",
		"Comparison":   "",
		"Emails":       []string{account.Email},
		"SlackDetails": []map[string]string{},
	})
	require.NotEmpty(t, alert.ID, "alert ID should not be empty")
	t.Logf("created uptime check alert: %s (ID: %s)", alert.Name, alert.ID)

	defer func() { deleteUptimeCheckAlert(t, tc, uptimeCheck.ID, alert.ID) }()

	// get uptime check alert
	t.Log("getting uptime check alert...")
	fetchedAlert := callTool[godo.UptimeAlert](t, "uptimecheck-alert-get", map[string]interface{}{
		"CheckID": uptimeCheck.ID,
		"AlertID": alert.ID,
	})
	require.Equal(t, alert.ID, fetchedAlert.ID)
	t.Logf("fetched uptime check alert: %s", fetchedAlert.Name)

	// list uptime check alerts
	t.Log("listing uptime check alerts...")
	alerts := callTool[[]godo.UptimeAlert](t, "uptimecheck-alert-list", map[string]interface{}{
		"CheckID": uptimeCheck.ID,
		"Page":    1,
		"PerPage": 50,
	})
	requireFoundInList(t, alerts, func(a godo.UptimeAlert) bool { return a.ID == alert.ID }, "alert")
	t.Logf("found alert in list (total: %d)", len(alerts))

	// update uptime check alert
	updatedAlertName := alertName + "-updated"
	t.Log("updating uptime check alert...")
	updatedAlert := callTool[godo.UptimeAlert](t, "uptimecheck-alert-update", map[string]interface{}{
		"CheckID":      uptimeCheck.ID,
		"AlertID":      alert.ID,
		"Name":         updatedAlertName,
		"Type":         "down",
		"Period":       "3m",
		"Comparison":   "",
		"Emails":       []string{account.Email},
		"SlackDetails": []map[string]string{},
	})
	require.Equal(t, updatedAlertName, updatedAlert.Name)
	t.Logf("updated alert name to: %s", updatedAlert.Name)
}

// TestAlertPolicyLifecycle tests the full lifecycle of an alert policy:
func TestAlertPolicyLifecycle(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	// Get account email (required for alert notifications - must be verified)
	t.Log("fetching account information...")
	account := callTool[godo.Account](t, "account-get-information", map[string]interface{}{})
	require.NotEmpty(t, account.Email, "account email should not be empty")
	t.Logf("using account email: %s", account.Email)

	policyDescription := fmt.Sprintf("test-policy-%d", time.Now().Unix())

	// create alert policy
	// Note: The DigitalOcean API requires at least one alert action (email or slack)
	t.Log("creating alert policy...")
	policy := callTool[godo.AlertPolicy](t, "alert-policy-create", map[string]interface{}{
		"Type":        "v1/insights/droplet/cpu",
		"Description": policyDescription,
		"Compare":     "GreaterThan",
		"Value":       80,
		"Window":      "5m",
		"Entities":    []string{},
		"Tags":        []string{"test-tag"},
		"Alerts": map[string]interface{}{
			"Email": []string{account.Email},
			"Slack": []interface{}{},
		},
		"Enabled": true,
	})
	require.NotEmpty(t, policy.UUID, "policy UUID should not be empty")
	t.Logf("created alert policy: %s (UUID: %s)", policy.Description, policy.UUID)

	defer func() { deleteAlertPolicy(t, tc, policy.UUID) }()

	// get alert policy
	t.Log("getting alert policy...")
	fetchedPolicy := callTool[godo.AlertPolicy](t, "alert-policy-get", map[string]interface{}{
		"UUID": policy.UUID,
	})
	require.Equal(t, policy.UUID, fetchedPolicy.UUID)
	t.Logf("fetched alert policy: %s", fetchedPolicy.Description)

	// list alert policies
	t.Log("listing alert policies...")
	policies := callTool[[]godo.AlertPolicy](t, "alert-policy-list", map[string]interface{}{
		"Page":    1,
		"PerPage": 50,
	})
	requireFoundInList(t, policies, func(p godo.AlertPolicy) bool { return p.UUID == policy.UUID }, "policy")
	t.Logf("found policy in list (total: %d)", len(policies))

	// update alert policy
	updatedDescription := policyDescription + "-updated"
	t.Log("updating alert policy...")
	updatedPolicy := callTool[godo.AlertPolicy](t, "alert-policy-update", map[string]interface{}{
		"UUID":        policy.UUID,
		"Type":        "v1/insights/droplet/cpu",
		"Description": updatedDescription,
		"Compare":     "GreaterThan",
		"Value":       90,
		"Window":      "10m",
		"Entities":    []string{},
		"Tags":        []string{"test-tag", "updated-tag"},
		"Alerts": map[string]interface{}{
			"Email": []string{account.Email},
			"Slack": []interface{}{},
		},
		"Enabled": true,
	})
	require.Equal(t, updatedDescription, updatedPolicy.Description)
	t.Logf("updated policy description to: %s", updatedPolicy.Description)
}
