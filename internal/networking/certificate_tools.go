package networking

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
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
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

// getCertificate retrieves a certificate by ID
func (c *CertificateTool) getCertificate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	certID := req.GetArguments()["ID"].(string)
	certificate, _, err := c.client.Certificates.Get(ctx, certID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCert, err := json.MarshalIndent(certificate, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(jsonCert)), nil
}

// Tools returns a list of certificate tools
func (c *CertificateTool) Tools() []server.ServerTool {
	return []server.ServerTool{
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
		{
			Handler: c.getCertificate,
			Tool: mcp.NewTool("digitalocean-certificate-get",
				mcp.WithDescription("Get details of a certificate"),
				mcp.WithString("ID", mcp.Required(), mcp.Description("ID of the certificate to retrieve")),
			),
		},
	}
}
