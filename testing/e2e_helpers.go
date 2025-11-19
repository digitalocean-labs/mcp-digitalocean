//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func verboseDump(t *testing.T, prefix string, v interface{}) {
	if os.Getenv("E2E_VERBOSE") == "true" {
		t.Logf("%s: %+v", prefix, v)
	} else {
		t.Logf("%s: (suppressed). Set E2E_VERBOSE=true for details", prefix)
	}
}

func formatID(id interface{}) string {
	switch v := id.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v)
	case float32:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

const (
	defaultTimeout = 2 * time.Minute
	pollInterval   = 5 * time.Second
)

// callToolJSON calls an MCP tool and unmarshals its text content into out.
func callToolJSON(ctx context.Context, c *client.Client, t *testing.T, name string, args map[string]interface{}, out interface{}) *mcp.CallToolResult {
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	require.NoError(t, err)

	// Provide helpful logging on error responses, then assert to fail the test consistently.
	if resp.IsError {
		if len(resp.Content) > 0 {
			if tc, ok := resp.Content[0].(mcp.TextContent); ok {
				t.Logf("%s error text: %s", name, tc.Text)
			} else {
				verboseDump(t, name+" error content", resp.Content)
				t.Logf("%s raw content: %+v", name, resp.Content)
			}
		} else {
			t.Logf("%s returned error: %v", name, resp)
		}
		require.False(t, resp.IsError, "%s returned error", name)
	}

	require.NotEmpty(t, resp.Content, "%s returned empty content", name)

	tc, ok := resp.Content[0].(mcp.TextContent)
	require.True(t, ok, "unexpected content type for %s", name)

	data := tc.Text
	require.NoError(t, json.Unmarshal([]byte(data), out), "failed to unmarshal %s response", name)
	return resp
}

// callToolUnmarshal calls an MCP tool and unmarshals its text content into a typed value.
func callToolUnmarshal[T any](ctx context.Context, c *client.Client, t *testing.T, name string, args map[string]interface{}) (T, *mcp.CallToolResult, error) {
	var zero T
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		return zero, resp, err
	}
	if resp.IsError {
		return zero, resp, fmt.Errorf("%s returned error: %+v", name, resp)
	}
	if len(resp.Content) == 0 {
		return zero, resp, fmt.Errorf("%s returned empty content", name)
	}
	tc, ok := resp.Content[0].(mcp.TextContent)
	if !ok {
		return zero, resp, fmt.Errorf("%s returned unexpected content type", name)
	}
	var v T
	if err := json.Unmarshal([]byte(tc.Text), &v); err != nil {
		return zero, resp, err
	}
	return v, resp, nil
}

// deferCleanupDroplet returns a closure suitable for deferring droplet cleanup in tests.
func deferCleanupDroplet(ctx context.Context, c *client.Client, t *testing.T, dropletID int) func() {
	return func() {
		resources := ListResources(ctx, c, t, "droplet", "before deletion", 1, 50)
		LogResourceList(t, "droplet", "before deletion", resources)
		DeleteResource(ctx, c, t, "droplet", float64(dropletID))
	}
}

func getSSHKeys(ctx context.Context, c *client.Client, t *testing.T) []interface{} {
	keys, _, err := callToolUnmarshal[[]map[string]interface{}](ctx, c, t, "key-list", map[string]interface{}{})
	require.NoError(t, err)
	require.NotEmpty(t, keys, "No SSH keys found in account. Please add at least one SSH key.")

	var keyIDs []interface{}
	for _, key := range keys {
		if id, ok := key["id"].(float64); ok {
			keyIDs = append(keyIDs, id)
		}
	}
	require.NotEmpty(t, keyIDs, "No valid SSH key IDs found")

	t.Logf("Found %d SSH key(s) in account", len(keyIDs))
	return keyIDs
}

