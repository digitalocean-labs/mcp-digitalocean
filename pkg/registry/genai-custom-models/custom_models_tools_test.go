package genaicustommodels

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func setupCustomModelsToolWithFailingClient() *CustomModelsTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return nil, context.Canceled
	}
	return NewCustomModelsTool(client)
}

func TestCustomModelsTool_Tools(t *testing.T) {
	tool := NewCustomModelsTool(func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{}, nil
	})

	tools := tool.Tools()
	require.Len(t, tools, 6, "should have 6 custom models tools")

	expectedTools := map[string]bool{
		"genai-custom-models-list":            false,
		"genai-custom-models-import":          false,
		"genai-custom-models-update-metadata": false,
		"genai-custom-models-get":             false,
		"genai-custom-models-delete":          false,
		"genai-models-unified-search":         false,
	}

	for _, st := range tools {
		name := st.Tool.Name
		_, ok := expectedTools[name]
		require.True(t, ok, "unexpected tool name: %s", name)
		expectedTools[name] = true
	}

	for name, found := range expectedTools {
		require.True(t, found, "missing expected tool: %s", name)
	}
}

func TestCustomModelsTool_listModels_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	_, err := tool.listModels(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_importModel_validation(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing source_type", args: map[string]any{
			"name":       "my-model",
			"source_ref": map[string]interface{}{"repo_id": "test/model"},
		}},
		{name: "missing source_ref", args: map[string]any{
			"name":        "my-model",
			"source_type": "SOURCE_TYPE_HUGGINGFACE",
		}},
		{name: "missing accept_terms_and_conditions", args: map[string]any{
			"name":        "my-model",
			"source_type": "SOURCE_TYPE_HUGGINGFACE",
			"source_ref":  map[string]interface{}{"repo_id": "test/model"},
		}},
		{name: "accept_terms_and_conditions false", args: map[string]any{
			"name":                        "my-model",
			"source_type":                 "SOURCE_TYPE_HUGGINGFACE",
			"source_ref":                  map[string]interface{}{"repo_id": "test/model"},
			"accept_terms_and_conditions": false,
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.importModel(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestImportModel_missingConsentDoesNotResolveHuggingFace(t *testing.T) {
	oldFetch := fetchHuggingFaceCommitSHA
	hfCalled := false
	fetchHuggingFaceCommitSHA = func(ctx context.Context, repoID, hfToken string) (string, error) {
		hfCalled = true
		return "", errors.New("Hugging Face resolution should not run without consent")
	}
	t.Cleanup(func() { fetchHuggingFaceCommitSHA = oldFetch })

	tool := setupCustomModelsToolWithFailingClient()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"source_type": "SOURCE_TYPE_HUGGINGFACE",
		"source_ref":  map[string]interface{}{"repo_id": "test/model"},
	}}}
	resp, err := tool.importModel(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
	require.False(t, hfCalled)
}

func TestCustomModelsTool_importModel_spacesSourceRef(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"name":        "my-spaces-model",
		"source_type": "SOURCE_TYPE_SPACES_BUCKET",
		"source_ref": map[string]interface{}{
			"bucket": "my-bucket",
			"region": "nyc3",
			"prefix": "models/mistral/",
		},
		"accept_terms_and_conditions": true,
	}}}
	_, err := tool.importModel(context.Background(), req)
	require.Error(t, err, "should fail due to client error, not validation")
}

