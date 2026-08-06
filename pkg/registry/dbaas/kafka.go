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

type KafkaTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

func NewKafkaTool(client func(ctx context.Context) (*godo.Client, error)) *KafkaTool {
	return &KafkaTool{client: client}
}

func (s *KafkaTool) getKafkaConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	cfg, _, err := client.Databases.GetKafkaConfig(ctx, id)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonCfg, err := response.CompactJSON(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonCfg), nil
}

func (s *KafkaTool) updateKafkaConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	cfgMap, ok := args["config"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Missing or invalid 'config' object"), nil
	}
	cfgBytes, err := json.Marshal(cfgMap)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	var config godo.KafkaConfig
	if err = json.Unmarshal(cfgBytes, &config); err != nil {
		return mcp.NewToolResultError("Invalid config object: " + err.Error()), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.UpdateKafkaConfig(ctx, id, &config)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Kafka config updated successfully"), nil
}

func (s *KafkaTool) listTopics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	opts := &godo.ListOptions{}
	if pStr, ok := args["page"].(string); ok && pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil {
			opts.Page = p
		}
	}
	if pp, ok := args["per_page"].(int); ok {
		opts.PerPage = pp
	}
	if wpStr, ok := args["with_projects"].(string); ok && wpStr != "" {
		if wp, err := strconv.ParseBool(wpStr); err == nil {
			opts.WithProjects = wp
		}
	}
	if odStr, ok := args["only_deployed"].(string); ok && odStr != "" {
		if od, err := strconv.ParseBool(odStr); err == nil {
			opts.Deployed = od
		}
	}
	if poStr, ok := args["public_only"].(string); ok && poStr != "" {
		if po, err := strconv.ParseBool(poStr); err == nil {
			opts.PublicOnly = po
		}
	}
	if ucStr, ok := args["usecases"].(string); ok && ucStr != "" {
		ucList := []string{}
		for _, u := range strings.Split(ucStr, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				ucList = append(ucList, u)
			}
		}
		opts.Usecases = ucList
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	topics, _, err := client.Databases.ListTopics(ctx, id, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonTopics, err := response.CompactJSON(topics)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonTopics), nil
}

func (s *KafkaTool) createTopic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Topic name is required"), nil
	}

	var partitionCount *uint32
	if pcStr, ok := args["partition_count"].(string); ok && pcStr != "" {
		if pc, err := strconv.ParseUint(pcStr, 10, 32); err == nil {
			pc32 := uint32(pc)
			partitionCount = &pc32
		}
	}
	var replicationFactor *uint32
	if rfStr, ok := args["replication_factor"].(string); ok && rfStr != "" {
		if rf, err := strconv.ParseUint(rfStr, 10, 32); err == nil {
			rf32 := uint32(rf)
			replicationFactor = &rf32
		}
	}

	var topicConfig *godo.TopicConfig
	if cfgMap, ok := args["config"].(map[string]any); ok {
		cfgBytes, _ := json.Marshal(cfgMap)
		var cfg godo.TopicConfig
		if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
			return mcp.NewToolResultError("Invalid config object: " + err.Error()), nil
		}
		topicConfig = &cfg
	}

	createReq := &godo.DatabaseCreateTopicRequest{
		Name:              name,
		PartitionCount:    partitionCount,
		ReplicationFactor: replicationFactor,
		Config:            topicConfig,
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	topic, _, err := client.Databases.CreateTopic(ctx, id, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonTopic, err := response.CompactJSON(topic)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonTopic), nil
}

func (s *KafkaTool) getTopic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Topic name is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	topic, _, err := client.Databases.GetTopic(ctx, id, name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	jsonTopic, err := response.CompactJSON(topic)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(jsonTopic), nil
}

func (s *KafkaTool) deleteTopic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Topic name is required"), nil
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.DeleteTopic(ctx, id, name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Topic deleted successfully"), nil
}

