package ghost

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-ost/go/base"
	ghostsql "github.com/github/gh-ost/go/sql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

var defaultConfig = struct {
	allowedRunningOnMaster              bool
	concurrentCountTableRows            bool
	timestampAllTable                   bool
	hooksStatusIntervalSec              int64
	heartbeatIntervalMilliseconds       int64
	niceRatio                           float64
	chunkSize                           int64
	dmlBatchSize                        int64
	maxLagMillisecondsThrottleThreshold int64
	defaultNumRetries                   int64
	cutoverLockTimeoutSeconds           int64
	exponentialBackoffMaxInterval       int64
	throttleHTTPIntervalMillis          int64
	throttleHTTPTimeoutMillis           int64
}{
	allowedRunningOnMaster:              true, // allow-on-master
	concurrentCountTableRows:            true, // concurrent-rowcount
	timestampAllTable:                   true, // doesn't have a gh-ost cli flag counterpart
	hooksStatusIntervalSec:              60,   // hooks-status-interval
	heartbeatIntervalMilliseconds:       100,  // heartbeat-interval-millis
	niceRatio:                           0,    // nice-ration
	chunkSize:                           1000, // chunk-size
	dmlBatchSize:                        10,   // dml-batch-size
	maxLagMillisecondsThrottleThreshold: 1500, // max-lag-millis
	defaultNumRetries:                   60,   // default-retries
	cutoverLockTimeoutSeconds:           60,   // cut-over-lock-timeout-seconds
	exponentialBackoffMaxInterval:       64,   // exponential-backoff-max-interval
	throttleHTTPIntervalMillis:          100,  // throttle-http-interval-millis
	throttleHTTPTimeoutMillis:           1000, // throttle-http-timeout-millis
}

type UserFlags struct {
	maxLoad                 *string
	chunkSize               *int
	initiallyDropGhostTable *bool
	maxLagMillis            *int
	allowOnMaster           *bool
	switchToRBR             *bool
}

var knownKeys = map[string]bool{
	"max-load":                   true,
	"chunk-size":                 true,
	"initially-drop-ghost-table": true,
	"max-lag-millis":             true,
	"allow-on-master":            true,
	"switch-to-rbr":              true,
}

func GetUserFlags(flags map[string]string) (*UserFlags, error) {
	for k := range flags {
		if !knownKeys[k] {
			return nil, errors.Errorf("unsupported flag: %s", k)
		}
	}

	f := &UserFlags{}
	if v, ok := flags["max-load"]; ok {
		if _, err := base.ParseLoadMap(v); err != nil {
			return nil, errors.Wrapf(err, "failed to parse max-load %q", v)
		}
		f.maxLoad = &v
	}
	if v, ok := flags["chunk-size"]; ok {
		chunkSize, err := strconv.Atoi(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert chunk-size %q to int", v)
		}
		f.chunkSize = &chunkSize
	}
	if v, ok := flags["initially-drop-ghost-table"]; ok {
		initiallyDropGhostTable, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert initially-drop-ghost-table %q to bool", v)
		}
		f.initiallyDropGhostTable = &initiallyDropGhostTable
	}
	if v, ok := flags["max-lag-millis"]; ok {
		maxLagMillis, err := strconv.Atoi(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert max-lag-millis %q to int", v)
		}
		f.maxLagMillis = &maxLagMillis
	}
	if v, ok := flags["allow-on-master"]; ok {
		allowOnMaster, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert allow-on-master %q to bool", v)
		}
		f.allowOnMaster = &allowOnMaster
	}
	if v, ok := flags["switch-to-rbr"]; ok {
		switchToRBR, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert switch-to-rbr %q to bool", v)
		}
		f.switchToRBR = &switchToRBR
	}
	return f, nil
}

func getSocketFilename(taskID int, databaseID int, databaseName string, tableName string) string {
	return fmt.Sprintf("/tmp/gh-ost.%v.%v.%v.%v.sock", taskID, databaseID, databaseName, tableName)
}

// GetPostponeFlagFilename gets the postpone flag filename for gh-ost.
func GetPostponeFlagFilename(taskID int, databaseID int, databaseName string, tableName string) string {
	return fmt.Sprintf("/tmp/gh-ost.%v.%v.%v.%v.postponeFlag", taskID, databaseID, databaseName, tableName)
}