func TestCustomModelsTool_importModel_clientError(t *testing.T) {
	oldFetch := fetchHuggingFaceCommitSHA
	fetchHuggingFaceCommitSHA = func(ctx context.Context, repoID, hfToken string) (string, error) {
		return "abc123def456", nil
	}
	t.Cleanup(func() { fetchHuggingFaceCommitSHA = oldFetch })

	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"name":                        "my-model",
		"source_type":                 "SOURCE_TYPE_HUGGINGFACE",
		"source_ref":                  map[string]interface{}{"repo_id": "test/model"},
		"accept_terms_and_conditions": true,
	}}}
	_, err := tool.importModel(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_updateMetadata_validation(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing uuid", args: map[string]any{
			"name": "new-name",
		}},
		{name: "empty uuid", args: map[string]any{
			"uuid": "",
			"name": "new-name",
		}},
		{name: "no update fields", args: map[string]any{
			"uuid": "test-uuid",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateMetadata(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestCustomModelsTool_updateMetadata_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"uuid": "test-uuid",
		"name": "new-name",
	}}}
	_, err := tool.updateMetadata(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_getModel_validation(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing uuid", args: map[string]any{}},
		{name: "empty uuid", args: map[string]any{"uuid": ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getModel(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestCustomModelsTool_getModel_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"uuid": "test-uuid",
	}}}
	_, err := tool.getModel(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_deleteModel_validation(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing uuid and name", args: map[string]any{}},
		{name: "empty uuid and name", args: map[string]any{"uuid": "", "name": ""}},
		{name: "whitespace name only", args: map[string]any{"name": "  "}},
		{name: "name wrong type", args: map[string]any{"name": float64(1)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteModel(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestCustomModelsTool_deleteModel_partialUUIDWithoutConsent(t *testing.T) {
	const modelUUID = "123e4567-e89b-12d3-a456-426614174000"
	tool := setupCustomModelsToolWithTestServer(t, []*CustomModel{
		{UUID: modelUUID, Name: "mcp-delete-test-3", Status: CustomModelStatusReady},
	})

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"uuid": "123e4567",
	}}}
	resp, err := tool.deleteModel(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
	requireToolErrorContains(t, resp, "partial uuid")
	requireToolErrorContains(t, resp, modelUUID)
}

func TestResolveCustomModelUUIDByName(t *testing.T) {
	models := []*CustomModel{
		{UUID: "uuid-1", Name: "my-llama-model", Status: CustomModelStatusReady},
		{UUID: "uuid-2", Name: "my-gpt-model", Status: CustomModelStatusReady},
		{UUID: "uuid-3", Name: "my-llama-model", Status: CustomModelStatusReady},
		{UUID: "uuid-deleted", Name: "old-llama", Status: CustomModelStatusDeleted},
	}

	t.Run("exact name match", func(t *testing.T) {
		uuid, unresolved, err := resolveCustomModelUUIDByName("my-gpt-model", models)
		require.NoError(t, err)
		require.Nil(t, unresolved)
		require.Equal(t, "uuid-2", uuid)
	})

	t.Run("partial name returns matches", func(t *testing.T) {
		uuid, unresolved, err := resolveCustomModelUUIDByName("llama", models)
		require.NoError(t, err)
		require.Empty(t, uuid)
		require.NotNil(t, unresolved)
		require.Len(t, unresolved.Matches, 2)
		require.Equal(t, "llama", unresolved.Query)
		require.True(t, unresolved.RequiresExactMatch)
		require.Equal(t, "name", unresolved.QueryField)
		require.True(t, unresolved.DoNotSubstituteFromList)
	})

	t.Run("single partial match still unresolved", func(t *testing.T) {
		only := []*CustomModel{
			{UUID: "uuid-1", Name: "mcp-delete-test-3", Status: CustomModelStatusReady},
		}
		uuid, unresolved, err := resolveCustomModelUUIDByName("mcp-delete-tes", only)
		require.NoError(t, err)
		require.Empty(t, uuid)
		require.NotNil(t, unresolved)
		require.Len(t, unresolved.Matches, 1)
		require.Contains(t, unresolved.Message, "mcp-delete-test-3")
	})

	t.Run("no match", func(t *testing.T) {
		_, unresolved, err := resolveCustomModelUUIDByName("mistral", models)
		require.NoError(t, err)
		require.NotNil(t, unresolved)
		require.Empty(t, unresolved.Matches)
	})

	t.Run("duplicate exact names", func(t *testing.T) {
		_, _, err := resolveCustomModelUUIDByName("my-llama-model", models)
		require.Error(t, err)
		require.Contains(t, err.Error(), "multiple custom models")
	})

	t.Run("deleted model excluded from partial match", func(t *testing.T) {
		onlyDeleted := []*CustomModel{
			{UUID: "uuid-deleted", Name: "old-llama", Status: CustomModelStatusDeleted},
		}
		_, unresolved, err := resolveCustomModelUUIDByName("llama", onlyDeleted)
		require.NoError(t, err)
		require.NotNil(t, unresolved)
		require.Empty(t, unresolved.Matches)
	})
}

func TestResolveCustomModelUUIDByUUID(t *testing.T) {
	fullUUID := "123e4567-e89b-12d3-a456-426614174000"
	models := []*CustomModel{
		{UUID: fullUUID, Name: "my-model", Status: CustomModelStatusReady},
		{UUID: "223e4567-e89b-12d3-a456-426614174001", Name: "other", Status: CustomModelStatusReady},
	}

	t.Run("exact uuid match", func(t *testing.T) {
		uuid, unresolved, err := resolveCustomModelUUIDByUUID(fullUUID, models)
		require.NoError(t, err)
		require.Nil(t, unresolved)
		require.Equal(t, fullUUID, uuid)
	})

	t.Run("partial uuid returns matches never deletes", func(t *testing.T) {
		uuid, unresolved, err := resolveCustomModelUUIDByUUID("123e4567", models)
		require.NoError(t, err)
		require.Empty(t, uuid)
		require.NotNil(t, unresolved)
		require.Len(t, unresolved.Matches, 1)
		require.Equal(t, "uuid", unresolved.QueryField)
	})

	t.Run("single partial uuid still unresolved", func(t *testing.T) {
		uuid, unresolved, err := resolveCustomModelUUIDByUUID("426614174000", models)
		require.NoError(t, err)
		require.Empty(t, uuid)
		require.NotNil(t, unresolved)
		require.Len(t, unresolved.Matches, 1)
	})

	t.Run("unknown exact uuid format", func(t *testing.T) {
		unknown := "00000000-0000-4000-8000-000000000099"
		_, unresolved, err := resolveCustomModelUUIDByUUID(unknown, models)
		require.NoError(t, err)
		require.NotNil(t, unresolved)
		require.Empty(t, unresolved.Matches)
	})
}

func TestCustomModelsTool_deleteModel_consentRequired(t *testing.T) {
	const modelUUID = "123e4567-e89b-12d3-a456-426614174000"
	tool := setupCustomModelsToolWithTestServer(t, []*CustomModel{
		{UUID: modelUUID, Name: "my-model", Status: CustomModelStatusReady},
	})

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing confirm_deletion by name", args: map[string]any{"name": "my-model"}},
		{name: "missing confirm_deletion by uuid", args: map[string]any{"uuid": modelUUID}},
		{name: "confirm_deletion false", args: map[string]any{
			"name":             "my-model",
			"confirm_deletion": false,
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteModel(context.Background(), req)
			require.NoError(t, err, "consent rejection should be a tool error, not a client failure")
			require.True(t, resp.IsError)
			requireToolErrorContains(t, resp, "confirm_deletion")
		})
	}
}

func TestCustomModelsTool_deleteModel_partialNameWithoutConsent(t *testing.T) {
	tool := setupCustomModelsToolWithTestServer(t, []*CustomModel{
		{UUID: "123e4567-e89b-12d3-a456-426614174001", Name: "mcp-delete-test-3", Status: CustomModelStatusReady},
	})

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"name": "mcp-delete-tes",
	}}}
	resp, err := tool.deleteModel(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
	requireToolErrorContains(t, resp, "requires_exact_match")
	requireToolErrorContains(t, resp, "mcp-delete-test-3")
}

func TestCustomModelsTool_deleteModel_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"uuid":             "123e4567-e89b-12d3-a456-426614174000",
		"confirm_deletion": true,
	}}}
	_, err := tool.deleteModel(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_unifiedSearch_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"query": "llama",
	}}}
	_, err := tool.unifiedSearch(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_unifiedSearch_emptyQuery_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	_, err := tool.unifiedSearch(context.Background(), req)
	require.Error(t, err, "empty query should still attempt API call and fail on client error")
}

