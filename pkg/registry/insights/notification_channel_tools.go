package insights

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

type NotificationChannelTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewNotificationChannelTool(client func(ctx context.Context) (*godo.Client, error)) *NotificationChannelTool {
	return &NotificationChannelTool{client: client}
}

func (t *NotificationChannelTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := stringArg(req.GetArguments(), "ID")
	if id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	ch, _, err := client.Insights.GetNotificationChannel(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return notificationChannelOut.Result(ch)
}

func (t *NotificationChannelTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	channels, _, err := client.Insights.ListNotificationChannels(ctx, &godo.ListOptions{
		Page:    intArg(args, "Page", defaultAlertPoliciesPage),
		PerPage: intArg(args, "PerPage", defaultAlertPoliciesPageSize),
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return notificationChannelsOut.Result(channels)
}

func (t *NotificationChannelTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	if stringArg(args, "Name") == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}
	if stringArg(args, "EmailTo") == "" && stringArg(args, "SlackChannel") == "" && stringArg(args, "WebhookURL") == "" {
		return mcp.NewToolResultError("Set EmailTo, SlackChannel, or WebhookURL"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	ch, _, err := client.Insights.CreateNotificationChannel(ctx, notificationChannelRequestFromArgs(args))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return notificationChannelOut.Result(ch)
}

func (t *NotificationChannelTool) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id := stringArg(args, "ID")
	if id == "" || stringArg(args, "Name") == "" {
		return mcp.NewToolResultError("ID and Name are required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	ch, _, err := client.Insights.UpdateNotificationChannel(ctx, id, notificationChannelRequestFromArgs(args))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return notificationChannelOut.Result(ch)
}

func (t *NotificationChannelTool) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := stringArg(req.GetArguments(), "ID")
	if id == "" {
		return mcp.NewToolResultError("ID is required"), nil
	}
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	if _, err := client.Insights.DeleteNotificationChannel(ctx, id); err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Notification channel deleted successfully"), nil
}

func channelConfigArgs() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("Name", mcp.Required(), mcp.Description("Channel name")),
		mcp.WithString("EmailTo", mcp.Description("Verified team-member emails, comma/space/semicolon separated")),
		mcp.WithString("SlackChannel", mcp.Description("Slack channel, e.g. #alerts")),
		mcp.WithString("SlackWebhookURL", mcp.Description("Slack webhook URL (write-only)")),
		mcp.WithString("WebhookURL", mcp.Description("HTTPS webhook URL")),
		mcp.WithString("WebhookUsername", mcp.Description("Optional webhook basic-auth username")),
		mcp.WithString("WebhookPassword", mcp.Description("Optional webhook basic-auth password (write-only)")),
		mcp.WithString("WebhookBearerToken", mcp.Description("Optional webhook bearer token (write-only; do not set with basic auth)")),
		mcp.WithString("WebhookSignatureSecret", mcp.Description("Optional HMAC signing secret (write-only)")),
	}
}

func (t *NotificationChannelTool) Tools() []server.ServerTool {
	createOpts := append([]mcp.ToolOption{
		common.WithHints(common.HintsCreate),
		common.WithRisk(common.RiskLow),
		notificationChannelOut.Schema(),
		mcp.WithDescription("PREFERRED — Create an Insights v2 notification channel. Set exactly one of EmailTo, SlackChannel, or WebhookURL."),
	}, channelConfigArgs()...)
	updateOpts := append([]mcp.ToolOption{
		common.WithHints(common.HintsToggle),
		common.WithRisk(common.RiskLow),
		notificationChannelOut.Schema(),
		mcp.WithDescription("PREFERRED — Update an Insights v2 notification channel. Leave SlackWebhookURL empty to keep the current secret."),
		mcp.WithString("ID", mcp.Required(), mcp.Description("Notification channel ID")),
	}, channelConfigArgs()...)

	return []server.ServerTool{
		{Handler: t.get, Tool: mcp.NewTool("insights-notification-channel-get",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), notificationChannelOut.Schema(),
			mcp.WithDescription("PREFERRED — Get an Insights v2 notification channel. Secrets are masked."),
			mcp.WithString("ID", mcp.Required(), mcp.Description("Notification channel ID")),
		)},
		{Handler: t.list, Tool: mcp.NewTool("insights-notification-channel-list",
			common.WithHints(common.HintsRead), common.WithRisk(common.RiskLow), notificationChannelsOut.Schema(),
			mcp.WithDescription("PREFERRED — List Insights v2 notification channels."),
			mcp.WithNumber("Page", mcp.DefaultNumber(defaultAlertPoliciesPage), mcp.Description("Page number")),
			mcp.WithNumber("PerPage", mcp.DefaultNumber(defaultAlertPoliciesPageSize), mcp.Description("Items per page")),
		)},
		{Handler: t.create, Tool: mcp.NewTool("insights-notification-channel-create", createOpts...)},
		{Handler: t.update, Tool: mcp.NewTool("insights-notification-channel-update", updateOpts...)},
		{Handler: t.delete, Tool: mcp.NewTool("insights-notification-channel-delete",
			common.WithHints(common.HintsDelete), common.WithRisk(common.RiskMedium),
			mcp.WithDescription("PREFERRED — Delete an Insights v2 notification channel. Returns 409 if alert rules still reference it."),
			mcp.WithString("ID", mcp.Required(), mcp.Description("Notification channel ID")),
		)},
	}
}
