package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// LoadBalancersTool provides load balancer management tools
type LoadBalancersTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewLoadBalancersTool creates a new LoadBalancersTool
func NewLoadBalancersTool(client func(ctx context.Context) (*godo.Client, error)) *LoadBalancersTool {
	return &LoadBalancersTool{
		client: client,
	}
}

func parseForwardingRules(rules []any) ([]godo.ForwardingRule, *mcp.CallToolResult) {
	forwardingRules := []godo.ForwardingRule{}
	for _, ruleData := range rules {
		rule, ok := ruleData.(map[string]any)
		if !ok {
			return nil, mcp.NewToolResultError("invalid rule format")
		}

		entryProtocol, ok := rule["EntryProtocol"].(string)
		if !ok {
			return nil, mcp.NewToolResultError("EntryProtocol must be a string")
		}
		entryPort, ok := rule["EntryPort"].(float64)
		if !ok {
			return nil, mcp.NewToolResultError("EntryPort must be a number")
		}
		targetProtocol, ok := rule["TargetProtocol"].(string)
		if !ok {
			return nil, mcp.NewToolResultError("TargetProtocol must be a string")
		}
		targetPort, ok := rule["TargetPort"].(float64)
		if !ok {
			return nil, mcp.NewToolResultError("TargetPort must be a number")
		}
		// set tlsPassthrough to false if not provided
		tlsPassthrough := false
		if val, ok := rule["TlsPassthrough"].(bool); ok {
			tlsPassthrough = val
		}
		// set the certificate id to empty string if not provided
		certificateID := ""
		if val, ok := rule["CertificateID"].(string); ok {
			certificateID = val
		}

		forwardingRule := godo.ForwardingRule{
			EntryProtocol:  entryProtocol,
			EntryPort:      int(entryPort),
			TargetProtocol: targetProtocol,
			TargetPort:     int(targetPort),
			TlsPassthrough: tlsPassthrough,
			CertificateID:  certificateID,
		}
		forwardingRules = append(forwardingRules, forwardingRule)
	}
	return forwardingRules, nil
}

func (l *LoadBalancersTool) createLoadBalancer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	name, ok := args["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}
	// Optional arguments
	lbType, _ := args["Type"].(string)
	network, _ := args["Network"].(string)
	sizeUnit, _ := args["SizeUnit"].(float64)
	networkStack, _ := args["NetworkStack"].(string)
	projectID, _ := args["ProjectID"].(string)

	lbr := &godo.LoadBalancerRequest{
		Name:         name,
		SizeUnit:     uint32(sizeUnit),
		Type:         lbType,
		Network:      network,
		NetworkStack: networkStack,
		ProjectID:    projectID,
	}

	// Global load balancer arguments
	if lbType == "GLOBAL" {
		targetLoadBalancerIDs, ok := args["TargetLoadBalancerIDs"].([]string)
		if ok && len(targetLoadBalancerIDs) > 0 {
			lbr.TargetLoadBalancerIDs = targetLoadBalancerIDs
		}

		// Parse GLB settings
		if glbSettings, ok := args["GLBSettings"].(map[string]any); ok && len(glbSettings) > 0 {
			targetProtocol, _ := glbSettings["TargetProtocol"].(string)
			targetPort, _ := glbSettings["TargetPort"].(float64)

			cdnSettings := &godo.CDNSettings{}
			if cdn, ok := glbSettings["CDN"].(map[string]any); ok {
				if isEnabled, ok := cdn["IsEnabled"].(bool); ok {
					cdnSettings.IsEnabled = isEnabled
				}
			}

			rp := make(map[string]uint32)
			if regionPriorities, ok := glbSettings["RegionPriorities"].(map[string]any); ok {
				for k, v := range regionPriorities {
					if val, ok := v.(float64); ok {
						rp[k] = uint32(val)
					}
				}
			}

			failoverThreshold, _ := glbSettings["FailoverThreshold"].(float64)

			lbr.GLBSettings = &godo.GLBSettings{
				TargetProtocol:    targetProtocol,
				TargetPort:        uint32(targetPort),
				CDN:               cdnSettings,
				RegionPriorities:  rp,
				FailoverThreshold: uint32(failoverThreshold),
			}
		}
	} else {
		// Regional load balancer arguments
		region, ok := args["Region"].(string)
		if !ok || region == "" {
			return mcp.NewToolResultError("Region is required for REGIONAL and REGIONAL_NETWORK load balancers"), nil
		}
		lbr.Region = region

		// Parse forwarding rules
		forwardingRules := []godo.ForwardingRule{}
		if rules, ok := args["ForwardingRules"]; ok && rules != nil {
			var err *mcp.CallToolResult
			forwardingRules, err = parseForwardingRules(rules.([]any))
			if err != nil {
				return err, nil
			}
		}

		if len(forwardingRules) == 0 {
			return mcp.NewToolResultError("At least one forwarding rule must be provided"), nil
		}

		lbr.ForwardingRules = forwardingRules
	}

	// Target identifiers are optional but only one can be provided
	tag, _ := args["Tag"].(string)
	dropletIDs, _ := args["DropletIDs"].([]any)
	if len(dropletIDs) > 0 && tag != "" {
		return mcp.NewToolResultError("Only one target identifier (e.g. tag, droplets) can be specified"), nil
	}

	// If droplet IDs are provided, make request with them
	if len(dropletIDs) > 0 {
		// Parse droplet IDs as ints
		intDropletIDs := make([]int, len(dropletIDs))
		for i, id := range dropletIDs {
			if did, ok := id.(float64); ok {
				intDropletIDs[i] = int(did)
			}
		}
		lbr.DropletIDs = intDropletIDs
	}
	// If tag is provided, make request with it
	if tag != "" {
		lbr.Tag = tag
	}

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	lb, _, err := client.LoadBalancers.Create(ctx, lbr)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonLB, err := json.MarshalIndent(lb, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonLB)), nil
}

