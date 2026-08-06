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

type MongoTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewMongoTool(client func(ctx context.Context) (*godo.Client, error)) *MongoTool {
	return &MongoTool{
		client: client,
	}
}

func (s *MongoTool) getMongoDBConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	cfg, _, err := client.Databases.GetMongoDBConfig(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCfg, err := response.CompactJSON(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCfg), nil
}

func (s *MongoTool) updateMongoDBConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	var config godo.MongoDBConfig
	if err := json.Unmarshal(cfgBytes, &config); err != nil {
		return mcp.NewToolResultError("Invalid config object: " + err.Error()), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.UpdateMongoDBConfig(ctx, id, &config)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("MongoDB config updated successfully"), nil
}

func (s *MongoTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.getMongoDBConfig,
			Tool: mcp.NewTool("db-cluster-get-mongodb-config",
				mcp.WithDescription("Get the MongoDB config for a cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
			),
		},
		{
			Handler: s.updateMongoDBConfig,
			Tool: mcp.NewTool("db-cluster-update-mongodb-config",
				mcp.WithDescription("Update the MongoDB config for a cluster by its id. Accepts a structured config object."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithObject("config",
					mcp.Required(),
					mcp.Description("Configuration parameters for MongoDB"),
					mcp.Properties(map[string]any{
						"default_read_concern": map[string]any{
							"type":        "string",
							"description": "Specifies the default read concern (e.g., 'local', 'majority')",
						},
						"default_write_concern": map[string]any{
							"type":        "string",
							"description": "Specifies the default write concern (e.g., 'majority')",
						},
						"transaction_lifetime_limit_seconds": map[string]any{
							"type":        "integer",
							"description": "Time in seconds a transaction can run before expiring",
						},
						"slow_op_threshold_ms": map[string]any{
							"type":        "integer",
							"description": "Threshold in milliseconds to log slow operations",
						},
						"verbosity": map[string]any{
							"type":        "integer",
							"description": "Level of MongoDB logging verbosity (typically 0–5)",
						},
					}),
				),
			),
		},
	}
}
