package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/internal/account"
	"mcp-digitalocean/internal/apps"
	"mcp-digitalocean/internal/cache"
	"mcp-digitalocean/internal/common"
	"mcp-digitalocean/internal/dbaas"
	"mcp-digitalocean/internal/doks"
	"mcp-digitalocean/internal/droplet"
	"mcp-digitalocean/internal/insights"
	"mcp-digitalocean/internal/marketplace"
	"mcp-digitalocean/internal/metrics"
	"mcp-digitalocean/internal/networking"
	"mcp-digitalocean/internal/ratelimit"
	"mcp-digitalocean/internal/spaces"
)

// supportedServices is a set of services that we support in this MCP server.
var supportedServices = map[string]struct{}{
	"apps":        {},
	"networking":  {},
	"droplets":    {},
	"accounts":    {},
	"spaces":      {},
	"databases":   {},
	"marketplace": {},
	"insights":    {},
	"doks":        {},
}

// Components holds the shared components for the MCP server
type Components struct {
	Client      *godo.Client
	Metrics     *metrics.Metrics
	Cache       *cache.Cache
	RateLimiter *ratelimit.RateLimiter
}

// registerAppTools registers the app platform tools with the MCP server.
func registerAppTools(s *server.MCPServer, comp *Components) error {
	appTools, err := apps.NewAppPlatformTool(comp.Client)
	if err != nil {
		return fmt.Errorf("failed to create apps tool: %w", err)
	}

	s.AddTools(appTools.Tools()...)
	return nil
}

// registerCommonTools registers the common tools with the MCP server.
func registerCommonTools(s *server.MCPServer, comp *Components) error {
	s.AddTools(common.NewRegionTools(comp.Client).Tools()...)
	return nil
}

// registerDropletTools registers the droplet tools with the MCP server.
func registerDropletTools(s *server.MCPServer, comp *Components) error {
	s.AddTools(droplet.NewDropletTool(comp.Client).Tools()...)
	s.AddTools(droplet.NewDropletActionsTool(comp.Client).Tools()...)
	s.AddTools(droplet.NewImagesTool(comp.Client).Tools()...)
	s.AddTools(droplet.NewSizesTool(comp.Client).Tools()...)
	return nil
}

// registerNetworkingTools registers the networking tools with the MCP server.
func registerNetworkingTools(s *server.MCPServer, comp *Components) error {
	s.AddTools(networking.NewCertificateTool(comp.Client).Tools()...)
	s.AddTools(networking.NewDomainsTool(comp.Client).Tools()...)
	s.AddTools(networking.NewFirewallTool(comp.Client).Tools()...)
	s.AddTools(networking.NewReservedIPTool(comp.Client).Tools()...)
	// Partner attachments doesn't have much users so this has been disabled
	// s.AddTools(networking.NewPartnerAttachmentTool(comp.Client).Tools()...)
	s.AddTools(networking.NewVPCTool(comp.Client).Tools()...)
	s.AddTools(networking.NewVPCPeeringTool(comp.Client).Tools()...)
	return nil
}

// registerAccountTools registers the account tools with the MCP server.
func registerAccountTools(s *server.MCPServer, comp *Components) error {
	s.AddTools(account.NewAccountTools(comp.Client).Tools()...)
	s.AddTools(account.NewActionTools(comp.Client).Tools()...)
	s.AddTools(account.NewBalanceTools(comp.Client).Tools()...)
	s.AddTools(account.NewBillingTools(comp.Client).Tools()...)
	s.AddTools(account.NewInvoiceTools(comp.Client).Tools()...)
	s.AddTools(account.NewKeysTool(comp.Client).Tools()...)
	return nil
}

// registerSpacesTools registers the spaces tools and resources with the MCP server.
func registerSpacesTools(s *server.MCPServer, comp *Components) error {
	// Register the tools for spaces keys
	s.AddTools(spaces.NewSpacesKeysTool(comp.Client).Tools()...)
	s.AddTools(spaces.NewCDNTool(comp.Client).Tools()...)
	return nil
}

// registerMarketplaceTools registers the marketplace tools with the MCP server.
func registerMarketplaceTools(s *server.MCPServer, comp *Components) error {
	s.AddTools(marketplace.NewOneClickTool(comp.Client).Tools()...)
	return nil
}