func (l *LoadBalancersTool) deleteLoadBalancer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lbID := req.GetArguments()["LoadBalancerID"].(string)

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.LoadBalancers.Delete(ctx, lbID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Load Balancer deleted successfully"), nil
}

func (l *LoadBalancersTool) deleteLoadBalancerCache(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lbID := req.GetArguments()["LoadBalancerID"].(string)

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.LoadBalancers.PurgeCache(ctx, lbID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Load Balancer cache deleted successfully"), nil
}

func (l *LoadBalancersTool) getLoadBalancer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lbID, ok := req.GetArguments()["LoadBalancerID"].(string)
	if !ok || lbID == "" {
		return mcp.NewToolResultError("LoadBalancer ID is required"), nil
	}

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	lb, _, err := client.LoadBalancers.Get(ctx, lbID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonLB, err := json.MarshalIndent(lb, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonLB)), nil
}

func (l *LoadBalancersTool) listLoadBalancers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page, ok := req.GetArguments()["Page"].(float64)
	if !ok {
		page = 1
	}
	perPage, ok := req.GetArguments()["PerPage"].(float64)
	if !ok {
		perPage = float64(20)
	}
	opt := &godo.ListOptions{
		Page:    int(page),
		PerPage: int(perPage),
	}

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	lbs, _, err := client.LoadBalancers.List(ctx, opt)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonLBs, err := json.MarshalIndent(lbs, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonLBs)), nil
}

func (l *LoadBalancersTool) addDroplets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lbID, ok := req.GetArguments()["LoadBalancerID"].(string)
	if !ok || lbID == "" {
		return mcp.NewToolResultError("Load Balancer ID is required"), nil
	}
	dropletIDs, ok := req.GetArguments()["DropletIDs"].([]any)
	if !ok || len(dropletIDs) == 0 {
		return mcp.NewToolResultError("Droplet IDs are required"), nil
	}
	dIDs := make([]int, len(dropletIDs))
	for i, id := range dropletIDs {
		if did, ok := id.(float64); ok {
			dIDs[i] = int(did)
		}
	}

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.LoadBalancers.AddDroplets(ctx, lbID, dIDs...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Droplets added successfully"), nil
}

