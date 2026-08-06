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

type PostgreSQLTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewPostgreSQLTool(client func(ctx context.Context) (*godo.Client, error)) *PostgreSQLTool {
	return &PostgreSQLTool{
		client: client,
	}
}

func (s *PostgreSQLTool) getPostgreSQLConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	cfg, _, err := client.Databases.GetPostgreSQLConfig(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCfg, err := response.CompactJSON(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCfg), nil
}

func (s *PostgreSQLTool) updatePostgreSQLConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	configMap, ok := args["config"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Missing or invalid 'config' object (must be a structured object)"), nil
	}

	cfgBytes, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	var config godo.PostgreSQLConfig
	if err := json.Unmarshal(cfgBytes, &config); err != nil {
		return mcp.NewToolResultError("Invalid config object: " + err.Error()), nil
	}

	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Databases.UpdatePostgreSQLConfig(ctx, id, &config)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("PostgreSQL config updated successfully"), nil
}

func (s *PostgreSQLTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.getPostgreSQLConfig,
			Tool: mcp.NewTool("db-cluster-get-postgresql-config",
				mcp.WithDescription("Get the PostgreSQL config for a cluster by its id"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
			),
		},
		{
			Handler: s.updatePostgreSQLConfig,
			Tool: mcp.NewTool("db-cluster-update-psql-config",
				mcp.WithDescription("Update the PostgreSQL config for a cluster by its id. Accepts a structured config object."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster UUID")),
				mcp.WithObject("config",
					mcp.Required(),
					mcp.Description("Configuration object for PostgreSQL database"),
					mcp.Properties(map[string]any{
						"autovacuum_max_workers":              map[string]any{"type": "integer"},
						"autovacuum_freeze_max_age":           map[string]any{"type": "integer"},
						"autovacuum_naptime":                  map[string]any{"type": "integer"},
						"autovacuum_vacuum_threshold":         map[string]any{"type": "integer"},
						"autovacuum_analyze_threshold":        map[string]any{"type": "integer"},
						"autovacuum_vacuum_scale_factor":      map[string]any{"type": "number"},
						"autovacuum_analyze_scale_factor":     map[string]any{"type": "number"},
						"autovacuum_vacuum_cost_delay":        map[string]any{"type": "integer"},
						"autovacuum_vacuum_cost_limit":        map[string]any{"type": "integer"},
						"bgwriter_delay":                      map[string]any{"type": "integer"},
						"bgwriter_flush_after":                map[string]any{"type": "integer"},
						"bgwriter_lru_maxpages":               map[string]any{"type": "integer"},
						"bgwriter_lru_multiplier":             map[string]any{"type": "number"},
						"deadlock_timeout":                    map[string]any{"type": "integer"},
						"default_toast_compression":           map[string]any{"type": "string"},
						"idle_in_transaction_session_timeout": map[string]any{"type": "integer"},
						"jit":                                 map[string]any{"type": "boolean"},
						"log_autovacuum_min_duration":         map[string]any{"type": "integer"},
						"log_error_verbosity":                 map[string]any{"type": "string"},
						"log_line_prefix":                     map[string]any{"type": "string"},
						"log_min_duration_statement":          map[string]any{"type": "integer"},
						"max_files_per_process":               map[string]any{"type": "integer"},
						"max_prepared_transactions":           map[string]any{"type": "integer"},
						"max_pred_locks_per_transaction":      map[string]any{"type": "integer"},
						"max_locks_per_transaction":           map[string]any{"type": "integer"},
						"max_stack_depth":                     map[string]any{"type": "integer"},
						"max_standby_archive_delay":           map[string]any{"type": "integer"},
						"max_standby_streaming_delay":         map[string]any{"type": "integer"},
						"max_replication_slots":               map[string]any{"type": "integer"},
						"max_logical_replication_workers":     map[string]any{"type": "integer"},
						"max_parallel_workers":                map[string]any{"type": "integer"},
						"max_parallel_workers_per_gather":     map[string]any{"type": "integer"},
						"max_worker_processes":                map[string]any{"type": "integer"},
						"pg_partman_bgw.role":                 map[string]any{"type": "string"},
						"pg_partman_bgw.interval":             map[string]any{"type": "integer"},
						"pg_stat_statements.track":            map[string]any{"type": "string"},
						"temp_file_limit":                     map[string]any{"type": "integer"},
						"timezone":                            map[string]any{"type": "string"},
						"track_activity_query_size":           map[string]any{"type": "integer"},
						"track_commit_timestamp":              map[string]any{"type": "string"},
						"track_functions":                     map[string]any{"type": "string"},
						"track_io_timing":                     map[string]any{"type": "string"},
						"max_wal_senders":                     map[string]any{"type": "integer"},
						"wal_sender_timeout":                  map[string]any{"type": "integer"},
						"wal_writer_delay":                    map[string]any{"type": "integer"},
						"shared_buffers_percentage":           map[string]any{"type": "number"},
						"backup_hour":                         map[string]any{"type": "integer"},
						"backup_minute":                       map[string]any{"type": "integer"},
						"work_mem":                            map[string]any{"type": "integer"},
						"synchronous_replication":             map[string]any{"type": "string"},
						"stat_monitor_enable":                 map[string]any{"type": "boolean"},
						"max_failover_replication_time_lag":   map[string]any{"type": "integer"},
						// For nested objects like pgbouncer/timescaledb, you can expand them or pass as JSON strings
						"pgbouncer":   map[string]any{"type": "object"},
						"timescaledb": map[string]any{"type": "object"},
					}),
				),
			),
		},
	}
}
