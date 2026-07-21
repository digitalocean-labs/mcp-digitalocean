//go:build integration

package testing

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// listModelRoutersResponse matches GET /v2/gen-ai/models/routers JSON.
type listModelRoutersResponse struct {
	Meta struct {
		Page  int `json:"page"`
		Pages int `json:"pages"`
		Total int `json:"total"`
	} `json:"meta"`
	ModelRouters []struct {
		UUID    string   `json:"uuid"`
		Name    string   `json:"name"`
		Regions []string `json:"regions"`
	} `json:"model_routers"`
}

// getModelRouterResponse matches GET /v2/gen-ai/models/routers/{uuid} JSON.
type getModelRouterResponse struct {
	ModelRouter struct {
		UUID      string   `json:"uuid"`
		Name      string   `json:"name"`
		Regions   []string `json:"regions"`
		CreatedAt string   `json:"created_at"`
		UpdatedAt string   `json:"updated_at"`
		Config    struct {
			FallbackModels []string `json:"fallback_models"`
			Policies       []any    `json:"policies"`
		} `json:"config"`
	} `json:"model_router"`
}

// deleteModelRouterResponse matches DELETE /v2/gen-ai/models/routers/{uuid} JSON.
type deleteModelRouterResponse struct {
	UUID string `json:"uuid"`
}

func inferenceRouterMCPErrorText(resp *mcp.CallToolResult) string {
	if resp == nil || len(resp.Content) == 0 {
		return ""
	}
	if tc, ok := resp.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return fmt.Sprintf("%v", resp.Content)
}

// callToolInferenceRouterDelete is like callTool for genai-inference-router-delete but explains 403 (token scope).
func callToolInferenceRouterDelete(t *testing.T, routerUUID string) deleteModelRouterResponse {
	t.Helper()
	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "genai-inference-router-delete",
			Arguments: map[string]interface{}{"UUID": routerUUID},
		},
	})
	require.NoError(t, err)
	if resp.IsError {
		errText := inferenceRouterMCPErrorText(resp)
		low := strings.ToLower(errText)
		if strings.Contains(errText, "403") || strings.Contains(low, "not authorized") {
			t.Fatalf("genai-inference-router-delete returned 403 for uuid=%s: DIGITALOCEAN_API_TOKEN cannot delete inference routers (read-only or insufficient scope). If the test created this router, remove it under INFERENCE > Inference Router in the control panel, or use a PAT with write access. API: %s", routerUUID, errText)
		}
		t.Fatalf("genai-inference-router-delete failed: %s", errText)
	}
	require.NotEmpty(t, resp.Content, "empty delete response")
	tc, ok := resp.Content[0].(mcp.TextContent)
	require.True(t, ok, "unexpected delete response type")
	var out deleteModelRouterResponse
	require.NoError(t, json.Unmarshal([]byte(tc.Text), &out))
	return out
}

// deleteInferenceRouter logs cleanup failures; used when a test aborts before explicit delete.
func deleteInferenceRouter(t *testing.T, routerUUID string) {
	t.Helper()
	if routerUUID == "" {
		return
	}
	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "genai-inference-router-delete",
			Arguments: map[string]interface{}{"UUID": routerUUID},
		},
	})
	if err != nil {
		t.Logf("inference router cleanup delete failed: %v", err)
		return
	}
	if resp != nil && resp.IsError {
		msg := inferenceRouterMCPErrorText(resp)
		t.Logf("inference router cleanup delete returned error: %s", msg)
		if strings.Contains(msg, "403") || strings.Contains(strings.ToLower(msg), "not authorized") {
			t.Logf("cleanup hint: token may lack delete permission; delete router %s manually (INFERENCE > Inference Router)", routerUUID)
		}
	}
}

