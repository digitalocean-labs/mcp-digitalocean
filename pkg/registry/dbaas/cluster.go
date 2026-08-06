package dbaas

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/pkg/response"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ClusterTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewClusterTool(client func(ctx context.Context) (*godo.Client, error)) *ClusterTool {
	return &ClusterTool{
		client: client,
	}
}

func (s *ClusterTool) listCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Optional pagination
	page := 0
	if pStr, ok := args["page"].(string); ok && pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil {
			page = p
		}
	}
	perPage := 0
	if pp, ok := args["per_page"].(int); ok {
		perPage = pp
	}

	var opts *godo.ListOptions
	if page > 0 || perPage > 0 {
		opts = &godo.ListOptions{Page: page, PerPage: perPage}
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	clusters, _, err := client.Databases.List(ctx, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonClusters, err := response.CompactJSON(clusters)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonClusters), nil
}

func (s *ClusterTool) getCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	cluster, _, err := client.Databases.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCluster, err := response.CompactJSON(cluster)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCluster), nil
}

func (s *ClusterTool) createCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	engine, _ := args["engine"].(string)
	version, _ := args["version"].(string)
	region, _ := args["region"].(string)
	size, _ := args["size"].(string)
	numNodes, _ := args["num_nodes"].(float64) // JSON numbers are float64

	tags := []string{}
	if tagsRaw, ok := args["tags"].(string); ok && tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	createReq := &godo.DatabaseCreateRequest{
		Name:       name,
		EngineSlug: engine,
		Version:    version,
		Region:     region,
		SizeSlug:   size,
		NumNodes:   int(numNodes),
		Tags:       tags,
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	cluster, _, err := client.Databases.Create(ctx, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCluster, err := response.CompactJSON(cluster)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCluster), nil
}

func (s *ClusterTool) deleteCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.Delete(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Cluster deleted successfully"), nil
}

