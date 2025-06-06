package tools

import (
	"context"
	"encoding/json"

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

// CreateCertificate creates a new certificate
func (c *CertificateTool) CreateCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("Name")
	if err != nil {
		return nil, err
	}
	
	privateKey, err := req.RequireString("PrivateKey")
	if err != nil {
		return nil, err
	}
	
	leafCertificate, err := req.RequireString("LeafCertificate")
	if err != nil {
		return nil, err
	}
	
	certificateChain, err := req.RequireString("CertificateChain")
	if err != nil {
		return nil, err
	}

	certRequest := &godo.CertificateRequest{
		Name:             name,
		PrivateKey:       privateKey,
		LeafCertificate:  leafCertificate,
		CertificateChain: certificateChain,
		Type:             "custom",
	}

	certificate, _, err := c.client.Certificates.Create(ctx, certRequest)
	if err != nil {
		return nil, err
	}

	jsonCert, err := json.MarshalIndent(certificate, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonCert)), nil
}

// DeleteCertificate deletes a certificate
func (c *CertificateTool) DeleteCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	certID, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}
	
	_, err = c.client.Certificates.Delete(ctx, certID)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("Certificate deleted successfully"), nil
}

// GetCertificate retrieves a certificate by ID
func (c *CertificateTool) GetCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	certID, err := req.RequireString("ID")
	if err != nil {
		return nil, err
	}
	
	certificate, _, err := c.client.Certificates.Get(ctx, certID)
	if err != nil {
		return nil, err
	}

	jsonCert, err := json.MarshalIndent(certificate, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonCert)), nil
}

// Tools returns a list of certificate tools
func (c *CertificateTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: c.CreateCertificate,
			Tool: mcp.NewTool("digitalocean-certificate-create",
				mcp.WithDescription("Create a new certificate"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the certificate")),
				mcp.WithString("PrivateKey", mcp.Required(), mcp.Description("Private key for the certificate")),
				mcp.WithString("LeafCertificate", mcp.Required(), mcp.Description("Leaf certificate")),
				mcp.WithString("CertificateChain", mcp.Required(), mcp.Description("Certificate chain")),
			),
		},
		{
			Handler: c.DeleteCertificate,
			Tool: mcp.NewTool("digitalocean-certificate-delete",
				mcp.WithDescription("Delete a certificate"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the certificate to delete")),
			),
		},
		{
			Handler: c.GetCertificate,
			Tool: mcp.NewTool("digitalocean-certificate-get",
				mcp.WithDescription("Get details of a certificate"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the certificate to retrieve")),
			),
		},
	}
}
