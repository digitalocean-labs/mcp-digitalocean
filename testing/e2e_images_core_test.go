//go:build integration

package testing

import (
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestImageList(t *testing.T) {
	t.Parallel()

	t.Logf("[List] Requesting distribution images (PerPage: %d)...", imageListPerPage)

	images := callTool[[]map[string]any](t, "image-list", map[string]any{
		"Type":    "distribution",
		"PerPage": imageListPerPage,
	})

	require.NotEmpty(t, images)
	require.Equal(t, "base", images[0]["type"])

	t.Logf("[List] Successfully listed %d distribution images:", len(images))
	for _, img := range images {
		name, _ := img["name"].(string)
		slug, _ := img["slug"].(string)
		id, _ := img["id"].(float64)
		t.Logf(" - %s (ID: %.0f, Slug: %s)", name, id, slug)
	}
}

func TestImageGet(t *testing.T) {
	t.Parallel()

	id, slug := getTestImage(t)

	t.Logf("[Get] Requesting details for Image ID: %.0f (Slug: %s)...", id, slug)

	image := callTool[godo.Image](t, "image-get", map[string]any{
		"ID": id,
	})

	require.Equal(t, int(id), image.ID)
	require.NotEmpty(t, image.Slug)

	t.Logf("[Get] Successfully retrieved image:")
	t.Logf("      Name: %s", image.Name)
	t.Logf("      ID: %d", image.ID)
	t.Logf("      Slug: %s", image.Slug)
	t.Logf("      Distribution: %s", image.Distribution)
	t.Logf("      Type: %s", image.Type)
	t.Logf("      Regions: %v", image.Regions)
}

func TestImageLifecycle(t *testing.T) {
	t.Parallel()
	image := CreateTestSnapshotImage(t, "mcp-e2e-image-lifecycle")

	newName := fmt.Sprintf("%s-renamed", image.Name)
	t.Logf("[Update] Renaming image %d to %s...", image.ID, newName)

	updatedImage := callTool[godo.Image](t, "image-update", map[string]any{
		"ID":   float64(image.ID),
		"Name": newName,
	})

	require.Equal(t, newName, updatedImage.Name)
	t.Logf("[Update] Success. New Name: %s", updatedImage.Name)

	DeleteResource(t, "image", float64(image.ID))

	t.Logf("[Verify] Waiting for image %d deletion...", image.ID)
	err := WaitForImageDeletion(t, image.ID, defaultPollInterval, imageDeleteTimeout)
	require.NoError(t, err, "Image was not deleted")
	t.Logf("[Verify] Confirmed image %d deletion", image.ID)
}