func (s *KafkaTool) updateTopic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Topic name is required"), nil
	}

	var partitionCount *uint32
	if pcStr, ok := args["partition_count"].(string); ok && pcStr != "" {
		if pc, err := strconv.ParseUint(pcStr, 10, 32); err == nil {
			pc32 := uint32(pc)
			partitionCount = &pc32
		}
	}
	var replicationFactor *uint32
	if rfStr, ok := args["replication_factor"].(string); ok && rfStr != "" {
		if rf, err := strconv.ParseUint(rfStr, 10, 32); err == nil {
			rf32 := uint32(rf)
			replicationFactor = &rf32
		}
	}

	var topicConfig *godo.TopicConfig
	if cfgMap, ok := args["config"].(map[string]any); ok {
		cfgBytes, _ := json.Marshal(cfgMap)
		var cfg godo.TopicConfig
		if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
			return mcp.NewToolResultError("Invalid config object: " + err.Error()), nil
		}
		topicConfig = &cfg
	}

	updateReq := &godo.DatabaseUpdateTopicRequest{
		PartitionCount:    partitionCount,
		ReplicationFactor: replicationFactor,
		Config:            topicConfig,
	}
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}
	_, err = client.Databases.UpdateTopic(ctx, id, name, updateReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("Topic updated successfully"), nil
}

