package dbaas

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/pkg/response"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MysqlTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewMysqlTool(client func(ctx context.Context) (*godo.Client, error)) *MysqlTool {
	return &MysqlTool{
		client: client,
	}
}

func (s *MysqlTool) getMySQLConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	cfg, _, err := client.Databases.GetMySQLConfig(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCfg, err := response.CompactJSON(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCfg), nil
}

func (s *MysqlTool) updateMySQLConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	configMap, ok := args["config"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Invalid or missing 'config' object (expected structured object)"), nil
	}

	cfgBytes, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	var config godo.MySQLConfig
	if err := json.Unmarshal(cfgBytes, &config); err != nil {
		return mcp.NewToolResultError("Invalid config object: " + err.Error()), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.UpdateMySQLConfig(ctx, id, &config)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("MySQL config updated successfully"), nil
}

func (s *MysqlTool) getSQLMode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	mode, _, err := client.Databases.GetSQLMode(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText(mode), nil
}

func (s *MysqlTool) setSQLMode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	modesStr, ok := args["modes"].(string)
	if !ok || modesStr == "" {
		return mcp.NewToolResultError("SQL modes are required (comma-separated)"), nil
	}
	modes := []string{}
	for _, m := range strings.Split(modesStr, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			modes = append(modes, m)
		}
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.SetSQLMode(ctx, id, modes...)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("SQL mode set successfully"), nil
}

func (s *MysqlTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.getMySQLConfig,
			Tool: mcp.NewTool("db-cluster-get-mysql-config",
				mcp.WithDescription("Get the MySQL config for a cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
			),
		},
		{
			Handler: s.updateMySQLConfig,
			Tool: mcp.NewTool("db-cluster-update-mysql-config",
				mcp.WithDescription("Update the MySQL config for a cluster by its id. Accepts a structured 'config' object."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithObject("config",
					mcp.Required(),
					mcp.Description("Structured configuration for MySQL"),
					mcp.Properties(map[string]any{
						"connect_timeout":                  map[string]any{"type": "integer"},
						"default_time_zone":                map[string]any{"type": "string"},
						"innodb_log_buffer_size":           map[string]any{"type": "integer"},
						"innodb_online_alter_log_max_size": map[string]any{"type": "integer"},
						"innodb_lock_wait_timeout":         map[string]any{"type": "integer"},
						"interactive_timeout":              map[string]any{"type": "integer"},
						"max_allowed_packet":               map[string]any{"type": "integer"},
						"net_read_timeout":                 map[string]any{"type": "integer"},
						"sort_buffer_size":                 map[string]any{"type": "integer"},
						"sql_mode":                         map[string]any{"type": "string"},
						"sql_require_primary_key":          map[string]any{"type": "boolean"},
						"wait_timeout":                     map[string]any{"type": "integer"},
						"net_write_timeout":                map[string]any{"type": "integer"},
						"group_concat_max_len":             map[string]any{"type": "integer"},
						"information_schema_stats_expiry":  map[string]any{"type": "integer"},
						"innodb_ft_min_token_size":         map[string]any{"type": "integer"},
						"innodb_ft_server_stopword_table":  map[string]any{"type": "string"},
						"innodb_print_all_deadlocks":       map[string]any{"type": "boolean"},
						"innodb_rollback_on_timeout":       map[string]any{"type": "boolean"},
						"internal_tmp_mem_storage_engine":  map[string]any{"type": "string"},
						"max_heap_table_size":              map[string]any{"type": "integer"},
						"tmp_table_size":                   map[string]any{"type": "integer"},
						"slow_query_log":                   map[string]any{"type": "boolean"},
						"long_query_time":                  map[string]any{"type": "number"},
						"backup_hour":                      map[string]any{"type": "integer"},
						"backup_minute":                    map[string]any{"type": "integer"},
						"binlog_retention_period":          map[string]any{"type": "integer"},
						"innodb_change_buffer_max_size":    map[string]any{"type": "integer"},
						"innodb_flush_neighbors":           map[string]any{"type": "integer"},
						"innodb_read_io_threads":           map[string]any{"type": "integer"},
						"innodb_thread_concurrency":        map[string]any{"type": "integer"},
						"innodb_write_io_threads":          map[string]any{"type": "integer"},
						"net_buffer_length":                map[string]any{"type": "integer"},
						"log_output":                       map[string]any{"type": "string"},
					}),
				),
			),
		},
		{
			Handler: s.getSQLMode,
			Tool: mcp.NewTool("db-cluster-get-sql-mode",
				mcp.WithDescription("Get the SQL mode for a cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
			),
		},
		{
			Handler: s.setSQLMode,
			Tool: mcp.NewTool("db-cluster-set-sql-mode",
				mcp.WithDescription("Set the SQL mode for a cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithString("modes", mcp.Required(), mcp.Description("Comma-separated SQL modes to set")),
			),
		},
	}
}