func registerInsightsTools(s *server.MCPServer, comp *Components) error {
	s.AddTools(insights.NewUptimeTool(comp.Client).Tools()...)
	s.AddTools(insights.NewUptimeCheckAlertTool(comp.Client).Tools()...)
	s.AddTools(insights.NewAlertPolicyTool(comp.Client).Tools()...)
	return nil
}

func registerDOKSTools(s *server.MCPServer, comp *Components) error {
	s.AddTools(doks.NewDoksTool(comp.Client).Tools()...)
	return nil
}

func registerDatabasesTools(s *server.MCPServer, comp *Components) error {
	s.AddTools(dbaas.NewClusterTool(comp.Client).Tools()...)
	s.AddTools(dbaas.NewFirewallTool(comp.Client).Tools()...)
	s.AddTools(dbaas.NewKafkaTool(comp.Client).Tools()...)
	s.AddTools(dbaas.NewMongoTool(comp.Client).Tools()...)
	s.AddTools(dbaas.NewMysqlTool(comp.Client).Tools()...)
	s.AddTools(dbaas.NewOpenSearchTool(comp.Client).Tools()...)
	s.AddTools(dbaas.NewPostgreSQLTool(comp.Client).Tools()...)
	s.AddTools(dbaas.NewRedisTool(comp.Client).Tools()...)
	s.AddTools(dbaas.NewUserTool(comp.Client).Tools()...)
	return nil
}

// Register registers the set of tools for the specified services with the MCP server.
// We either register a subset of tools of the services are specified, or we register all tools if no services are specified.
// This is the legacy function for backward compatibility.
func Register(logger *slog.Logger, s *server.MCPServer, c *godo.Client, servicesToActivate ...string) error {
	comp := &Components{
		Client:      c,
		Metrics:     metrics.New(),
		Cache:       cache.New(0, false), // Disabled by default for backward compatibility
		RateLimiter: ratelimit.New(0, false), // Disabled by default for backward compatibility
	}
	
	return RegisterWithComponents(logger, s, c, comp.Metrics, comp.Cache, comp.RateLimiter, servicesToActivate...)
}

// RegisterWithComponents registers the set of tools with enhanced components support
func RegisterWithComponents(logger *slog.Logger, s *server.MCPServer, c *godo.Client, 
	metricsCollector *metrics.Metrics, cacheInstance *cache.Cache, rateLimiter *ratelimit.RateLimiter,
	servicesToActivate ...string) error {
	
	comp := &Components{
		Client:      c,
		Metrics:     metricsCollector,
		Cache:       cacheInstance,
		RateLimiter: rateLimiter,
	}
	
	if len(servicesToActivate) == 0 {
		logger.Warn("no services specified, loading all supported services")
		for k := range supportedServices {
			servicesToActivate = append(servicesToActivate, k)
		}
	}
	
	for _, svc := range servicesToActivate {
		logger.Debug("Registering tools and resources for service", "service", svc)
		
		// Wrap service registration with metrics
		err := comp.Metrics.Middleware(svc, func(ctx context.Context) error {
			return registerServiceTools(s, comp, svc)
		})(context.Background())
		
		if err != nil {
			return fmt.Errorf("failed to register %s tools: %w", svc, err)
		}
	}

	// Common tools are always registered because they provide common functionality for all services such as region resources
	if err := registerCommonTools(s, comp); err != nil {
		return fmt.Errorf("failed to register common tools: %w", err)
	}

	return nil
}

// registerServiceTools registers tools for a specific service
func registerServiceTools(s *server.MCPServer, comp *Components, service string) error {
	switch service {
	case "apps":
		return registerAppTools(s, comp)
	case "networking":
		return registerNetworkingTools(s, comp)
	case "droplets":
		return registerDropletTools(s, comp)
	case "accounts":
		return registerAccountTools(s, comp)
	case "spaces":
		return registerSpacesTools(s, comp)
	case "databases":
		return registerDatabasesTools(s, comp)
	case "marketplace":
		return registerMarketplaceTools(s, comp)
	case "insights":
		return registerInsightsTools(s, comp)
	case "doks":
		return registerDOKSTools(s, comp)
	default:
		return fmt.Errorf("unsupported service: %s, supported services are: %v", service, setToString(supportedServices))
	}
}

func setToString(set map[string]struct{}) string {
	var result []string
	for key := range set {
		result = append(result, key)
	}

	return strings.Join(result, ",")
}
