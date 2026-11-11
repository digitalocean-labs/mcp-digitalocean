//go:build integration

package testing

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestNfsShareLifecycleAndGet(t *testing.T) {
	t.Parallel()

	newShare := CreateTestNfsShare(t, "nfs-e2e-lifecycle")

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

	newShare := CreateTestNfsShare(t, "nfs-e2e-resize")

	activeShare := WaitForNfsShareActive(t, newShare.ID, defaultActionTimeout)

	resizeAction := callTool[godo.NfsAction](t, "nfs-resize", map[string]any{
		"ShareID":       activeShare.ID,
		"SizeGibibytes": 100,
	})
	t.Logf("[Resize] Action: ID=%s Type=%s Status=%s", resizeAction.ID, resizeAction.Type, resizeAction.Status)

	var resizedShare godo.Nfs
	require.Eventually(t, func() bool {
		resizedShare = callTool[godo.Nfs](t, "nfs-file-share-get", map[string]any{
			"ID": activeShare.ID,
		})
		t.Logf("[Resize] nfs share: %s State: %s Size: %d", activeShare.Name, resizedShare.Status, resizedShare.SizeGib)
		return resizedShare.SizeGib == 100
	}, defaultActionTimeout, defaultPollInterval, "nfs share did not resize in time")

	t.Logf("[Resize] Successfully resized nfs share: %s to %d GiB", activeShare.Name, resizedShare.SizeGib)
	t.Logf("      ID: %s", resizedShare.ID)
	t.Logf("      Size: %d", resizedShare.SizeGib)

	t.Logf("[Delete] Deleting nfs share: %s...", activeShare.Name)

	DeleteResource(t, "nfs-file-share", activeShare.ID)
}

func TestNfsSnapshot(t *testing.T) {
	t.Skip("Skipping due to godo client having no way to poll nfs action. Nfs action is a uuid, while all methods require int id")

	newShare := CreateTestNfsShare(t, "nfs-e2e-snapshot")

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
	t.Parallel()

	newShare := CreateTestNfsShare(t, "nfs-e2e-attach")

	activeShare := WaitForNfsShareActive(t, newShare.ID, defaultActionTimeout)

	vpcId := activeShare.VpcIDs[0]
	t.Logf("[Detach] Attempting to detach nfs share: %s from %s", activeShare.Name, vpcId)
	detachAction := callTool[godo.NfsAction](t, "nfs-detach", map[string]any{
		"ShareID": activeShare.ID,
		"VpcID":   vpcId,
	})
	t.Logf("[Detach] Action: ID=%s Type=%s Status=%s", detachAction.ID, detachAction.Type, detachAction.Status)

	var detachedShare godo.Nfs
	require.Eventually(t, func() bool {
		detachedShare = callTool[godo.Nfs](t, "nfs-file-share-get", map[string]any{
			"ID": activeShare.ID,
		})
		t.Logf("[Detach] nfs share: %s State: %s VPC IDs: %v", activeShare.Name, detachedShare.Status, detachedShare.VpcIDs)
		// A share with no VPC left reports INACTIVE, a status godo has no
		// constant for, so the VPC list is what this waits on.
		return !slices.Contains(detachedShare.VpcIDs, vpcId)
	}, defaultActionTimeout, defaultPollInterval, "nfs share did not detach in time")
	t.Logf("[Detach] Successfully detached nfs share: %s from %s", activeShare.Name, vpcId)

	t.Logf("[Attach] Attempting to attach nfs share: %s to %s", activeShare.Name, vpcId)
	attachAction := callTool[godo.NfsAction](t, "nfs-attach", map[string]any{
		"ShareID": activeShare.ID,
		"VpcID":   vpcId,
	})
	t.Logf("[Attach] Action: ID=%s Type=%s Status=%s", attachAction.ID, attachAction.Type, attachAction.Status)

	var attachedShare godo.Nfs
	require.Eventually(t, func() bool {
		attachedShare = callTool[godo.Nfs](t, "nfs-file-share-get", map[string]any{
			"ID": activeShare.ID,
		})
		t.Logf("[Attach] nfs share: %s State: %s VPC IDs: %v", attachedShare.Name, attachedShare.Status, attachedShare.VpcIDs)
		return attachedShare.Status == godo.NfsShareActive && slices.Contains(attachedShare.VpcIDs, vpcId)
	}, defaultActionTimeout, defaultPollInterval, "nfs share did not attach in time")

	t.Logf("[Attach] Successfully attached nfs share: %s to %s", activeShare.Name, vpcId)
	t.Logf("      ID: %s", attachedShare.ID)
	t.Logf("      Name: %s", attachedShare.Name)
	t.Logf("      Size: %d", attachedShare.SizeGib)
	t.Logf("      VPC IDs: %v", attachedShare.VpcIDs)

	t.Logf("[Delete] Deleting nfs share: %s...", activeShare.Name)

	DeleteResource(t, "nfs-file-share", activeShare.ID)
}
