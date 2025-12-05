//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"mcp-digitalocean/internal/testhelpers"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

const (
	dbaasClusterStatusOnline = "online"

	// Configuration Defaults
	defaultDropletSize   = "s-1vcpu-1gb"
	defaultTestImageSlug = "ubuntu-22-04-x64"

	// Polling Intervals
	defaultPollInterval  = 2 * time.Second
	resourcePollInterval = 3 * time.Second

	// Timeouts
	defaultActionTimeout     = 5 * time.Minute
	dropletActiveTimeout     = 10 * time.Minute
	dropletDeleteTimeout     = 2 * time.Minute
	imageAvailableTimeout    = 5 * time.Minute
	snapshotDiscoveryTimeout = 2 * time.Minute
	renameVerifyTimeout      = 30 * time.Second
	ipv6AssignTimeout        = 1 * time.Minute
	backupsEnableTimeout     = 1 * time.Minute
	imageDeleteTimeout       = 1 * time.Minute
	restoreActionTimeout     = 2 * time.Minute
	rebuildActionTimeout     = 5 * time.Minute

	// Pagination
	defaultPerPage   = 50
	imageListPerPage = 20
	defaultPage      = 1

	// Test Regions
	testRegionNYC1 = "nyc1"
	testRegionNYC3 = "nyc3"
)

var testClients sync.Map // map[*testing.T]*client.Client

func getTestClient(t *testing.T) (context.Context, *client.Client) {
	t.Helper()

	if cached, ok := testClients.Load(t); ok {
		return context.Background(), cached.(*client.Client)
	}

	ctx := context.Background()
	c := initializeClient(ctx, t)

	testClients.Store(t, c)
	t.Cleanup(func() {
		c.Close()
		testClients.Delete(t)
	})

	return ctx, c
}

// Deprecated: Use getTestClient(t) in helper functions instead.
func setupTest(t *testing.T) (context.Context, *client.Client) {
	t.Helper()
	return getTestClient(t)
}

type genericActionWaiter func(ctx context.Context, client *godo.Client, resourceID, actionID int, interval, timeout time.Duration) (*godo.Action, error)

func triggerGenericActionAndWait(t *testing.T, tool string, args map[string]any, resourceID int, waiter genericActionWaiter, timeout time.Duration) {
	t.Helper()

	ctx := context.Background()
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	action := callTool[godo.Action](t, tool, args)
	require.NotZero(t, action.ID, "Action ID should not be zero")

	LogActionStatus(t, tool, action)

	final, err := waiter(ctx, gclient, resourceID, action.ID, defaultPollInterval, timeout)
	require.NoError(t, err, fmt.Sprintf("Action %s failed to complete", tool))

	LogActionStatus(t, tool, *final)
}

func triggerActionAndWait(t *testing.T, tool string, args map[string]any, resourceID int) {
	t.Helper()
	triggerGenericActionAndWait(t, tool, args, resourceID, testhelpers.WaitForAction, defaultActionTimeout)
}

func triggerImageActionAndWait(t *testing.T, tool string, args map[string]any, imageID int) {
	t.Helper()
	triggerGenericActionAndWait(t, tool, args, imageID, testhelpers.WaitForImageAction, defaultActionTimeout)
}