// TestInferenceRouterList exercises genai-inference-router-list against the live API.
func TestInferenceRouterList(t *testing.T) {
	t.Log("listing model routers (first page)...")
	out := callTool[listModelRoutersResponse](t, "genai-inference-router-list", map[string]interface{}{
		"Page":    float64(1),
		"PerPage": float64(20),
	})

	require.GreaterOrEqual(t, out.Meta.Total, 0, "meta.total should be present")
	require.Equal(t, 1, out.Meta.Page)
	t.Logf("total routers: %d, pages: %d, this page count: %d", out.Meta.Total, out.Meta.Pages, len(out.ModelRouters))

	if out.Meta.Total > 0 {
		require.NotEmpty(t, out.ModelRouters, "non-zero total should return routers on page 1")
		for i, r := range out.ModelRouters {
			require.NotEmpty(t, r.UUID, "router %d missing uuid", i)
			require.NotEmpty(t, r.Name, "router %d missing name", i)
		}
	}
}

// listTaskPresetsResponse matches GET /v2/gen-ai/models/routers/tasks/presets JSON.
type listTaskPresetsResponse struct {
	Meta struct {
		Page  int `json:"page"`
		Pages int `json:"pages"`
		Total int `json:"total"`
	} `json:"meta"`
	Tasks []struct {
		TaskSlug string `json:"task_slug"`
		Name     string `json:"name"`
	} `json:"tasks"`
}

// TestInferenceRouterListTaskPresets exercises genai-inference-router-task-presets against the live API.
func TestInferenceRouterListTaskPresets(t *testing.T) {
	t.Log("listing inference router task presets (first page)...")
	out := callTool[listTaskPresetsResponse](t, "genai-inference-router-task-presets", map[string]interface{}{
		"Page":    float64(1),
		"PerPage": float64(50),
	})
	require.GreaterOrEqual(t, out.Meta.Total, 0, "meta.total should be present")
	require.Equal(t, 1, out.Meta.Page)
	t.Logf("total presets: %d, this page: %d", out.Meta.Total, len(out.Tasks))
	if out.Meta.Total > 0 {
		require.NotEmpty(t, out.Tasks, "non-zero total should return tasks on page 1")
		for i, task := range out.Tasks {
			require.NotEmpty(t, task.TaskSlug, "task %d missing task_slug", i)
			require.NotEmpty(t, task.Name, "task %d missing name", i)
		}
	}
}

// TestInferenceRouterListPagination checks page and per_page are honored when multiple pages exist.
func TestInferenceRouterListPagination(t *testing.T) {
	first := callTool[listModelRoutersResponse](t, "genai-inference-router-list", map[string]interface{}{
		"Page":    float64(1),
		"PerPage": float64(2),
	})
	if first.Meta.Total <= 2 {
		t.Skip("need more than 2 routers to assert pagination; skipping")
	}

	second := callTool[listModelRoutersResponse](t, "genai-inference-router-list", map[string]interface{}{
		"Page":    float64(2),
		"PerPage": float64(2),
	})

	require.Equal(t, 2, second.Meta.Page)
	require.NotEmpty(t, second.ModelRouters)
	// Page 1 and 2 should not return the same primary UUID when totals exceed page size.
	require.NotEqual(t, first.ModelRouters[0].UUID, second.ModelRouters[0].UUID)
}

// TestInferenceRouterGet fetches one router by UUID from list results.
func TestInferenceRouterGet(t *testing.T) {
	listOut := callTool[listModelRoutersResponse](t, "genai-inference-router-list", map[string]interface{}{
		"Page":    float64(1),
		"PerPage": float64(10),
	})
	if len(listOut.ModelRouters) == 0 {
		t.Skip("no model routers in account; skipping get test")
	}

	uuid := listOut.ModelRouters[0].UUID
	name := listOut.ModelRouters[0].Name
	t.Logf("getting model router %s (%s)...", uuid, name)

	got := callTool[getModelRouterResponse](t, "genai-inference-router-get", map[string]interface{}{
		"UUID": uuid,
	})

	require.Equal(t, uuid, got.ModelRouter.UUID)
	require.Equal(t, name, got.ModelRouter.Name)
	require.NotEmpty(t, got.ModelRouter.CreatedAt)
	t.Logf("config: %d policies, %d fallback models",
		len(got.ModelRouter.Config.Policies), len(got.ModelRouter.Config.FallbackModels))
}

