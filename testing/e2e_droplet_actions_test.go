//go:build integration

package testing

import (
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestDropletReboot(t *testing.T) {
	t.Parallel()

	d := CreateTestDroplet(t, "mcp-e2e-reboot")

	triggerActionAndWait(t, "reboot-droplet", map[string]interface{}{"ID": d.ID}, d.ID)
}

func TestDropletPowerCycle(t *testing.T) {
	t.Parallel()

	d := CreateTestDroplet(t, "mcp-e2e-powercycle")

	triggerActionAndWait(t, "power-cycle-droplet", map[string]interface{}{"ID": d.ID}, d.ID)
}

func TestDropletSnapshotAction(t *testing.T) {
	t.Parallel()

	d := CreateTestDroplet(t, "mcp-e2e-snap-action")

	snapName := fmt.Sprintf("e2e-snap-%d", time.Now().Unix())
	imageID := CreateDropletSnapshot(t, d.ID, snapName)

	t.Logf("Snapshot verified. Image ID: %d", imageID)
}

func TestDropletRename(t *testing.T) {
	t.Parallel()

	d := CreateTestDroplet(t, "mcp-e2e-rename")

	newName := fmt.Sprintf("%s-renamed", d.Name)
	triggerActionAndWait(t, "rename-droplet", map[string]interface{}{
		"ID":   d.ID,
		"Name": newName,
	}, d.ID)

	// Verify Name Change
	refreshed, err := WaitForDropletCondition(t, d.ID, func(droplet *godo.Droplet) bool {
		return droplet != nil && droplet.Name == newName
	}, defaultPollInterval, renameVerifyTimeout)

	require.NoError(t, err, "Failed to verify rename")
	require.Equal(t, newName, refreshed.Name)

	t.Logf("Droplet successfully renamed to: %s", refreshed.Name)
}

func TestDropletEnableIPv6(t *testing.T) {
	t.Parallel()

	d := CreateTestDroplet(t, "mcp-e2e-ipv6")

	triggerActionAndWait(t, "enable-ipv6-droplet", map[string]interface{}{"ID": d.ID}, d.ID)

	// Verify IPv6 Address Assignment
	refreshed, err := WaitForDropletCondition(t, d.ID, func(droplet *godo.Droplet) bool {
		return droplet != nil && len(droplet.Networks.V6) > 0
	}, resourcePollInterval, ipv6AssignTimeout)

	require.NoError(t, err, "IPv6 address was not assigned within timeout")
	t.Logf("IPv6 Assigned: %s", refreshed.Networks.V6[0].IPAddress)
}

func TestDropletEnableBackups(t *testing.T) {
	t.Parallel()

	d := CreateTestDroplet(t, "mcp-e2e-backups")

	triggerActionAndWait(t, "enable-backups-droplet", map[string]interface{}{"ID": d.ID}, d.ID)

	// Verify Backups Enabled
	refreshed, err := WaitForDropletCondition(t, d.ID, func(droplet *godo.Droplet) bool {
		if droplet == nil {
			return false
		}
		// Check standard fields
		if len(droplet.BackupIDs) > 0 || droplet.NextBackupWindow != nil {
			return true
		}
		// Fallback: Check explicit policy
		policy := callTool[godo.DropletBackupPolicy](t, "droplet-backup-policy", map[string]interface{}{"ID": droplet.ID})
		return policy.BackupEnabled
	}, resourcePollInterval, backupsEnableTimeout)

	require.NoError(t, err, "Backups were not enabled within timeout")
	t.Logf("Backups Verified for Droplet %d", refreshed.ID)
}

func TestDropletActionTool(t *testing.T) {
	t.Parallel()

	d := CreateTestDroplet(t, "mcp-e2e-action-tool")

	cycleAction := callTool[godo.Action](t, "power-cycle-droplet", map[string]interface{}{"ID": d.ID})
	require.NotZero(t, cycleAction.ID)

	fetchedAction := callTool[godo.Action](t, "droplet-action", map[string]interface{}{
		"DropletID": float64(d.ID),
		"ActionID":  float64(cycleAction.ID),
	})

	t.Logf("Retrieved Action via Tool: ID=%d, Type=%s, Status=%s", fetchedAction.ID, fetchedAction.Type, fetchedAction.Status)

	require.Equal(t, cycleAction.ID, fetchedAction.ID)
	require.Equal(t, "power_cycle", fetchedAction.Type)

	final := WaitForActionComplete(t, d.ID, cycleAction.ID, defaultActionTimeout)

	LogActionStatus(t, "ActionToolTest", final)
}
