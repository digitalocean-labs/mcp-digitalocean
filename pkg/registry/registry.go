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

// supportedServices maps service names to their default category.
var supportedServices = map[string]string{
	"apps":        "basic",
	"networking":  "basic",
	"droplets":    "basic",
	"accounts":    "basic",
	"spaces":      "basic",
	"databases":   "basic",
	"marketplace": "basic",
	"insights":    "basic",
	"doks":        "basic",
}

// parseServiceFilters parses service specifications with optional categories.
// Format: "service" or "service:category" or "service:cat1,service:cat2"
// If no category specified, uses "basic" as default.
// Use "service:all" to load all tools for a service.
func parseServiceFilters(services []string) map[string][]string {
	result := make(map[string][]string)

	for _, svc := range services {
		if idx := strings.Index(svc, ":"); idx != -1 {
			serviceName := svc[:idx]
			category := svc[idx+1:]
			if category != "" {
				result[serviceName] = append(result[serviceName], category)
			}
		} else {
			// No category specified - use default "basic"
			if _, exists := result[svc]; !exists {
				result[svc] = []string{"basic"}
			}
		}
	}

	return result
}

// hasCategory checks if a category is in the list, or if "all" is specified.
func hasCategory(categories []string, cat string) bool {
	for _, c := range categories {
		if c == cat || c == "all" {
			return true
		}
	}
	return false
}

// registerAppTools registers app platform tools.
// Categories: basic, all
func registerAppTools(s *server.MCPServer, getClient getClientFn, categories []string) error {
	appTools, err := apps.NewAppPlatformTool(getClient)
	if err != nil {
		return fmt.Errorf("failed to create apps tool: %w", err)
	}
	// Apps currently has no sub-categories, always load all
	s.AddTools(appTools.Tools()...)
	return nil
}

// registerCommonTools registers common tools (always loaded).
func registerCommonTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(common.NewRegionTools(getClient).Tools()...)
	return nil
}

// registerDropletTools registers droplet tools.
// Categories: basic, actions, images, sizes, all
func registerDropletTools(s *server.MCPServer, getClient getClientFn, categories []string) error {
	if hasCategory(categories, "basic") {
		s.AddTools(droplet.NewDropletTool(getClient).Tools()...)
	}
	if hasCategory(categories, "actions") {
		s.AddTools(droplet.NewDropletActionsTool(getClient).Tools()...)
	}
	if hasCategory(categories, "images") {
		s.AddTools(droplet.NewImageTool(getClient).Tools()...)
		s.AddTools(droplet.NewImageActionsTool(getClient).Tools()...)
	}
	if hasCategory(categories, "sizes") {
		s.AddTools(droplet.NewSizesTool(getClient).Tools()...)
	}
	return nil
}

// registerNetworkingTools registers networking tools.
// Categories: basic (lb), lb, firewall, dns, vpc, ip, all
func registerNetworkingTools(s *server.MCPServer, getClient getClientFn, categories []string) error {
	// "basic" for networking means load balancers (most common use case)
	if hasCategory(categories, "basic") || hasCategory(categories, "lb") {
		s.AddTools(networking.NewLoadBalancersTool(getClient).Tools()...)
	}
	if hasCategory(categories, "firewall") {
		s.AddTools(networking.NewFirewallTool(getClient).Tools()...)
	}
	if hasCategory(categories, "dns") {
		s.AddTools(networking.NewDomainsTool(getClient).Tools()...)
		s.AddTools(networking.NewCertificateTool(getClient).Tools()...)
	}
	if hasCategory(categories, "vpc") {
		s.AddTools(networking.NewVPCTool(getClient).Tools()...)
		s.AddTools(networking.NewVPCPeeringTool(getClient).Tools()...)
	}
	if hasCategory(categories, "ip") {
		s.AddTools(networking.NewReservedIPTool(getClient).Tools()...)
		s.AddTools(networking.NewBYOIPPrefixTool(getClient).Tools()...)
	}
	return nil
}

// registerAccountTools registers account tools.
// Categories: basic (info), info, billing, keys, actions, all
func registerAccountTools(s *server.MCPServer, getClient getClientFn, categories []string) error {
	if hasCategory(categories, "basic") || hasCategory(categories, "info") {
		s.AddTools(account.NewAccountTools(getClient).Tools()...)
	}
	if hasCategory(categories, "billing") {
		s.AddTools(account.NewBalanceTools(getClient).Tools()...)
		s.AddTools(account.NewBillingTools(getClient).Tools()...)
		s.AddTools(account.NewInvoiceTools(getClient).Tools()...)
	}
	if hasCategory(categories, "keys") {
		s.AddTools(account.NewKeysTool(getClient).Tools()...)
	}
	if hasCategory(categories, "actions") {
		s.AddTools(account.NewActionTools(getClient).Tools()...)
	}
	return nil
}

