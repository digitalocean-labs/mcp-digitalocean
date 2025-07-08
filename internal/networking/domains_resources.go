package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/common"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const DomainURI = "domains://"

type DomainMCPResource struct {
	client *godo.Client
}

func NewDomainMCPResource(client *godo.Client) *DomainMCPResource {
	return &DomainMCPResource{
		client: client,
	}
}

func (d *DomainMCPResource) getDomainResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		DomainURI+"{name}",
		"Domain",
		mcp.WithTemplateDescription("Returns domain information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (d *DomainMCPResource) getDomainRecordResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		DomainURI+"{name}/records/{record_id}",
		"Domain Record",
		mcp.WithTemplateDescription("Returns information about a domain record"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (d *DomainMCPResource) handleGetDomainResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	domainName, err := common.ExtractStringIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid domain URI: %w", err)
	}

	domain, _, err := d.client.Domains.Get(ctx, domainName)
	if err != nil {
		return nil, fmt.Errorf("error fetching domain: %w", err)
	}

	jsonData, err := json.MarshalIndent(domain, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing domain: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (d *DomainMCPResource) handleGetDomainRecordResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	domainName, recordID, err := extractDomainAndRecordFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid domain record URI: %w", err)
	}

	record, _, err := d.client.Domains.Record(ctx, domainName, recordID)
	if err != nil {
		return nil, fmt.Errorf("error fetching domain record: %w", err)
	}

	jsonData, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing domain record: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func extractDomainAndRecordFromURI(uri string) (string, int, error) {
	// First extract the domain name part
	uri = strings.TrimPrefix(uri, DomainURI)
	parts := strings.Split(uri, "/records/")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid domain record URI format")
	}

	domainName := parts[0]
	if domainName == "" {
		return "", 0, fmt.Errorf("empty domain name")
	}

	recordID, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid record ID: %w", err)
	}

	return domainName, recordID, nil
}

func (d *DomainMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		d.getDomainResourceTemplate():       d.handleGetDomainResource,
		d.getDomainRecordResourceTemplate(): d.handleGetDomainRecordResource,
	}
}
