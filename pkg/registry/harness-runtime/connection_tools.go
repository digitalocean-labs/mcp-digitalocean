package harnessruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/pkg/registry/common"
)

const (
	providerAuthPath     = "v2/harness-runtime/auth"
	webhookProvidersPath = "v2/harness-runtime/webhook-providers"
)

// --- Connection types ---

// ConnectionOAuth carries a connection's granted OAuth detail.
type ConnectionOAuth struct {
	Scopes    []string   `json:"scopes,omitempty"`
	GrantedAt *time.Time `json:"granted_at,omitempty"`
}

// Connection is one external-provider connection as returned by the API.
type Connection struct {
	ID                  string           `json:"id"`
	Provider            string           `json:"provider"`
	ProviderDisplayName string           `json:"provider_display_name,omitempty"`
	ActorID             string           `json:"user_id"`
	Status              string           `json:"status"`
	OwningUserID        string           `json:"owning_user_id,omitempty"`
	DeletionProtected   bool             `json:"deletion_protected"`
	CreatedAt           *time.Time       `json:"created_at,omitempty"`
	UpdatedAt           *time.Time       `json:"updated_at,omitempty"`
	OAuth               *ConnectionOAuth `json:"oauth,omitempty"`
}

// ConnectionAuthorization is the interactive step a still-pending connection
// needs. Omitted once the connection is active.
type ConnectionAuthorization struct {
	Status           string     `json:"status,omitempty"`
	ConnectURL       string     `json:"connect_url,omitempty"`
	VerificationCode string     `json:"verification_code,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

// ConnectionEnvelope is a single connection plus any authorization still
// outstanding — the shape create, get, and delete all return.
type ConnectionEnvelope struct {
	Connection    Connection               `json:"connection"`
	Authorization *ConnectionAuthorization `json:"authorization,omitempty"`
}

// ConnectionPagination is the page descriptor for a connection list.
type ConnectionPagination struct {
	Page    int32 `json:"page"`
	PerPage int32 `json:"per_page"`
	Total   int32 `json:"total"`
}

// ListConnectionsResponse is one page of connections.
type ListConnectionsResponse struct {
	Connections []Connection         `json:"connections"`
	Pagination  ConnectionPagination `json:"pagination"`
}

// --- Provider Auth types ---

// ProviderAuthStartResponse is returned by POST /v2/harness-runtime/auth/{provider}.
type ProviderAuthStartResponse struct {
	Provider         string     `json:"provider"`
	Status           string     `json:"status"`
	ConnectURL       string     `json:"connect_url,omitempty"`
	PollURL          string     `json:"poll_url,omitempty"`
	VerificationCode string     `json:"verification_code,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

// ProviderAuthPollResponse is returned by GET /v2/harness-runtime/auth/{provider}/poll.
type ProviderAuthPollResponse struct {
	Provider  string     `json:"provider"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// --- Webhook Provider types ---

// WebhookProviderSignature describes how a provider signs deliveries.
type WebhookProviderSignature struct {
	Header string `json:"header,omitempty"`
	Scheme string `json:"scheme,omitempty"`
}

// WebhookProvider is a static registry entry for the create-trigger UI.
type WebhookProvider struct {
	Key         string                    `json:"key,omitempty"`
	DisplayName string                    `json:"display_name,omitempty"`
	Description string                    `json:"description,omitempty"`
	DocsURL     string                    `json:"docs_url,omitempty"`
	PasteHint   string                    `json:"paste_hint,omitempty"`
	Signature   *WebhookProviderSignature `json:"signature,omitempty"`
}

// ListWebhookProvidersResponse is returned by ListWebhookProviders.
type ListWebhookProvidersResponse struct {
	Providers []WebhookProvider `json:"providers"`
}

// --- Handlers ---

// validProvider checks that the provider string is a recognised external provider slug.
func validProvider(provider string) bool {
	switch provider {
	case "github", "gitlab":
		return true
	}
	return false
}

// listConnections lists the team's connections for an external provider.
func (t *Tool) listConnections(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	provider, _ := args["Provider"].(string)
	if provider == "" {
		return mcp.NewToolResultError("Provider is required"), nil
	}
	if !validProvider(provider) {
		return mcp.NewToolResultError("Provider must be one of: github, gitlab"), nil
	}

	query := url.Values{}
	if v, _ := args["ActorID"].(string); v != "" {
		query.Set("user_id", v)
	}
	if v, _ := args["Status"].(string); v != "" {
		query.Set("status", v)
	}
	if v, ok := args["Page"].(float64); ok {
		if v != math.Trunc(v) || v < 1 {
			return mcp.NewToolResultError("Page must be a positive integer"), nil
		}
		query.Set("page", strconv.Itoa(int(v)))
	}
	if v, ok := args["PerPage"].(float64); ok {
		if v != math.Trunc(v) || v < 1 || v > 100 {
			return mcp.NewToolResultError("PerPage must be an integer between 1 and 100"), nil
		}
		query.Set("per_page", strconv.Itoa(int(v)))
	}

	path := providerAuthPath + "/" + provider + "/connections"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListConnectionsResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list connections", err), nil
	}
	if out.Connections == nil {
		out.Connections = []Connection{}
	}
	return connectionListOut.Result(out)
}

// createConnection creates (or resumes) a connection binding an actor to the provider.
func (t *Tool) createConnection(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	provider, _ := args["Provider"].(string)
	if provider == "" {
		return mcp.NewToolResultError("Provider is required"), nil
	}
	if !validProvider(provider) {
		return mcp.NewToolResultError("Provider must be one of: github, gitlab"), nil
	}

	actorID, _ := args["ActorID"].(string)
	if actorID == "" {
		return mcp.NewToolResultError("ActorID is required"), nil
	}

	body := map[string]any{
		"user_id": actorID,
	}

	// Scopes is optional; the API provides the provider's full configured scope set when omitted.
	if rawScopes, ok := args["Scopes"]; ok && rawScopes != nil {
		scopesSlice, ok := rawScopes.([]any)
		if !ok {
			return mcp.NewToolResultError("Scopes must be an array of strings"), nil
		}
		scopes := make([]string, 0, len(scopesSlice))
		for _, s := range scopesSlice {
			str, ok := s.(string)
			if !ok {
				return mcp.NewToolResultError("Each element of Scopes must be a string"), nil
			}
			scopes = append(scopes, str)
		}
		if len(scopes) > 0 {
			body["scopes"] = scopes
		}
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to encode request body", err), nil
	}

	path := providerAuthPath + "/" + provider + "/connections"
	apiReq, err := client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Body = io.NopCloser(bytes.NewReader(encoded))
	apiReq.ContentLength = int64(len(encoded))

	var out ConnectionEnvelope
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create connection", err), nil
	}
	return connectionOut.Result(out)
}

// getConnection reads one connection by id.
func (t *Tool) getConnection(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	provider, _ := args["Provider"].(string)
	if provider == "" {
		return mcp.NewToolResultError("Provider is required"), nil
	}
	if !validProvider(provider) {
		return mcp.NewToolResultError("Provider must be one of: github, gitlab"), nil
	}

	connectionID, _ := args["ConnectionID"].(string)
	if connectionID == "" {
		return mcp.NewToolResultError("ConnectionID is required"), nil
	}

	path := providerAuthPath + "/" + provider + "/connections/" + connectionID
	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ConnectionEnvelope
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get connection", err), nil
	}
	return connectionOut.Result(out)
}

// deleteConnection revokes one connection by id.
func (t *Tool) deleteConnection(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	provider, _ := args["Provider"].(string)
	if provider == "" {
		return mcp.NewToolResultError("Provider is required"), nil
	}
	if !validProvider(provider) {
		return mcp.NewToolResultError("Provider must be one of: github, gitlab"), nil
	}

	connectionID, _ := args["ConnectionID"].(string)
	if connectionID == "" {
		return mcp.NewToolResultError("ConnectionID is required"), nil
	}

	path := providerAuthPath + "/" + provider + "/connections/" + connectionID
	apiReq, err := client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ConnectionEnvelope
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to delete connection", err), nil
	}
	return connectionOut.Result(out)
}

// startProviderAuth begins (or resumes) the team-scoped connect flow for an
// external provider. The server derives the team from the authenticated
// principal; there is no request body.
func (t *Tool) startProviderAuth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	provider, _ := req.GetArguments()["Provider"].(string)
	if provider == "" {
		return mcp.NewToolResultError("Provider is required"), nil
	}
	if !validProvider(provider) {
		return mcp.NewToolResultError("Provider must be one of: github, gitlab"), nil
	}

	path := providerAuthPath + "/" + provider
	apiReq, err := client.NewRequest(ctx, http.MethodPost, path, struct{}{})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ProviderAuthStartResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to start provider auth", err), nil
	}
	return providerAuthStartOut.Result(out)
}

// pollProviderAuth checks whether a pending connect link has been authorized.
func (t *Tool) pollProviderAuth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	provider, _ := args["Provider"].(string)
	if provider == "" {
		return mcp.NewToolResultError("Provider is required"), nil
	}
	if !validProvider(provider) {
		return mcp.NewToolResultError("Provider must be one of: github, gitlab"), nil
	}

	pollURL, _ := args["PollURL"].(string)
	if pollURL == "" {
		return mcp.NewToolResultError("PollURL is required"), nil
	}

	query := url.Values{}
	query.Set("poll_url", pollURL)
	path := providerAuthPath + "/" + provider + "/poll?" + query.Encode()

	apiReq, err := client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ProviderAuthPollResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to poll provider auth", err), nil
	}
	return providerAuthPollOut.Result(out)
}

// listWebhookProviders returns the static set of webhook providers.
func (t *Tool) listWebhookProviders(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := client.NewRequest(ctx, http.MethodGet, webhookProvidersPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create request", err), nil
	}

	var out ListWebhookProvidersResponse
	if _, err := client.Do(ctx, apiReq, &out); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to list webhook providers", err), nil
	}
	if out.Providers == nil {
		out.Providers = []WebhookProvider{}
	}
	return webhookProviderListOut.Result(out)
}

// connectionTools returns the connection and provider auth MCP tools to be merged into Tools().
func (t *Tool) connectionTools() []server.ServerTool {
	return []server.ServerTool{
		// ---- Connections ----
		{
			Handler: t.listConnections,
			Tool: mcp.NewTool("harness-runtime-list-connections",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				connectionListOut.Schema(),
				mcp.WithDescription("List the team's connections for an external provider (e.g. GitHub, GitLab). Connections link external provider accounts to the team so agents can access repos. Paginated with page/per_page (default 20, max 100)."),
				mcp.WithString("Provider", mcp.Required(), mcp.Description("External provider slug: 'github' or 'gitlab'")),
				mcp.WithString("ActorID", mcp.Description("Optional filter by actor (user_id) — the actor identifier the manifest's secret actor: binds to")),
				mcp.WithString("Status", mcp.Description("Optional filter by connection status")),
				mcp.WithNumber("Page", mcp.Description("Page number (1-based, default 1)")),
				mcp.WithNumber("PerPage", mcp.Description("Results per page (1-100, default 20)")),
			),
		},
		{
			Handler: t.createConnection,
			Tool: mcp.NewTool("harness-runtime-create-connection",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				connectionOut.Schema(),
				mcp.WithDescription("Create (or resume) a connection binding an actor to an external provider. A still-pending connection comes back with an Authorization the user must complete in a browser; an already-active one has none. The ActorID names the actor the connection acts for (maps to user_id on the wire)."),
				mcp.WithString("Provider", mcp.Required(), mcp.Description("External provider slug: 'github' or 'gitlab'")),
				mcp.WithString("ActorID", mcp.Required(), mcp.Description("Actor identifier the connection acts for (user_id on the wire)")),
				mcp.WithArray("Scopes", mcp.Description("Optional OAuth scopes to request; omit for the provider's full configured scope set"), mcp.Items(map[string]any{"type": "string"})),
			),
		},
		{
			Handler: t.getConnection,
			Tool: mcp.NewTool("harness-runtime-get-connection",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				connectionOut.Schema(),
				mcp.WithDescription("Get one connection by id, including its status and any pending authorization. Useful for polling a pending connection to completion."),
				mcp.WithString("Provider", mcp.Required(), mcp.Description("External provider slug: 'github' or 'gitlab'")),
				mcp.WithString("ConnectionID", mcp.Required(), mcp.Description("The connection id")),
			),
		},
		{
			Handler: t.deleteConnection,
			Tool: mcp.NewTool("harness-runtime-delete-connection",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				connectionOut.Schema(),
				mcp.WithDescription("Revoke and delete one connection by id. Returns the connection's final state. Connections with deletion_protected=true (i.e. referenced by an active agent environment) will be refused — remove the environment dependency first."),
				mcp.WithString("Provider", mcp.Required(), mcp.Description("External provider slug: 'github' or 'gitlab'")),
				mcp.WithString("ConnectionID", mcp.Required(), mcp.Description("The connection id to delete")),
			),
		},
		// ---- Provider Auth ----
		{
			Handler: t.startProviderAuth,
			Tool: mcp.NewTool("harness-runtime-start-provider-auth",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				providerAuthStartOut.Schema(),
				mcp.WithDescription("Begin (or resume) the OAuth device-flow connect for an external provider (e.g. GitHub). The team is derived from your authenticated principal. Returns a connect_url the user must open in a browser and a verification_code to enter. After the user authorizes, call harness-runtime-poll-provider-auth with the returned poll_url to check completion. Status is 'pending' when the user must still authorize, or 'success' when already connected."),
				mcp.WithString("Provider", mcp.Required(), mcp.Description("External provider slug: 'github' or 'gitlab'")),
			),
		},
		{
			Handler: t.pollProviderAuth,
			Tool: mcp.NewTool("harness-runtime-poll-provider-auth",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				providerAuthPollOut.Schema(),
				mcp.WithDescription("Check whether a pending provider authorization has been completed. Call repeatedly after start-provider-auth until status is 'success'. PollURL is the poll_url returned by start-provider-auth."),
				mcp.WithString("Provider", mcp.Required(), mcp.Description("External provider slug: 'github' or 'gitlab'")),
				mcp.WithString("PollURL", mcp.Required(), mcp.Description("The poll_url returned by harness-runtime-start-provider-auth")),
			),
		},
		// ---- Webhook Providers ----
		{
			Handler: t.listWebhookProviders,
			Tool: mcp.NewTool("harness-runtime-list-webhook-providers",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				webhookProviderListOut.Schema(),
				mcp.WithDescription("List the supported webhook providers for trigger creation. Returns a static catalog of providers (e.g. GitHub, GitLab, Custom) with their display names, documentation URLs, paste hints, and signature verification schemes. No pagination — the full list is returned."),
			),
		},
	}
}
