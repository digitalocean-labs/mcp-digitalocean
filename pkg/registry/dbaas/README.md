## DBaaS MCP Tools

This directory contains tools and resources for managing DigitalOcean managed database resources via the MCP Server. These tools enable you to create, modify, and query clusters, users, firewalls, configuration, topics, and other database-related resources.

---

## Supported Tools

### Cluster Tools

- **`db-cluster-list`**

  - Get list of clusters.
  - **Arguments:**
    - `page` (optional, integer as string): Page number for pagination
    - `per_page` (optional, integer): Number of results per page

- **`db-cluster-get`**

  - Get a cluster by its ID.
  - **Arguments:**
    - `id` (required): The ID of the cluster to retrieve

- **`db-cluster-get-ca`**

  - Get the CA certificate for a cluster by its ID.
  - **Arguments:**
    - `id` (required): The ID of the cluster to retrieve the CA for

- **`db-cluster-create`**

  - Create a new database cluster.
  - **Arguments:**
    - `name` (required): The name of the cluster
    - `engine` (required): The engine slug (e.g., valkey, pg, mysql)
    - `version` (required): The engine version (e.g., 14, 8.0, etc.)
    - `region` (required): The region slug (e.g., nyc1)
    - `size` (required): The size slug (e.g., db-s-2vcpu-4gb)
    - `num_nodes` (required, number): The number of nodes
    - `tags` (optional, string): Comma-separated tags

- **`db-cluster-delete`**

  - Delete a database cluster by its ID.
  - **Arguments:**
    - `id` (required): The ID of the cluster to delete

- **`db-cluster-resize`**

  - Resize a database cluster by its ID. At least one of the following must be provided: `size`, `num_nodes`, or `storage_size_mib`.
  - **Arguments:**
    - `id` (required): The ID of the cluster to resize
    - `size` (optional): The new cluster size (e.g., db-s-4vcpu-8gb)
    - `num_nodes` (optional, number): The new number of nodes
    - `storage_size_mib` (optional, number): New storage size in MiB

- **`db-cluster-list-backups`**

  - List backups for a database cluster by its ID.
  - **Arguments:**
    - `id` (required): The ID of the cluster
    - `page` (optional, integer as string): Page number
    - `per_page` (optional, integer): Results per page

- **`db-cluster-list-options`**

  - List available cluster creation options, including engines, sizes, and regions.
  - **Arguments:** None

- **`db-cluster-upgrade-major-version`**

  - Upgrade the major database version of a cluster.
  - **Arguments:**
    - `id` (required): The cluster ID
    - `version` (required): Target major version (e.g., 15)

- **`db-cluster-start-online-migration`**

  - Start an online migration for a cluster.
  - **Arguments:**
    - `id` (required): The cluster ID
    - `source` (required, object): Source DB connection info
      - `host` (string): Hostname or IP
      - `port` (integer): Source port
      - `dbname` (string): Source database name
      - `username` (string): Connection username
      - `password` (string): Connection password
    - `disable_ssl` (optional, boolean): Disable SSL
    - `ignore_dbs` (optional, string): Comma-separated DBs to ignore

- **`db-cluster-stop-online-migration`**

  - Cancel an ongoing online migration.
  - **Arguments:**
    - `id` (required): Cluster ID
    - `migration_id` (required): Migration ID to stop

- **`db-cluster-get-migration`**

  - Query the current status of an online migration.
  - **Arguments:**
    - `id` (required): Cluster ID


### Firewall Tools

- **`db-cluster-get-firewall-rules`**

  - Get the firewall rules for a cluster by its ID.
  - **Arguments:**
    - `id` (required, string): The cluster UUID

- **`db-cluster-update-firewall-rules`**

  - Update the firewall rules for a cluster by its ID.
  - **Arguments:**
    - `id` (required, string): The cluster UUID
    - `rules` (required, array of objects): The list of firewall rules to apply. Each rule supports:
      - `uuid` (optional, string): Rule UUID (for updating existing rules)
      - `cluster_uuid` (optional, string): UUID of the associated cluster
      - `type` (required, string): Type of rule (`ip_addr`, `droplet`, `tag`, `app`, etc.)
      - `value` (required, string): IP address, tag name, or droplet ID

### Kafka Tools

- **`db-cluster-list-topics`**

  - List topics for a Kafka cluster by its ID. Supports list options and filters.
  - **Arguments:**
    - `id` (required, string): The Kafka cluster UUID
    - `page` (optional, integer as string): Page number for pagination
    - `per_page` (optional, integer): Number of results per page
    - `with_projects` (optional, bool as string): Include project fields
    - `only_deployed` (optional, bool as string): Only list deployed topics
    - `public_only` (optional, bool as string): Only include public topics
    - `usecases` (optional, string): Comma-separated list of usecases to include

