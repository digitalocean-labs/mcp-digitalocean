package doks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	_ "embed"
)

//go:embed spec/cluster-create-schema.json
var clusterCreateSchemaJSON []byte

//go:embed spec/node-pool-create-schema.json
var nodePoolCreateSchemaJSON []byte

type DoksTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewDoksTool creates a new DOKS tool
func NewDoksTool(client func(ctx context.Context) (*godo.Client, error)) *DoksTool {
	return &DoksTool{client: client}
}

// getDoksCluster gets a DOKS cluster
func (d *DoksTool) getDoksCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	cluster, _, err := client.Kubernetes.Get(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// Marshal the response
	clusterJSON, err := json.MarshalIndent(cluster, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(clusterJSON)), nil
}

// ListDOKSClusters lists DOKS clusters
func (d *DoksTool) listDOKSClusters(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get list options from the request
	args := req.GetArguments()

	// Extract page
	page := 1
	if pageFloat, ok := args["Page"].(float64); ok {
		page = int(pageFloat)
	}

	// Extract per page
	perPage := 20
	if perPageFloat, ok := args["PerPage"].(float64); ok {
		perPage = int(perPageFloat)
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	clusters, _, err := client.Kubernetes.List(ctx, &godo.ListOptions{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	// Marshal the response
	clustersJSON, err := json.MarshalIndent(clusters, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(clustersJSON)), nil
}

// CreateDOKSCluster creates a new Kubernetes cluster
func (d *DoksTool) createDOKSCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(req.GetArguments())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	createRequest := &godo.KubernetesClusterCreateRequest{}
	if err := json.Unmarshal(jsonBytes, createRequest); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to parse cluster create request", err), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	cluster, resp, err := client.Kubernetes.Create(ctx, createRequest)
	if err != nil {
		// Include more context in the error message for better debugging
		if resp != nil {
			return mcp.NewToolResultErrorFromErr(fmt.Sprintf("failed to create cluster: (status: %d)", resp.StatusCode), err), nil
		}
		return mcp.NewToolResultErrorFromErr("failed to create cluster", err), nil
	}

	// Marshal the response
	clusterJSON, err := json.MarshalIndent(cluster, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal cluster", err), nil
	}

	return mcp.NewToolResultText(string(clusterJSON)), nil
}

// UpdateDOKSCluster updates a Kubernetes cluster
func (d *DoksTool) updateDOKSCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	// Extract name if provided
	name := ""
	if nameArg, ok := args["Name"].(string); ok {
		name = nameArg
	}

	// Extract maintenance policy if provided
	var maintenancePolicy *godo.KubernetesMaintenancePolicy
	if mpArg, ok := args["MaintenancePolicy"].(map[string]any); ok && mpArg != nil {
		startTime, startTimeOk := mpArg["StartTime"].(string)
		day, dayOk := mpArg["Day"].(string)

		if startTimeOk && dayOk {
			maintenancePolicy = &godo.KubernetesMaintenancePolicy{
				StartTime: startTime,
				Day:       godo.KubernetesMaintenancePolicyDay(getDayFromString(day)),
			}
		} else {
			return mcp.NewToolResultError("MaintenancePolicy requires both 'StartTime' and 'Day' fields"), nil
		}
	}

	// Extract auto upgrade if provided
	var autoUpgrade *bool
	if au, ok := args["AutoUpgrade"].(bool); ok {
		autoUpgrade = &au
	}

	// Extract surge upgrade if provided
	surgeUpgrade := false
	if su, ok := args["SurgeUpgrade"].(bool); ok {
		surgeUpgrade = su
	}

	// Extract tags if provided
	var tags []string
	if tagList, ok := args["Tags"].([]any); ok {
		for _, tag := range tagList {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	// Create the request
	updateRequest := &godo.KubernetesClusterUpdateRequest{
		Name:              name,
		MaintenancePolicy: maintenancePolicy,
		AutoUpgrade:       autoUpgrade,
		SurgeUpgrade:      surgeUpgrade,
		Tags:              tags,
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	cluster, _, err := client.Kubernetes.Update(ctx, clusterID, updateRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to update cluster", err), nil
	}

	// Marshal the response
	clusterJSON, err := json.MarshalIndent(cluster, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal cluster", err), nil
	}

	return mcp.NewToolResultText(string(clusterJSON)), nil
}

// DeleteDOKSCluster deletes a Kubernetes cluster
func (d *DoksTool) deleteDOKSCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	_, err = client.Kubernetes.Delete(ctx, clusterID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to delete cluster", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Cluster %s deleted successfully", clusterID)), nil
}

// UpgradeDOKSCluster upgrades a Kubernetes cluster
func (d *DoksTool) upgradeDOKSCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	// Extract version
	version, ok := args["VersionSlug"].(string)
	if !ok {
		return mcp.NewToolResultError("VersionSlug is required and must be a string"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	_, err = client.Kubernetes.Upgrade(ctx, clusterID, &godo.KubernetesClusterUpgradeRequest{
		VersionSlug: version,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to upgrade cluster", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Cluster %s upgraded to %s", clusterID, version)), nil
}

// GetDOKSClusterUpgrades gets the available upgrades for a cluster
func (d *DoksTool) getDOKSClusterUpgrades(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	upgrades, _, err := client.Kubernetes.GetUpgrades(ctx, clusterID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get upgrades", err), nil
	}

	// Marshal the response
	upgradesJSON, err := json.MarshalIndent(upgrades, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal upgrades", err), nil
	}

	return mcp.NewToolResultText(string(upgradesJSON)), nil
}

// GetDOKSClusterKubeConfig gets the kubeconfig for a cluster
func (d *DoksTool) getDOKSClusterKubeConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	kubecfg, _, err := client.Kubernetes.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get kubeconfig", err), nil
	}

	return mcp.NewToolResultText(string(kubecfg.KubeconfigYAML)), nil
}

// GetDOKSClusterCredentials gets the credentials for a cluster
func (d *DoksTool) getDOKSClusterCredentials(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	credentials, _, err := client.Kubernetes.GetCredentials(ctx, clusterID, &godo.KubernetesClusterCredentialsGetRequest{})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get credentials", err), nil
	}

	// Build response
	var result struct {
		Server                   string `json:"server"`
		CertificateAuthorityData string `json:"certificate_authority_data"`
		ClientCertificateData    string `json:"client_certificate_data"`
		ClientKeyData            string `json:"client_key_data"`
		Token                    string `json:"token"`
		ExpiresAt                string `json:"expires_at"`
	}

	result.Server = credentials.Server
	result.CertificateAuthorityData = string(credentials.CertificateAuthorityData)
	result.ClientCertificateData = string(credentials.ClientCertificateData)
	result.ClientKeyData = string(credentials.ClientKeyData)
	result.Token = credentials.Token
	result.ExpiresAt = credentials.ExpiresAt.String()

	// Marshal the response
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// CreateDOKSNodePool creates a new node pool for a cluster
func (d *DoksTool) createDOKSNodePool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["cluster_id"].(string)
	if !ok {
		return mcp.NewToolResultError("cluster_id is required and must be a string"), nil
	}

	// Extract cluster ID
	createNPRequest, ok := args["node_pool_create_request"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("node_pool_create_request is required, and must be a json as []byte"), nil
	}

	jsonBytes, err := json.Marshal(createNPRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("marshal error", err), nil
	}

	createRequest := &godo.KubernetesNodePoolCreateRequest{}
	if err := json.Unmarshal(jsonBytes, createRequest); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to parse node pool create request", err), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	nodePool, _, err := client.Kubernetes.CreateNodePool(ctx, clusterID, createRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create node pool", err), nil
	}

	// Marshal the response
	nodePoolJSON, err := json.MarshalIndent(nodePool, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal node pool", err), nil
	}

	return mcp.NewToolResultText(string(nodePoolJSON)), nil
}

