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
	dedicatedinference "mcp-digitalocean/pkg/registry/dedicated-inference"
	"mcp-digitalocean/pkg/registry/docr"
	"mcp-digitalocean/pkg/registry/docs"
	"mcp-digitalocean/pkg/registry/doks"
	"mcp-digitalocean/pkg/registry/droplet"
	"mcp-digitalocean/pkg/registry/functions"
	"mcp-digitalocean/pkg/registry/genai"
	genaibi "mcp-digitalocean/pkg/registry/genai-batchinference"
	genaicm "mcp-digitalocean/pkg/registry/genai-custom-models"
	genaiinferencerouter "mcp-digitalocean/pkg/registry/genai-inferencerouter"
	inferencemodelcatalog "mcp-digitalocean/pkg/registry/inference-modelcatalog"
	"mcp-digitalocean/pkg/registry/insights"
	"mcp-digitalocean/pkg/registry/marketplace"
	"mcp-digitalocean/pkg/registry/networking"
	"mcp-digitalocean/pkg/registry/nfs"
	"mcp-digitalocean/pkg/registry/spaces"
	"mcp-digitalocean/pkg/registry/volumes"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
)

type getClientFn func(ctx context.Context) (*godo.Client, error)

// supportedServices is a set of services that we support in this MCP server.
var supportedServices = map[string]struct{}{
	"apps":                   {},
	"networking":             {},
	"droplets":               {},
	"accounts":               {},
	"spaces":                 {},
	"databases":              {},
	"marketplace":            {},
	"dedicated-inference":    {},
	"inference-modelcatalog": {},
	"genai-evaluation":       {},
	"genai-custom-models":    {},
	"genai-batchinference":   {},
	"genai-inferencerouter":  {},
	"insights":               {},
	"doks":                   {},
	"docr":                   {},
	"docs":                   {},
	"volumes":                {},
	"functions":              {},
	"nfs":                    {},
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
	s.AddTools(droplet.NewImageTool(getClient).Tools()...)
	s.AddTools(droplet.NewImageActionsTool(getClient).Tools()...)
	s.AddTools(droplet.NewSizesTool(getClient).Tools()...)
	return nil
}

// registerNetworkingTools registers the networking tools with the MCP server.
func registerNetworkingTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(networking.NewCertificateTool(getClient).Tools()...)
	s.AddTools(networking.NewDomainsTool(getClient).Tools()...)
	s.AddTools(networking.NewFirewallTool(getClient).Tools()...)
	s.AddTools(networking.NewLoadBalancersTool(getClient).Tools()...)
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

// registerDedicatedInferenceTools registers the Dedicated Inference tools with the MCP server.
func registerDedicatedInferenceTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(dedicatedinference.NewDedicatedInferenceTool(getClient).Tools()...)
	return nil
}

// registerModelCatalogTools registers the model catalog tools with the MCP server.
func registerModelCatalogTools(s *server.MCPServer, getClient getClientFn) error {
	modelTool := inferencemodelcatalog.NewModelTool(getClient)
	s.AddTools(modelTool.Tools()...)
	s.AddPrompts(modelTool.Prompts()...)
	return nil
}

// registerGenAIEvaluationTools registers the GenAI evaluation tools with the MCP server.
func registerGenAIEvaluationTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(genai.NewEvaluationTool(getClient).Tools()...)
	s.AddTools(genai.NewModelEvaluationTool(getClient).Tools()...)
	return nil
}

// registerGenAICustomModelsTools registers the GenAI custom models tools with the MCP server.
func registerGenAICustomModelsTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(genaicm.NewCustomModelsTool(getClient).Tools()...)
	return nil
}

// registerGenAIBatchInferenceTools registers the GenAI batch inference tools with the MCP server.
func registerGenAIBatchInferenceTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(genaibi.NewBatchInferenceTool(getClient).Tools()...)
	return nil
}

// registerGenAIInferenceRouterTools registers the GenAI model router tools with the MCP server.
func registerGenAIInferenceRouterTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(genaiinferencerouter.NewRouterTool(getClient).Tools()...)
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

// registerDocsTools registers the documentation tools with the MCP server.
// Unlike other services, docs tools do not require a DigitalOcean API client
// since they access public documentation.
func registerDocsTools(s *server.MCPServer) error {
	s.AddTools(docs.NewDocsTool().Tools()...)
	return nil
}

func registerDOCRTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(docr.NewRegistryTool(getClient).Tools()...)
	s.AddTools(docr.NewRepositoryTool(getClient).Tools()...)
	s.AddTools(docr.NewGarbageCollectionTool(getClient).Tools()...)
	s.AddTools(docr.NewSubscriptionTool(getClient).Tools()...)
	return nil
}

func registerFunctionsTools(s *server.MCPServer, getClient getClientFn) error {
	resolver := functions.NewOWResolver(getClient)
	s.AddTools(functions.NewNamespaceTool(getClient).Tools()...)
	s.AddTools(functions.NewTriggerTool(getClient).Tools()...)
	s.AddTools(functions.NewActionTool(resolver).Tools()...)
	s.AddTools(functions.NewPackageTool(resolver).Tools()...)
	s.AddTools(functions.NewActivationTool(resolver).Tools()...)
	s.AddTools(functions.NewDeploymentGuideTool().Tools()...)
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

func registerVolumesTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(volumes.NewVolumeTool(getClient).Tools()...)
	s.AddTools(volumes.NewVolumeActionsTool(getClient).Tools()...)
	return nil
}

func registerNfsTools(s *server.MCPServer, getClient getClientFn) error {
	s.AddTools(nfs.NewNfsTool(getClient).Tools()...)
	s.AddTools(nfs.NewNfsActionsTool(getClient).Tools()...)
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
		case "dedicated-inference":
			if err := registerDedicatedInferenceTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register dedicated-inference tools: %w", err)
			}
		case "inference-modelcatalog":
			if err := registerModelCatalogTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register inference-modelcatalog tools: %w", err)
			}
		case "genai-evaluation":
			if err := registerGenAIEvaluationTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register genai-evaluation tools: %w", err)
			}
		case "genai-custom-models":
			if err := registerGenAICustomModelsTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register genai-custom-models tools: %w", err)
			}
		case "genai-batchinference":
			if err := registerGenAIBatchInferenceTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register genai-batchinference tools: %w", err)
			}
		case "genai-inferencerouter":
			if err := registerGenAIInferenceRouterTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register genai-inferencerouter tools: %w", err)
			}
		case "insights":
			if err := registerInsightsTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register insights tools: %w", err)
			}
		case "doks":
			if err := registerDOKSTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register DOKS tools: %w", err)
			}
		case "docr":
			if err := registerDOCRTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register DOCR tools: %w", err)
			}
		case "docs":
			if err := registerDocsTools(s); err != nil {
				return fmt.Errorf("failed to register docs tools: %w", err)
			}
		case "volumes":
			if err := registerVolumesTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register volumes tools: %w", err)
			}
		case "functions":
			if err := registerFunctionsTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register functions tools: %w", err)
			}
		case "nfs":
			if err := registerNfsTools(s, getClient); err != nil {
				return fmt.Errorf("failed to register nfs tools: %w", err)
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