- **`db-cluster-create-topic`**

  - Create a topic for a Kafka database cluster.
  - **Arguments:**
    - `id` (required, string): The Kafka cluster UUID
    - `name` (required, string): The topic name
    - `partition_count` (optional, integer as string): Number of partitions
    - `replication_factor` (optional, integer as string): Replication factor
    - `config` (optional, object): Configuration for the topic with the following fields:
      - `cleanup_policy`, `compression_type`, `delete_retention_ms`, `flush_messages`, `flush_ms`,
      - `index_interval_bytes`, `max_compaction_lag_ms`, `max_message_bytes`,
      - `message_down_conversion_enable`, `message_format_version`, `message_timestamp_difference_max_ms`,
      - `message_timestamp_type`, `min_cleanable_dirty_ratio`, `min_compaction_lag_ms`, `min_insync_replicas`,
      - `preallocate`, `retention_bytes`, `retention_ms`, `segment_bytes`, `segment_index_bytes`,
      - `segment_jitter_ms`, `segment_ms`

- **`db-cluster-get-topic`**

  - Get a topic’s details from a Kafka database cluster.
  - **Arguments:**
    - `id` (required, string): The Kafka cluster UUID
    - `name` (required, string): The topic name

- **`db-cluster-delete-topic`**

  - Delete a topic for a Kafka database cluster.
  - **Arguments:**
    - `id` (required, string): The Kafka cluster UUID
    - `name` (required, string): The topic name

- **`db-cluster-update-topic`**

  - Update a topic's partition count, replication factor, or config settings.
  - **Arguments:**
    - `id` (required, string): The Kafka cluster UUID
    - `name` (required, string): Topic name
    - `partition_count` (optional, integer as string): Updated number of partitions
    - `replication_factor` (optional, integer as string): Updated replication factor
    - `config` (optional, object): Same fields as `create-topic`'s `config` object

- **`db-cluster-get-kafka-config`**

  - Get the Kafka configuration for a cluster.
  - **Arguments:**
    - `id` (required, string): The Kafka cluster UUID

- **`db-cluster-update-kafka-config`**

  - Update the Kafka cluster configuration using a structured object.
  - **Arguments:**
    - `id` (required, string): The Kafka cluster UUID
    - `config` (required, object): Configuration object supporting:
      - `group_initial_rebalance_delay_ms`, `group_min_session_timeout_ms`, `group_max_session_timeout_ms`,
      - `message_max_bytes`, `log_cleaner_delete_retention_ms`, `log_cleaner_min_compaction_lag_ms`,
      - `log_flush_interval_ms`, `log_index_interval_bytes`, `log_message_downconversion_enable`,
      - `log_message_timestamp_difference_max_ms`, `log_preallocate`, `log_retention_bytes`, `log_retention_hours`,
      - `log_retention_ms`, `log_roll_jitter_ms`, `log_segment_delete_delay_ms`,
      - `auto_create_topics_enable`

### Mongo Tools

- **`db-cluster-get-mongodb-config`**

  - Get the MongoDB config for a cluster by its ID.
  - **Arguments:**
    - `id` (required, string): The cluster UUID

- **`db-cluster-update-mongodb-config`**

  - Update the MongoDB config for a cluster by its ID using a structured object.
  - **Arguments:**
    - `id` (required, string): The cluster UUID
    - `config` (required, object): Configuration parameters for MongoDB:
      - `default_read_concern` (optional, string): e.g., `local`, `majority`
      - `default_write_concern` (optional, string): e.g., `majority`
      - `transaction_lifetime_limit_seconds` (optional, integer): Transaction timeout
      - `slow_op_threshold_ms` (optional, integer): Log slow ops above this threshold (ms)
      - `verbosity` (optional, integer): Logging verbosity (0–5)

### MySQL Tools

- **`db-cluster-get-mysql-config`**

  - Get the MySQL config for a cluster by its ID.
  - **Arguments:**
    - `ID` (required): The cluster UUID

- **`db-cluster-update-mysql-config`**

  - Update the MySQL config for a cluster by its ID. Accepts a JSON string for the config.
  - **Arguments:**
    - `ID` (required): The cluster UUID
    - `config_json` (required): JSON for the MySQLConfig to set

- **`db-cluster-get-sql-mode`**

  - Get the SQL mode for a cluster by its ID.
  - **Arguments:**
    - `ID` (required): The cluster UUID