func (s *KafkaTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.listTopics,
			Tool: mcp.NewTool("db-cluster-list-topics",
				mcp.WithDescription("List topics for a Kafka cluster by its ID. Supports pagination and filtering."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The Kafka cluster UUID")),
				mcp.WithString("page", mcp.Description("Page number (string)")),
				mcp.WithNumber("per_page", mcp.Description("Number of results per page (integer)")),
				mcp.WithString("with_projects", mcp.Description("Include project field (bool as string)")),
				mcp.WithString("only_deployed", mcp.Description("Only deployed topics (bool as string)")),
				mcp.WithString("public_only", mcp.Description("Only public models (bool as string)")),
				mcp.WithString("usecases", mcp.Description("Comma-separated usecases (optional)")),
			),
		},
		{
			Handler: s.createTopic,
			Tool: mcp.NewTool("db-cluster-create-topic",
				mcp.WithDescription("Create a topic for a Kafka cluster."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Kafka cluster UUID")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Topic name")),
				mcp.WithString("partition_count", mcp.Description("Number of partitions")),
				mcp.WithString("replication_factor", mcp.Description("Replication factor")),
				mcp.WithObject("config",
					mcp.Description("Kafka topic configuration (optional)"),
					mcp.Properties(map[string]any{
						"cleanup_policy":                      map[string]any{"type": "string"},
						"compression_type":                    map[string]any{"type": "string"},
						"delete_retention_ms":                 map[string]any{"type": "integer"},
						"flush_messages":                      map[string]any{"type": "integer"},
						"flush_ms":                            map[string]any{"type": "integer"},
						"index_interval_bytes":                map[string]any{"type": "integer"},
						"max_compaction_lag_ms":               map[string]any{"type": "integer"},
						"max_message_bytes":                   map[string]any{"type": "integer"},
						"message_down_conversion_enable":      map[string]any{"type": "boolean"},
						"message_format_version":              map[string]any{"type": "string"},
						"message_timestamp_difference_max_ms": map[string]any{"type": "integer"},
						"message_timestamp_type":              map[string]any{"type": "string"},
						"min_cleanable_dirty_ratio":           map[string]any{"type": "number"},
						"min_compaction_lag_ms":               map[string]any{"type": "integer"},
						"min_insync_replicas":                 map[string]any{"type": "integer"},
						"preallocate":                         map[string]any{"type": "boolean"},
						"retention_bytes":                     map[string]any{"type": "integer"},
						"retention_ms":                        map[string]any{"type": "integer"},
						"segment_bytes":                       map[string]any{"type": "integer"},
						"segment_index_bytes":                 map[string]any{"type": "integer"},
						"segment_jitter_ms":                   map[string]any{"type": "integer"},
						"segment_ms":                          map[string]any{"type": "integer"},
					}),
				),
			),
		},
		{
			Handler: s.getTopic,
			Tool: mcp.NewTool("db-cluster-get-topic",
				mcp.WithDescription("Get a Kafka topic by name."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Kafka cluster UUID")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Topic name")),
			),
		},
		{
			Handler: s.deleteTopic,
			Tool: mcp.NewTool("db-cluster-delete-topic",
				mcp.WithDescription("Delete a Kafka topic by name."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Kafka cluster UUID")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Topic name")),
			),
		},
		{
			Handler: s.updateTopic,
			Tool: mcp.NewTool("db-cluster-update-topic",
				mcp.WithDescription("Update a Kafka topic's partition count, replication factor, or config."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Kafka cluster UUID")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Topic name")),
				mcp.WithString("partition_count", mcp.Description("Number of partitions")),
				mcp.WithString("replication_factor", mcp.Description("Replication factor")),
				mcp.WithObject("config",
					mcp.Description("Kafka topic configuration (optional)"),
					mcp.Properties(map[string]any{
						"cleanup_policy":                      map[string]any{"type": "string"},
						"compression_type":                    map[string]any{"type": "string"},
						"delete_retention_ms":                 map[string]any{"type": "integer"},
						"flush_messages":                      map[string]any{"type": "integer"},
						"flush_ms":                            map[string]any{"type": "integer"},
						"index_interval_bytes":                map[string]any{"type": "integer"},
						"max_compaction_lag_ms":               map[string]any{"type": "integer"},
						"max_message_bytes":                   map[string]any{"type": "integer"},
						"message_down_conversion_enable":      map[string]any{"type": "boolean"},
						"message_format_version":              map[string]any{"type": "string"},
						"message_timestamp_difference_max_ms": map[string]any{"type": "integer"},
						"message_timestamp_type":              map[string]any{"type": "string"},
						"min_cleanable_dirty_ratio":           map[string]any{"type": "number"},
						"min_compaction_lag_ms":               map[string]any{"type": "integer"},
						"min_insync_replicas":                 map[string]any{"type": "integer"},
						"preallocate":                         map[string]any{"type": "boolean"},
						"retention_bytes":                     map[string]any{"type": "integer"},
						"retention_ms":                        map[string]any{"type": "integer"},
						"segment_bytes":                       map[string]any{"type": "integer"},
						"segment_index_bytes":                 map[string]any{"type": "integer"},
						"segment_jitter_ms":                   map[string]any{"type": "integer"},
						"segment_ms":                          map[string]any{"type": "integer"},
					}),
				),
			),
		},
		{
			Handler: s.getKafkaConfig,
			Tool: mcp.NewTool("db-cluster-get-kafka-config",
				mcp.WithDescription("Get the Kafka config for a cluster."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Kafka cluster UUID")),
			),
		},
		{
			Handler: s.updateKafkaConfig,
			Tool: mcp.NewTool("db-cluster-update-kafka-config",
				mcp.WithDescription("Update the Kafka cluster configuration."),
				mcp.WithString("id", mcp.Required(), mcp.Description("Kafka cluster UUID")),
				mcp.WithObject("config",
					mcp.Required(),
					mcp.Description("Kafka configuration object"),
					mcp.Properties(map[string]any{
						"group_initial_rebalance_delay_ms":        map[string]any{"type": "integer"},
						"group_min_session_timeout_ms":            map[string]any{"type": "integer"},
						"group_max_session_timeout_ms":            map[string]any{"type": "integer"},
						"message_max_bytes":                       map[string]any{"type": "integer"},
						"log_cleaner_delete_retention_ms":         map[string]any{"type": "integer"},
						"log_cleaner_min_compaction_lag_ms":       map[string]any{"type": "integer"},
						"log_flush_interval_ms":                   map[string]any{"type": "integer"},
						"log_index_interval_bytes":                map[string]any{"type": "integer"},
						"log_message_downconversion_enable":       map[string]any{"type": "boolean"},
						"log_message_timestamp_difference_max_ms": map[string]any{"type": "integer"},
						"log_preallocate":                         map[string]any{"type": "boolean"},
						"log_retention_bytes":                     map[string]any{"type": "integer"},
						"log_retention_hours":                     map[string]any{"type": "integer"},
						"log_retention_ms":                        map[string]any{"type": "integer"},
						"log_roll_jitter_ms":                      map[string]any{"type": "integer"},
						"log_segment_delete_delay_ms":             map[string]any{"type": "integer"},
						"auto_create_topics_enable":               map[string]any{"type": "boolean"},
					}),
				),
			),
		},
	}
}
