//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"mcp-digitalocean/internal/testhelpers"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

const dbaasClusterStatusOnline = "online"

// setupTest initializes context, MCP client (via Docker/HTTP), and Godo client.
func setupTest(t *testing.T) (context.Context, *client.Client, *godo.Client, func()) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	return ctx, c, gclient, func() {
		c.Close()
	}
}

// genericActionWaiter defines the signature for wait functions
type genericActionWaiter func(ctx context.Context, client *godo.Client, resourceID, actionID int, interval, timeout time.Duration) (*godo.Action, error)

// triggerGenericActionAndWait is a DRY helper to execute an action tool and wait for the result.
func triggerGenericActionAndWait(t *testing.T, ctx context.Context, c *client.Client, gclient *godo.Client, tool string, args map[string]any, resourceID int, waiter genericActionWaiter, timeout time.Duration) {
	action := callTool[godo.Action](ctx, c, t, tool, args)
	require.NotZero(t, action.ID, "Action ID should not be zero")

	// Log 1: Initial State
	LogActionStatus(t, tool, action)

	final, err := waiter(ctx, gclient, resourceID, action.ID, 2*time.Second, timeout)
	require.NoError(t, err, fmt.Sprintf("Action %s failed to complete", tool))

	// Log 2: Final State
	LogActionStatus(t, tool, *final)
}

// triggerActionAndWait is a specific helper for Droplet actions
func triggerActionAndWait(t *testing.T, ctx context.Context, c *client.Client, gclient *godo.Client, tool string, args map[string]any, resourceID int) {
	triggerGenericActionAndWait(t, ctx, c, gclient, tool, args, resourceID, testhelpers.WaitForAction, 5*time.Minute)
}

// triggerImageActionAndWait calls an image action tool and waits for completion.
func triggerImageActionAndWait(t *testing.T, ctx context.Context, c *client.Client, gclient *godo.Client, tool string, args map[string]any, imageID int) {
	triggerGenericActionAndWait(t, ctx, c, gclient, tool, args, imageID, testhelpers.WaitForImageAction, 5*time.Minute)
}

// callTool calls an MCP tool and returns the unmarshaled result T.
func callTool[T any](ctx context.Context, c *client.Client, t *testing.T, name string, args map[string]any) T {
	var result T
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	require.NoError(t, err)

	if resp.IsError {
		var errText string
		if len(resp.Content) > 0 {
			if tc, ok := resp.Content[0].(mcp.TextContent); ok {
				errText = tc.Text
			} else {
				errText = fmt.Sprintf("%v", resp.Content)
			}
		}
		t.Fatalf("Tool %s failed: %s", name, errText)
	}

	require.NotEmpty(t, resp.Content, "Tool %s returned empty content", name)
	tc, ok := resp.Content[0].(mcp.TextContent)
	require.True(t, ok, "Unexpected content type for %s", name)

	err = json.Unmarshal([]byte(tc.Text), &result)
	require.NoError(t, err, "Failed to unmarshal %s response", name)

	return result
}

// --- Resource Lifecycle Helpers ---

func CreateTestDroplet(ctx context.Context, c *client.Client, t *testing.T, namePrefix string) godo.Droplet {
	sshKeys := getSSHKeys(ctx, c, t)
	region := selectRegion(ctx, c, t)
	imageID, imageSlug := getTestImage(ctx, c, t)

	dropletName := fmt.Sprintf("%s-%d", namePrefix, time.Now().Unix())

	t.Logf("Creating Droplet: %s (Image: %s [ID: %.0f], Region: %s)...", dropletName, imageSlug, imageID, region)

	droplet := callTool[godo.Droplet](ctx, c, t, "droplet-create", map[string]any{
		"Name":       dropletName,
		"Size":       "s-1vcpu-1gb",
		"ImageID":    imageID,
		"Region":     region,
		"Backup":     false,
		"Monitoring": true,
		"SSHKeys":    sshKeys,
	})

	// Log 1: Initial State
	LogResourceCreated(t, "droplet", droplet.ID, droplet.Name, droplet.Status, region)

	activeDroplet := WaitForDropletActive(ctx, c, t, droplet.ID, 2*time.Minute)

	// Log 2: Confirmation of Active state
	LogResourceCreated(t, "droplet", activeDroplet.ID, activeDroplet.Name, activeDroplet.Status, activeDroplet.Region.Slug)

	return activeDroplet
}