- **`db-cluster-set-sql-mode`**
  - Set the SQL mode for a cluster by its ID.
  - **Arguments:**
    - `ID` (required): The cluster UUID
    - `modes` (required): Comma-separated SQL modes to set


Absolutely! Based on your provided updated `MysqlTool` code, here's the ✅ updated **MySQL Tools** section of your README in your preferred format:

### MySQL Tools

- **`db-cluster-get-sql-mode`**

  - Get the SQL mode for a cluster.
  - **Arguments:**
    - `id` (required, string): The cluster UUID

- **`db-cluster-set-sql-mode`**

  - Set the SQL mode for a cluster using a comma-separated list.
  - **Arguments:**
    - `id` (required, string): The cluster UUID
    - `modes` (required, string): Comma-separated SQL modes to set


### Opensearch Tools

- **`db-cluster-get-opensearch-config`**

  - Get the OpenSearch config for a cluster by its ID.
  - **Arguments:**
    - `id` (required, string): The cluster UUID

- **`db-cluster-update-os-config`**

  - Update the OpenSearch config for a cluster using a structured object.
  - **Arguments:**
    - `id` (required, string): The cluster UUID
    - `config` (required, object): A structured object that supports dozens of OpenSearch configuration parameters. A few examples include:
      - `http_max_content_length_bytes` (integer)
      - `indices_query_bool_max_clause_count` (integer)
      - `ism_enabled` (boolean)
      - `search_max_buckets` (integer)
      - `thread_pool_write_queue_size` (integer)
      - `cluster_max_shards_per_node` (integer)
      - `script_max_compilations_rate` (string)
      - `reindex_remote_whitelist` (array of strings)
      - `plugins_alerting_filter_by_backend_roles_enabled` (boolean)

### Postgres Tools

- **`db-cluster-get-postgresql-config`**

  - Get the PostgreSQL config for a cluster by its ID.
  - **Arguments:**
    - `id` (required, string): The cluster UUID

- **`db-cluster-update-psql-config`**

  - Update the PostgreSQL config for a cluster using a structured object.
  - **Arguments:**
    - `id` (required, string): The cluster UUID
    - `config` (required, object): A structured configuration object that supports dozens of PostgreSQL tuning parameters. Examples include:
      - `autovacuum_max_workers` (integer)
      - `autovacuum_vacuum_scale_factor` (number)
      - `shared_buffers_percentage` (number)
      - `timezone` (string)
      - `temp_file_limit` (integer)
      - `jit` (boolean)
      - `pgbouncer` (object)
      - `timescaledb` (object)
      - ...and many more.

### Redis Tools

- **`db-cluster-get-redis-config`**

  - Get the Redis config for a cluster by its ID.
  - **Arguments:**
    - `id` (required, string): The cluster UUID

- **`db-cluster-update-redis-config`**

  - Update the Redis config for a cluster by its ID using a structured `config` object.
  - **Arguments:**
    - `id` (required, string): The cluster UUID
    - `config` (required, object): Configuration for the Redis cluster. Includes:
      - `redis_maxmemory_policy` (string): Eviction policy (e.g., `allkeys-lru`)
      - `redis_pubsub_client_output_buffer_limit` (integer)
      - `redis_number_of_databases` (integer)
      - `redis_io_threads` (integer)
      - `redis_lfu_log_factor` (integer)
      - `redis_lfu_decay_time` (integer)
      - `redis_ssl` (boolean)
      - `redis_timeout` (integer)
      - `redis_notify_keyspace_events` (string)
      - `redis_persistence` (string): e.g., `aof`, `rdb`, or `none`
      - `redis_acl_channels_default` (string)

### User Tools

- **`db-cluster-get-user`**

  - Get a database user by cluster ID and user name.
  - **Arguments:**
    - `id` (required, string): The cluster ID (UUID)
    - `user` (required, string): The user name

- **`db-cluster-list-users`**

  - List database users for a cluster by its ID.
  - **Arguments:**
    - `id` (required, string): The cluster ID (UUID)
    - `page` (optional, string): Page number for pagination
    - `per_page` (optional, integer): Number of results per page

- **`db-cluster-create-user`**

  - Create a database user for a cluster.
  - **Arguments:**
    - `id` (required, string): The cluster ID
    - `name` (required, string): The user name
    - `mysql_auth_plugin` (optional, string): MySQL auth plugin (e.g., `mysql_native_password`)
    - `settings` (optional, object): Structured settings object including:
      - `acl` (array of objects):
        - `id` (string)
        - `permission` (string)
        - `topic` (string)
      - `opensearch_acl` (array of objects):
        - `index` (string)
        - `permission` (string)
      - `mongo_user_settings` (object):
        - `databases` (array of strings)
        - `role` (string)

