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

type RedisTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewRedisTool(client func(ctx context.Context) (*godo.Client, error)) *RedisTool {
	return &RedisTool{
		client: client,
	}
}

func (s *RedisTool) getRedisConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	cfg, _, err := client.Databases.GetRedisConfig(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonCfg, err := response.CompactJSON(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCfg), nil
}

func (s *RedisTool) updateRedisConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	configMap, ok := args["config"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Missing or invalid 'config' object (expected structured object)"), nil
	}

	cfgBytes, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	var config godo.RedisConfig
	if err := json.Unmarshal(cfgBytes, &config); err != nil {
		return mcp.NewToolResultError("Invalid config object: " + err.Error()), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Databases.UpdateRedisConfig(ctx, id, &config)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("Redis config updated successfully"), nil
}

func (s *RedisTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.getRedisConfig,
			Tool: mcp.NewTool("db-cluster-get-redis-config",
				mcp.WithDescription("Get the Redis config for a cluster by its id."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
			),
		},
		{
			Handler: s.updateRedisConfig,
			Tool: mcp.NewTool("db-cluster-update-redis-config",
				mcp.WithDescription("Update the Redis config for a cluster by its id. Accepts a structured config object."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithObject("config",
					mcp.Required(),
					mcp.Description("Structured configuration for Redis database"),
					mcp.Properties(map[string]any{
						"redis_maxmemory_policy": map[string]any{
							"type":        "string",
							"description": "Policy for eviction when memory is full (e.g., allkeys-lru)",
						},
						"redis_pubsub_client_output_buffer_limit": map[string]any{
							"type": "integer",
						},
						"redis_number_of_databases": map[string]any{
							"type": "integer",
						},
						"redis_io_threads": map[string]any{
							"type": "integer",
						},
						"redis_lfu_log_factor": map[string]any{
							"type": "integer",
						},
						"redis_lfu_decay_time": map[string]any{
							"type": "integer",
						},
						"redis_ssl": map[string]any{
							"type": "boolean",
						},
						"redis_timeout": map[string]any{
							"type": "integer",
						},
						"redis_notify_keyspace_events": map[string]any{
							"type": "string",
						},
						"redis_persistence": map[string]any{
							"type":        "string",
							"description": "Persistence mode (e.g., aof, rdb, none)",
						},
						"redis_acl_channels_default": map[string]any{
							"type": "string",
						},
					}),
				),
			),
		},
	}
}
