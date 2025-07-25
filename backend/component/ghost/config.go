package ghost

import (
	"context"
	"strconv"
	"strings"

	ghostbase "github.com/github/gh-ost/go/base"
	ghostsql "github.com/github/gh-ost/go/sql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	secretcomp "github.com/bytebase/bytebase/backend/component/secret"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

var defaultConfig = struct {
	attemptInstantDDL                   bool
	allowedRunningOnMaster              bool
	concurrentCountTableRows            bool
	timestampOldTable                   bool
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
	attemptInstantDDL:                   true,  // attempt-instant-ddl
	allowedRunningOnMaster:              true,  // allow-on-master
	concurrentCountTableRows:            true,  // concurrent-rowcount
	timestampOldTable:                   false, // doesn't have a gh-ost cli flag counterpart
	hooksStatusIntervalSec:              60,    // hooks-status-interval
	heartbeatIntervalMilliseconds:       100,   // heartbeat-interval-millis
	niceRatio:                           0,     // nice-ratio
	chunkSize:                           1000,  // chunk-size
	dmlBatchSize:                        10,    // dml-batch-size
	maxLagMillisecondsThrottleThreshold: 1500,  // max-lag-millis
	defaultNumRetries:                   60,    // default-retries
	cutoverLockTimeoutSeconds:           10,    // cut-over-lock-timeout-seconds
	exponentialBackoffMaxInterval:       64,    // exponential-backoff-max-interval
	throttleHTTPIntervalMillis:          100,   // throttle-http-interval-millis
	throttleHTTPTimeoutMillis:           1000,  // throttle-http-timeout-millis
}

type UserFlags struct {
	maxLoad                       *string
	chunkSize                     *int64
	dmlBatchSize                  *int64
	defaultRetries                *int64
	cutoverLockTimeoutSeconds     *int64
	exponentialBackoffMaxInterval *int64
	maxLagMillis                  *int64
	allowOnMaster                 *bool
	switchToRBR                   *bool
	assumeRBR                     *bool
	heartbeatIntervalMillis       *int64
	niceRatio                     *float64
	throttleControlReplicas       *string
	attemptInstantDDL             *bool
	assumeMasterHost              *bool // use datasource host if true
}

var knownKeys = map[string]bool{
	"max-load":                         true,
	"chunk-size":                       true,
	"dml-batch-size":                   true,
	"default-retries":                  true,
	"cut-over-lock-timeout-seconds":    true,
	"exponential-backoff-max-interval": true,
	"max-lag-millis":                   true,
	"allow-on-master":                  true,
	"switch-to-rbr":                    true,
	"assume-rbr":                       true,
	"heartbeat-interval-millis":        true,
	"nice-ratio":                       true,
	"throttle-control-replicas":        true,
	"attempt-instant-ddl":              true,
	"assume-master-host":               true,
}

func GetUserFlags(flags map[string]string) (*UserFlags, error) {
	f := &UserFlags{}
	if flags == nil {
		return f, nil
	}

	for k := range flags {
		if !knownKeys[k] {
			return nil, errors.Errorf("unsupported flag: %s", k)
		}
	}

	if v, ok := flags["max-load"]; ok {
		if _, err := ghostbase.ParseLoadMap(v); err != nil {
			return nil, errors.Wrapf(err, "failed to parse max-load %q", v)
		}
		f.maxLoad = &v
	}
	if v, ok := flags["chunk-size"]; ok {
		chunkSize, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert chunk-size %q to int", v)
		}
		f.chunkSize = &chunkSize
	}
	if v, ok := flags["dml-batch-size"]; ok {
		dmlBatchSize, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert dml-batch-size %q to int", v)
		}
		f.dmlBatchSize = &dmlBatchSize
	}
	if v, ok := flags["default-retries"]; ok {
		defaultRetries, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert default-retries %q to int", v)
		}
		f.defaultRetries = &defaultRetries
	}
	if v, ok := flags["cut-over-lock-timeout-seconds"]; ok {
		cutoverLockTimeoutSeconds, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert cut-over-lock-timeout-seconds %q to int", v)
		}
		f.cutoverLockTimeoutSeconds = &cutoverLockTimeoutSeconds
	}
	if v, ok := flags["exponential-backoff-max-interval"]; ok {
		exponentialBackoffMaxInterval, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert exponential-backoff-max-interval %q to int", v)
		}
		f.exponentialBackoffMaxInterval = &exponentialBackoffMaxInterval
	}
	if v, ok := flags["max-lag-millis"]; ok {
		maxLagMillis, err := strconv.ParseInt(v, 10, 64)
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
	if v, ok := flags["assume-rbr"]; ok {
		assumeRBR, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert assume-rbr %q to bool", v)
		}
		f.assumeRBR = &assumeRBR
	}
	if v, ok := flags["heartbeat-interval-millis"]; ok {
		heartbeatIntervalMillis, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert heartbeat-interval-millis %q to int", v)
		}
		f.heartbeatIntervalMillis = &heartbeatIntervalMillis
	}
	if v, ok := flags["nice-ratio"]; ok {
		niceRatio, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert nice-ratio %q to float", v)
		}
		f.niceRatio = &niceRatio
	}
	if v, ok := flags["throttle-control-replicas"]; ok {
		f.throttleControlReplicas = &v
	}
	if v, ok := flags["attempt-instant-ddl"]; ok {
		attemptInstantDDL, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert attempt-instant-ddl %q to bool", v)
		}
		f.attemptInstantDDL = &attemptInstantDDL
	}
	if v, ok := flags["assume-master-host"]; ok {
		assumeMasterHost, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert assume-master-host %q to bool", v)
		}
		f.assumeMasterHost = &assumeMasterHost
	}
	return f, nil
}