func getTestImage(ctx context.Context, c *client.Client, t *testing.T) float64 {
	images, _, err := callToolUnmarshal[[]map[string]interface{}](ctx, c, t, "image-list", map[string]interface{}{"Type": "distribution"})
	require.NoError(t, err)
	require.NotEmpty(t, images, "No images found")

	for _, img := range images {
		if slug, ok := img["slug"].(string); ok {
			if slug == "ubuntu-22-04-x64" {
				imageID := img["id"].(float64)
				t.Logf("Using Ubuntu 22.04 LTS image (ID: %.0f)", imageID)
				return imageID
			}
		}
	}

	imageID := images[0]["id"].(float64)
	t.Logf("Using fallback image (ID: %.0f)", imageID)
	return imageID
}

func selectRegion(ctx context.Context, c *client.Client, t *testing.T) string {
	if rg := os.Getenv("TEST_REGION"); rg != "" {
		t.Logf("Using TEST_REGION from env: %s", rg)
		return rg
	}

	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "region-list",
			Arguments: map[string]interface{}{"Page": 1, "PerPage": 100},
		},
	})
	require.NoError(t, err)
	if resp.IsError {
		if len(resp.Content) > 0 {
			t.Fatalf("region-list failed: %+v", resp.Content)
		}
		t.Fatalf("region-list failed: %+v", resp)
	}

	if len(resp.Content) == 0 {
		t.Fatalf("region-list returned empty content")
	}

	var regions []map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &regions); err != nil {
		t.Fatalf("failed to parse region-list response: %v", err)
	}

	for _, r := range regions {
		if slug, ok := r["slug"].(string); ok && slug != "" {
			if avail, ok := r["available"].(bool); ok {
				if avail {
					t.Logf("Selected region: %s (available=true)", slug)
					return slug
				}
				continue
			}
			t.Logf("Selected region: %s", slug)
			return slug
		}
	}

	t.Fatalf("no suitable region found from region-list")
	return ""
}

func CreateTestDropletWithImage(ctx context.Context, c *client.Client, t *testing.T, namePrefix string, imageID float64) godo.Droplet {
	sshKeys := getSSHKeys(ctx, c, t)
	region := selectRegion(ctx, c, t)

	dropletName := fmt.Sprintf("%s-%d", namePrefix, time.Now().Unix())
	createResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "droplet-create",
			Arguments: map[string]interface{}{
				"Name":       dropletName,
				"Size":       "s-1vcpu-1gb",
				"ImageID":    imageID,
				"Region":     region,
				"Backup":     false,
				"Monitoring": true,
				"SSHKeys":    sshKeys,
			},
		},
	})
	require.NoError(t, err)
	if createResp.IsError {
		if len(createResp.Content) > 0 {
			t.Fatalf("droplet-create returned error. Content: %+v", createResp.Content)
		}
		t.Fatalf("droplet-create returned error: %+v", createResp)
	}

	if len(createResp.Content) == 0 {
		t.Fatalf("droplet-create returned empty content")
	}

	var droplet godo.Droplet
	if err := json.Unmarshal([]byte(createResp.Content[0].(mcp.TextContent).Text), &droplet); err != nil {
		t.Fatalf("failed to unmarshal droplet-create response: %v", err)
	}

	refreshed := WaitForDropletActive(ctx, c, t, droplet.ID, 2*time.Minute)
	return refreshed
}

func CreateTestDroplet(ctx context.Context, c *client.Client, t *testing.T, namePrefix string) godo.Droplet {
	imageID := getTestImage(ctx, c, t)
	return CreateTestDropletWithImage(ctx, c, t, namePrefix, imageID)
}

func WaitForDropletActive(ctx context.Context, c *client.Client, t *testing.T, dropletID int, timeout time.Duration) godo.Droplet {
	poll := func() (godo.Droplet, string, error) {
		d, _, err := callToolUnmarshal[godo.Droplet](ctx, c, t, "droplet-get", map[string]interface{}{"ID": float64(dropletID)})
		if err != nil {
			return godo.Droplet{}, "", err
		}
		return d, d.Status, nil
	}

	done := func(d godo.Droplet, status string) bool { return status == "active" }
	return WaitForResult(ctx, t, poll, done, timeout)
}

