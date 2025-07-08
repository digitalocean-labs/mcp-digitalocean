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

const CertificateURI = "certificates://"

type CertificateMCPResource struct {
	client *godo.Client
}

func NewCertificateMCPResource(client *godo.Client) *CertificateMCPResource {
	return &CertificateMCPResource{
		client: client,
	}
}

func (c *CertificateMCPResource) getCertificateResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		CertificateURI+"{id}",
		"Certificate",
		mcp.WithTemplateDescription("Returns certificate information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (c *CertificateMCPResource) handleGetCertificateResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	certID, err := common.ExtractStringIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid certificate URI: %w", err)
	}

	certificate, _, err := c.client.Certificates.Get(ctx, certID)
	if err != nil {
		return nil, fmt.Errorf("error fetching certificate: %w", err)
	}

	jsonData, err := json.MarshalIndent(certificate, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing certificate: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (c *CertificateMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc {
	return map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc{
		c.getCertificateResourceTemplate(): c.handleGetCertificateResource,
	}
}
