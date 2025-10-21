package registry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"mcp-digitalocean/pkg/registry/account"
	"mcp-digitalocean/pkg/registry/apps"
	"mcp-digitalocean/pkg/registry/common"
	"mcp-digitalocean/pkg/registry/dbaas"
	"mcp-digitalocean/pkg/registry/doks"
	"mcp-digitalocean/pkg/registry/droplet"
	"mcp-digitalocean/pkg/registry/insights"
	"mcp-digitalocean/pkg/registry/marketplace"
	"mcp-digitalocean/pkg/registry/networking"
	"mcp-digitalocean/pkg/registry/spaces"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
)

type getClientFn func(ctx context.Context) (*godo.Client, error)

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

// registerAppTools registers the app platform tools with the MCP server.
func registerAppTools(s *server.MCPServer, getClient getClientFn) error {
	appTools, err := apps.NewAppPlatformTool(getClient)
	if err != nil {
		return fmt.Errorf("failed to create apps tool: %w", err)
	}

	s.AddTools(appTools.Tools()...)

	return nil
}

// registerCommonTools registers the common tools with the MCP server.
func registerCommonTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(common.NewRegionTools(getClient).Tools()...)

	return nil
}

// registerDropletTools registers the droplet tools with the MCP server.
func registerDropletTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(droplet.NewDropletTool(getClient).Tools()...)
	s.AddTools(droplet.NewDropletActionsTool(getClient).Tools()...)
	s.AddTools(droplet.NewImagesTool(getClient).Tools()...)
	s.AddTools(droplet.NewSizesTool(getClient).Tools()...)
	return nil
}

// registerNetworkingTools registers the networking tools with the MCP server.
func registerNetworkingTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(networking.NewCertificateTool(getClient).Tools()...)
	s.AddTools(networking.NewDomainsTool(getClient).Tools()...)
	s.AddTools(networking.NewFirewallTool(getClient).Tools()...)
	s.AddTools(networking.NewReservedIPTool(getClient).Tools()...)
	s.AddTools(networking.NewBYOIPPrefixTool(getClient).Tools()...)
	// Partner attachments doesn't have much users so this has been disabled
	// s.AddTools(networking.NewPartnerAttachmentTool(c).Tools()...)
	s.AddTools(networking.NewVPCTool(getClient).Tools()...)
	s.AddTools(networking.NewVPCPeeringTool(getClient).Tools()...)
	return nil
}

// registerAccountTools registers the account tools with the MCP server.
func registerAccountTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(account.NewAccountTools(getClient).Tools()...)
	s.AddTools(account.NewActionTools(getClient).Tools()...)
	s.AddTools(account.NewBalanceTools(getClient).Tools()...)
	s.AddTools(account.NewBillingTools(getClient).Tools()...)
	s.AddTools(account.NewInvoiceTools(getClient).Tools()...)
	s.AddTools(account.NewKeysTool(getClient).Tools()...)

	return nil
}

// registerSpacesTools registers the spaces tools and resources with the MCP server.
func registerSpacesTools(s *server.MCPServer, getClient getClientFn) error {
	// Register the tools for spaces keys
	s.AddTools(spaces.NewSpacesKeysTool(getClient).Tools()...)
	s.AddTools(spaces.NewCDNTool(getClient).Tools()...)

	return nil
}

// registerMarketplaceTools registers the marketplace tools with the MCP server.
func registerMarketplaceTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(marketplace.NewOneClickTool(getClient).Tools()...)

	return nil
}

func registerInsightsTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(insights.NewUptimeTool(getClient).Tools()...)
	s.AddTools(insights.NewUptimeCheckAlertTool(getClient).Tools()...)
	s.AddTools(insights.NewAlertPolicyTool(getClient).Tools()...)
	return nil
}

func registerDOKSTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(doks.NewDoksTool(getClient).Tools()...)

	return nil
}

func registerDatabasesTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(dbaas.NewClusterTool(getClient).Tools()...)
	s.AddTools(dbaas.NewFirewallTool(getClient).Tools()...)
	s.AddTools(dbaas.NewKafkaTool(getClient).Tools()...)
	s.AddTools(dbaas.NewMongoTool(getClient).Tools()...)
	s.AddTools(dbaas.NewMysqlTool(getClient).Tools()...)
	s.AddTools(dbaas.NewOpenSearchTool(getClient).Tools()...)
	s.AddTools(dbaas.NewPostgreSQLTool(getClient).Tools()...)
	s.AddTools(dbaas.NewRedisTool(getClient).Tools()...)
	s.AddTools(dbaas.NewUserTool(getClient).Tools()...)

	return nil
}

// Register registers the set of tools for the specified services with the MCP server.
// We either register a subset of tools of the services are specified, or we register all tools if no services are specified.
func Register(logger *slog.Logger, s *server.MCPServer, getClient getClientFn, servicesToActivate ...string) error {
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
			if err := registerAppTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register app tools: %w", err)
			}
		case "networking":
			if err := registerNetworkingTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register networking tools: %w", err)
			}
		case "droplets":
			if err := registerDropletTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register droplets tool: %w", err)
			}
		case "accounts":
			if err := registerAccountTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register account tools: %w", err)
			}
		case "spaces":
			if err := registerSpacesTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register spaces tools: %w", err)
			}
		case "databases":
			if err := registerDatabasesTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register databases tools: %w", err)
			}
		case "marketplace":
			if err := registerMarketplaceTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register marketplace tools: %w", err)
			}
		case "insights":
			if err := registerInsightsTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register insights tools: %w", err)
			}
		case "doks":
			if err := registerDOKSTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register DOKS tools: %w", err)
			}
		default:
			return fmt.Errorf("unsupported service: %s, supported service are: %v", svc, setToString(supportedServices))
		}
	}

	// Common tools are always registered because they provide common functionality for all services such as region resources
	if err := registerCommonTools(s, getClient); err != nil {
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