// GetDOKSNodePool gets a node pool for a cluster
func (d *DoksTool) getDOKSNodePool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	// Extract node pool ID
	nodePoolID, ok := args["NodePoolID"].(string)
	if !ok {
		return mcp.NewToolResultError("NodePoolID is required and must be a string"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	nodePool, _, err := client.Kubernetes.GetNodePool(ctx, clusterID, nodePoolID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get node pool", err), nil
	}

	// Marshal the response
	nodePoolJSON, err := json.MarshalIndent(nodePool, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal node pool", err), nil
	}

	return mcp.NewToolResultText(string(nodePoolJSON)), nil
}

// ListDOKSNodePools lists node pools for a cluster
func (d *DoksTool) listDOKSNodePools(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	nodePools, _, err := client.Kubernetes.ListNodePools(ctx, clusterID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list node pools", err), nil
	}

	// Marshal the response
	nodePoolsJSON, err := json.MarshalIndent(nodePools, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal node pools", err), nil
	}

	return mcp.NewToolResultText(string(nodePoolsJSON)), nil
}

// UpdateDOKSNodePool updates a node pool for a cluster
func (d *DoksTool) updateDOKSNodePool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	// Extract node pool ID
	nodePoolID, ok := args["NodePoolID"].(string)
	if !ok {
		return mcp.NewToolResultError("NodePoolID is required and must be a string"), nil
	}

	// Extract name if provided
	name := ""
	if nameArg, ok := args["Name"].(string); ok {
		name = nameArg
	}

	// Extract count if provided
	var count *int
	if countFloat, ok := args["Count"].(float64); ok {
		countInt := int(countFloat)
		count = &countInt
	}

	// Extract auto scale if provided
	var autoScale *bool
	var minNodes, maxNodes *int
	if as, ok := args["AutoScale"].(bool); ok {
		autoScale = &as

		// Min nodes
		if minFloat, ok := args["MinNodes"].(float64); ok {
			minInt := int(minFloat)
			minNodes = &minInt
		}

		// Max nodes
		if maxFloat, ok := args["MaxNodes"].(float64); ok {
			maxInt := int(maxFloat)
			maxNodes = &maxInt
		}
	}

	// Extract labels if provided
	var labels map[string]string
	if labelsMap, ok := args["Labels"].(map[string]any); ok {
		labels = make(map[string]string)
		for k, v := range labelsMap {
			if strVal, ok := v.(string); ok {
				labels[k] = strVal
			}
		}
	}

	// Extract taints if provided
	var taints []godo.Taint
	if taintList, ok := args["Taints"].([]any); ok {
		for _, taintArg := range taintList {
			if taintMap, ok := taintArg.(map[string]any); ok {
				key, keyOk := taintMap["Key"].(string)
				value, valueOk := taintMap["Value"].(string)
				effect, effectOk := taintMap["Effect"].(string)

				if keyOk && valueOk && effectOk {
					taints = append(taints, godo.Taint{
						Key:    key,
						Value:  value,
						Effect: effect,
					})
				}
			}
		}
	}

	// Extract tags if provided
	var tags []string
	if tagList, ok := args["Tags"].([]any); ok {
		for _, tag := range tagList {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	// Create the request
	updateRequest := &godo.KubernetesNodePoolUpdateRequest{
		Name:      name,
		Count:     count,
		Tags:      tags,
		Labels:    labels,
		Taints:    &taints,
		AutoScale: autoScale,
		MinNodes:  minNodes,
		MaxNodes:  maxNodes,
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	nodePool, _, err := client.Kubernetes.UpdateNodePool(ctx, clusterID, nodePoolID, updateRequest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to update node pool", err), nil
	}

	// Marshal the response
	nodePoolJSON, err := json.MarshalIndent(nodePool, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal node pool", err), nil
	}

	return mcp.NewToolResultText(string(nodePoolJSON)), nil
}

// DeleteDOKSNodePool deletes a node pool for a cluster
func (d *DoksTool) deleteDOKSNodePool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	// Extract node pool ID
	nodePoolID, ok := args["NodePoolID"].(string)
	if !ok {
		return mcp.NewToolResultError("NodePoolID is required and must be a string"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	_, err = client.Kubernetes.DeleteNodePool(ctx, clusterID, nodePoolID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to delete node pool", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Node pool %s deleted successfully", nodePoolID)), nil
}

// DeleteDOKSNode deletes a node from a node pool
func (d *DoksTool) deleteDOKSNode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	// Extract node pool ID
	nodePoolID, ok := args["NodePoolID"].(string)
	if !ok {
		return mcp.NewToolResultError("NodePoolID is required and must be a string"), nil
	}

	// Extract node ID
	nodeID, ok := args["NodeID"].(string)
	if !ok {
		return mcp.NewToolResultError("NodeID is required and must be a string"), nil
	}

	// Extract skip drain if provided
	skipDrain := false
	if sd, ok := args["SkipDrain"].(bool); ok {
		skipDrain = sd
	}

	// Extract replace if provided
	replace := false
	if r, ok := args["Replace"].(bool); ok {
		replace = r
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	_, err = client.Kubernetes.DeleteNode(ctx, clusterID, nodePoolID, nodeID, &godo.KubernetesNodeDeleteRequest{
		SkipDrain: skipDrain,
		Replace:   replace,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to delete node", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Node %s deleted successfully", nodeID)), nil
}

// RecycleDOKSNodes recycles nodes in a node pool
func (d *DoksTool) recycleDOKSNodes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Extract cluster ID
	clusterID, ok := args["ClusterID"].(string)
	if !ok {
		return mcp.NewToolResultError("ClusterID is required and must be a string"), nil
	}

	// Extract node pool ID
	nodePoolID, ok := args["NodePoolID"].(string)
	if !ok {
		return mcp.NewToolResultError("NodePoolID is required and must be a string"), nil
	}

	// Extract node IDs
	var nodeIDs []string
	if nodeIDList, ok := args["NodeIDs"].([]any); ok {
		for _, id := range nodeIDList {
			if idStr, ok := id.(string); ok {
				nodeIDs = append(nodeIDs, idStr)
			}
		}
	}

	// If no node IDs provided, return error
	if len(nodeIDs) == 0 {
		return mcp.NewToolResultError("NodeIDs is required and must be a non-empty array of strings"), nil
	}

	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call
	_, err = client.Kubernetes.RecycleNodePoolNodes(ctx, clusterID, nodePoolID, &godo.KubernetesNodePoolRecycleNodesRequest{
		Nodes: nodeIDs,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to recycle nodes", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully recycled %d nodes in node pool %s", len(nodeIDs), nodePoolID)), nil
}

// getKubernetesOptions gets available Kubernetes options including versions, regions, and sizes
func (d *DoksTool) getKubernetesOptions(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := d.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// Make the API call to get Kubernetes options
	options, _, err := client.Kubernetes.GetOptions(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get kubernetes options", err), nil
	}

	// Marshal the response
	optionsJSON, err := json.MarshalIndent(options, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal kubernetes options", err), nil
	}

	return mcp.NewToolResultText(string(optionsJSON)), nil
}

// getDayFromString converts a day string to the format expected by the API
func getDayFromString(day string) int {
	// Normalize the day string
	day = strings.ToLower(day)
	day = strings.TrimSpace(day)

	// Map days to their expected values
	dayMap := map[string]int{
		"any":       0, // KubernetesMaintenanceDayAny
		"monday":    1, // KubernetesMaintenanceDayMonday
		"mon":       1,
		"tuesday":   2, // KubernetesMaintenanceDayTuesday
		"tue":       2,
		"wednesday": 3, // KubernetesMaintenanceDayWednesday
		"wed":       3,
		"thursday":  4, // KubernetesMaintenanceDayThursday
		"thu":       4,
		"friday":    5, // KubernetesMaintenanceDayFriday
		"fri":       5,
		"saturday":  6, // KubernetesMaintenanceDaySaturday
		"sat":       6,
		"sunday":    7, // KubernetesMaintenanceDaySunday
		"sun":       7,
	}

	if mappedDay, ok := dayMap[day]; ok {
		return mappedDay
	}

	// Default to Any if not recognized
	return 0
}

// Tools returns the tools provided by this tool
func (d *DoksTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.getDoksCluster,
			Tool: mcp.NewTool("doks-get-cluster",
				mcp.WithDescription("Get a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
			),
		},
		{
			Handler: d.listDOKSClusters,
			Tool: mcp.NewTool("doks-list-clusters",
				mcp.WithDescription("List all DigitalOcean Kubernetes clusters"),
				mcp.WithNumber("Page", mcp.DefaultNumber(1), mcp.Description("Page number of the results to fetch")),
				mcp.WithNumber("PerPage", mcp.DefaultNumber(20), mcp.Description("Number of items returned per page")),
			),
		},
		{
			Handler: d.createDOKSCluster,
			Tool: mcp.NewToolWithRawSchema("doks-create-cluster",
				"Create a new DigitalOcean Kubernetes cluster", clusterCreateSchemaJSON,
			),
		},
		{
			Handler: d.updateDOKSCluster,
			Tool: mcp.NewTool("doks-update-cluster",
				mcp.WithDescription("Update a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
				mcp.WithString("Name", mcp.Description("The name of the Kubernetes cluster")),
				mcp.WithObject("MaintenancePolicy", mcp.Description("Maintenance window policy for the cluster")),
				mcp.WithBoolean("AutoUpgrade", mcp.Description("Whether the cluster will be automatically upgraded")),
				mcp.WithBoolean("SurgeUpgrade", mcp.Description("Whether to enable surge upgrades for the cluster")),
				mcp.WithArray("Tags", mcp.Description("A list of tags to apply to the cluster")),
			),
		},
		{
			Handler: d.deleteDOKSCluster,
			Tool: mcp.NewTool("doks-delete-cluster",
				mcp.WithDescription("Delete a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
			),
		},
		{
			Handler: d.upgradeDOKSCluster,
			Tool: mcp.NewTool("doks-upgrade-cluster",
				mcp.WithDescription("Upgrade a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
				mcp.WithString("VersionSlug", mcp.Required(), mcp.Description("The Kubernetes version to upgrade to")),
			),
		},
		{
			Handler: d.getDOKSClusterUpgrades,
			Tool: mcp.NewTool("doks-get-cluster-upgrades",
				mcp.WithDescription("Get available upgrades for a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
			),
		},
		{
			Handler: d.getDOKSClusterKubeConfig,
			Tool: mcp.NewTool("doks-get-kubeconfig",
				mcp.WithDescription("Get kubeconfig for a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
			),
		},
		{
			Handler: d.getDOKSClusterCredentials,
			Tool: mcp.NewTool("doks-get-credentials",
				mcp.WithDescription("Get credentials for a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
			),
		},
		{
			Handler: d.createDOKSNodePool,
			Tool: mcp.NewToolWithRawSchema("doks-create-nodepool",
				"Create a new node pool in a DigitalOcean Kubernetes cluster", nodePoolCreateSchemaJSON,
			),
		},
		{
			Handler: d.getDOKSNodePool,
			Tool: mcp.NewTool("doks-get-nodepool",
				mcp.WithDescription("Get a node pool in a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
				mcp.WithString("NodePoolID", mcp.Required(), mcp.Description("The ID of the node pool")),
			),
		},
		{
			Handler: d.listDOKSNodePools,
			Tool: mcp.NewTool("doks-list-nodepools",
				mcp.WithDescription("List all node pools in a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
			),
		},
		{
			Handler: d.updateDOKSNodePool,
			Tool: mcp.NewTool("doks-update-nodepool",
				mcp.WithDescription("Update a node pool in a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
				mcp.WithString("NodePoolID", mcp.Required(), mcp.Description("The ID of the node pool")),
				mcp.WithString("Name", mcp.Description("The name of the node pool")),
				mcp.WithNumber("Count", mcp.Description("The number of nodes in the node pool")),
				mcp.WithArray("Tags", mcp.Description("A list of tags to apply to the node pool")),
				mcp.WithObject("Labels", mcp.Description("A map of Kubernetes labels to apply to the nodes")),
				mcp.WithArray("Taints", mcp.Description("A list of Kubernetes taints to apply to the nodes")),
				mcp.WithBoolean("AutoScale", mcp.Description("Whether to enable auto-scaling for the node pool")),
				mcp.WithNumber("MinNodes", mcp.Description("The minimum number of nodes for auto-scaling")),
				mcp.WithNumber("MaxNodes", mcp.Description("The maximum number of nodes for auto-scaling")),
			),
		},
		{
			Handler: d.deleteDOKSNodePool,
			Tool: mcp.NewTool("doks-delete-nodepool",
				mcp.WithDescription("Delete a node pool in a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
				mcp.WithString("NodePoolID", mcp.Required(), mcp.Description("The ID of the node pool")),
			),
		},
		{
			Handler: d.deleteDOKSNode,
			Tool: mcp.NewTool("doks-delete-node",
				mcp.WithDescription("Delete a node from a node pool in a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
				mcp.WithString("NodePoolID", mcp.Required(), mcp.Description("The ID of the node pool")),
				mcp.WithString("NodeID", mcp.Required(), mcp.Description("The ID of the node")),
				mcp.WithBoolean("SkipDrain", mcp.Description("Whether to skip draining the node before deletion")),
				mcp.WithBoolean("Replace", mcp.Description("Whether to replace the node after deletion")),
			),
		},
		{
			Handler: d.recycleDOKSNodes,
			Tool: mcp.NewTool("doks-recycle-nodes",
				mcp.WithDescription("Recycle specific nodes in a node pool in a DigitalOcean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster")),
				mcp.WithString("NodePoolID", mcp.Required(), mcp.Description("The ID of the node pool")),
				mcp.WithArray("NodeIDs", mcp.Required(), mcp.Description("List of node IDs to recycle")),
			),
		},
		{
			Handler: d.getKubernetesOptions,
			Tool: mcp.NewTool("doks-list-options",
				mcp.WithDescription("List available Kubernetes options including versions, regions, and sizes"),
			),
		},
	}
}
