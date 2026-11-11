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

func TestValidateRepositoryName(t *testing.T) {
	valid := []string{
		"myapp",
		"my-app",
		"my_app",
		"my__app",
		"my.app",
		"team/my-app",
		"a/b/c",
		"registry.digitalocean.com/myreg/repo",
		"app123",
		strings.Repeat("a", 255),
	}
	for _, name := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			require.NoError(t, validateRepositoryName(name))
		})
	}
	invalid := []string{
		"",
		"MyApp",    // uppercase
		"-myapp",   // separator at start
		"myapp-",   // separator at end
		"my..app",  // consecutive dots
		"my/",      // empty component
		"/myapp",   // empty component
		"my app",   // space
		"my%2fapp", // percent
		"my\\app",  // backslash
		strings.Repeat("a", 256),
		".",
		"..",
		"../droplets",
		"team/../../droplets/123",
		"..%2f..%2fdroplets",
	}
	for _, name := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			require.Error(t, validateRepositoryName(name))
		})
	}
}

func TestValidateTag(t *testing.T) {
	valid := []string{"latest", "v1.2.3", "1.0", "RELEASE_2024", "a", "sha-abc123", "_x", strings.Repeat("a", 128)}
	for _, tag := range valid {
		t.Run("valid/"+tag, func(t *testing.T) {
			require.NoError(t, validateTag(tag))
		})
	}
	invalid := []string{"", ".hidden", "-x", "a/b", "a b", "a%2fb", "a:b", strings.Repeat("a", 129), "..", "../../../../droplets/123"}
	for _, tag := range invalid {
		t.Run("invalid/"+tag, func(t *testing.T) {
			require.Error(t, validateTag(tag))
		})
	}
}

func TestValidateDigest(t *testing.T) {
	valid := []string{
		"sha256:" + strings.Repeat("a", 64),
		"sha512:" + strings.Repeat("0", 128),
		"sha256:abc", // shape only; the API checks the length
	}
	for _, d := range valid {
		t.Run("valid/"+d, func(t *testing.T) {
			require.NoError(t, validateDigest(d))
		})
	}
	invalid := []string{"", strings.Repeat("a", 64), "sha256", "sha256:", "SHA256:abc", "sha256:abc/def", "sha256:../../droplets", "../droplets/123"}
	for _, d := range invalid {
		t.Run("invalid/"+d, func(t *testing.T) {
			require.Error(t, validateDigest(d))
		})
	}
}

func TestValidateGarbageCollectionUUID(t *testing.T) {
	valid := []string{"3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01", "00000000-0000-4000-8000-000000000000"}
	for _, id := range valid {
		t.Run("valid/"+id, func(t *testing.T) {
			require.NoError(t, validateGarbageCollectionUUID(id))
		})
	}
	invalid := []string{"", "gc-uuid-1", "3f2b1c6e-8d4a-4f1b-9c2e", "../droplets/123", "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01/../x"}
	for _, id := range invalid {
		t.Run("invalid/"+id, func(t *testing.T) {
			require.Error(t, validateGarbageCollectionUUID(id))
		})
	}
}

// TestPathArgumentsRejectedBeforeAPICall checks that every handler refuses a
// path-traversal value in each path-bound argument without calling the API.
// The gomock controller fails the test on any unexpected call.
func TestPathArgumentsRejectedBeforeAPICall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockRegistriesService(ctrl)

	registry := setupRegistryToolWithMock(mock)
	repository := setupRepositoryToolWithMock(mock)
	gc := setupGarbageCollectionToolWithMock(mock)

	type handlerFn = func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	cases := []struct {
		tool    string
		handler handlerFn
		args    []string // path-bound arguments the handler reads
	}{
		{"docr-get", registry.get, []string{"RegistryName"}},
		{"docr-delete", registry.delete, []string{"RegistryName"}},
		{"docr-docker-credentials", registry.dockerCredentials, []string{"RegistryName"}},
		{"docr-repository-list", repository.listRepositories, []string{"RegistryName"}},
		{"docr-repository-tag-list", repository.listRepositoryTags, []string{"RegistryName", "Repository"}},
		{"docr-repository-tag-delete", repository.deleteTag, []string{"RegistryName", "Repository", "Tag"}},
		{"docr-repository-manifest-list", repository.listRepositoryManifests, []string{"RegistryName", "Repository"}},
		{"docr-repository-manifest-delete", repository.deleteManifest, []string{"RegistryName", "Repository", "Digest"}},
		{"docr-garbage-collection-start", gc.startGarbageCollection, []string{"RegistryName"}},
		{"docr-garbage-collection-get", gc.getGarbageCollection, []string{"RegistryName"}},
		{"docr-garbage-collection-list", gc.listGarbageCollections, []string{"RegistryName"}},
		{"docr-garbage-collection-update", gc.updateGarbageCollection, []string{"RegistryName", "GarbageCollectionUUID"}},
	}

	validArgs := map[string]any{
		"RegistryName":          "my-registry",
		"Repository":            "team/my-app",
		"Tag":                   "latest",
		"Digest":                "sha256:" + strings.Repeat("a", 64),
		"Type":                  "untagged manifests only",
		"GarbageCollectionUUID": "3f2b1c6e-8d4a-4f1b-9c2e-7a5d6b8e9f01",
		"Cancel":                true,
	}

	for _, tc := range cases {
		for _, arg := range tc.args {
			t.Run(tc.tool+"/"+arg, func(t *testing.T) {
				args := map[string]any{}
				for k, v := range validArgs {
					args[k] = v
				}
				args[arg] = "../../../../droplets/12345678"
				req := mcp.CallToolRequest{}
				req.Params.Arguments = args

				resp, err := tc.handler(context.Background(), req)
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError, "expected a tool error result")
				text, ok := resp.Content[0].(mcp.TextContent)
				require.True(t, ok)
				require.Contains(t, text.Text, arg)
			})
		}
	}
}
