package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CertificateTool provides tools for managing certificates
type CertificateTool struct {
	client *godo.Client
}

// NewCertificateTool creates a new certificate tool
func NewCertificateTool(client *godo.Client) *CertificateTool {
	return &CertificateTool{
		client: client,
	}
}

// createCertificate creates a new certificate
func (c *CertificateTool) createCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	certificate, _, err := c.client.Certificates.Create(ctx, certRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCert, err := json.MarshalIndent(certificate, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonCert)), nil
}

// deleteCertificate deletes a certificate
func (c *CertificateTool) deleteCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	certID := req.GetArguments()["ID"].(string)
	_, err := c.client.Certificates.Delete(ctx, certID)
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

	certificate, _, err := c.client.Certificates.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCert, err := json.MarshalIndent(certificate, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonCert)), nil
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
	certs, _, err := c.client.Certificates.List(ctx, &godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCerts, err := json.MarshalIndent(certs, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonCerts)), nil
}

// Tools returns a list of certificate tools
func (c *CertificateTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: c.getCertificate,
			Tool: mcp.NewTool("digitalocean-certificate-get",
				mcp.WithDescription("Get certificate information by ID"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the certificate")),
			),
		},
		{
			Handler: c.listCertificates,
			Tool: mcp.NewTool("digitalocean-certificate-list",
				mcp.WithDescription("List certificates with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: c.createCertificate,
			Tool: mcp.NewTool("digitalocean-certificate-create",
				mcp.WithDescription("Create a new certificate"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the certificate")),
				mcp.WithString("PrivateKey", mcp.Required(), mcp.Description("Private key for the certificate")),
				mcp.WithString("LeafCertificate", mcp.Required(), mcp.Description("Leaf certificate")),
				mcp.WithString("CertificateChain", mcp.Required(), mcp.Description("Certificate chain")),
			),
		},
		{
			Handler: c.deleteCertificate,
			Tool: mcp.NewTool("digitalocean-certificate-delete",
				mcp.WithDescription("Delete a certificate"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the certificate to delete")),
			),
		},
	}
}
