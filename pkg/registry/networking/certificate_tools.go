package networking

import (
	"context"
	"fmt"
	"mcp-digitalocean/pkg/response"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mcp-digitalocean/pkg/registry/common"
)

// CertificateTool provides tools for managing certificates
type CertificateTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewCertificateTool creates a new certificate tool
func NewCertificateTool(client func(ctx context.Context) (*godo.Client, error)) *CertificateTool {
	return &CertificateTool{
		client: client,
	}
}

// createCustomCertificate creates a new certificate
func (c *CertificateTool) createCustomCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)
	privateKey := req.GetArguments()["PrivateKey"].(string)
	leafCertificate := req.GetArguments()["LeafCertificate"].(string)
	certificateChain := req.GetArguments()["CertificateChain"].(string)

	certRequest := &godo.CertificateRequest{
		Name:             name,
		PrivateKey:       privateKey,
		LeafCertificate:  leafCertificate,
		CertificateChain: certificateChain,
		Type:             "custom",
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	certificate, _, err := client.Certificates.Create(ctx, certRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCert, err := response.CompactJSON(certificate)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(jsonCert), nil
}

// createLetsEncryptCertificate creates a new LetsEncrypt certificate
func (c *CertificateTool) createLetsEncryptCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetArguments()["Name"].(string)
	dnsNames := req.GetArguments()["DnsNames"].([]any)
	dnsNamesStr := make([]string, len(dnsNames))
	for i, dnsName := range dnsNames {
		dnsNamesStr[i] = dnsName.(string)
	}

	certRequest := &godo.CertificateRequest{
		Name:     name,
		DNSNames: dnsNamesStr,
		Type:     "lets_encrypt",
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	certificate, _, err := client.Certificates.Create(ctx, certRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCert, err := response.CompactJSON(certificate)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(jsonCert), nil
}

// deleteCertificate deletes a certificate
func (c *CertificateTool) deleteCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	certID := req.GetArguments()["ID"].(string)

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Certificates.Delete(ctx, certID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Certificate deleted successfully"), nil
}

// getCertificate fetches certificate information by ID
func (c *CertificateTool) getCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["ID"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Certificate ID is required"), nil
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	certificate, _, err := client.Certificates.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCert, err := response.CompactJSON(certificate)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(jsonCert), nil
}

// listCertificates lists certificates with pagination support
func (c *CertificateTool) listCertificates(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := 1
	perPage := 20
	if v, ok := req.GetArguments()["Page"].(float64); ok && int(v) > 0 {
		page = int(v)
	}
	if v, ok := req.GetArguments()["PerPage"].(float64); ok && int(v) > 0 {
		perPage = int(v)
	}

	client, err := c.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	certs, _, err := client.Certificates.List(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCerts, err := response.CompactJSON(certs)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCerts), nil
}

// Tools returns a list of certificate tools
func (c *CertificateTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: c.getCertificate,
			Tool: mcp.NewTool("certificate-get",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("Get certificate information by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the certificate")),
			),
		},
		{
			Handler: c.listCertificates,
			Tool: mcp.NewTool("certificate-list",
				common.WithHints(common.HintsRead),
				common.WithRisk(common.RiskLow),
				mcp.WithDescription("List certificates with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: c.createCustomCertificate,
			Tool: mcp.NewTool("custom-certificate-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Create a new custom certificate"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the certificate")),
				mcp.WithString("PrivateKey", mcp.Required(), mcp.Description("Private key for the certificate")),
				mcp.WithString("LeafCertificate", mcp.Required(), mcp.Description("Leaf certificate")),
				mcp.WithString("CertificateChain", mcp.Required(), mcp.Description("Certificate chain")),
			),
		},
		{
			Handler: c.createLetsEncryptCertificate,
			Tool: mcp.NewTool("lets-encrypt-certificate-create",
				common.WithHints(common.HintsCreate),
				common.WithRisk(common.RiskMedium),
				mcp.WithDescription("Create a new let's encrypt certificate"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the certificate")),
				mcp.WithArray("DnsNames", mcp.Required(), mcp.Description("DNS names of the certificate"), mcp.Items(map[string]any{
					"type":        "string",
					"description": "DNS name for the certificate, including wildcard domains",
				})),
			),
		},
		{
			Handler: c.deleteCertificate,
			Tool: mcp.NewTool("certificate-delete",
				common.WithHints(common.HintsDelete),
				common.WithRisk(common.RiskHigh),
				mcp.WithDescription("Delete a certificate"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the certificate to delete")),
			),
		},
	}
}
