package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/common"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	ReservedIPv4URI = "reserved_ipv4://"
	ReservedIPv6URI = "reserved_ipv6://"
)

type ReservedIPMCPResource struct {
	client *godo.Client
}

func NewReservedIPMCPResource(client *godo.Client) *ReservedIPMCPResource {
	return &ReservedIPMCPResource{
		client: client,
	}
}

func (r *ReservedIPMCPResource) getIPv4ResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		ReservedIPv4URI+"{ip}",
		"Reserved IPv4",
		mcp.WithTemplateDescription("Returns information about a reserved IPv4"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (r *ReservedIPMCPResource) getIPv6ResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		ReservedIPv6URI+"{ip}",
		"Reserved IPv6",
		mcp.WithTemplateDescription("Returns information about a reserved IPv6"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (r *ReservedIPMCPResource) handleGetIPv4Resource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	ip, err := common.ExtractStringIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid reserved IPv4 URI: %w", err)
	}

	reservedIP, _, err := r.client.ReservedIPs.Get(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("error fetching reserved IPv4: %w", err)
	}

	jsonData, err := json.MarshalIndent(reservedIP, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing reserved IPv4: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (r *ReservedIPMCPResource) handleGetIPv6Resource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	ip, err := common.ExtractStringIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid reserved IPv6 URI: %w", err)
	}

	reservedIP, _, err := r.client.ReservedIPV6s.Get(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("error fetching reserved IPv6: %w", err)
	}

	jsonData, err := json.MarshalIndent(reservedIP, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing reserved IPv6: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (r *ReservedIPMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		r.getIPv4ResourceTemplate(): r.handleGetIPv4Resource,
		r.getIPv6ResourceTemplate(): r.handleGetIPv6Resource,
	}
}
