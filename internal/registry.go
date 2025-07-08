package internal

import (
	"fmt"
	"log/slog"
	"strings"

	"mcp-digitalocean/internal/account"
	"mcp-digitalocean/internal/apps"
	"mcp-digitalocean/internal/common"
	"mcp-digitalocean/internal/droplet"
	"mcp-digitalocean/internal/networking"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
)

// supportedServices is a set of services that we support in this MCP server.
var supportedServices = map[string]struct{}{
	"apps":       {},
	"networking": {},
	"droplets":   {},
	"accounts":   {},
}

// registerAppTools registers the app platform tools with the MCP server.
func registerAppTools(s *server.MCPServer, c *godo.Client) error {
	appTools, err := apps.NewAppPlatformTool(c)
	if err != nil {
		return fmt.Errorf("failed to create apps tool: %w", err)
	}

	s.AddTools(appTools.Tools()...)

	return nil
}

func registerCommonTools(s *server.MCPServer, c *godo.Client) error {
	for resource, handler := range common.NewRegionMCPResource(c).Resources() {
		s.AddResource(resource, handler)
	}

	return nil
}

// registerDropletTools registers the tools and resources for droplets with the MCP server.
func registerDropletTools(s *server.MCPServer, c *godo.Client) error {
	// Register the tools and resources for droplets
	s.AddTools(droplet.NewDropletTool(c).Tools()...)

	// Register the resources for droplets
	imageResource := droplet.NewImagesMCPResource(c)
	for resource, handler := range imageResource.Resources() {
		s.AddResource(resource, handler)
	}
	for template, handler := range imageResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}
	sizesResource := droplet.NewSizesMCPResource(c)
	for resource, handler := range sizesResource.Resources() {
		s.AddResource(resource, handler)
	}
	dropletResource := droplet.NewDropletMCPResource(c)
	for template, handler := range dropletResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	return nil
}

// registerNetworkingTools registers the networking tools and resources with the MCP server.
func registerNetworkingTools(s *server.MCPServer, c *godo.Client) error {
	s.AddTools(networking.NewCDNTool(c).Tools()...)
	s.AddTools(networking.NewCertificateTool(c).Tools()...)
	s.AddTools(networking.NewDomainsTool(c).Tools()...)
	s.AddTools(networking.NewFirewallTool(c).Tools()...)
	s.AddTools(networking.NewReservedIPTool(c).Tools()...)
	s.AddTools(networking.NewPartnerAttachmentTool(c).Tools()...)
	s.AddTools(networking.NewVPCTool(c).Tools()...)

	// Register the resources for networking
	cdnResource := networking.NewCDNMCPResource(c)
	for template, handler := range cdnResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register certificate resource and resource templates
	certificateResource := networking.NewCertificateMCPResource(c)
	for template, handler := range certificateResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register domains resource
	domainsResource := networking.NewDomainMCPResource(c)
	for template, handler := range domainsResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register firewall resource
	firewallResource := networking.NewFirewallMCPResource(c)
	for template, handler := range firewallResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register reserved IP resources
	reservedIPResource := networking.NewReservedIPMCPResource(c)
	for template, handler := range reservedIPResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	partnerAttachmentResource := networking.NewPartnerAttachmentMCPResource(c)
	for template, handler := range partnerAttachmentResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	// Register VPC resource
	vpcResource := networking.NewVPCMCPResource(c)
	for template, handler := range vpcResource.ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	return nil
}

// registerAccountTools account resource and resource templates
func registerAccountTools(s *server.MCPServer, c *godo.Client) error {
	s.AddTools(account.NewKeysTool(c).Tools()...)

	for resource, handler := range account.NewAccountMCPResource(c).Resources() {
		s.AddResource(resource, handler)
	}
	for resource, handler := range account.NewBalanceMCPResource(c).Resources() {
		s.AddResource(resource, handler)
	}

	for resource, handler := range account.NewInvoicesMCPResource(c).ResourceTemplates() {
		s.AddResourceTemplate(resource, handler)
	}
	for template, handler := range account.NewBillingMCPResource(c).ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}
	for template, handler := range account.NewActionMCPResource(c).ResourcesTemplates() {
		s.AddResourceTemplate(template, handler)
	}
	for template, handler := range account.NewKeyMCPResource(c).ResourceTemplates() {
		s.AddResourceTemplate(template, handler)
	}

	return nil
}

// Register registers the set of tools for the specified services with the MCP server.
// We either register a subset of tools of the services are specified, or we register all tools if no services are specified.
func Register(logger *slog.Logger, s *server.MCPServer, c *godo.Client, servicesToActivate ...string) error {
	if len(servicesToActivate) == 0 {
		logger.Warn("no services specified, loading all supported services")
		for k := range supportedServices {
			servicesToActivate = append(servicesToActivate, k)
		}
	}
	for _, svc := range servicesToActivate {
		logger.Debug(fmt.Sprintf("Registering tool and resources for service: %s", svc))
		switch svc {
		case "apps":
			if err := registerAppTools(s, c); err != nil {
				return fmt.Errorf("failed to register app tools: %w", err)
			}
		case "networking":
			if err := registerNetworkingTools(s, c); err != nil {
				return fmt.Errorf("failed to register networking tools: %w", err)
			}
		case "droplets":
			if err := registerDropletTools(s, c); err != nil {
				return fmt.Errorf("failed to register droplets tool: %w", err)
			}
		case "accounts":
			if err := registerAccountTools(s, c); err != nil {
				return fmt.Errorf("failed to register account tools: %w", err)
			}
		default:
			return fmt.Errorf("unsupported service: %s, supported service are: %v", svc, setToString(supportedServices))
		}
	}

	// Common tools are always registered because they provide common functionality for all services such as region resources
	if err := registerCommonTools(s, c); err != nil {
		return fmt.Errorf("failed to register common tools: %w", err)
	}

	return nil
}

func setToString(set map[string]struct{}) string {
	var result []string
	for key := range set {
		result = append(result, key)
	}

	return strings.Join(result, ",")
}