// NewMigrationContext is the context for gh-ost migration.
func NewMigrationContext(ctx context.Context, taskID int, database *store.DatabaseMessage, dataSource *storepb.DataSource, tableName string, tmpTableNameSuffix string, statement string, noop bool, flags map[string]string, serverIDOffset uint) (*ghostbase.MigrationContext, error) {
	password, err := secretcomp.ReplaceExternalSecret(ctx, dataSource.GetPassword(), dataSource.GetExternalSecret())
	if err != nil {
		return nil, err
	}

	migrationContext := ghostbase.NewMigrationContext()
	migrationContext.Log = newGhostLogger()
	migrationContext.InspectorConnectionConfig.Key.Hostname = dataSource.GetHost()
	port := 3306
	if dataSource.GetPort() != "" {
		dsPort, err := strconv.Atoi(dataSource.GetPort())
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert port from string to int")
		}
		port = dsPort
	}
	if dataSource.GetUseSsl() {
		migrationContext.UseTLS = true
		migrationContext.TLSCACertificate = dataSource.GetSslCa()
		migrationContext.TLSCertificate = dataSource.GetSslCert()
		migrationContext.TLSKey = dataSource.GetSslKey()
		migrationContext.TLSAllowInsecure = true
		if err := migrationContext.SetupTLS(); err != nil {
			return nil, errors.Wrapf(err, "failed to set up tls")
		}
	}
	migrationContext.InspectorConnectionConfig.Key.Port = port
	migrationContext.CliUser = dataSource.GetUsername()
	migrationContext.CliPassword = password
	// GhostDatabaseName is our homemade parameter to allow creating temporary tables under another database.
	// Use MySQL/TiDB backup database name for gh-ost
	migrationContext.GhostDatabaseName = common.BackupDatabaseNameOfEngine(storepb.Engine_MYSQL)
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
	migrationContext.AttemptInstantDDL = defaultConfig.attemptInstantDDL
	migrationContext.AllowedRunningOnMaster = defaultConfig.allowedRunningOnMaster
	migrationContext.ConcurrentCountTableRows = defaultConfig.concurrentCountTableRows
	migrationContext.HooksStatusIntervalSec = defaultConfig.hooksStatusIntervalSec
	migrationContext.CutOverType = ghostbase.CutOverAtomic
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
	migrationContext.ServeSocketFile = ""
	migrationContext.PostponeCutOverFlagFile = ""
	migrationContext.TimestampOldTable = defaultConfig.timestampOldTable
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

	userFlags, err := GetUserFlags(flags)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get user flags")
	}
	if v := userFlags.attemptInstantDDL; v != nil {
		migrationContext.AttemptInstantDDL = *v
	}
	if v := userFlags.maxLoad; v != nil {
		if err := migrationContext.ReadMaxLoad(*v); err != nil {
			return nil, errors.Wrapf(err, "failed to parse max load %q", *v)
		}
	}
	if v := userFlags.chunkSize; v != nil {
		migrationContext.SetChunkSize(*v)
	}
	if v := userFlags.dmlBatchSize; v != nil {
		migrationContext.SetDMLBatchSize(*v)
	}
	if v := userFlags.defaultRetries; v != nil {
		migrationContext.SetDefaultNumRetries(*v)
	}
	if v := userFlags.cutoverLockTimeoutSeconds; v != nil {
		if err := migrationContext.SetCutOverLockTimeoutSeconds(*v); err != nil {
			return nil, errors.Wrapf(err, "failed to set cutover lock timeout %q", *v)
		}
	}
	if v := userFlags.exponentialBackoffMaxInterval; v != nil {
		if err := migrationContext.SetExponentialBackoffMaxInterval(*v); err != nil {
			return nil, errors.Wrapf(err, "failed to set exponential backoff max interval %q", *v)
		}
	}
	if v := userFlags.maxLagMillis; v != nil {
		migrationContext.SetMaxLagMillisecondsThrottleThreshold(*v)
	}
	if v := userFlags.allowOnMaster; v != nil {
		migrationContext.AllowedRunningOnMaster = *v
	}
	if v := userFlags.switchToRBR; v != nil {
		migrationContext.SwitchToRowBinlogFormat = *v
	}
	if v := userFlags.assumeRBR; v != nil {
		migrationContext.AssumeRBR = *v
	}
	if v := userFlags.heartbeatIntervalMillis; v != nil {
		migrationContext.SetHeartbeatIntervalMilliseconds(*v)
	}
	if v := userFlags.niceRatio; v != nil {
		migrationContext.SetNiceRatio(*v)
	}
	if v := userFlags.throttleControlReplicas; v != nil {
		if err := migrationContext.ReadThrottleControlReplicaKeys(*v); err != nil {
			return nil, errors.Wrapf(err, "failed to set throttleControlReplicas")
		}
	}
	if v := userFlags.assumeMasterHost; v != nil && *v {
		migrationContext.AssumeMasterHostname = dataSource.GetHost()
		if dataSource.GetPort() != "" {
			migrationContext.AssumeMasterHostname += ":" + dataSource.GetPort()
		}
	}
	// Uses specified port. GCP, Aliyun, Azure are equivalent here.
	migrationContext.GoogleCloudPlatform = true

	migrationContext.ForceTmpTableName = tableName + tmpTableNameSuffix

	if migrationContext.SwitchToRowBinlogFormat && migrationContext.AssumeRBR {
		return nil, errors.Errorf("switchToRBR and assumeRBR are mutually exclusive")
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