func (s *ClusterTool) resizeCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	size, _ := args["size"].(string)
	numNodes := 0
	if n, ok := args["num_nodes"].(float64); ok {
		numNodes = int(n)
	}
	storageSizeMib := uint64(0)
	if ssm, ok := args["storage_size_mib"].(float64); ok {
		storageSizeMib = uint64(ssm)
	}

	resizeReq := &godo.DatabaseResizeRequest{}
	if size != "" {
		resizeReq.SizeSlug = size
	}
	if numNodes > 0 {
		resizeReq.NumNodes = numNodes
	}
	if storageSizeMib > 0 {
		resizeReq.StorageSizeMib = storageSizeMib
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.Resize(ctx, id, resizeReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Cluster resize initiated successfully"), nil
}

func (s *ClusterTool) getCA(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, ok := req.GetArguments()["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	ca, _, err := client.Databases.GetCA(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCA, err := response.CompactJSON(ca)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCA), nil
}

func (s *ClusterTool) listBackups(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	// Optional pagination
	page := 0
	if pStr, ok := args["page"].(string); ok && pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil {
			page = p
		}
	}
	perPage := 0
	if pp, ok := args["per_page"].(int); ok {
		perPage = pp
	}

	var opts *godo.ListOptions
	if page > 0 || perPage > 0 {
		opts = &godo.ListOptions{Page: page, PerPage: perPage}
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	backups, _, err := client.Databases.ListBackups(ctx, id, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonBackups, err := response.CompactJSON(backups)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonBackups), nil
}

func (s *ClusterTool) listOptions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	options, _, err := client.Databases.ListOptions(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonOptions, err := response.CompactJSON(options)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonOptions), nil
}

func (s *ClusterTool) upgradeMajorVersion(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	version, ok := args["version"].(string)
	if !ok || version == "" {
		return mcp.NewToolResultError("Target version is required"), nil
	}
	upgradeReq := &godo.UpgradeVersionRequest{Version: version}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.UpgradeMajorVersion(ctx, id, upgradeReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Major version upgrade initiated successfully"), nil
}

// Handler implementation for startOnlineMigration using structured object
func (s *ClusterTool) startOnlineMigration(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	sourceMap, ok := args["source"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Missing or invalid 'source' object (expected structured object)"), nil
	}

	sourceBytes, err := json.Marshal(sourceMap)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	var source godo.DatabaseOnlineMigrationConfig
	if err := json.Unmarshal(sourceBytes, &source); err != nil {
		return mcp.NewToolResultError("Invalid source object: " + err.Error()), nil
	}
	disableSSL := false
	if dssl, ok := args["disable_ssl"].(bool); ok {
		disableSSL = dssl
	}
	var ignoreDBs []string
	if ignoreStr, ok := args["ignore_dbs"].(string); ok && ignoreStr != "" {
		for _, db := range strings.Split(ignoreStr, ",") {
			db = strings.TrimSpace(db)
			if db != "" {
				ignoreDBs = append(ignoreDBs, db)
			}
		}
	}
	startReq := &godo.DatabaseStartOnlineMigrationRequest{
		Source:     &source,
		DisableSSL: disableSSL,
		IgnoreDBs:  ignoreDBs,
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	status, _, err := client.Databases.StartOnlineMigration(ctx, id, startReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonStatus, err := response.CompactJSON(status)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonStatus), nil
}

func (s *ClusterTool) stopOnlineMigration(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	migrationID, ok := args["migration_id"].(string)
	if !ok || migrationID == "" {
		return mcp.NewToolResultError("migration_id is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.StopOnlineMigration(ctx, id, migrationID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Online migration stopped successfully"), nil
}

func (s *ClusterTool) getOnlineMigrationStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	status, _, err := client.Databases.GetOnlineMigrationStatus(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonStatus, err := response.CompactJSON(status)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonStatus), nil
}

func (s *ClusterTool) Tools() []server.ServerTool {
	return []server.ServerTool{

		{
			Handler: s.listCluster,
			Tool: mcp.NewTool("db-cluster-list",
				mcp.WithDescription("Get list of  Cluster"),
				mcp.WithString("page", mcp.Description("Page number for pagination (optional, integer as string)")),
				mcp.WithNumber("per_page", mcp.Description("Number of results per page (optional, integer)")),
			),
		},
		{
			Handler: s.getCluster,
			Tool: mcp.NewTool("db-cluster-get",
				mcp.WithDescription("Get a cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The id of the cluster to retrieve")),
			),
		},
		{
			Handler: s.getCA,
			Tool: mcp.NewTool("db-cluster-get-ca",
				mcp.WithDescription("Get the CA certificate for a cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The id of the cluster to retrieve the CA for")),
			),
		},
		{
			Handler: s.createCluster,
			Tool: mcp.NewTool("db-cluster-create",
				mcp.WithDescription("Create a new database cluster"),
				mcp.WithString("name", mcp.Required(), mcp.Description("The name of the cluster")),
				mcp.WithString("engine", mcp.Required(), mcp.Description("The engine slug (e.g., valkey, pg, mysql, etc.)")),
				mcp.WithString("version", mcp.Required(), mcp.Description("The version of the engine")),
				mcp.WithString("region", mcp.Required(), mcp.Description("The region slug (e.g., nyc1)")),
				mcp.WithString("size", mcp.Required(), mcp.Description("The size slug (e.g., db-s-2vcpu-4gb)")),
				mcp.WithNumber("num_nodes", mcp.Required(), mcp.Description("The number of nodes")),
				mcp.WithString("tags", mcp.Description("Comma-separated tags to apply to the cluster")),
			),
		},
		{
			Handler: s.deleteCluster,
			Tool: mcp.NewTool("db-cluster-delete",
				mcp.WithDescription("Delete a database cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The id of the cluster to delete")),
			),
		},
		{
			Handler: s.resizeCluster,
			Tool: mcp.NewTool("db-cluster-resize",
				mcp.WithDescription("Resize a database cluster by its id. At least one of size, num_nodes, or storage_size_mib must be provided."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The id of the cluster to resize")),
				mcp.WithString("size", mcp.Description("The new size slug (e.g., db-s-2vcpu-4gb)")),
				mcp.WithNumber("num_nodes", mcp.Description("The new number of nodes")),
				mcp.WithNumber("storage_size_mib", mcp.Description("The new storage size in MiB")),
			),
		},
		{
			Handler: s.listBackups,
			Tool: mcp.NewTool("db-cluster-list-backups",
				mcp.WithDescription("List backups for a database cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The id of the cluster to list backups for")),
				mcp.WithString("page", mcp.Description("Page number for pagination (optional, integer as string)")),
				mcp.WithNumber("per_page", mcp.Description("Number of results per page (optional, integer)")),
			),
		},
		{
			Handler: s.listOptions,
			Tool: mcp.NewTool("db-cluster-list-options",
				mcp.WithDescription("List available database options (engines, versions, sizes, regions, etc) for DigitalOcean managed databases."),
			),
		},
		{
			Handler: s.upgradeMajorVersion,
			Tool: mcp.NewTool("db-cluster-upgrade-major-version",
				mcp.WithDescription("Upgrade the major version of a database cluster by its id. Requires the target version."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithString("version", mcp.Required(), mcp.Description("The target major version to upgrade to (e.g., 15 for PostgreSQL)")),
			),
		},
		{
			Handler: s.startOnlineMigration,
			Tool: mcp.NewTool("db-cluster-start-online-migration",
				mcp.WithDescription("Start an online migration for a database cluster by its id."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithObject("source",
					mcp.Required(),
					mcp.Description("The source database configuration"),
					mcp.Properties(map[string]any{
						"host": map[string]any{
							"type":        "string",
							"description": "Hostname or IP of the source database",
						},
						"port": map[string]any{
							"type":        "integer",
							"description": "Source database port",
						},
						"dbname": map[string]any{
							"type":        "string",
							"description": "Name of the database to migrate",
						},
						"username": map[string]any{
							"type":        "string",
							"description": "Username for source connection",
						},
						"password": map[string]any{
							"type":        "string",
							"description": "Password for source connection",
						},
					}),
				),
				mcp.WithBoolean("disable_ssl", mcp.Description("Disable SSL on source connection (optional)")),
				mcp.WithString("ignore_dbs", mcp.Description("Comma-separated list of databases to ignore")),
			),
		},
		{
			Handler: s.stopOnlineMigration,
			Tool: mcp.NewTool("db-cluster-stop-online-migration",
				mcp.WithDescription("Stop an online migration for a database cluster by its id and migration_id."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithString("migration_id", mcp.Required(), mcp.Description("The migration id to stop")),
			),
		},
		{
			Handler: s.getOnlineMigrationStatus,
			Tool: mcp.NewTool("db-cluster-get-migration",
				mcp.WithDescription("Get the online migration status for a database cluster by its id."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
			),
		},
	}
}
