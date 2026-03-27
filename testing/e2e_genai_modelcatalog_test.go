//go:build integration

package testing

import (
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// TestModelCatalogSearch tests searching for models in the catalog
func TestModelCatalogSearch(t *testing.T) {
	// Search for a common model name
	t.Log("searching for 'llama' models...")
	result := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "genai-model-catalog-search", map[string]interface{}{
		"SearchQuery": "llama",
	})

	require.NotNil(t, result)
	require.Equal(t, "llama", result.SearchQuery)
	t.Logf("found %d models matching 'llama'", result.Count)

	// Search with empty string to get all models
	t.Log("searching with empty string to get all models...")
	allModels := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "genai-model-catalog-search", map[string]interface{}{
		"SearchQuery": "",
	})

	require.NotNil(t, allModels)
	require.Equal(t, "", allModels.SearchQuery)
	require.Greater(t, allModels.Count, 0, "empty search should return all models")
	t.Logf("found %d total models with empty search", allModels.Count)

	// Search with missing parameter (should also return all models)
	t.Log("searching with missing parameter to get all models...")
	allModelsNoParam := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "genai-model-catalog-search", map[string]interface{}{})

	require.NotNil(t, allModelsNoParam)
	require.Equal(t, "", allModelsNoParam.SearchQuery, "missing parameter should default to empty string")
	require.Greater(t, allModelsNoParam.Count, 0, "missing parameter should return all models")
	require.Equal(t, allModels.Count, allModelsNoParam.Count, "empty string and missing parameter should return same results")
	t.Logf("found %d total models with missing parameter", allModelsNoParam.Count)

	// Search for a model that likely doesn't exist
	t.Log("searching for non-existent model...")
	emptyResult := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "genai-model-catalog-search", map[string]interface{}{
		"SearchQuery": "nonexistent-model-xyz-123",
	})

	require.NotNil(t, emptyResult)
	require.Equal(t, 0, emptyResult.Count)
	require.Empty(t, emptyResult.ModelUUIDs)
	t.Log("correctly returned empty results for non-existent model")
}

// TestModelCatalogGetCard tests retrieving model metadata
func TestModelCatalogGetCard(t *testing.T) {
	// First search for models to get a valid UUID
	searchQuery := "llama"
	t.Logf("searching for '%s' models to get a valid UUID...", searchQuery)
	searchResult := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "genai-model-catalog-search", map[string]interface{}{
		"SearchQuery": searchQuery,
	})

	if len(searchResult.ModelUUIDs) == 0 {
		t.Skip("no models found to test with, skipping model card test")
	}

	// Use the first UUID to get model details
	testUUID := searchResult.ModelUUIDs[0]
	t.Logf("getting model card for UUID: %s", testUUID)

	model := callTool[struct {
		UUID              string          `json:"uuid"`
		Name              string          `json:"name"`
		Agreement         *godo.Agreement `json:"agreement,omitempty"`
		ModelAvailability string          `json:"model_availability,omitempty"`
	}](t, "genai-model-catalog-get-card", map[string]interface{}{
		"ModelUUID": testUUID,
	})

	require.Equal(t, testUUID, model.UUID)
	require.NotEmpty(t, model.Name, "model should have a name")
	require.Contains(t, strings.ToLower(model.Name), strings.ToLower(searchQuery), "model name should contain the search query")
	t.Logf("successfully retrieved model card: %s", model.Name)
	if model.ModelAvailability != "" {
		t.Logf("model availability: %s", model.ModelAvailability)
	}

	// Verify model metadata structure
	if model.Agreement != nil {
		t.Logf("model agreement: %s", model.Agreement.Name)
	}
}

// TestModelCatalogGetCardNotFound tests error handling for non-existent model UUID
func TestModelCatalogGetCardNotFound(t *testing.T) {
	ctx, c := getTestClient(t)

	fakeUUID := "99999999-9999-9999-9999-999999999999"
	t.Logf("attempting to get model card for non-existent UUID: %s", fakeUUID)

	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "genai-model-catalog-get-card",
			Arguments: map[string]interface{}{
				"ModelUUID": fakeUUID,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError, "should return error for non-existent UUID")

	if len(resp.Content) > 0 {
		if tc, ok := resp.Content[0].(mcp.TextContent); ok {
			t.Logf("error message: %s", tc.Text)
			require.Contains(t, tc.Text, "not found")
		}
	}
}

// TestModelCatalogWorkflow tests a complete workflow: search -> get details
func TestModelCatalogWorkflow(t *testing.T) {
	t.Log("testing complete model catalog workflow...")

	// Step 1: Search for a specific model
	searchQuery := "gpt"
	t.Logf("step 1: searching for '%s' models...", searchQuery)

	searchResult := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "genai-model-catalog-search", map[string]interface{}{
		"SearchQuery": searchQuery,
	})

	t.Logf("found %d models matching '%s'", searchResult.Count, searchQuery)

	if len(searchResult.ModelUUIDs) == 0 {
		t.Skip("no models found, skipping workflow test")
	}

	// Step 2: Get details for each model found
	t.Logf("step 2: getting details for %d models...", len(searchResult.ModelUUIDs))

	for i, uuid := range searchResult.ModelUUIDs {
		if i >= 3 {
			// Limit to first 3 models to avoid too many API calls
			t.Logf("limiting to first 3 models")
			break
		}

		model := callTool[struct {
			UUID              string          `json:"uuid"`
			Name              string          `json:"name"`
			Agreement         *godo.Agreement `json:"agreement,omitempty"`
			ModelAvailability string          `json:"model_availability,omitempty"`
		}](t, "genai-model-catalog-get-card", map[string]interface{}{
			"ModelUUID": uuid,
		})

		require.Equal(t, uuid, model.UUID)
		require.Contains(t, strings.ToLower(model.Name), strings.ToLower(searchQuery), "model name should contain the search query")
		if model.ModelAvailability != "" {
			t.Logf("  - %s (availability: %s)", model.Name, model.ModelAvailability)
		} else {
			t.Logf("  - %s", model.Name)
		}
	}

	t.Log("workflow completed successfully")
}