// registerSpacesTools registers spaces/object storage tools.
// Categories: basic (keys), keys, cdn, all
func registerSpacesTools(s *server.MCPServer, getClient getClientFn, categories []string) error {
	if hasCategory(categories, "basic") || hasCategory(categories, "keys") {
		s.AddTools(spaces.NewSpacesKeysTool(getClient).Tools()...)
	}
	if hasCategory(categories, "cdn") {
		s.AddTools(spaces.NewCDNTool(getClient).Tools()...)
	}
	return nil
}

// registerMarketplaceTools registers marketplace tools.
// Categories: basic, all (marketplace has limited tools)
func registerMarketplaceTools(s *server.MCPServer, getClient getClientFn, categories []string) error {
	s.AddTools(marketplace.NewOneClickTool(getClient).Tools()...)
	return nil
}

// registerInsightsTools registers monitoring/insights tools.
// Categories: basic (uptime), uptime, alerts, all
func registerInsightsTools(s *server.MCPServer, getClient getClientFn, categories []string) error {
	if hasCategory(categories, "basic") || hasCategory(categories, "uptime") {
		s.AddTools(insights.NewUptimeTool(getClient).Tools()...)
		s.AddTools(insights.NewUptimeCheckAlertTool(getClient).Tools()...)
	}
	if hasCategory(categories, "alerts") {
		s.AddTools(insights.NewAlertPolicyTool(getClient).Tools()...)
	}
	return nil
}

// registerDOKSTools registers Kubernetes tools.
// Categories: basic, all (DOKS has single tool set)
func registerDOKSTools(s *server.MCPServer, getClient getClientFn, categories []string) error {
	s.AddTools(doks.NewDoksTool(getClient).Tools()...)
	return nil
}

// registerDatabasesTools registers database tools.
// Categories: basic (cluster), cluster, postgresql, mysql, mongodb, redis, kafka, opensearch, users, firewall, all
func registerDatabasesTools(s *server.MCPServer, getClient getClientFn, categories []string) error {
	if hasCategory(categories, "basic") || hasCategory(categories, "cluster") {
		s.AddTools(dbaas.NewClusterTool(getClient).Tools()...)
	}
	if hasCategory(categories, "postgresql") {
		s.AddTools(dbaas.NewPostgreSQLTool(getClient).Tools()...)
	}
	if hasCategory(categories, "mysql") {
		s.AddTools(dbaas.NewMysqlTool(getClient).Tools()...)
	}
	if hasCategory(categories, "mongodb") {
		s.AddTools(dbaas.NewMongoTool(getClient).Tools()...)
	}
	if hasCategory(categories, "redis") {
		s.AddTools(dbaas.NewRedisTool(getClient).Tools()...)
	}
	if hasCategory(categories, "kafka") {
		s.AddTools(dbaas.NewKafkaTool(getClient).Tools()...)
	}
	if hasCategory(categories, "opensearch") {
		s.AddTools(dbaas.NewOpenSearchTool(getClient).Tools()...)
	}
	if hasCategory(categories, "users") {
		s.AddTools(dbaas.NewUserTool(getClient).Tools()...)
	}
	if hasCategory(categories, "firewall") {
		s.AddTools(dbaas.NewFirewallTool(getClient).Tools()...)
	}
	return nil
}

// Register registers tools for the specified services with the MCP server.
// Services can be specified with categories: "service:category" (e.g., "droplets:basic").
// If no category is specified, "basic" is used as default.
// Use "service:all" to load all tools for a service.
func Register(logger *slog.Logger, s *server.MCPServer, getClient getClientFn, servicesToActivate ...string) error {
	if len(servicesToActivate) == 0 {
		logger.Warn("no services specified, loading basic tools for all services")
		for k := range supportedServices {
			servicesToActivate = append(servicesToActivate, k)
		}
	}

	serviceFilters := parseServiceFilters(servicesToActivate)

	for svc, categories := range serviceFilters {
		logger.Debug(fmt.Sprintf("Registering tools for service: %s, categories: %v", svc, categories))

		if _, ok := supportedServices[svc]; !ok {
			return fmt.Errorf("unsupported service: %s, supported services are: %v", svc, setToString(supportedServices))
		}

		var err error
		switch svc {
		case "apps":
			err = registerAppTools(s, getClient, categories)
		case "networking":
			err = registerNetworkingTools(s, getClient, categories)
		case "droplets":
			err = registerDropletTools(s, getClient, categories)
		case "accounts":
			err = registerAccountTools(s, getClient, categories)
		case "spaces":
			err = registerSpacesTools(s, getClient, categories)
		case "databases":
			err = registerDatabasesTools(s, getClient, categories)
		case "marketplace":
			err = registerMarketplaceTools(s, getClient, categories)
		case "insights":
			err = registerInsightsTools(s, getClient, categories)
		case "doks":
			err = registerDOKSTools(s, getClient, categories)
		}
		if err != nil {
			return fmt.Errorf("failed to register %s tools: %w", svc, err)
		}
	}

	// Common tools always registered
	if err := registerCommonTools(s, getClient); err != nil {
		return fmt.Errorf("failed to register common tools: %w", err)
	}

	return nil
}

func setToString(set map[string]string) string {
	var result []string
	for key := range set {
		result = append(result, key)
	}
	return strings.Join(result, ",")
}
