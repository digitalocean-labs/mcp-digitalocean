//go:build integration

package testing

import (
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestDropletLifecycle(t *testing.T) {
	t.Parallel()

	droplet := CreateTestDroplet(t, "mcp-e2e-test")

	type dropletShort struct {
		ID int `json:"id"`
	}
	droplets := callTool[[]dropletShort](t, "droplet-list", map[string]interface{}{"Page": defaultPage, "PerPage": defaultPerPage})

	found := false
	for _, d := range droplets {
		if d.ID == droplet.ID {
			found = true
			break
		}
	}
	require.True(t, found, "Created droplet not found in list")

	DeleteResource(t, "droplet", droplet.ID)

	err := WaitForDropletDeletion(t, droplet.ID, resourcePollInterval, dropletDeleteTimeout)
	if err != nil {
		t.Logf("Warning: WaitForDropletDeletion failed: %v", err)
	} else {
		t.Logf("Confirmed droplet deletion via direct API")
	}
}

func TestDropletSnapshot(t *testing.T) {
	t.Parallel()

	droplet := CreateTestDroplet(t, "mcp-e2e-snapshot")

	snapshotName := fmt.Sprintf("snapshot-%d", time.Now().Unix())
	imageID := CreateDropletSnapshot(t, droplet.ID, snapshotName)

	t.Logf("Snapshot verified. Image ID: %d", imageID)
}

func TestDropletRebuildBySlug(t *testing.T) {
	t.Parallel()

	type imageShort struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
	}
	images := callTool[[]imageShort](t, "image-list", map[string]interface{}{"Page": defaultPage, "PerPage": imageListPerPage, "Type": "distribution"})

	var imageSlug string
	for _, img := range images {
		if img.Slug == defaultTestImageSlug {
			imageSlug = img.Slug
			break
		}
	}
	if imageSlug == "" && len(images) > 0 {
		imageSlug = images[0].Slug
	}
	if imageSlug == "" {
		t.Skip("No suitable image slug found")
	}

	droplet := CreateTestDroplet(t, "mcp-e2e-rebuild")

	action := callTool[godo.Action](t, "rebuild-droplet-by-slug", map[string]interface{}{
		"ID":        droplet.ID,
		"ImageSlug": imageSlug,
	})

	LogActionStatus(t, "Rebuild", action)
	completed := WaitForActionComplete(t, droplet.ID, action.ID, rebuildActionTimeout)
	LogActionStatus(t, "Rebuild", completed)
}

func TestDropletRestore(t *testing.T) {
	t.Parallel()

	droplet := CreateTestDroplet(t, "mcp-e2e-restore")

	snapName := fmt.Sprintf("restore-snap-%d", time.Now().Unix())
	imageID := CreateDropletSnapshot(t, droplet.ID, snapName)

	t.Logf("Restoring from Image ID: %d", imageID)

	restoreAction := callTool[godo.Action](t, "restore-droplet", map[string]interface{}{
		"ID":      droplet.ID,
		"ImageID": float64(imageID),
	})

	LogActionStatus(t, "Restore", restoreAction)
	completed := WaitForActionComplete(t, droplet.ID, restoreAction.ID, restoreActionTimeout)
	LogActionStatus(t, "Restore", completed)
}