// TestInferenceRouterGetNotFound expects an error result for an unknown UUID.
func TestInferenceRouterGetNotFound(t *testing.T) {
	ctx, c := getTestClient(t)
	fakeUUID := "00000000-0000-0000-0000-000000000001"
	t.Logf("getting non-existent router %s...", fakeUUID)

	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "genai-inference-router-get",
			Arguments: map[string]interface{}{"UUID": fakeUUID},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError, "expected error for missing router")

	if len(resp.Content) > 0 {
		if tc, ok := resp.Content[0].(mcp.TextContent); ok {
			t.Logf("error text: %s", tc.Text)
		}
	}
}

// TestInferenceRouterWorkflow runs list -> get for a few routers (bounded API usage).
func TestInferenceRouterWorkflow(t *testing.T) {
	t.Log("workflow: list -> get for up to 3 routers")
	listOut := callTool[listModelRoutersResponse](t, "genai-inference-router-list", map[string]interface{}{
		"Page":    float64(1),
		"PerPage": float64(25),
	})
	if len(listOut.ModelRouters) == 0 {
		t.Skip("no model routers; skipping workflow")
	}

	limit := 3
	for i, r := range listOut.ModelRouters {
		if i >= limit {
			t.Logf("stopping after %d gets", limit)
			break
		}
		detail := callTool[getModelRouterResponse](t, "genai-inference-router-get", map[string]interface{}{
			"UUID": r.UUID,
		})
		require.Equal(t, r.UUID, detail.ModelRouter.UUID)
		require.Equal(t, r.Name, detail.ModelRouter.Name)
		t.Logf("  ok: %s", r.Name)
	}
}

// TestInferenceRouterCreateDeleteLifecycle creates a router, verifies get, deletes, and expects get to fail.
func TestInferenceRouterCreateDeleteLifecycle(t *testing.T) {
	routerName := fmt.Sprintf("mcp-e2e-infer-router-%s", uuid.New().String())

	var routerUUID string
	t.Cleanup(func() {
		if routerUUID != "" {
			deleteInferenceRouter(t, routerUUID)
		}
	})

	// Use custom_task so create does not depend on built-in task_slug catalog (avoids "task slug not found" in some environments).
	policies := `[{"custom_task":{"name":"MCP E2E","description":"Integration test policy for inference router lifecycle."},"models":["openai-gpt-oss-120b"],"selection_policy":{"prefer":"fastest"}}]`
	t.Logf("creating model router %s...", routerName)
	created := callTool[getModelRouterResponse](t, "genai-inference-router-create", map[string]interface{}{
		"Name":           routerName,
		"PoliciesJson":   policies,
		"FallbackModels": []interface{}{"openai-gpt-oss-120b"},
	})

	routerUUID = created.ModelRouter.UUID
	require.NotEmpty(t, routerUUID)
	require.Equal(t, routerName, created.ModelRouter.Name)
	require.NotEmpty(t, created.ModelRouter.Config.Policies, "create should persist policies")

	t.Logf("get by uuid %s...", routerUUID)
	detail := callTool[getModelRouterResponse](t, "genai-inference-router-get", map[string]interface{}{
		"UUID": routerUUID,
	})
	require.Equal(t, routerUUID, detail.ModelRouter.UUID)
	require.Equal(t, routerName, detail.ModelRouter.Name)

	t.Log("deleting router...")
	del := callToolInferenceRouterDelete(t, routerUUID)
	require.Equal(t, routerUUID, del.UUID)
	routerUUID = ""

	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "genai-inference-router-get",
			Arguments: map[string]interface{}{"UUID": del.UUID},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError, "get after delete should fail")
}

// TestInferenceRouterDeleteNotFound expects an error when deleting a non-existent router.
func TestInferenceRouterDeleteNotFound(t *testing.T) {
	ctx, c := getTestClient(t)
	fakeUUID := "00000000-0000-0000-0000-000000000099"
	t.Logf("deleting non-existent router %s...", fakeUUID)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "genai-inference-router-delete",
			Arguments: map[string]interface{}{"UUID": fakeUUID},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError, "delete of missing router should error")
}
