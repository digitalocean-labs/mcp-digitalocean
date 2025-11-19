//go:build integration

package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestDropletLifecycle(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	droplet := CreateTestDroplet(ctx, c, t, "mcp-e2e-test")
	LogResourceCreated(t, "droplet", droplet.ID, droplet.Name, droplet.Status, droplet.Region.Slug)

	require.Equal(t, "active", droplet.Status)
	t.Logf("Retrieved droplet successfully")

	var droplets []map[string]interface{}
	callToolJSON(ctx, c, t, "droplet-list", map[string]interface{}{"Page": 1, "PerPage": 50}, &droplets)
	require.NotEmpty(t, droplets)

	found := false
	for _, d := range droplets {
		if int(d["id"].(float64)) == droplet.ID {
			found = true
			break
		}
	}
	require.True(t, found, "Created droplet not found in list")
	LogResourceList(t, "droplet", "in list", droplets)

	resources := ListResources(ctx, c, t, "droplet", "before deletion", 1, 50)
	LogResourceList(t, "droplet", "before deletion", resources)

	deleteResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "droplet-delete",
			Arguments: map[string]interface{}{
				"ID": float64(droplet.ID),
			},
		},
	})
	require.NoError(t, err)
	LogResourceDeleted(t, "droplet", droplet.ID, err, deleteResp)
}

func TestDropletSnapshot(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	droplet := CreateTestDroplet(ctx, c, t, "mcp-e2e-snapshot")
	defer deferCleanupDroplet(ctx, c, t, droplet.ID)()
	LogResourceCreated(t, "droplet", droplet.ID, droplet.Name, droplet.Status, droplet.Region.Slug)

	snapshotName := fmt.Sprintf("snapshot-%d", time.Now().Unix())
	var action godo.Action
	callToolJSON(ctx, c, t, "snapshot-droplet", map[string]interface{}{"ID": float64(droplet.ID), "Name": snapshotName}, &action)
	require.NotEmpty(t, action.ID)
	t.Logf("Snapshot action initiated: ID=%s, Name=%s", formatID(action.ID), snapshotName)
	completedAction := WaitForActionComplete(ctx, c, t, action.ID, 2*time.Minute)
	LogActionCompleted(t, "Snapshot", completedAction)

	var refreshed godo.Droplet
	callToolJSON(ctx, c, t, "droplet-get", map[string]interface{}{"ID": float64(droplet.ID)}, &refreshed)
	if len(refreshed.SnapshotIDs) > 0 {
		snapshotImageID := float64(refreshed.SnapshotIDs[0])
		t.Logf("Snapshot image ID: %s", formatID(snapshotImageID))
		// Ensure snapshot image is cleaned up after the test
		defer deferCleanupImage(ctx, c, t, snapshotImageID)()
	}
}

func TestDropletRebuildBySlug(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	var images []map[string]interface{}
	callToolJSON(ctx, c, t, "image-list", map[string]interface{}{"Page": 1, "PerPage": 10}, &images)
	require.NotEmpty(t, images)

	var imageSlug string
	var imageID float64
	for _, img := range images {
		if slug, ok := img["slug"].(string); ok && slug == "ubuntu-22-04-x64" {
			imageSlug = slug
			imageID = img["id"].(float64)
			break
		}
	}
	if imageSlug == "" {
		for _, img := range images {
			if slug, ok := img["slug"].(string); ok && slug != "" {
				imageSlug = slug
				imageID = img["id"].(float64)
				break
			}
		}
	}
	if imageSlug == "" {
		t.Skip("No image with slug found; skipping rebuild-by-slug test")
	}
	t.Logf("Using image slug: %s (ID: %s)", imageSlug, formatID(imageID))

	droplet := CreateTestDroplet(ctx, c, t, "mcp-e2e-rebuild-slug")
	LogResourceCreated(t, "droplet", droplet.ID, droplet.Name, droplet.Status, droplet.Region.Slug)
	defer deferCleanupDroplet(ctx, c, t, droplet.ID)()

	var action godo.Action
	callToolJSON(ctx, c, t, "rebuild-droplet-by-slug", map[string]interface{}{"ID": float64(droplet.ID), "ImageSlug": imageSlug}, &action)
	require.NotEmpty(t, action.ID)
	t.Logf("Rebuild by slug action initiated: ID=%s, ImageSlug=%s", formatID(action.ID), imageSlug)
	completedAction := WaitForActionComplete(ctx, c, t, action.ID, 2*time.Minute)
	LogActionCompleted(t, "Rebuild", completedAction)
}

func TestDropletRestore(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	droplet := CreateTestDroplet(ctx, c, t, "mcp-e2e-restore")
	LogResourceCreated(t, "droplet", droplet.ID, droplet.Name, droplet.Status, droplet.Region.Slug)
	defer deferCleanupDroplet(ctx, c, t, droplet.ID)()

	snapshotName := fmt.Sprintf("restore-snapshot-%d", time.Now().Unix())
	var snapshotAction godo.Action
	callToolJSON(ctx, c, t, "snapshot-droplet", map[string]interface{}{"ID": float64(droplet.ID), "Name": snapshotName}, &snapshotAction)
	t.Logf("Snapshot created: %s", snapshotName)

	_ = WaitForActionComplete(ctx, c, t, snapshotAction.ID, 2*time.Minute)

	var refreshedDroplet godo.Droplet
	callToolJSON(ctx, c, t, "droplet-get", map[string]interface{}{"ID": float64(droplet.ID)}, &refreshedDroplet)
	require.NotEmpty(t, refreshedDroplet.SnapshotIDs, "Droplet should have at least one snapshot")

	snapshotImageID := float64(refreshedDroplet.SnapshotIDs[0])
	t.Logf("Refreshed droplet: ID=%s, Status=%s, Name=%s, Snapshots=%v", formatID(refreshedDroplet.ID), refreshedDroplet.Status, refreshedDroplet.Name, refreshedDroplet.SnapshotIDs)
	t.Logf("Using snapshot image ID: %s", formatID(snapshotImageID))
	// Ensure snapshot image created for restore is removed during cleanup
	defer deferCleanupImage(ctx, c, t, snapshotImageID)()

	var restoreAction godo.Action
	callToolJSON(ctx, c, t, "restore-droplet", map[string]interface{}{"ID": float64(droplet.ID), "ImageID": snapshotImageID}, &restoreAction)
	require.NotEmpty(t, restoreAction.ID)
	t.Logf("Restore action initiated: ID=%s, ImageID=%s", formatID(restoreAction.ID), formatID(snapshotImageID))
	completedAction := WaitForActionComplete(ctx, c, t, restoreAction.ID, 2*time.Minute)
	LogActionCompleted(t, "Restore", completedAction)
}
