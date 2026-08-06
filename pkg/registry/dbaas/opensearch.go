package dbaas

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/pkg/response"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type OpenSearchTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewOpenSearchTool(client func(ctx context.Context) (*godo.Client, error)) *OpenSearchTool {
	return &OpenSearchTool{
		client: client,
	}
}

func (s *OpenSearchTool) getOpensearchConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	cfg, _, err := client.Databases.GetOpensearchConfig(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCfg, err := response.CompactJSON(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCfg), nil
}

func (s *OpenSearchTool) updateOpensearchConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	cfgMap, ok := args["config"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Missing or invalid 'config' object (expected structured object)"), nil
	}
	cfgBytes, err := json.Marshal(cfgMap)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	var config godo.OpensearchConfig
	if err := json.Unmarshal(cfgBytes, &config); err != nil {
		return mcp.NewToolResultError("Invalid config object: " + err.Error()), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.UpdateOpensearchConfig(ctx, id, &config)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Opensearch config updated successfully"), nil
}

func (s *OpenSearchTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.getOpensearchConfig,
			Tool: mcp.NewTool("db-cluster-get-opensearch-config",
				mcp.WithDescription("Get the Opensearch config for a cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
			),
		},
		{
			Handler: s.updateOpensearchConfig,
			Tool: mcp.NewTool("db-cluster-update-os-config",
				mcp.WithDescription("Update the Opensearch config for a cluster by its id. Accepts a structured config object."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithObject("config",
					mcp.Required(),
					mcp.Description("Structured configuration for Opensearch cluster"),
					mcp.Properties(map[string]any{
						"http_max_content_length_bytes":                         map[string]any{"type": "integer"},
						"http_max_header_size_bytes":                            map[string]any{"type": "integer"},
						"http_max_initial_line_length_bytes":                    map[string]any{"type": "integer"},
						"indices_query_bool_max_clause_count":                   map[string]any{"type": "integer"},
						"indices_fielddata_cache_size_percentage":               map[string]any{"type": "integer"},
						"indices_memory_index_buffer_size_percentage":           map[string]any{"type": "integer"},
						"indices_memory_min_index_buffer_size_mb":               map[string]any{"type": "integer"},
						"indices_memory_max_index_buffer_size_mb":               map[string]any{"type": "integer"},
						"indices_queries_cache_size_percentage":                 map[string]any{"type": "integer"},
						"indices_recovery_max_mb_per_sec":                       map[string]any{"type": "integer"},
						"indices_recovery_max_concurrent_file_chunks":           map[string]any{"type": "integer"},
						"thread_pool_search_size":                               map[string]any{"type": "integer"},
						"thread_pool_search_throttled_size":                     map[string]any{"type": "integer"},
						"thread_pool_get_size":                                  map[string]any{"type": "integer"},
						"thread_pool_analyze_size":                              map[string]any{"type": "integer"},
						"thread_pool_write_size":                                map[string]any{"type": "integer"},
						"thread_pool_force_merge_size":                          map[string]any{"type": "integer"},
						"thread_pool_search_queue_size":                         map[string]any{"type": "integer"},
						"thread_pool_search_throttled_queue_size":               map[string]any{"type": "integer"},
						"thread_pool_get_queue_size":                            map[string]any{"type": "integer"},
						"thread_pool_analyze_queue_size":                        map[string]any{"type": "integer"},
						"thread_pool_write_queue_size":                          map[string]any{"type": "integer"},
						"ism_enabled":                                           map[string]any{"type": "boolean"},
						"ism_history_enabled":                                   map[string]any{"type": "boolean"},
						"ism_history_max_age_hours":                             map[string]any{"type": "integer"},
						"ism_history_max_docs":                                  map[string]any{"type": "integer"},
						"ism_history_rollover_check_period_hours":               map[string]any{"type": "integer"},
						"ism_history_rollover_retention_period_days":            map[string]any{"type": "integer"},
						"search_max_buckets":                                    map[string]any{"type": "integer"},
						"action_auto_create_index_enabled":                      map[string]any{"type": "boolean"},
						"enable_security_audit":                                 map[string]any{"type": "boolean"},
						"action_destructive_requires_name":                      map[string]any{"type": "boolean"},
						"cluster_max_shards_per_node":                           map[string]any{"type": "integer"},
						"override_main_response_version":                        map[string]any{"type": "boolean"},
						"script_max_compilations_rate":                          map[string]any{"type": "string"},
						"cluster_routing_allocation_node_concurrent_recoveries": map[string]any{"type": "integer"},
						"reindex_remote_whitelist":                              map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"plugins_alerting_filter_by_backend_roles_enabled":      map[string]any{"type": "boolean"},
					}),
				),
			),
		},
	}
}