// CreateTestSnapshotImage creates a droplet, snapshots it, then deletes the droplet.
func CreateTestSnapshotImage(ctx context.Context, c *client.Client, t *testing.T, namePrefix string) godo.Image {
	// 1. Create Droplet
	droplet := CreateTestDroplet(ctx, c, t, namePrefix+"-setup")
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)
	// Ensure cleanup of the setup droplet
	defer func() {
		DeleteResource(ctx, c, t, "droplet", float64(droplet.ID))
		testhelpers.WaitForDropletDeleted(ctx, gclient, droplet.ID, 2*time.Second, 2*time.Minute)
	}()

	// 2. Create Snapshot
	snapName := fmt.Sprintf("%s-snap-%d", namePrefix, time.Now().Unix())
	t.Logf("Creating snapshot %s from droplet %d...", snapName, droplet.ID)

	// Reuse the standard action trigger/wait helper
	triggerActionAndWait(t, ctx, c, gclient, "snapshot-droplet", map[string]any{
		"ID":   float64(droplet.ID),
		"Name": snapName,
	}, droplet.ID)

	// 3. Find the new image
	// The snapshot action doesn't return the image ID directly in the action response,
	// but the droplet's SnapshotIDs field will be updated.
	updatedDroplet, err := testhelpers.WaitForDroplet(ctx, gclient, droplet.ID, func(d *godo.Droplet) bool {
		return len(d.SnapshotIDs) > 0
	}, 3*time.Second, 1*time.Minute)
	require.NoError(t, err, "Failed to find created snapshot on droplet")

	imageID := updatedDroplet.SnapshotIDs[0]

	// 4. Wait for image to be available
	img := WaitForImageAvailable(ctx, c, t, imageID, 5*time.Minute)
	t.Logf("Created test image: %d (%s)", img.ID, img.Name)

	return img
}

func DeleteResource(ctx context.Context, c *client.Client, t *testing.T, resourceType string, id any) {
	// Use "image-delete" for both images and snapshots as they share the endpoint
	toolName := fmt.Sprintf("%s-delete", resourceType)
	if resourceType == "snapshot" {
		toolName = "image-delete"
	}

	args := map[string]any{
		"ID": id,
	}

	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	LogResourceDeleted(t, resourceType, id, err, resp)
}

func ListResources(ctx context.Context, c *client.Client, t *testing.T, resourceType string, page, perPage int) []map[string]any {
	return callTool[[]map[string]any](ctx, c, t, fmt.Sprintf("%s-list", resourceType), map[string]any{
		"Page":    page,
		"PerPage": perPage,
	})
}

func createDbaasCluster(ctx context.Context, t *testing.T, c *client.Client, name string, engine string, version string, region string, size string, numNodes int) godo.Database {
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-create",
			Arguments: map[string]interface{}{
				"name":      name,
				"engine":    engine,
				"version":   version,
				"region":    region,
				"size":      size,
				"num_nodes": numNodes,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	var cluster godo.Database
	clusterJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(clusterJSON), &cluster)
	require.NoError(t, err)
	t.Logf("Created cluster: %v", cluster.Name)

	return cluster
}

