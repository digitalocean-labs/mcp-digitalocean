package docr

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	defaultRepoPageSize = 20
	defaultRepoPage     = 1
)

// RepositoryTool provides container registry repository management tools
type RepositoryTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewRepositoryTool creates a new RepositoryTool
func NewRepositoryTool(client func(ctx context.Context) (*godo.Client, error)) *RepositoryTool {
	return &RepositoryTool{
		client: client,
	}
}

// listRepositories lists repositories in a container registry
func (r *RepositoryTool) listRepositories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryName, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || registryName == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	page := defaultRepoPage
	perPage := defaultRepoPageSize
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}
	pageToken, _ := req.GetArguments()["PageToken"].(string)

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	repos, resp, err := client.Registries.ListRepositoriesV2(ctx, registryName, &godo.TokenListOptions{
		Page:    page,
		PerPage: perPage,
		Token:   pageToken,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	result := struct {
		Repositories []*godo.RepositoryV2 `json:"repositories"`
		Meta         *godo.Meta           `json:"meta,omitempty"`
	}{
		Repositories: repos,
		Meta:         resp.Meta,
	}

	jsonRepos, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonRepos)), nil
}

// listRepositoryTags lists tags for a repository
func (r *RepositoryTool) listRepositoryTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryName, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || registryName == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	repository, ok := req.GetArguments()["Repository"].(string)
	if !ok || repository == "" {
		return mcp.NewToolResultError("Repository is required"), nil
	}

	page := defaultRepoPage
	perPage := defaultRepoPageSize
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	tags, resp, err := client.Registries.ListRepositoryTags(ctx, registryName, repository, &godo.ListOptions{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	result := struct {
		Tags []*godo.RepositoryTag `json:"tags"`
		Meta *godo.Meta            `json:"meta,omitempty"`
	}{
		Tags: tags,
		Meta: resp.Meta,
	}

	jsonTags, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonTags)), nil
}

// deleteTag deletes a tag from a repository
func (r *RepositoryTool) deleteTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryName, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || registryName == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	repository, ok := req.GetArguments()["Repository"].(string)
	if !ok || repository == "" {
		return mcp.NewToolResultError("Repository is required"), nil
	}

	tag, ok := req.GetArguments()["Tag"].(string)
	if !ok || tag == "" {
		return mcp.NewToolResultError("Tag is required"), nil
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Registries.DeleteTag(ctx, registryName, repository, tag)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("repository tag deleted successfully"), nil
}

// listRepositoryManifests lists manifests for a repository
func (r *RepositoryTool) listRepositoryManifests(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryName, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || registryName == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	repository, ok := req.GetArguments()["Repository"].(string)
	if !ok || repository == "" {
		return mcp.NewToolResultError("Repository is required"), nil
	}

	page := defaultRepoPage
	perPage := defaultRepoPageSize
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	manifests, resp, err := client.Registries.ListRepositoryManifests(ctx, registryName, repository, &godo.ListOptions{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	result := struct {
		Manifests []*godo.RepositoryManifest `json:"manifests"`
		Meta      *godo.Meta                 `json:"meta,omitempty"`
	}{
		Manifests: manifests,
		Meta:      resp.Meta,
	}

	jsonManifests, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonManifests)), nil
}

// deleteManifest deletes a manifest from a repository
func (r *RepositoryTool) deleteManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryName, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || registryName == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	repository, ok := req.GetArguments()["Repository"].(string)
	if !ok || repository == "" {
		return mcp.NewToolResultError("Repository is required"), nil
	}

	digest, ok := req.GetArguments()["Digest"].(string)
	if !ok || digest == "" {
		return mcp.NewToolResultError("Digest is required"), nil
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Registries.DeleteManifest(ctx, registryName, repository, digest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("repository manifest deleted successfully"), nil
}

// Tools returns a list of tool functions for repository management
func (r *RepositoryTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: r.listRepositories,
			Tool: mcp.NewTool("docr-repository-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List repositories in a container registry"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultRepoPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultRepoPageSize), mcp.Description("Items per page")),
				mcp.WithString("PageToken", mcp.Description("Token for paginating through results")),
			),
		},
		{
			Handler: r.listRepositoryTags,
			Tool: mcp.NewTool("docr-repository-tag-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List tags for a repository in a container registry"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithString("Repository", mcp.Required(), mcp.Description("Name of the repository")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultRepoPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultRepoPageSize), mcp.Description("Items per page")),
			),
		},
		{
			Handler: r.deleteTag,
			Tool: mcp.NewTool("docr-repository-tag-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a tag from a repository in a container registry"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithString("Repository", mcp.Required(), mcp.Description("Name of the repository")),
				mcp.WithString("Tag", mcp.Required(), mcp.Description("Tag to delete")),
			),
		},
		{
			Handler: r.listRepositoryManifests,
			Tool: mcp.NewTool("docr-repository-manifest-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List manifests for a repository in a container registry"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithString("Repository", mcp.Required(), mcp.Description("Name of the repository")),
				mcp.WithNumber("Page", mcp.DefaultNumber(defaultRepoPage), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultRepoPageSize), mcp.Description("Items per page")),
			),
		},
		{
			Handler: r.deleteManifest,
			Tool: mcp.NewTool("docr-repository-manifest-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a manifest from a repository in a container registry"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithString("Repository", mcp.Required(), mcp.Description("Name of the repository")),
				mcp.WithString("Digest", mcp.Required(), mcp.Description("Digest of the manifest to delete (e.g., 'sha256:abc123...')")),
			),
		},
	}
}