func (l *LoadBalancersTool) removeDroplets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lbID, ok := req.GetArguments()["LoadBalancerID"].(string)
	if !ok || lbID == "" {
		return mcp.NewToolResultError("Load Balancer ID is required"), nil
	}
	dropletIDs, ok := req.GetArguments()["DropletIDs"].([]any)
	if !ok || len(dropletIDs) == 0 {
		return mcp.NewToolResultError("Droplet IDs are required"), nil
	}
	dIDs := make([]int, len(dropletIDs))
	for i, id := range dropletIDs {
		if did, ok := id.(float64); ok {
			dIDs[i] = int(did)
		}
	}

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.LoadBalancers.RemoveDroplets(ctx, lbID, dIDs...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Droplets removed successfully"), nil
}

func (l *LoadBalancersTool) updateLoadBalancer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	lbID, ok := args["LoadBalancerID"].(string)
	if !ok || lbID == "" {
		return mcp.NewToolResultError("Load Balancer ID is required"), nil
	}
	name, ok := args["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}
	// Type is required for update with MCP to validate type-specific required arguments
	// For example, Region is required for REGIONAL load balancers
	// and GLBSettings is required for GLOBAL load balancers
	// If Type is not provided and the existing load balancer is a GLOBAL load balancer
	// and Region is not provided
	// then api returns an error even though region is not required for GLOBAL load balancers
	lbType, ok := args["Type"].(string)
	if !ok || lbType == "" {
		return mcp.NewToolResultError("Type is required"), nil
	}

	// Optional arguments
	network, _ := args["Network"].(string)
	sizeUnit, _ := args["SizeUnit"].(float64)
	networkStack, _ := args["NetworkStack"].(string)
	projectID, _ := args["ProjectID"].(string)

	lbr := &godo.LoadBalancerRequest{
		Name:         name,
		SizeUnit:     uint32(sizeUnit),
		Type:         lbType,
		Network:      network,
		NetworkStack: networkStack,
		ProjectID:    projectID,
	}

	if lbType == "GLOBAL" {
		targetLoadBalancerIDs, ok := args["TargetLoadBalancerIDs"].([]string)
		if ok && len(targetLoadBalancerIDs) > 0 {
			lbr.TargetLoadBalancerIDs = targetLoadBalancerIDs
		}

		// Parse GLB settings
		if glbSettings, ok := args["GLBSettings"].(map[string]any); ok && len(glbSettings) > 0 {
			targetProtocol, _ := glbSettings["TargetProtocol"].(string)
			targetPort, _ := glbSettings["TargetPort"].(float64)

			cdnSettings := &godo.CDNSettings{}
			if cdn, ok := glbSettings["CDN"].(map[string]any); ok {
				if isEnabled, ok := cdn["IsEnabled"].(bool); ok {
					cdnSettings.IsEnabled = isEnabled
				}
			}

			rp := make(map[string]uint32)
			if regionPriorities, ok := glbSettings["RegionPriorities"].(map[string]any); ok {
				for k, v := range regionPriorities {
					if val, ok := v.(float64); ok {
						rp[k] = uint32(val)
					}
				}
			}

			failoverThreshold, _ := glbSettings["FailoverThreshold"].(float64)

			lbr.GLBSettings = &godo.GLBSettings{
				TargetProtocol:    targetProtocol,
				TargetPort:        uint32(targetPort),
				CDN:               cdnSettings,
				RegionPriorities:  rp,
				FailoverThreshold: uint32(failoverThreshold),
			}
		}
	} else {
		// Regional load balancer arguments
		region, ok := args["Region"].(string)
		if !ok || region == "" {
			return mcp.NewToolResultError("Region is required for REGIONAL and REGIONAL_NETWORK load balancers"), nil
		}
		lbr.Region = region

		// Parse forwarding rules
		forwardingRules := []godo.ForwardingRule{}
		if rules, ok := args["ForwardingRules"]; ok && rules != nil {
			var err *mcp.CallToolResult
			forwardingRules, err = parseForwardingRules(rules.([]any))
			if err != nil {
				return err, nil
			}
		}
		lbr.ForwardingRules = forwardingRules
	}

	// Target identifiers are optional but only one can be provided
	tag, _ := args["Tag"].(string)
	dropletIDs, _ := args["DropletIDs"].([]any)
	if len(dropletIDs) > 0 && tag != "" {
		return mcp.NewToolResultError("Only one target identifier (e.g. tag, droplets) can be specified"), nil
	}

	// If droplet IDs are provided, make request with them
	if len(dropletIDs) > 0 {
		// Parse droplet IDs as ints
		intDropletIDs := make([]int, len(dropletIDs))
		for i, id := range dropletIDs {
			if did, ok := id.(float64); ok {
				intDropletIDs[i] = int(did)
			}
		}
		lbr.DropletIDs = intDropletIDs
	}
	// If tag is provided, make request with it
	if tag != "" {
		lbr.Tag = tag
	}

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	lb, _, err := client.LoadBalancers.Update(ctx, lbID, lbr)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonLB, err := json.MarshalIndent(lb, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonLB)), nil
}

