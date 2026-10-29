package docr

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestValidateRegistryName(t *testing.T) {
	valid := []string{
		"a",
		"my-registry",
		"registry1",
		"r2d2",
		"mcp-e2e-docr-1761580000",
		strings.Repeat("a", 63),
	}
	for _, name := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			require.NoError(t, validateRegistryName(name))
		})
	}

	invalid := []string{
		"",
		"My-Registry", // uppercase
		"1registry",   // must start with a letter
		"-registry",   // must start with a letter
		"registry-",   // must end with a letter or digit
		"my_registry", // underscore
		"my.registry", // dot
		"my registry", // space
		"my/registry", // slash
		strings.Repeat("a", 64),
		// Path-traversal payloads observed against the hosted endpoint.
		"../droplets/12345678",
		"../../../../v2/droplets/docker-credentials",
		"..%2f..%2f..%2fv2%2fdroplets",
		"%2e%2e%2f%2e%2e%2fv2%2fdroplets",
		"..;/..;/v2/droplets",
		"..%5C..%5Cv2%5Cdroplets",
		"%c0%ae%c0%ae/%c0%ae%c0%ae/v2/droplets",
		"..%252f..%252fv2%252fdroplets",
		"....//....//v2/droplets",
		"https://api.digitalocean.com/v2/droplets",
		"//api.digitalocean.com/v2/droplets",
		"/v2/droplets",
	}
	for _, name := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			require.Error(t, validateRegistryName(name))
		})
	}
}

// TestRegistryNameRejectedBeforeAPICall checks that every handler taking a
// RegistryName refuses an invalid name without calling the API. The gomock
// controller fails the test on any unexpected call.
func TestRegistryNameRejectedBeforeAPICall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockRegistriesService(ctrl)

	registry := setupRegistryToolWithMock(mock)
	repository := setupRepositoryToolWithMock(mock)
	gc := setupGarbageCollectionToolWithMock(mock)

	handlers := map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error){
		"docr-get":                        registry.get,
		"docr-delete":                     registry.delete,
		"docr-docker-credentials":         registry.dockerCredentials,
		"docr-repository-list":            repository.listRepositories,
		"docr-repository-tags-list":       repository.listRepositoryTags,
		"docr-repository-tag-delete":      repository.deleteTag,
		"docr-repository-manifests-list":  repository.listRepositoryManifests,
		"docr-repository-manifest-delete": repository.deleteManifest,
		"docr-garbage-collection-start":   gc.startGarbageCollection,
		"docr-garbage-collection-get":     gc.getGarbageCollection,
		"docr-garbage-collection-list":    gc.listGarbageCollections,
		"docr-garbage-collection-update":  gc.updateGarbageCollection,
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"RegistryName":          "../droplets/12345678",
				"RepositoryName":        "repo",
				"Tag":                   "latest",
				"ManifestDigest":        "sha256:0000000000000000000000000000000000000000000000000000000000000000",
				"Type":                  "untagged manifests only",
				"GarbageCollectionUUID": "00000000-0000-4000-8000-000000000000",
				"Cancel":                true,
			}
			resp, err := handler(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError, "expected a tool error result")
			text, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			require.Contains(t, text.Text, "RegistryName")
		})
	}
}