func deleteDbaasCluster(ctx context.Context, t *testing.T, c *client.Client, id string) {
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-delete",
			Arguments: map[string]interface{}{
				"id": id,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	if resp.IsError {
		t.Fatalf("Tool call returned error: %v", resp.Content)
	}

	t.Logf("Deleted cluster with ID: %s", id)
}

func dbaasAssertClusterExists(ctx context.Context, t *testing.T, c *client.Client, clusterID string) {
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-list",
			Arguments: map[string]interface{}{
				"page":     1,
				"per_page": 50,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "Tool call returned error: %v", resp.Content)

	var clusters []godo.Database
	err = json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &clusters)
	require.NoError(t, err)

	for _, cl := range clusters {
		if cl.ID == clusterID {
			t.Logf("Cluster %s found in list", clusterID)
			return
		}
	}

	t.Fatalf("Cluster %s not found in list", clusterID)
}

// --- Prerequisite Helpers ---

func getSSHKeys(ctx context.Context, c *client.Client, t *testing.T) []interface{} {
	keys := callTool[[]map[string]interface{}](ctx, c, t, "key-list", map[string]interface{}{})
	var keyIDs []interface{}
	for _, key := range keys {
		if id, ok := key["id"].(float64); ok {
			keyIDs = append(keyIDs, id)
		}
	}
	return keyIDs
}

func getTestImage(ctx context.Context, c *client.Client, t *testing.T) (float64, string) {
	images := callTool[[]map[string]any](ctx, c, t, "image-list", map[string]any{"Type": "distribution"})

	for _, img := range images {
		if slug, ok := img["slug"].(string); ok && slug == "ubuntu-22-04-x64" {
			return img["id"].(float64), slug
		}
	}
	require.NotEmpty(t, images, "No images found")

	firstID := images[0]["id"].(float64)
	firstSlug, _ := images[0]["slug"].(string)
	return firstID, firstSlug
}

func selectRegion(ctx context.Context, c *client.Client, t *testing.T) string {
	if rg := os.Getenv("TEST_REGION"); rg != "" {
		return rg
	}

	regions := callTool[[]map[string]any](ctx, c, t, "region-list", map[string]any{"Page": 1, "PerPage": 100})

	for _, r := range regions {
		slug, _ := r["slug"].(string)
		avail, _ := r["available"].(bool)
		if slug != "" && avail {
			return slug
		}
	}
	t.Fatal("No available region found")
	return ""
}

// --- Wait Wrappers ---

func WaitForDropletActive(ctx context.Context, _ *client.Client, t *testing.T, dropletID int, timeout time.Duration) godo.Droplet {
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)
	d, err := testhelpers.WaitForDroplet(ctx, gclient, dropletID, testhelpers.IsDropletActive, 3*time.Second, timeout)
	require.NoError(t, err, "WaitForDropletActive failed")
	return *d
}

func WaitForImageAvailable(ctx context.Context, _ *client.Client, t *testing.T, imageID int, timeout time.Duration) godo.Image {
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)
	img, err := testhelpers.WaitForImage(ctx, gclient, imageID, testhelpers.IsImageAvailable, 3*time.Second, timeout)
	require.NoError(t, err, "WaitForImageAvailable failed")
	return *img
}

func WaitForActionComplete(ctx context.Context, c *client.Client, t *testing.T, dropletID int, actionID int, timeout time.Duration) godo.Action {
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	// Verify tool works
	act := callTool[godo.Action](ctx, c, t, "droplet-action", map[string]any{
		"DropletID": float64(dropletID),
		"ActionID":  float64(actionID),
	})
	require.Equal(t, actionID, act.ID)

	final, err := testhelpers.WaitForAction(ctx, gclient, dropletID, actionID, 2*time.Second, timeout)
	require.NoError(t, err, "WaitForActionComplete failed")
	return *final
}

