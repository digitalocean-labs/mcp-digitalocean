//go:build integration

package testing

import (
	"slices"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestImageTransfer(t *testing.T) {
	t.Parallel()

	image := CreateTestSnapshotImage(t, "mcp-e2e-img-transfer")

	require.NotEmpty(t, image.Regions)

	// Determine a target region different from current
	currentRegion := image.Regions[0]
	targetRegion := testRegionNYC1
	if currentRegion == testRegionNYC1 {
		targetRegion = testRegionNYC3
	}

	t.Logf("Transferring image %d from %s to %s...", image.ID, currentRegion, targetRegion)

	triggerImageActionAndWait(t, "image-action-transfer", map[string]any{
		"ID":     float64(image.ID),
		"Region": targetRegion,
	}, image.ID)

	refreshedImage := callTool[godo.Image](t, "image-get", map[string]any{
		"ID": float64(image.ID),
	})

	found := slices.Contains(refreshedImage.Regions, targetRegion)

	require.True(t, found, "Image does not list target region after transfer")
	t.Logf("Image transfer verified. Regions: %v", refreshedImage.Regions)
}