- **`db-cluster-update-user`**

  - Update a user’s settings in a given database cluster.
  - **Arguments:**
    - `id` (required, string): The cluster ID
    - `user` (required, string): The user name
    - `settings` (optional, object): Same structure as in `create-user`

- **`db-cluster-delete-user`**

  - Delete a user from a database cluster.
  - **Arguments:**
    - `id` (required, string): The cluster ID
    - `user` (required, string): The user name to delete


---


## Example Queries Using DBaaS MCP Tools

Below are some example natural language queries you might use, along with the corresponding tool and arguments. Each example shows how a user might phrase a request, and how it maps to the underlying tool and parameters.

### Clusters

| Example Query                                 | Tool                                 | Arguments                                                                                                           |
|----------------------------------------------|--------------------------------------|---------------------------------------------------------------------------------------------------------------------|
| Could you please list all DBaaS clusters?     | db-cluster-list  | `{ "page": "1", "per_page": 10 }`                                                                                   |
| Show me details for cluster ``  | db-cluster-get   | `{ "id": "" }`                                                                                       |
| Create a DBaaS cluster called "my-db" in nyc1 | db-cluster-create| `{ "name": "my-db", "engine": "mysql", "version": "8", "region": "nyc1", "size": "db-s-1vcpu-1gb", "num_nodes": 1 }`|
| Delete the cluster ``           | db-cluster-delete| `{ "id": "" }`                                                                                       |
| Resize cluster `` to 2 nodes    | db-cluster-resize| `{ "id": "", "num_nodes": 2 }`                                                                       |

### Users

| Example Query                                            | Tool                                      | Arguments                                                                                |
|----------------------------------------------------------|-------------------------------------------|-------------------------------------------------------------------------------------------|
| List all users for cluster ``              | db-cluster-list-users | `{ "id": "" }`                                                             |
| Add a user named "readonly" to cluster ``  | db-cluster-create-user| `{ "id": "", "name": "readonly" }`                                         |
| Remove the user "readonly" from cluster `` | db-cluster-delete-user| `{ "id": "", "user": "readonly" }`                                         |
| Update user "readonly" with ACL settings                 | db-cluster-update-user| `{ "id": "", "user": "readonly", "settings": { "acl": [{...}] } }`         |

### Firewalls

| Example Query                                             | Tool                                           | Arguments                                                                                           |
|-----------------------------------------------------------|------------------------------------------------|------------------------------------------------------------------------------------------------------|
| What are the firewall rules for cluster ``? | db-cluster-get-firewall-rules   | `{ "id": "" }`                                                                        |
| Update firewall rules for cluster ``        | db-cluster-update-firewall-rules| `{ "id": "", "rules": [ { "type": "ip_addr", "value": "1.2.3.4" } ] }`                |

### Configuration

| Example Query                                              | Tool                                                | Arguments                                                             |
|------------------------------------------------------------|-----------------------------------------------------|-----------------------------------------------------------------------|
| Show me the MySQL config for cluster ``      | db-cluster-get-mysql-config     | `{ "id": "" }`                                          |
| Update the MongoDB config for cluster ``     | db-cluster-update-mongodb-config| `{ "id": "", "config": { "verbosity": 3 } }`           |
| Get the Redis config for cluster ``          | db-cluster-get-redis-config     | `{ "id": "" }`                                          |
| Update the PostgreSQL config for cluster ``  | db-cluster-update-psql-config | `{ "id": "", "config": { "timezone": "UTC" } }`     |

### Kafka Topics

| Example Query                                                      | Tool                                       | Arguments                                                                            |
|--------------------------------------------------------------------|--------------------------------------------|---------------------------------------------------------------------------------------|
| List all topics in Kafka cluster ``                  | db-cluster-list-topics | `{ "id": "" }`                                                         |
| Create a topic named "my-topic" in cluster ``        | db-cluster-create-topic| `{ "id": "", "name": "my-topic" }`                                     |
| Delete the topic "my-topic" from Kafka cluster ``    | db-cluster-delete-topic| `{ "id": "", "name": "my-topic" }`                                     |
| Update topic "events" to have 6 partitions                         | db-cluster-update-topic| `{ "id": "", "name": "events", "partition_count": "6" }`               |
| Get Kafka config for cluster ``                      | db-cluster-get-kafka-config | `{ "id": "" }`                                                    |
| Update Kafka config                                                | db-cluster-update-kafka-config | `{ "id": "", "config": { "log_retention_ms": 86400000 } }`       |




---

Feel free to use these queries as a starting point for interacting with the DBaaS MCP tools!
