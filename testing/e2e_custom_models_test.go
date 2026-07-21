//go:build integration

package testing

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

const e2eHFRepoID = "Qwen/Qwen2.5-0.5B"

// TestCustomModelsListModels calls genai-custom-models-list against the live GenAI API.
// Unfiltered list returns one custom-models table with every UUID (including STATUS_FAILED).
func TestCustomModelsListModels(t *testing.T) {
	t.Parallel()

	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "genai-custom-models-list", Arguments: map[string]any{}},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	text := callToolResultText(resp)
	require.Contains(t, text, "## Custom Models")
	require.Contains(t, text, "| UUID | Name | Source | Status | Architecture |")
	require.Contains(t, text, "STATUS_FAILED")
	require.NotContains(t, text, "## Model Catalog")
	require.Greater(t, strings.Count(text, "STATUS_FAILED"), 1, "failed models should appear as table rows, not only in a summary")
	t.Logf("unfiltered custom list returned table (%d bytes)", len(text))
}

// TestCustomModelsListModelsWithPagination tests pagination on custom models list.
func TestCustomModelsListModelsWithPagination(t *testing.T) {
	t.Parallel()

	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "genai-custom-models-list",
			Arguments: map[string]any{
				"page":     float64(1),
				"per_page": float64(5),
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	text := callToolResultText(resp)
	require.Contains(t, text, "## Custom Models")
	require.Contains(t, text, "| UUID | Name | Source | Status | Architecture |")
	require.NotContains(t, text, "## Model Catalog")
	t.Logf("paginated list returned custom table (%d bytes)", len(text))
}

// TestCustomModelsListModelsWithStatusFilter tests status filtering on custom models list.
func TestCustomModelsListModelsWithStatusFilter(t *testing.T) {
	t.Parallel()

	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "genai-custom-models-list",
			Arguments: map[string]any{"status": "STATUS_READY"},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	text := callToolResultText(resp)
	require.Contains(t, text, "## Custom Models")
	require.Contains(t, text, "STATUS_READY")
	require.NotContains(t, text, "## Model Catalog")
	t.Logf("status-filtered list returned custom table (%d bytes)", len(text))
}

// TestCustomModelsImportHuggingFaceResolvesCommitSHA imports without commit_sha and
// verifies the MCP server resolved it from Hugging Face before calling DigitalOcean.
func TestCustomModelsImportHuggingFaceResolvesCommitSHA(t *testing.T) {
	modelName := fmt.Sprintf("it-hf-commit-%d", time.Now().Unix())

	type sourceRef struct {
		RepoID    string `json:"repo_id"`
		CommitSHA string `json:"commit_sha"`
	}
	type customModel struct {
		UUID      string    `json:"uuid"`
		Name      string    `json:"name"`
		Status    string    `json:"status"`
		SourceRef sourceRef `json:"source_ref"`
	}
	type importResponse struct {
		Model customModel `json:"model"`
	}

	ctx, c := getTestClient(t)
	importArgs := map[string]any{
		"name":        modelName,
		"source_type": "SOURCE_TYPE_HUGGINGFACE",
		"source_ref": map[string]any{
			"repo_id":     e2eHFRepoID,
			"access_type": "ACCESS_TYPE_PUBLIC",
		},
		"accept_terms_and_conditions": true,
	}

	const importRetryAttempts = 4
	var resp *mcp.CallToolResult
	var lastErrText string
	for attempt := 0; attempt < importRetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		var err error
		resp, err = c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{Name: "genai-custom-models-import", Arguments: importArgs},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		if !resp.IsError {
			break
		}
		lastErrText = callToolResultText(resp)
		if strings.Contains(lastErrText, "already imported") && strings.Contains(lastErrText, e2eHFRepoID) {
			require.Regexp(t, `[0-9a-f]{40}`, lastErrText, "error should reference resolved commit revision")
			t.Logf("revision already imported (commit_sha reached API): %s", lastErrText)
			return
		}
		if strings.Contains(lastErrText, "404") && strings.Contains(lastErrText, "not found") {
			t.Logf("attempt %d/%d: transient 404 on import: %s", attempt+1, importRetryAttempts, lastErrText)
			continue
		}
		t.Fatalf("Tool genai-custom-models-import failed: %s", lastErrText)
	}
	if resp != nil && resp.IsError {
		if strings.Contains(lastErrText, "404") {
			t.Skipf("GenAI import returned transient 404 after %d attempts: %s", importRetryAttempts, lastErrText)
		}
		t.Fatalf("Tool genai-custom-models-import failed: %s", lastErrText)
	}

	var out importResponse
	require.NoError(t, json.Unmarshal([]byte(callToolResultText(resp)), &out))
	require.NotEmpty(t, out.Model.UUID)
	require.Len(t, out.Model.SourceRef.CommitSHA, 40, "import should pin commit_sha from Hugging Face Hub")
	require.Regexp(t, `^[0-9a-f]{40}$`, out.Model.SourceRef.CommitSHA)
	require.Equal(t, e2eHFRepoID, out.Model.SourceRef.RepoID)
	t.Logf("import started model uuid=%s name=%s commit_sha=%s status=%s",
		out.Model.UUID, out.Model.Name, out.Model.SourceRef.CommitSHA, out.Model.Status)

	t.Cleanup(func() {
		callTool[map[string]any](t, "genai-custom-models-delete", map[string]any{
			"name":             out.Model.Name,
			"uuid":             out.Model.UUID,
			"confirm_deletion": true,
		})
	})
}

func callToolResultText(resp *mcp.CallToolResult) string {
	if resp == nil || len(resp.Content) == 0 {
		return ""
	}
	if tc, ok := resp.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return fmt.Sprintf("%v", resp.Content)
}

// TestCustomModelsImportHuggingFaceInvalidRepoFailsBeforeImport ensures an unknown
// Hugging Face repo fails during commit_sha resolution without starting an import job.
func TestCustomModelsImportHuggingFaceInvalidRepoFailsBeforeImport(t *testing.T) {
	t.Parallel()

	errText := callToolExpectError(t, "genai-custom-models-import", map[string]any{
		"name":        fmt.Sprintf("it-hf-invalid-%d", time.Now().Unix()),
		"source_type": "SOURCE_TYPE_HUGGINGFACE",
		"source_ref": map[string]any{
			"repo_id":     "this-org/does-not-exist-xyz-404",
			"access_type": "ACCESS_TYPE_PUBLIC",
		},
		"accept_terms_and_conditions": true,
	})

	require.True(t,
		strings.Contains(errText, "commit_sha") || strings.Contains(errText, "404"),
		"expected commit_sha resolution failure, got: %s", errText)
}

func callToolExpectError(t *testing.T, name string, args map[string]any) string {
	t.Helper()

	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	require.NoError(t, err)
	require.True(t, resp.IsError, "tool %s should return an error", name)

	if len(resp.Content) == 0 {
		return ""
	}
	if tc, ok := resp.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return fmt.Sprintf("%v", resp.Content)
}
