package registry

import (
	"mcp-digitalocean/internal/resources"
	"mcp-digitalocean/internal/tools"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterTools(s *server.MCPServer, c *godo.Client) {
	s.AddTools(tools.NewDropletTool(c).Tools()...)
	s.AddTools(tools.NewCDNTool(c).Tools()...)
	s.AddTools(tools.NewCertificateTool(c).Tools()...)
	s.AddTools(tools.NewDomainsTool(c).Tools()...)
	s.AddTools(tools.NewFirewallTool(c).Tools()...)
	s.AddTools(tools.NewKeysTool(c).Tools()...)
	s.AddTools(tools.NewReservedIPTool(c).Tools()...)
	s.AddTools(tools.NewPartnerAttachmentTool(c).Tools()...)
	s.AddTools(tools.NewVPCTool(c).Tools()...)
	s.AddTools(tools.NewDatabaseTool(c).Tools()...)
}

func RegisterResources(s *server.MCPServer, c *godo.Client) {
	// Register droplet resource and resource templates
	dropletResource := resources.NewDropletMCPResource(c)
	for template, handler := range dropletResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register sizes resource
	sizesResource := resources.NewSizesMCPResource(c)
	for resource, handler := range sizesResource.Resources() {
		s.AddResource(resource, handler)
	}

	// Register images resources and templates
	imageResource := resources.NewImagesMCPResource(c)
	for resource, handler := range imageResource.Resources() {
		s.AddResource(resource, handler)
	}
	for template, handler := range imageResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register account resource
	for resource, handler := range resources.NewAccountMCPResource(c).Resources() {
		s.AddResource(resource, handler)
	}

	// Register action resource
	for template, handler := range resources.NewActionMCPResource(c).Resources() {
		s.AddResourceTemplate(template, handler)
	}

	// Register balance resource
	for resource, handler := range resources.NewBalanceMCPResource(c).Resources() {
		s.AddResource(resource, handler)
	}

	// Register billing resource
	for template, handler := range resources.NewBillingMCPResource(c).ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register CDN resource and resource templates
	cdnResource := resources.NewCDNResource(c)
	for template, handler := range cdnResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register certificate resource and resource templates
	certificateResource := resources.NewCertificateMCPResource(c)
	for template, handler := range certificateResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register domains resource
	domainsResource := resources.NewDomainsMCPResource(c)
	for template, handler := range domainsResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register firewall resource
	firewallResource := resources.NewFirewallMCPResource(c)
	for template, handler := range firewallResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register keys resource
	keysResource := resources.NewKeysMCPResource(c)
	for template, handler := range keysResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register regions resource
	regionsResource := resources.NewRegionsMCPResource(c)
	for resource, handler := range regionsResource.Resources() {
		s.AddResource(resource, handler)
	}

	// Register reserved IP resources
	reservedIPResource := resources.NewReservedIPResource(c)
	for template, handler := range reservedIPResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	partnerAttachmentResource := resources.NewPartnerAttachmentMCPResource(c)
	for template, handler := range partnerAttachmentResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register invoices resource
	invoicesResource := resources.NewInvoicesMCPResource(c)
	for resource, handler := range invoicesResource.Resources() {
		s.AddResource(resource, handler)
	}

	// Register VPC resource
	vpcResource := resources.NewVPCMCPResource(c)
	for template, handler := range vpcResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}
}
