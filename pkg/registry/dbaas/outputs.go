package dbaas

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// Every payload here is a bare godo resource or a bare array, so each one takes
// an envelope to satisfy MCP's object-root requirement; no handler in this
// package hand-wraps its payload, so the text content is unchanged throughout.
//
// Several vars are shared, because the tools behind them return the same
// resource: clusterOut by db-cluster-get and db-cluster-create, topicOut by
// db-cluster-get-topic and db-cluster-create-topic, userOut by the get, create
// and update user tools, and migrationOut by both the start and the status
// online-migration tools.
//
// The per-engine config tools collapse to one var each rather than one shared
// var, because godo models every engine's configuration as its own type; they
// do share the "config" envelope key, which is what a client keys off.
//
// Left text-only: every update/set tool (db-cluster-update-{kafka,mongodb,
// mysql,os,psql,redis}-config, db-cluster-update-firewall-rules,
// db-cluster-update-topic, db-cluster-set-sql-mode), every delete tool
// (db-cluster-delete, db-cluster-delete-topic, db-cluster-delete-user) and the
// remaining cluster actions (db-cluster-resize,
// db-cluster-upgrade-major-version, db-cluster-stop-online-migration) return a
// fixed success message rather than a resource, so an output schema would
// describe nothing.
//
// sqlModeOut is the one exception to "no handler hand-wraps its payload":
// db-cluster-get-sql-mode answers with a bare SQL-mode string, which cannot be
// an object root, so it takes an envelope. Its text content stays the bare
// string it has always been, which is why that tool uses ResultWithText.
var (
	clusterOut          = common.NewOutput[*godo.Database]("cluster")
	clusterListOut      = common.NewOutput[[]godo.Database]("databases")
	caOut               = common.NewOutput[*godo.DatabaseCA]("ca")
	backupListOut       = common.NewOutput[[]godo.DatabaseBackup]("backups")
	optionsOut          = common.NewOutput[*godo.DatabaseOptions]("options")
	migrationOut        = common.NewOutput[*godo.DatabaseOnlineMigrationStatus]("migration")
	firewallRuleListOut = common.NewOutput[[]godo.DatabaseFirewallRule]("rules")
	topicOut            = common.NewOutput[*godo.DatabaseTopic]("topic")
	topicListOut        = common.NewOutput[[]godo.DatabaseTopic]("topics")
	userOut             = common.NewOutput[*godo.DatabaseUser]("user")
	userListOut         = common.NewOutput[[]godo.DatabaseUser]("users")
	kafkaConfigOut      = common.NewOutput[*godo.KafkaConfig]("config")
	mongoConfigOut      = common.NewOutput[*godo.MongoDBConfig]("config")
	mysqlConfigOut      = common.NewOutput[*godo.MySQLConfig]("config")
	opensearchConfigOut = common.NewOutput[*godo.OpensearchConfig]("config")
	postgresConfigOut   = common.NewOutput[*godo.PostgreSQLConfig]("config")
	redisConfigOut      = common.NewOutput[*godo.RedisConfig]("config")
	sqlModeOut          = common.NewOutput[string]("sql_mode")
)
