//go:build integration

package testing

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func skipIfMicroDropletUnavailable(t *testing.T, resp *mcp.CallToolResult) {
	t.Helper()
	if resp == nil || !resp.IsError {
		return
	}
	text := callToolResultText(resp)
	lower := strings.ToLower(text)
	if strings.Contains(lower, "401") ||
		strings.Contains(lower, "403") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "not enabled") ||
		strings.Contains(lower, "flipper") {
		t.Skipf("microvms not available for this token: %s", text)
	}
}

func TestMicroDropletList(t *testing.T) {
	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "microvm-list",
			Arguments: map[string]any{
				"page":     float64(1),
				"per_page": float64(10),
			},
		},
	})
	require.NoError(t, err)
	skipIfMicroDropletUnavailable(t, resp)
	require.False(t, resp.IsError, callToolResultText(resp))

	text := callToolResultText(resp)
	require.NotEmpty(t, text)
	require.Contains(t, text, "microvms")

	var payload struct {
		MicroVMs []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"microvms"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	t.Logf("listed %d microvms", len(payload.MicroVMs))

	if len(payload.MicroVMs) == 0 {
		return
	}

	id := payload.MicroVMs[0].ID
	require.NotEmpty(t, id)

	getResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "microvm-get",
			Arguments: map[string]any{"id": id},
		},
	})
	require.NoError(t, err)
	skipIfMicroDropletUnavailable(t, getResp)
	require.False(t, getResp.IsError, callToolResultText(getResp))
	require.Contains(t, callToolResultText(getResp), id)
}

func TestMicroDropletCheckpointList(t *testing.T) {
	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "microvm-checkpoint-list",
			Arguments: map[string]any{
				"page":     float64(1),
				"per_page": float64(10),
			},
		},
	})
	require.NoError(t, err)
	skipIfMicroDropletUnavailable(t, resp)
	require.False(t, resp.IsError, callToolResultText(resp))

	text := callToolResultText(resp)
	require.NotEmpty(t, text)
	require.Contains(t, text, "checkpoints")
}

// TestMicroDropletCreateDelete exercises create→get→delete when
// MICROVM_E2E_OCI_REF is set (e.g. docker.io/library/nginx:1.27).
// Skipped by default so CI tokens without MicroVM create rights stay green.
func TestMicroDropletCreateDelete(t *testing.T) {
	ociRef := os.Getenv("MICROVM_E2E_OCI_REF")
	if ociRef == "" {
		t.Skip("MICROVM_E2E_OCI_REF not set")
	}

	region := os.Getenv("MICROVM_E2E_REGION")
	if region == "" {
		region = "nyc1"
	}

	ctx, c := getTestClient(t)
	name := fmt.Sprintf("mcp-e2e-mdrop-%d", time.Now().Unix())

	createResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "microvm-create",
			Arguments: map[string]any{
				"name":   name,
				"region": region,
				"size":   map[string]any{"cpu": float64(1), "memory": float64(1024)},
				"source": map[string]any{"oci_ref": ociRef},
				"tags":   []any{"mcp-e2e"},
			},
		},
	})
	require.NoError(t, err)
	skipIfMicroDropletUnavailable(t, createResp)
	require.False(t, createResp.IsError, callToolResultText(createResp))

	var created struct {
		MicroVM struct {
			ID string `json:"id"`
		} `json:"microvm"`
	}
	require.NoError(t, json.Unmarshal([]byte(callToolResultText(createResp)), &created))
	id := created.MicroVM.ID
	require.NotEmpty(t, id, "create response missing microvm.id")

	t.Cleanup(func() {
		delResp, delErr := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "microvm-delete",
				Arguments: map[string]any{"id": id},
			},
		})
		if delErr != nil {
			t.Logf("cleanup microvm-delete error: %v", delErr)
			return
		}
		if delResp.IsError {
			t.Logf("cleanup microvm-delete: %s", callToolResultText(delResp))
		}
	})

	getResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "microvm-get",
			Arguments: map[string]any{"id": id},
		},
	})
	require.NoError(t, err)
	require.False(t, getResp.IsError, callToolResultText(getResp))
	require.Contains(t, callToolResultText(getResp), id)
}