// NewMigrationContext is the context for gh-ost migration.
func NewMigrationContext(taskID int, database *store.DatabaseMessage, dataSource *store.DataSourceMessage, secret string, tableName string, statement string, noop bool, serverIDOffset uint) (*base.MigrationContext, error) {
	password, err := common.Unobfuscate(dataSource.ObfuscatedPassword, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get password")
	}

	migrationContext := base.NewMigrationContext()
	migrationContext.InspectorConnectionConfig.Key.Hostname = dataSource.Host
	port := 3306
	if dataSource.Port != "" {
		dsPort, err := strconv.Atoi(dataSource.Port)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert port from string to int")
		}
		port = dsPort
	}
	migrationContext.InspectorConnectionConfig.Key.Port = port
	migrationContext.CliUser = dataSource.Username
	migrationContext.CliPassword = password
	migrationContext.DatabaseName = database.DatabaseName
	migrationContext.OriginalTableName = tableName
	migrationContext.AlterStatement = strings.Join(strings.Fields(statement), " ")
	migrationContext.Noop = noop
	// On the source and each replica, you must set the server_id system variable to establish a unique replication ID. For each server, you should pick a unique positive integer in the range from 1 to 2^32 − 1, and each ID must be different from every other ID in use by any other source or replica in the replication topology. Example: server-id=3.
	// https://dev.mysql.com/doc/refman/5.7/en/replication-options-source.html
	// Here we use serverID = offset + task.ID to avoid potential conflicts.
	migrationContext.ReplicaServerId = serverIDOffset + uint(taskID)
	// set defaults
	if err := migrationContext.SetConnectionConfig(""); err != nil {
		return nil, err
	}
	migrationContext.AllowedRunningOnMaster = defaultConfig.allowedRunningOnMaster
	migrationContext.ConcurrentCountTableRows = defaultConfig.concurrentCountTableRows
	migrationContext.HooksStatusIntervalSec = defaultConfig.hooksStatusIntervalSec
	migrationContext.CutOverType = base.CutOverAtomic
	migrationContext.ThrottleHTTPIntervalMillis = defaultConfig.throttleHTTPIntervalMillis
	migrationContext.ThrottleHTTPTimeoutMillis = defaultConfig.throttleHTTPTimeoutMillis

	if migrationContext.AlterStatement == "" {
		return nil, errors.Errorf("alterStatement must be provided and must not be empty")
	}
	parser := ghostsql.NewParserFromAlterStatement(migrationContext.AlterStatement)
	migrationContext.AlterStatementOptions = parser.GetAlterStatementOptions()

	if migrationContext.DatabaseName == "" {
		if !parser.HasExplicitSchema() {
			return nil, errors.Errorf("database must be provided and database name must not be empty, or alterStatement must specify database name")
		}
		migrationContext.DatabaseName = parser.GetExplicitSchema()
	}
	if migrationContext.OriginalTableName == "" {
		if !parser.HasExplicitTable() {
			return nil, errors.Errorf("table must be provided and table name must not be empty, or alterStatement must specify table name")
		}
		migrationContext.OriginalTableName = parser.GetExplicitTable()
	}
	migrationContext.ServeSocketFile = getSocketFilename(taskID, database.UID, database.DatabaseName, tableName)
	migrationContext.PostponeCutOverFlagFile = GetPostponeFlagFilename(taskID, database.UID, database.DatabaseName, tableName)
	migrationContext.TimestampAllTable = defaultConfig.timestampAllTable
	migrationContext.SetHeartbeatIntervalMilliseconds(defaultConfig.heartbeatIntervalMilliseconds)
	migrationContext.SetNiceRatio(defaultConfig.niceRatio)
	migrationContext.SetChunkSize(defaultConfig.chunkSize)
	migrationContext.SetDMLBatchSize(defaultConfig.dmlBatchSize)
	migrationContext.SetMaxLagMillisecondsThrottleThreshold(defaultConfig.maxLagMillisecondsThrottleThreshold)
	migrationContext.SetDefaultNumRetries(defaultConfig.defaultNumRetries)
	migrationContext.ApplyCredentials()
	if err := migrationContext.SetCutOverLockTimeoutSeconds(defaultConfig.cutoverLockTimeoutSeconds); err != nil {
		return nil, err
	}
	if err := migrationContext.SetExponentialBackoffMaxInterval(defaultConfig.exponentialBackoffMaxInterval); err != nil {
		return nil, err
	}
	return migrationContext, nil
}

// GetTableNameFromStatement gets the table name from statement for gh-ost.
func GetTableNameFromStatement(statement string) (string, error) {
	// Trim the statement for the parser.
	// This in effect removes all leading and trailing spaces, substitute multiple spaces with one.
	statement = strings.Join(strings.Fields(statement), " ")
	parser := ghostsql.NewParserFromAlterStatement(statement)
	if !parser.HasExplicitTable() {
		return "", errors.Errorf("failed to parse table name from statement, statement: %v", statement)
	}
	return parser.GetExplicitTable(), nil
}