func WaitForImageActionComplete(ctx context.Context, c *client.Client, t *testing.T, imageID int, actionID int, timeout time.Duration) godo.Action {
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	// Verify tool works
	act := callTool[godo.Action](ctx, c, t, "image-action-get", map[string]any{
		"ImageID":  float64(imageID),
		"ActionID": float64(actionID),
	})
	require.Equal(t, actionID, act.ID)

	final, err := testhelpers.WaitForImageAction(ctx, gclient, imageID, actionID, 2*time.Second, timeout)
	require.NoError(t, err, "WaitForImageActionComplete failed")
	return *final
}

func waitForDbaasClusterActive(ctx context.Context, c *client.Client, t *testing.T, clusterID string, timeout time.Duration) (godo.Database, error) {
	var result godo.Database

	require.Eventually(t, func() bool {
		// Call MCP tool
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "db-cluster-get",
				Arguments: map[string]interface{}{
					"id": clusterID,
				},
			},
		})

		if err != nil || resp.IsError {
			// Keep retrying — do NOT fail inside Eventually
			return false
		}

		var db godo.Database
		dbJSON := resp.Content[0].(mcp.TextContent).Text
		if json.Unmarshal([]byte(dbJSON), &db) != nil {
			return false
		}

		// Log only when status changes
		if result.Status != db.Status {
			t.Logf("Cluster %s status changed: %s → %s", clusterID, result.Status, db.Status)
		}

		result = db
		return db.Status == dbaasClusterStatusOnline

	}, timeout, 2*time.Second, "cluster did not become active in time")

	return result, nil
}

// --- Cleanup & Logging ---

func deferCleanupDroplet(ctx context.Context, c *client.Client, t *testing.T, dropletID int) func() {
	return func() {
		t.Logf("Cleaning up droplet %d...", dropletID)
		DeleteResource(ctx, c, t, "droplet", float64(dropletID))
	}
}

func deferCleanupImage(ctx context.Context, c *client.Client, t *testing.T, imageID float64) func() {
	return func() {
		t.Logf("Cleaning up snapshot image %.0f...", imageID)
		DeleteResource(ctx, c, t, "image", imageID)
	}
}

func LogResourceCreated(t *testing.T, resourceType string, id any, name, status, region string) {
	t.Logf("[Created] %s %s: Name=%s, Status=%s, Region=%s", resourceType, formatID(id), name, status, region)
}

func LogResourceDeleted(t *testing.T, resourceType string, id any, err error, resp *mcp.CallToolResult) {
	if err != nil {
		t.Logf("[Delete] Failed %s %s: %v", resourceType, formatID(id), err)
		return
	}

	if resp != nil && resp.IsError {
		var errorMsg string
		if len(resp.Content) > 0 {
			if tc, ok := resp.Content[0].(mcp.TextContent); ok {
				errorMsg = tc.Text
			}
		}
		if errorMsg == "" {
			errorMsg = "unknown error"
		}

		// Check for common "already deleted" scenarios:
		// 1. 404 Not Found
		// 2. 422 Unprocessable Entity with "already deleted" message (DigitalOcean specific for images)
		lowerMsg := strings.ToLower(errorMsg)
		isAlreadyDeleted := strings.Contains(errorMsg, "404") ||
			strings.Contains(lowerMsg, "not found") ||
			(strings.Contains(errorMsg, "422") && strings.Contains(lowerMsg, "already deleted"))

		if isAlreadyDeleted {
			t.Logf("[Delete] Success %s %s (already deleted)", resourceType, formatID(id))
		} else {
			t.Logf("[Delete] Failed %s %s: %s", resourceType, formatID(id), errorMsg)
		}
		return
	}

	t.Logf("[Delete] Success %s %s", resourceType, formatID(id))
}

// LogActionStatus logs the action state.
func LogActionStatus(t *testing.T, context string, action godo.Action) {
	t.Logf("[Action] %s: ID=%d, Type=%s, Status=%s", context, action.ID, action.Type, action.Status)
}

func formatID(id any) string {
	switch v := id.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v)
	case float32:
		return fmt.Sprintf("%.0f", v)
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