func WaitForDropletActiveDefault(ctx context.Context, c *client.Client, t *testing.T, dropletID int) godo.Droplet {
	return WaitForDropletActive(ctx, c, t, dropletID, 2*time.Minute)
}

func LogResourceCreated(t *testing.T, resourceType string, id interface{}, name, status, region string) {
	t.Logf("Created %s: ID=%s, Name=%s, Status=%s, Region=%s", resourceType, formatID(id), name, status, region)
}

func LogResourceList(t *testing.T, resourceType, context string, resources []map[string]interface{}) {
	for _, r := range resources {
		t.Logf("%s %s: ID=%s, Status=%v, Name=%v", context, resourceType, formatID(r["id"]), r["status"], r["name"])
	}
	t.Logf("Found %d %s %s", len(resources), resourceType, context)
}

func LogResourceDeleted(t *testing.T, resourceType string, id interface{}, err error, resp *mcp.CallToolResult) {
	if err != nil {
		t.Logf("Failed to delete %s %s: %v", resourceType, formatID(id), err)
	} else if resp != nil && resp.IsError {
		t.Logf("%s-delete returned error for %s %s: %+v", resourceType, resourceType, formatID(id), resp.Content)
	} else {
		t.Logf("%s %s deleted successfully", resourceType, formatID(id))
	}
}

func LogActionCompleted(t *testing.T, actionType string, action godo.Action) {
	t.Logf("%s completed: ActionID=%d, Status=%s, ResourceID=%s", actionType, action.ID, action.Status, formatID(action.ResourceID))
}

func DeleteResource(ctx context.Context, c *client.Client, t *testing.T, resourceType string, id interface{}) *mcp.CallToolResult {
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      fmt.Sprintf("%s-delete", resourceType),
			Arguments: map[string]interface{}{"ID": id, "ImageID": id},
		},
	})
	LogResourceDeleted(t, resourceType, id, err, resp)
	return resp
}

func deferCleanupImage(ctx context.Context, c *client.Client, t *testing.T, imageID float64) func() {
	return func() {
		// Use existing DeleteResource which maps to the "<resource>-delete" tool (snapshot-delete)
		DeleteResource(ctx, c, t, "snapshot", imageID)
	}
}

func ListResources(ctx context.Context, c *client.Client, t *testing.T, resourceType, context string, page, perPage int) []map[string]interface{} {
	var resources []map[string]interface{}
	name := fmt.Sprintf("%s-list", resourceType)
	v, _, err := callToolUnmarshal[[]map[string]interface{}](ctx, c, t, name, map[string]interface{}{"Page": page, "PerPage": perPage})
	require.NoError(t, err)
	resources = v
	return resources
}

func WaitForActionComplete(ctx context.Context, c *client.Client, t *testing.T, actionID int, timeout time.Duration) godo.Action {
	poll := func() (godo.Action, string, error) {
		a, _, err := callToolUnmarshal[godo.Action](ctx, c, t, "action-get", map[string]interface{}{"ID": float64(actionID)})
		if err != nil {
			return godo.Action{}, "", err
		}
		return a, a.Status, nil
	}

	done := func(a godo.Action, status string) bool { return status == "completed" }
	return WaitForResult(ctx, t, poll, done, timeout)
}

// WaitForResult is a generic polling helper used by resource-specific wait functions.
func WaitForResult[T any](ctx context.Context, t *testing.T, poll func() (T, string, error), done func(T, string) bool, timeout time.Duration) T {
	var zero T
	deadline := time.Now().Add(timeout)
	var lastStatus string

	for time.Now().Before(deadline) {
		val, status, err := poll()
		if err != nil {
			t.Logf("poll error: %v", err)
			time.Sleep(pollInterval)
			continue
		}

		if status != lastStatus {
			if lastStatus == "" {
				t.Logf("initial status: %s", status)
			} else {
				t.Logf("status changed: %s -> %s", lastStatus, status)
			}
			lastStatus = status
		}

		if done(val, status) {
			return val
		}

		time.Sleep(pollInterval)
	}

	t.Fatalf("timed out waiting for condition after %s", timeout)
	return zero
}