func setupCustomModelsToolWithTestServer(t *testing.T, models []*CustomModel) *CustomModelsTool {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/v2/gen-ai/custom_models") {
			http.Error(w, "unexpected request", http.StatusInternalServerError)
			return
		}

		if strings.Contains(r.URL.Path, "/custom_models/") {
			uuid := strings.TrimPrefix(r.URL.Path, "/v2/gen-ai/custom_models/")
			for _, m := range models {
				if m.UUID == uuid {
					_ = json.NewEncoder(w).Encode(GetCustomModelOutput{Model: m})
					return
				}
			}
			http.NotFound(w, r)
			return
		}

		_ = json.NewEncoder(w).Encode(ListCustomModelsOutput{Models: models})
	}))
	t.Cleanup(srv.Close)

	httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}))
	client, err := godo.New(httpClient, godo.SetBaseURL(srv.URL))
	require.NoError(t, err)

	return NewCustomModelsTool(func(ctx context.Context) (*godo.Client, error) {
		return client, nil
	})
}

func requireToolErrorContains(t *testing.T, resp *mcp.CallToolResult, substr string) {
	t.Helper()
	require.True(t, resp.IsError)
	if len(resp.Content) == 0 {
		return
	}
	if tc, ok := resp.Content[0].(mcp.TextContent); ok {
		require.Contains(t, tc.Text, substr)
	}
}