func (l *LoadBalancersTool) addForwardingRules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lbID, ok := req.GetArguments()["LoadBalancerID"].(string)
	if !ok || lbID == "" {
		return mcp.NewToolResultError("Load Balancer ID is required"), nil
	}

	// Parse forwarding rules
	forwardingRules := []godo.ForwardingRule{}
	if rules, ok := req.GetArguments()["ForwardingRules"]; ok && rules != nil {
		var err *mcp.CallToolResult
		forwardingRules, err = parseForwardingRules(rules.([]any))
		if err != nil {
			return err, nil
		}
	}
	if len(forwardingRules) == 0 {
		return mcp.NewToolResultError("At least one forwarding rule must be provided"), nil
	}

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.LoadBalancers.AddForwardingRules(ctx, lbID, forwardingRules...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Forwarding rules added successfully"), nil
}

func (l *LoadBalancersTool) removeForwardingRules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lbID, ok := req.GetArguments()["LoadBalancerID"].(string)
	if !ok || lbID == "" {
		return mcp.NewToolResultError("Load Balancer ID is required"), nil
	}

	// Parse forwarding rules
	forwardingRules := []godo.ForwardingRule{}
	if rules, ok := req.GetArguments()["ForwardingRules"]; ok && rules != nil {
		var err *mcp.CallToolResult
		forwardingRules, err = parseForwardingRules(rules.([]any))
		if err != nil {
			return err, nil
		}
	}
	if len(forwardingRules) == 0 {
		return mcp.NewToolResultError("At least one forwarding rule must be provided"), nil
	}

	client, err := l.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.LoadBalancers.RemoveForwardingRules(ctx, lbID, forwardingRules...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Forwarding rules removed successfully"), nil
}

