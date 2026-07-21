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

// SubscriptionTool provides container registry subscription management tools
type SubscriptionTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewSubscriptionTool creates a new SubscriptionTool
func NewSubscriptionTool(client func(ctx context.Context) (*godo.Client, error)) *SubscriptionTool {
	return &SubscriptionTool{
		client: client,
	}
}

// getSubscription gets the current subscription for the registry
func (s *SubscriptionTool) getSubscription(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	subscription, _, err := client.Registries.GetSubscription(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonSubscription, err := json.MarshalIndent(subscription, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonSubscription)), nil
}

// updateSubscription updates the subscription tier for the registry
func (s *SubscriptionTool) updateSubscription(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tierSlug, ok := req.GetArguments()["TierSlug"].(string)
	if !ok || tierSlug == "" {
		return mcp.NewToolResultError("TierSlug is required"), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	subscription, _, err := client.Registries.UpdateSubscription(ctx, &godo.RegistrySubscriptionUpdateRequest{
		TierSlug: tierSlug,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonSubscription, err := json.MarshalIndent(subscription, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonSubscription)), nil
}

// Tools returns a list of tool functions for subscription management
func (s *SubscriptionTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.getSubscription,
			Tool: mcp.NewTool("docr-subscription-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get the current container registry subscription information"),
			),
		},
		{
			Handler: s.updateSubscription,
			Tool: mcp.NewTool("docr-subscription-update",
				common.WithHints(common.HintsToggle),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Update the container registry subscription tier"),
				mcp.WithString("TierSlug", mcp.Required(), mcp.Description("Subscription tier slug to update to (e.g., 'starter', 'basic', 'professional')")),
			),
		},
	}
}
