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

// RegistryTool provides container registry management tools
type RegistryTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewRegistryTool creates a new RegistryTool
func NewRegistryTool(client func(ctx context.Context) (*godo.Client, error)) *RegistryTool {
	return &RegistryTool{
		client: client,
	}
}

// get fetches a container registry by name
func (r *RegistryTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	registry, _, err := client.Registries.Get(ctx, name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonRegistry, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonRegistry)), nil
}

// list lists all container registries
func (r *RegistryTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	registries, _, err := client.Registries.List(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonRegistries, err := json.MarshalIndent(registries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonRegistries)), nil
}

// create creates a new container registry
func (r *RegistryTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := req.GetArguments()["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}

	subscriptionTierSlug, _ := req.GetArguments()["SubscriptionTierSlug"].(string)
	region, _ := req.GetArguments()["Region"].(string)

	createRequest := &godo.RegistryCreateRequest{
		Name:                 name,
		SubscriptionTierSlug: subscriptionTierSlug,
		Region:               region,
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	registry, _, err := client.Registries.Create(ctx, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonRegistry, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonRegistry)), nil
}

// delete deletes a container registry
func (r *RegistryTool) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Registries.Delete(ctx, name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("container registry deleted successfully"), nil
}

// dockerCredentials retrieves Docker credentials for a container registry
func (r *RegistryTool) dockerCredentials(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := req.GetArguments()["RegistryName"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("RegistryName is required"), nil
	}

	readWrite := false
	if v, ok := req.GetArguments()["ReadWrite"].(bool); ok {
		readWrite = v
	}

	credRequest := &godo.RegistryDockerCredentialsRequest{
		ReadWrite: readWrite,
	}

	if v, ok := req.GetArguments()["ExpirySeconds"].(float64); ok && int(v) > 0 {
		expiry := int(v)
		credRequest.ExpirySeconds = &expiry
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	creds, _, err := client.Registries.DockerCredentials(ctx, name, credRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText(string(creds.DockerConfigJSON)), nil
}

// getOptions retrieves available registry options
func (r *RegistryTool) getOptions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	options, _, err := client.Registries.GetOptions(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonOptions, err := json.MarshalIndent(options, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonOptions)), nil
}

// validateName validates a registry name
func (r *RegistryTool) validateName(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := req.GetArguments()["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}

	client, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Registries.ValidateName(ctx, &godo.RegistryValidateNameRequest{
		Name: name,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("registry name %q is available", name)), nil
}

// Tools returns a list of tool functions for registry management
func (r *RegistryTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: r.get,
			Tool: mcp.NewTool("docr-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get a container registry by name"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
			),
		},
		{
			Handler: r.list,
			Tool: mcp.NewTool("docr-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List all container registries"),
			),
		},
		{
			Handler: r.create,
			Tool: mcp.NewTool("docr-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Create a new container registry"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithString("SubscriptionTierSlug", mcp.Description("Subscription tier slug (e.g., 'starter', 'basic', 'professional')")),
				mcp.WithString("Region", mcp.Description("Region slug for the registry (e.g., 'nyc3', 'sfo3')")),
			),
		},
		{
			Handler: r.delete,
			Tool: mcp.NewTool("docr-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a container registry"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry to delete")),
			),
		},
		{
			Handler: r.dockerCredentials,
			Tool: mcp.NewTool("docr-docker-credentials",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get Docker credentials for a container registry"),
				mcp.WithString("RegistryName", mcp.Required(), mcp.Description("Name of the container registry")),
				mcp.WithBoolean("ReadWrite", mcp.Description("Whether the credentials should have read-write access (default: false, read-only)")),
				mcp.WithNumber("ExpirySeconds", mcp.Description("Number of seconds until the credentials expire. If not set, credentials do not expire")),
			),
		},
		{
			Handler: r.getOptions,
			Tool: mcp.NewTool("docr-options",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get available container registry options including subscription tiers and regions"),
			),
		},
		{
			Handler: r.validateName,
			Tool: mcp.NewTool("docr-validate-name",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Check if a container registry name is available"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name to validate for availability")),
			),
		},
	}
}