func (l *LoadBalancersTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: l.createLoadBalancer,
			Tool: mcp.NewTool("load-balancer-create",
				mcp.WithDescription("Create a new Load Balancer"),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the load balancer")),
				mcp.WithString("Region", mcp.Description("Region slug (e.g., nyc3)")),
				mcp.WithArray("DropletIDs", mcp.Description("IDs of the Droplets assigned to the load balancer")),
				mcp.WithString("Tag", mcp.Description("Droplet tag corresponding to Droplets assigned to the load balancer")),
				mcp.WithArray("ForwardingRules", mcp.Description("Forwarding rules for a load balancer")),
				mcp.WithString("Type", mcp.Description("Type of the load balancer (REGIONAL, REGIONAL_NETWORK, GLOBAL)")),
				mcp.WithString("Network", mcp.Description("Network type of the load balancer (EXTERNAL, INTERNAL)")),
				mcp.WithNumber("SizeUnit", mcp.DefaultNumber(2), mcp.Description("Size of the load balancer in units appropriate to its type")),
				mcp.WithString("NetworkStack", mcp.Description("Network stack of the load balancer (IPV4, DUALSTACK)")),
				mcp.WithString("ProjectID", mcp.Description("Project ID to which the load balancer will be assigned")),
				mcp.WithArray("TargetLoadBalancerIDs", mcp.Description("IDs of the target regional load balancers for a global load balancer")),
				mcp.WithObject("GLBSettings", mcp.Description("Forward configurations for a global load balancer")),
			),
		},
		{
			Handler: l.deleteLoadBalancer,
			Tool: mcp.NewTool("load-balancer-delete",
				mcp.WithDestructiveHintAnnotation(true),
				mcp.WithDescription("Delete a Load Balancer by ID"),
				mcp.WithString("LoadBalancerID", mcp.Required(), mcp.Description("ID of the load balancer")),
			),
		},
		{
			Handler: l.deleteLoadBalancerCache,
			Tool: mcp.NewTool("load-balancer-delete-cache",
				mcp.WithDestructiveHintAnnotation(true),
				mcp.WithDescription("Delete the CDN cache of a global load balancer by ID"),
				mcp.WithString("LoadBalancerID", mcp.Required(), mcp.Description("ID of the load balancer")),
			),
		},
		{
			Handler: l.getLoadBalancer,
			Tool: mcp.NewTool("load-balancer-get",
				mcp.WithDescription("Get a Load Balancer by ID"),
				mcp.WithString("LoadBalancerID", mcp.Required(), mcp.Description("ID of the load balancer")),
			),
		},
		{
			Handler: l.listLoadBalancers,
			Tool: mcp.NewTool("load-balancer-list",
				mcp.WithDescription("List Load Balancers with pagination"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Items per page")),
			),
		},
		{
			Handler: l.addDroplets,
			Tool: mcp.NewTool("load-balancer-add-droplets",
				mcp.WithDescription("Add Droplets to a Load Balancer"),
				mcp.WithString("LoadBalancerID", mcp.Required(), mcp.Description("ID of the load balancer")),
				mcp.WithArray("DropletIDs", mcp.Required(), mcp.Description("IDs of the droplets to add")),
			),
		},
		{
			Handler: l.removeDroplets,
			Tool: mcp.NewTool("load-balancer-remove-droplets",
				mcp.WithDescription("Remove Droplets from a Load Balancer"),
				mcp.WithString("LoadBalancerID", mcp.Required(), mcp.Description("ID of the load balancer")),
				mcp.WithArray("DropletIDs", mcp.Required(), mcp.Description("IDs of the droplets to remove")),
			),
		},
		{
			Handler: l.updateLoadBalancer,
			Tool: mcp.NewTool("load-balancer-update",
				mcp.WithDescription("Update a Load Balancer"),
				mcp.WithString("LoadBalancerID", mcp.Required(), mcp.Description("ID of the load balancer")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the load balancer")),
				mcp.WithString("Region", mcp.Description("Region slug (e.g., nyc3)")),
				mcp.WithArray("DropletIDs", mcp.Description("IDs of the Droplets assigned to the load balancer")),
				mcp.WithString("Tag", mcp.Description("Droplet tag corresponding to Droplets assigned to the load balancer")),
				mcp.WithArray("ForwardingRules", mcp.Description("Forwarding rules for a load balancer")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of the load balancer (REGIONAL, REGIONAL_NETWORK, GLOBAL)")),
				mcp.WithString("Network", mcp.Description("Network type of the load balancer (EXTERNAL, INTERNAL)")),
				mcp.WithNumber("SizeUnit", mcp.DefaultNumber(2), mcp.Description("Size of the load balancer in units appropriate to its type")),
				mcp.WithString("NetworkStack", mcp.Description("Network stack of the load balancer (IPV4, DUALSTACK)")),
				mcp.WithString("ProjectID", mcp.Description("Project ID to which the load balancer will be assigned")),
				mcp.WithArray("TargetLoadBalancerIDs", mcp.Description("IDs of the target regional load balancers for a global load balancer")),
				mcp.WithObject("GLBSettings", mcp.Description("Forward configurations for a global load balancer")),
			),
		},
		{
			Handler: l.addForwardingRules,
			Tool: mcp.NewTool("load-balancer-add-forwarding-rules",
				mcp.WithDescription("Add Forwarding Rules to a Load Balancer"),
				mcp.WithString("LoadBalancerID", mcp.Required(), mcp.Description("ID of the load balancer")),
				mcp.WithArray("ForwardingRules", mcp.Required(), mcp.Description("Forwarding rules to add")),
			),
		},
		{
			Handler: l.removeForwardingRules,
			Tool: mcp.NewTool("load-balancer-remove-forwarding-rules",
				mcp.WithDescription("Remove Forwarding Rules from a Load Balancer"),
				mcp.WithString("LoadBalancerID", mcp.Required(), mcp.Description("ID of the load balancer")),
				mcp.WithArray("ForwardingRules", mcp.Required(), mcp.Description("Forwarding rules to remove")),
			),
		},
	}
}
