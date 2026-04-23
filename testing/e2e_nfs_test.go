//go:build integration

package testing

import (
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestNfsShareLifecycleAndGet(t *testing.T) {
	t.Parallel()

	newShare := CreateTestNfsShare(t, "mcp-e2e-nfs")

	activeShare := WaitForNfsShareActive(t, newShare.ID, defaultActionTimeout)

	getShare := callTool[godo.Nfs](t, "nfs-file-share-get", map[string]any{
		"ID": activeShare.ID,
	})

	require.Equal(t, activeShare.ID, getShare.ID)
	t.Logf("[Get] Successfully retrieved nfs share:")
	t.Logf("      Name: %s", getShare.Name)
	t.Logf("      ID: %s", getShare.ID)
	t.Logf("      Size: %d", getShare.SizeGib)

	t.Logf("[Delete] Deleting nfs share: %s...", activeShare.Name)

	DeleteResource(t, "nfs-file-share", activeShare.ID)
}

func TestNfsResize(t *testing.T) {
	t.Parallel()

	newShare := CreateTestNfsShare(t, "mcp-e2e-nfs")

	activeShare := WaitForNfsShareActive(t, newShare.ID, defaultActionTimeout)

	_ = callTool[godo.NfsAction](t, "nfs-resize", map[string]any{
		"ShareID":       activeShare.ID,
		"SizeGibibytes": 100,
	})

	_ = callTool[godo.Nfs](t, "nfs-file-share-get", map[string]any{
		"ID": activeShare.ID,
	})

	var resizedShare godo.Nfs
	require.Eventually(t, func() bool {
		resizedShare = callTool[godo.Nfs](t, "nfs-file-share-get", map[string]any{
			"ID": activeShare.ID,
		})
		return resizedShare.SizeGib == 100
	}, defaultActionTimeout, defaultPollInterval, "nfs share did not resize in time")

	t.Logf("[Resize] Successfully resized nfs share: %s to %d GiB", activeShare.Name, resizedShare.SizeGib)
	t.Logf("      ID: %s", resizedShare.ID)
	t.Logf("      Size: %d", resizedShare.SizeGib)

	t.Logf("[Delete] Deleting nfs share: %s...", activeShare.Name)

	DeleteResource(t, "nfs-file-share", activeShare.ID)
}

func TestNfsSnapshot(t *testing.T) {
	t.Skip("Skipping due to flaky NFS share name collision. Will re-enable after cleanup fix.")
	t.Parallel()

	newShare := CreateTestNfsShare(t, "mcp-e2e-nfs")

	activeShare := WaitForNfsShareActive(t, newShare.ID, defaultActionTimeout)
	snapshotName := fmt.Sprintf("e2e-snap-%d", time.Now().Unix())
	action := callTool[godo.NfsAction](t, "nfs-snapshot", map[string]any{
		"ShareID":      activeShare.ID,
		"SnapshotName": snapshotName,
	})

	getSnapshot := callTool[godo.NfsSnapshot](t, "nfs-snapshot-get", map[string]any{
		"ID": action.ResourceID,
	})

	require.Equal(t, snapshotName, getSnapshot.Name)
	require.Equal(t, activeShare.ID, getSnapshot.ShareID)
	t.Logf("[Snapshot] Successfully created snapshot: %s", getSnapshot.Name)
	t.Logf("      ID: %s", getSnapshot.ID)
	t.Logf("      Name: %s", getSnapshot.Name)
	t.Logf("      Share ID: %s", getSnapshot.ShareID)
	t.Logf("      Size: %d", getSnapshot.SizeGib)
	t.Logf("      Region: %s", getSnapshot.Region)
	t.Logf("      Created At: %s", getSnapshot.CreatedAt)
	t.Logf("      Status: %s", getSnapshot.Status)

	t.Logf("[Delete] Deleting snapshot: %s...", "mcp-e2e-nfs-snapshot")

	DeleteResource(t, "nfs-snapshot", getSnapshot.ID)

	t.Logf("[Delete] Deleting nfs share: %s...", activeShare.Name)

	DeleteResource(t, "nfs-file-share", activeShare.ID)
}

func TestNfsDetachAndAttach(t *testing.T) {
	t.Skip("Skipping due to flaky NFS share name collision. Will re-enable after cleanup fix.")
	t.Parallel()

	newShare := CreateTestNfsShare(t, "mcp-e2e-nfs")

	activeShare := WaitForNfsShareActive(t, newShare.ID, defaultActionTimeout)

	vpcId := activeShare.VpcIDs[0]
	t.Logf("[Detach] Attempting to detach nfs share: %s from %s", activeShare.Name, vpcId)
	_ = callTool[godo.NfsAction](t, "nfs-detach", map[string]any{
		"ShareID": activeShare.ID,
		"VpcID":   vpcId,
	})

	var detachedShare godo.Nfs
	require.Eventually(t, func() bool {
		detachedShare = callTool[godo.Nfs](t, "nfs-file-share-get", map[string]any{
			"ID": activeShare.ID,
		})
		t.Logf("[Detach] Detached nfs share: %s State: %s", activeShare.Name, detachedShare.Status)
		return detachedShare.Status == "INACTIVE"
	}, defaultActionTimeout, defaultPollInterval, "nfs share did not detach in time")
	t.Logf("[Detach] Successfully detached nfs share: %s from %s", activeShare.Name, vpcId)

	t.Logf("[Attach] Attempting to attach nfs share: %s to %s", activeShare.Name, vpcId)
	_ = callTool[godo.NfsAction](t, "nfs-attach", map[string]any{
		"ShareID": activeShare.ID,
		"VpcID":   vpcId,
	})

	var attachedShare godo.Nfs
	require.Eventually(t, func() bool {
		attachedShare = callTool[godo.Nfs](t, "nfs-file-share-get", map[string]any{
			"ID": activeShare.ID,
		})
		t.Logf("[Attach] Attached nfs share: %s State: %s", attachedShare.Name, attachedShare.Status)
		return attachedShare.Status == godo.NfsShareActive
	}, defaultActionTimeout, defaultPollInterval, "nfs share did not attach in time")

	t.Logf("[Attach] Successfully attached nfs share: %s to %s", activeShare.Name, vpcId)
	t.Logf("      ID: %s", attachedShare.ID)
	t.Logf("      Name: %s", attachedShare.Name)
	t.Logf("      Size: %d", attachedShare.SizeGib)
	t.Logf("      VPC IDs: %v", attachedShare.VpcIDs)

	t.Logf("[Delete] Deleting nfs share: %s...", activeShare.Name)

	DeleteResource(t, "nfs-file-share", activeShare.ID)
}