func callTool[T any](t *testing.T, name string, args map[string]any) T {
	t.Helper()

	ctx, c := getTestClient(t)

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

// testContext bundles context and client for cleanup helpers.
type testContext struct {
	ctx    context.Context
	client *client.Client
}

// requireFoundInList asserts that at least one item in the list matches the predicate.
func requireFoundInList[T any](t *testing.T, items []T, match func(T) bool, itemName string) {
	for _, item := range items {
		if match(item) {
			return
		}
	}
	t.Fatalf("%s not found in list", itemName)
}

// --- Resource Lifecycle Helpers ---
func CreateTestDroplet(t *testing.T, namePrefix string) godo.Droplet {
	t.Helper()

	sshKeys := getSSHKeys(t)
	region := selectRegion(t)
	imageID, imageSlug := getTestImage(t)

	dropletName := fmt.Sprintf("%s-%d", namePrefix, time.Now().Unix())

	t.Logf("Creating Droplet: %s (Image: %s [ID: %.0f], Region: %s)...", dropletName, imageSlug, imageID, region)

	droplet := callTool[godo.Droplet](t, "droplet-create", map[string]any{
		"Name":       dropletName,
		"Size":       defaultDropletSize,
		"ImageID":    imageID,
		"Region":     region,
		"Backup":     false,
		"Monitoring": true,
		"SSHKeys":    sshKeys,
	})

	RegisterResourceCleanup(t, "droplet", float64(droplet.ID))

	LogResourceCreated(t, "droplet", droplet.ID, droplet.Name, droplet.Status, region)

	activeDroplet := WaitForDropletActive(t, droplet.ID, dropletActiveTimeout)

	LogResourceCreated(t, "droplet", activeDroplet.ID, activeDroplet.Name, activeDroplet.Status, activeDroplet.Region.Slug)

	return activeDroplet
}

func CreateDropletSnapshot(t *testing.T, dropletID int, snapshotName string) int {
	t.Helper()

	t.Logf("Creating snapshot %s from droplet %d...", snapshotName, dropletID)

	triggerActionAndWait(t, "snapshot-droplet", map[string]any{
		"ID":   float64(dropletID),
		"Name": snapshotName,
	}, dropletID)

	updatedDroplet, err := WaitForDropletCondition(t, dropletID, func(d *godo.Droplet) bool {
		return len(d.SnapshotIDs) > 0
	}, resourcePollInterval, snapshotDiscoveryTimeout)
	require.NoError(t, err, "Failed to find created snapshot on droplet")
	require.NotEmpty(t, updatedDroplet.SnapshotIDs, "Droplet should have at least one snapshot")

	imageID := updatedDroplet.SnapshotIDs[len(updatedDroplet.SnapshotIDs)-1]

	img := WaitForImageAvailable(t, imageID, imageAvailableTimeout)

	RegisterResourceCleanup(t, "image", float64(img.ID))

	t.Logf("Created snapshot: %d (%s) from droplet %d", img.ID, img.Name, dropletID)

	return img.ID
}

func CreateTestSnapshotImage(t *testing.T, namePrefix string) godo.Image {
	t.Helper()

	droplet := CreateTestDroplet(t, namePrefix+"-setup")

	snapName := fmt.Sprintf("%s-snap-%d", namePrefix, time.Now().Unix())
	imageID := CreateDropletSnapshot(t, droplet.ID, snapName)

	img := WaitForImageAvailable(t, imageID, imageAvailableTimeout)

	t.Logf("Created test image: %d (%s)", img.ID, img.Name)

	return img
}

func DeleteResource(t *testing.T, resourceType string, id any) {
	t.Helper()

	ctx, c := getTestClient(t)

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

func deleteResourceWithGodo(t *testing.T, ctx context.Context, gclient *godo.Client, resourceType string, id any) {
	t.Helper()

	var err error
	var idInt int

	switch v := id.(type) {
	case int:
		idInt = v
	case float64:
		idInt = int(v)
	default:
		t.Logf("[Delete] Failed %s %s: invalid ID type %T", resourceType, formatID(id), id)
		return
	}

	switch resourceType {
	case "droplet":
		_, err = gclient.Droplets.Delete(ctx, idInt)
	case "image", "snapshot":
		_, err = gclient.Images.Delete(ctx, idInt)
	case "volume":
		_, err = gclient.Storage.DeleteVolume(ctx, fmt.Sprintf("%d", idInt))
	default:
		t.Logf("[Delete] Failed %s %s: unsupported resource type", resourceType, formatID(id))
		return
	}

	if err != nil {
		errMsg := err.Error()
		errMsgLower := strings.ToLower(errMsg)
		if strings.Contains(errMsg, "404") ||
			strings.Contains(errMsgLower, "not found") ||
			strings.Contains(errMsg, "422") && strings.Contains(errMsgLower, "already deleted") {
			t.Logf("[Delete] Success %s %s (already deleted)", resourceType, formatID(id))
			return
		}
		t.Logf("[Delete] Failed %s %s: %v", resourceType, formatID(id), err)
		return
	}

	t.Logf("[Delete] Success %s %s", resourceType, formatID(id))
}

func ListResources(t *testing.T, resourceType string, page, perPage int) []map[string]any {
	t.Helper()
	return callTool[[]map[string]any](t, fmt.Sprintf("%s-list", resourceType), map[string]any{
		"Page":    page,
		"PerPage": perPage,
	})
}

func createDbaasCluster(ctx context.Context, t *testing.T, c *client.Client, name string, engine string, version string, region string, size string, numNodes int) godo.Database {
	t.Helper()
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
	t.Helper()
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
	t.Helper()
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "db-cluster-list",
			Arguments: map[string]interface{}{
				"page":     defaultPage,
				"per_page": defaultPerPage,
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

func getSSHKeys(t *testing.T) []interface{} {
	t.Helper()
	keys := callTool[[]map[string]interface{}](t, "key-list", map[string]interface{}{})
	var keyIDs []interface{}
	for _, key := range keys {
		if id, ok := key["id"].(float64); ok {
			keyIDs = append(keyIDs, id)
		}
	}
	return keyIDs
}

func getTestImage(t *testing.T) (float64, string) {
	t.Helper()
	images := callTool[[]map[string]any](t, "image-list", map[string]any{"Type": "distribution"})

	for _, img := range images {
		if slug, ok := img["slug"].(string); ok && slug == defaultTestImageSlug {
			return img["id"].(float64), slug
		}
	}
	require.NotEmpty(t, images, "No images found")

	firstID := images[0]["id"].(float64)
	firstSlug, _ := images[0]["slug"].(string)
	return firstID, firstSlug
}

func selectRegion(t *testing.T) string {
	t.Helper()
	if rg := os.Getenv("TEST_REGION"); rg != "" {
		return rg
	}

	regions := callTool[[]map[string]any](t, "region-list", map[string]any{"Page": defaultPage, "PerPage": defaultPerPage})

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

func WaitForDropletActive(t *testing.T, dropletID int, timeout time.Duration) godo.Droplet {
	t.Helper()

	d, err := WaitForDropletCondition(t, dropletID, testhelpers.IsDropletActive, resourcePollInterval, timeout)
	require.NoError(t, err, "WaitForDropletActive failed")
	return *d
}

func WaitForImageAvailable(t *testing.T, imageID int, timeout time.Duration) godo.Image {
	t.Helper()

	img, err := WaitForImageCondition(t, imageID, testhelpers.IsImageAvailable, resourcePollInterval, timeout)
	require.NoError(t, err, "WaitForImageAvailable failed")
	return *img
}

func WaitForActionComplete(t *testing.T, dropletID int, actionID int, timeout time.Duration) godo.Action {
	t.Helper()

	ctx := context.Background()
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	act := callTool[godo.Action](t, "droplet-action", map[string]any{
		"DropletID": float64(dropletID),
		"ActionID":  float64(actionID),
	})
	require.Equal(t, actionID, act.ID)

	final, err := testhelpers.WaitForAction(ctx, gclient, dropletID, actionID, defaultPollInterval, timeout)
	require.NoError(t, err, "WaitForActionComplete failed")
	return *final
}

func WaitForImageActionComplete(t *testing.T, imageID int, actionID int, timeout time.Duration) godo.Action {
	t.Helper()

	ctx := context.Background()
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	act := callTool[godo.Action](t, "image-action-get", map[string]any{
		"ImageID":  float64(imageID),
		"ActionID": float64(actionID),
	})
	require.Equal(t, actionID, act.ID)

	final, err := testhelpers.WaitForImageAction(ctx, gclient, imageID, actionID, defaultPollInterval, timeout)
	require.NoError(t, err, "WaitForImageActionComplete failed")
	return *final
}

func WaitForDropletCondition(t *testing.T, dropletID int, condition func(*godo.Droplet) bool, interval, timeout time.Duration) (*godo.Droplet, error) {
	t.Helper()

	ctx := context.Background()
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	return testhelpers.WaitForDroplet(ctx, gclient, dropletID, condition, interval, timeout)
}

func WaitForImageCondition(t *testing.T, imageID int, condition func(*godo.Image) bool, interval, timeout time.Duration) (*godo.Image, error) {
	t.Helper()

	ctx := context.Background()
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	return testhelpers.WaitForImage(ctx, gclient, imageID, condition, interval, timeout)
}

func WaitForDropletDeletion(t *testing.T, dropletID int, interval, timeout time.Duration) error {
	t.Helper()

	ctx := context.Background()
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	return testhelpers.WaitForDropletDeleted(ctx, gclient, dropletID, interval, timeout)
}

func WaitForImageDeletion(t *testing.T, imageID int, interval, timeout time.Duration) error {
	t.Helper()

	ctx := context.Background()
	gclient, err := testhelpers.MustGodoClient(ctx, t.Name())
	require.NoError(t, err)

	return testhelpers.WaitForImageDeleted(ctx, gclient, imageID, interval, timeout)
}

func waitForDbaasClusterActive(ctx context.Context, c *client.Client, t *testing.T, clusterID string, timeout time.Duration) (godo.Database, error) {
	t.Helper()
	var result godo.Database

	require.Eventually(t, func() bool {
		resp, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "db-cluster-get",
				Arguments: map[string]interface{}{
					"id": clusterID,
				},
			},
		})

		if err != nil || resp.IsError {
			return false
		}

		var db godo.Database
		dbJSON := resp.Content[0].(mcp.TextContent).Text
		if json.Unmarshal([]byte(dbJSON), &db) != nil {
			return false
		}

		if result.Status != db.Status {
			t.Logf("Cluster %s status changed: %s → %s", clusterID, result.Status, db.Status)
		}

		result = db
		return db.Status == dbaasClusterStatusOnline
	}, timeout, defaultPollInterval, "cluster did not become active in time")

	return result, nil
}

func RegisterResourceCleanup(t *testing.T, resourceType string, resourceID interface{}) {
	t.Helper()

	t.Cleanup(func() {
		t.Logf("Cleaning up %s %s...", resourceType, formatID(resourceID))

		cleanupCtx, cancel := context.WithTimeout(context.Background(), dropletDeleteTimeout)
		defer cancel()

		gclient, err := testhelpers.MustGodoClient(cleanupCtx, t.Name())
		if err != nil {
			t.Logf("Failed to create godo client for cleanup: %v", err)
			return
		}

		deleteResourceWithGodo(t, cleanupCtx, gclient, resourceType, resourceID)
	})
}

func LogResourceCreated(t *testing.T, resourceType string, id any, name, status, region string) {
	t.Helper()
	t.Logf("[Created] %s %s: Name=%s, Status=%s, Region=%s", resourceType, formatID(id), name, status, region)
}

func LogResourceDeleted(t *testing.T, resourceType string, id any, err error, resp *mcp.CallToolResult) {
	t.Helper()
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

func LogActionStatus(t *testing.T, context string, action godo.Action) {
	t.Helper()
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
