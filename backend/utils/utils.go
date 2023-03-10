// Package utils is a utility library for server.
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-ost/go/base"
	ghostsql "github.com/github/gh-ost/go/sql"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/store"
)

// GetLatestSchemaVersion gets the latest schema version for a database.
func GetLatestSchemaVersion(ctx context.Context, store *store.Store, driver db.Driver, instanceID int, databaseID int, databaseName string) (string, error) {
	// TODO(d): support semantic versioning.
	limit := 1
	find := &db.MigrationHistoryFind{
		InstanceID: instanceID,
		Database:   &databaseName,
		DatabaseID: &databaseID,
		Limit:      &limit,
	}

	if driver.GetType() == db.Redis || driver.GetType() == db.Oracle {
		history, err := store.FindInstanceChangeHistoryList(ctx, find)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get migration history for database %q", databaseName)
		}
		var schemaVersion string
		if len(history) == 1 {
			schemaVersion = history[0].Version
		}
		return schemaVersion, nil
	}

	history, err := driver.FindMigrationHistoryList(ctx, find)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get migration history for database %q", databaseName)
	}
	var schemaVersion string
	if len(history) == 1 {
		schemaVersion = history[0].Version
	}
	return schemaVersion, nil
}

// DataSourceFromInstanceWithType gets a typed data source from an instance.
func DataSourceFromInstanceWithType(instance *store.InstanceMessage, dataSourceType api.DataSourceType) *store.DataSourceMessage {
	for _, dataSource := range instance.DataSources {
		if dataSource.Type == dataSourceType {
			return dataSource
		}
	}
	return nil
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

// GhostConfig is the configuration for gh-ost migration.
type GhostConfig struct {
	// serverID should be unique
	serverID             uint
	host                 string
	port                 string
	user                 string
	password             string
	database             string
	table                string
	alterStatement       string
	socketFilename       string
	postponeFlagFilename string
	noop                 bool

	// vendor related
	isAWS bool
}

// GetGhostConfig returns a gh-ost configuration for migration.
func GetGhostConfig(taskID int, database *store.DatabaseMessage, dataSource *store.DataSourceMessage, secret string, instanceUsers []*store.InstanceUserMessage, tableName string, statement string, noop bool, serverIDOffset uint) (GhostConfig, error) {
	var isAWS bool
	for _, user := range instanceUsers {
		if user.Name == "'rdsadmin'@'localhost'" && strings.Contains(user.Grant, "SUPER") {
			isAWS = true
			break
		}
	}
	password, err := common.Unobfuscate(dataSource.ObfuscatedPassword, secret)
	if err != nil {
		return GhostConfig{}, err
	}
	return GhostConfig{
		host:                 dataSource.Host,
		port:                 dataSource.Port,
		user:                 dataSource.Username,
		password:             password,
		database:             database.DatabaseName,
		table:                tableName,
		alterStatement:       statement,
		socketFilename:       getSocketFilename(taskID, database.UID, database.DatabaseName, tableName),
		postponeFlagFilename: GetPostponeFlagFilename(taskID, database.UID, database.DatabaseName, tableName),
		noop:                 noop,
		// On the source and each replica, you must set the server_id system variable to establish a unique replication ID. For each server, you should pick a unique positive integer in the range from 1 to 2^32 − 1, and each ID must be different from every other ID in use by any other source or replica in the replication topology. Example: server-id=3.
		// https://dev.mysql.com/doc/refman/5.7/en/replication-options-source.html
		// Here we use serverID = offset + task.ID to avoid potential conflicts.
		serverID: serverIDOffset + uint(taskID),
		// https://github.com/github/gh-ost/blob/master/doc/rds.md
		isAWS: isAWS,
	}, nil
}

func getSocketFilename(taskID int, databaseID int, databaseName string, tableName string) string {
	return fmt.Sprintf("/tmp/gh-ost.%v.%v.%v.%v.sock", taskID, databaseID, databaseName, tableName)
}

// GetPostponeFlagFilename gets the postpone flag filename for gh-ost.
func GetPostponeFlagFilename(taskID int, databaseID int, databaseName string, tableName string) string {
	return fmt.Sprintf("/tmp/gh-ost.%v.%v.%v.%v.postponeFlag", taskID, databaseID, databaseName, tableName)
}

// NewMigrationContext is the context for gh-ost migration.
func NewMigrationContext(config GhostConfig) (*base.MigrationContext, error) {
	const (
		allowedRunningOnMaster              = true
		concurrentCountTableRows            = true
		timestampAllTable                   = true
		hooksStatusIntervalSec              = 60
		heartbeatIntervalMilliseconds       = 100
		niceRatio                           = 0
		chunkSize                           = 1000
		dmlBatchSize                        = 10
		maxLagMillisecondsThrottleThreshold = 1500
		defaultNumRetries                   = 60
		cutoverLockTimeoutSeconds           = 3
		exponentialBackoffMaxInterval       = 64
		throttleHTTPIntervalMillis          = 100
		throttleHTTPTimeoutMillis           = 1000
	)
	statement := strings.Join(strings.Fields(config.alterStatement), " ")
	migrationContext := base.NewMigrationContext()
	migrationContext.InspectorConnectionConfig.Key.Hostname = config.host
	port := 3306
	if config.port != "" {
		configPort, err := strconv.Atoi(config.port)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert port from string to int")
		}
		port = configPort
	}
	migrationContext.InspectorConnectionConfig.Key.Port = port
	migrationContext.CliUser = config.user
	migrationContext.CliPassword = config.password
	migrationContext.DatabaseName = config.database
	migrationContext.OriginalTableName = config.table
	migrationContext.AlterStatement = statement
	migrationContext.Noop = config.noop
	migrationContext.ReplicaServerId = config.serverID
	if config.isAWS {
		migrationContext.AssumeRBR = true
	}
	// set defaults
	migrationContext.AllowedRunningOnMaster = allowedRunningOnMaster
	migrationContext.ConcurrentCountTableRows = concurrentCountTableRows
	migrationContext.HooksStatusIntervalSec = hooksStatusIntervalSec
	migrationContext.CutOverType = base.CutOverAtomic
	migrationContext.ThrottleHTTPIntervalMillis = throttleHTTPIntervalMillis
	migrationContext.ThrottleHTTPTimeoutMillis = throttleHTTPTimeoutMillis

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
	migrationContext.ServeSocketFile = config.socketFilename
	migrationContext.PostponeCutOverFlagFile = config.postponeFlagFilename
	migrationContext.TimestampAllTable = timestampAllTable
	migrationContext.SetHeartbeatIntervalMilliseconds(heartbeatIntervalMilliseconds)
	migrationContext.SetNiceRatio(niceRatio)
	migrationContext.SetChunkSize(chunkSize)
	migrationContext.SetDMLBatchSize(dmlBatchSize)
	migrationContext.SetMaxLagMillisecondsThrottleThreshold(maxLagMillisecondsThrottleThreshold)
	migrationContext.SetDefaultNumRetries(defaultNumRetries)
	migrationContext.ApplyCredentials()
	if err := migrationContext.SetCutOverLockTimeoutSeconds(cutoverLockTimeoutSeconds); err != nil {
		return nil, err
	}
	if err := migrationContext.SetExponentialBackoffMaxInterval(exponentialBackoffMaxInterval); err != nil {
		return nil, err
	}
	return migrationContext, nil
}

// GetActiveStage returns an active stage among all stages.
func GetActiveStage(stages []*store.StageMessage) *store.StageMessage {
	for _, stage := range stages {
		if stage.Active {
			return stage
		}
	}
	return nil
}

// isMatchExpression checks whether a databases matches the query.
// labels is a mapping from database label key to value.
func isMatchExpression(labels map[string]string, expression *api.LabelSelectorRequirement) bool {
	switch expression.Operator {
	case api.InOperatorType:
		value, ok := labels[expression.Key]
		if !ok {
			return false
		}
		for _, exprValue := range expression.Values {
			if exprValue == value {
				return true
			}
		}
		return false
	case api.ExistsOperatorType:
		_, ok := labels[expression.Key]
		return ok
	default:
		return false
	}
}

func isMatchExpressions(labels map[string]string, expressionList []*api.LabelSelectorRequirement) bool {
	// Empty expression list matches no databases.
	if len(expressionList) == 0 {
		return false
	}
	// Expressions are ANDed.
	for _, expression := range expressionList {
		if !isMatchExpression(labels, expression) {
			return false
		}
	}
	return true
}

// GetDatabaseMatrixFromDeploymentSchedule gets a pipeline based on deployment schedule.
// The matrix will include the stage even if the stage has no database.
func GetDatabaseMatrixFromDeploymentSchedule(schedule *api.DeploymentSchedule, databaseList []*store.DatabaseMessage) ([][]*store.DatabaseMessage, error) {
	var matrix [][]*store.DatabaseMessage

	// idToLabels maps databaseID -> label key -> label value
	idToLabels := make(map[int]map[string]string)
	databaseMap := make(map[int]*store.DatabaseMessage)
	for _, database := range databaseList {
		databaseMap[database.UID] = database
		idToLabels[database.UID] = database.Labels
	}

	// idsSeen records database id which is already in a stage.
	idsSeen := make(map[int]bool)

	// For each stage, we loop over all databases to see if it is a match.
	for _, deployment := range schedule.Deployments {
		// For each stage, we will get a list of matched databases.
		var matchedDatabaseList []int
		// Loop over databaseList instead of idToLabels to get determinant results.
		for _, database := range databaseList {
			// Skip if the database is already in a stage.
			if _, ok := idsSeen[database.UID]; ok {
				continue
			}
			// Skip if the database is not found.
			if database.SyncState == api.NotFound {
				continue
			}

			if isMatchExpressions(idToLabels[database.UID], deployment.Spec.Selector.MatchExpressions) {
				matchedDatabaseList = append(matchedDatabaseList, database.UID)
				idsSeen[database.UID] = true
			}
		}

		var databaseList []*store.DatabaseMessage
		for _, id := range matchedDatabaseList {
			databaseList = append(databaseList, databaseMap[id])
		}
		// sort databases in stage based on IDs.
		if len(databaseList) > 0 {
			sort.Slice(databaseList, func(i, j int) bool {
				return databaseList[i].UID < databaseList[j].UID
			})
		}

		matrix = append(matrix, databaseList)
	}

	return matrix, nil
}

// RefreshToken is a token refresher that stores the latest access token configuration to repository.
func RefreshToken(ctx context.Context, store *store.Store, webURL string) common.TokenRefresher {
	return func(token, refreshToken string, expiresTs int64) error {
		_, err := store.PatchRepository(ctx, &api.RepositoryPatch{
			WebURL:       &webURL,
			UpdaterID:    api.SystemBotID,
			AccessToken:  &token,
			ExpiresTs:    &expiresTs,
			RefreshToken: &refreshToken,
		})
		return err
	}
}

// GetTaskStatement gets the statement of a task.
func GetTaskStatement(taskPayload string) (string, error) {
	var taskStatement struct {
		Statement string `json:"statement"`
	}
	if err := json.Unmarshal([]byte(taskPayload), &taskStatement); err != nil {
		return "", err
	}
	return taskStatement.Statement, nil
}

// GetTaskSkippedAndReason gets skipped and skippedReason from a task.
func GetTaskSkippedAndReason(task *api.Task) (bool, string, error) {
	var payload struct {
		Skipped       bool   `json:"skipped,omitempty"`
		SkippedReason string `json:"skippedReason,omitempty"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return false, "", err
	}
	return payload.Skipped, payload.SkippedReason, nil
}

// MergeTaskCreateLists merges a matrix of taskCreate and taskIndexDAG to a list of taskCreate and taskIndexDAG.
// The index of returned taskIndexDAG list is set regarding the merged taskCreate.
func MergeTaskCreateLists(taskCreateLists [][]api.TaskCreate, taskIndexDAGLists [][]api.TaskIndexDAG) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	if len(taskCreateLists) != len(taskIndexDAGLists) {
		return nil, nil, errors.Errorf("expect taskCreateLists and taskIndexDAGLists to have the same length, get %d, %d respectively", len(taskCreateLists), len(taskIndexDAGLists))
	}
	var resTaskCreateList []api.TaskCreate
	var resTaskIndexDAGList []api.TaskIndexDAG
	offset := 0
	for i := range taskCreateLists {
		taskCreateList := taskCreateLists[i]
		taskIndexDAGList := taskIndexDAGLists[i]

		resTaskCreateList = append(resTaskCreateList, taskCreateList...)
		for _, dag := range taskIndexDAGList {
			resTaskIndexDAGList = append(resTaskIndexDAGList, api.TaskIndexDAG{
				FromIndex: dag.FromIndex + offset,
				ToIndex:   dag.ToIndex + offset,
			})
		}
		offset += len(taskCreateList)
	}
	return resTaskCreateList, resTaskIndexDAGList, nil
}

// PassAllCheck checks whether a task has passed all task checks.
func PassAllCheck(task *store.TaskMessage, allowedStatus api.TaskCheckStatus, taskCheckRuns []*store.TaskCheckRunMessage, engine db.Type) (bool, error) {
	var runs []*store.TaskCheckRunMessage
	for _, run := range taskCheckRuns {
		if run.TaskID == task.ID {
			runs = append(runs, run)
		}
	}
	// schema update, data update and gh-ost sync task have required task check.
	if task.Type == api.TaskDatabaseSchemaUpdate || task.Type == api.TaskDatabaseSchemaUpdateSDL || task.Type == api.TaskDatabaseDataUpdate || task.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		pass, err := passCheck(runs, api.TaskCheckDatabaseConnect, allowedStatus)
		if err != nil {
			return false, err
		}
		if !pass {
			return false, nil
		}

		pass, err = passCheck(runs, api.TaskCheckInstanceMigrationSchema, allowedStatus)
		if err != nil {
			return false, err
		}
		if !pass {
			return false, nil
		}

		if api.IsSyntaxCheckSupported(engine) {
			ok, err := passCheck(runs, api.TaskCheckDatabaseStatementSyntax, allowedStatus)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}

		if api.IsSQLReviewSupported(engine) {
			ok, err := passCheck(runs, api.TaskCheckDatabaseStatementAdvise, allowedStatus)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}

		if engine == db.Postgres {
			ok, err := passCheck(runs, api.TaskCheckDatabaseStatementType, allowedStatus)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
	}

	if task.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		ok, err := passCheck(runs, api.TaskCheckGhostSync, allowedStatus)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

// Returns true only if the task check run result is at least the minimum required level.
// For PendingApproval->Pending transitions, the minimum level is SUCCESS.
// For Pending->Running transitions, the minimum level is WARN.
func passCheck(taskCheckRunList []*store.TaskCheckRunMessage, checkType api.TaskCheckType, allowedStatus api.TaskCheckStatus) (bool, error) {
	var lastRun *store.TaskCheckRunMessage
	for _, run := range taskCheckRunList {
		if checkType != run.Type {
			continue
		}
		if lastRun == nil || lastRun.ID < run.ID {
			lastRun = run
		}
	}

	if lastRun == nil || lastRun.Status != api.TaskCheckRunDone {
		return false, nil
	}
	checkResult := &api.TaskCheckRunResultPayload{}
	if err := json.Unmarshal([]byte(lastRun.Result), checkResult); err != nil {
		return false, err
	}
	for _, result := range checkResult.ResultList {
		if result.Status.LessThan(allowedStatus) {
			return false, nil
		}
	}

	return true, nil
}

// ExecuteMigration executes migration.
func ExecuteMigration(ctx context.Context, store *store.Store, driver db.Driver, m *db.MigrationInfo, statement string) (migrationHistoryID string, updatedSchema string, resErr error) {
	var prevSchemaBuf bytes.Buffer
	// Don't record schema if the database hasn't existed yet or is schemaless (e.g. Mongo).
	if !m.CreateDatabase {
		// For baseline migration, we also record the live schema to detect the schema drift.
		// See https://bytebase.com/blog/what-is-database-schema-drift
		if _, err := driver.Dump(ctx, m.Database, &prevSchemaBuf, true /*schemaOnly*/); err != nil {
			return "", "", err
		}
	}

	insertedID, err := beginMigration(ctx, store, m, prevSchemaBuf.String(), statement)
	if err != nil {
		if common.ErrorCode(err) == common.MigrationAlreadyApplied {
			return insertedID, prevSchemaBuf.String(), nil
		}
		return "", "", errors.Wrapf(err, "failed to begin migration for issue %s", m.IssueID)
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := endMigration(ctx, store, startedNs, insertedID, updatedSchema, db.BytebaseDatabase, resErr == nil /*isDone*/); err != nil {
			log.Error("Failed to update migration history record",
				zap.Error(err),
				zap.String("migration_id", migrationHistoryID),
			)
		}
	}()

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could has non-empty sql but will not execute.
	// https://github.com/bytebase/bytebase/issues/394
	doMigrate := true
	if statement == "" {
		doMigrate = false
	}
	if m.Type == db.Baseline {
		doMigrate = false
	}
	if doMigrate {
		// TODO(p0ny): migrate to instance change history
		if _, _, err := driver.ExecuteMigration(ctx, m, statement); err != nil {
			return "", "", err
		}
	}

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if _, err := driver.Dump(ctx, m.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
		// We will ignore the dump error if the database is dropped.
		if strings.Contains(err.Error(), "not found") {
			return insertedID, "", nil
		}
		return "", "", err
	}

	return insertedID, afterSchemaBuf.String(), nil
}

// beginMigration checks before executing migration and inserts a migration history record with pending status.
func beginMigration(ctx context.Context, store *store.Store, m *db.MigrationInfo, prevSchema string, statement string) (string, error) {
	// Convert version to stored version.
	storedVersion, err := util.ToStoredVersion(m.UseSemanticVersion, m.Version, m.SemanticVersionSuffix)
	if err != nil {
		return "", errors.Wrap(err, "failed to convert to stored version")
	}
	// Phase 1 - Pre-check before executing migration
	// Check if the same migration version has already been applied.
	if list, err := store.FindInstanceChangeHistoryList(ctx, &db.MigrationHistoryFind{
		InstanceID: m.InstanceID,
		DatabaseID: m.DatabaseID,
		Version:    &m.Version,
	}); err != nil {
		return "", errors.Wrap(err, "failed to check duplicate version")
	} else if len(list) > 0 {
		migrationHistory := list[0]
		switch migrationHistory.Status {
		case db.Done:
			if migrationHistory.IssueID != m.IssueID {
				return migrationHistory.ID, common.Errorf(common.MigrationFailed, "database %q has already applied version %s by issue %s", m.Database, m.Version, migrationHistory.IssueID)
			}
			return migrationHistory.ID, common.Errorf(common.MigrationAlreadyApplied, "database %q has already applied version %s", m.Database, m.Version)
		case db.Pending:
			err := errors.Errorf("database %q version %s migration is already in progress", m.Database, m.Version)
			log.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return migrationHistory.ID, nil
			}
			return "", common.Wrap(err, common.MigrationPending)
		case db.Failed:
			err := errors.Errorf("database %q version %s migration has failed, please check your database to make sure things are fine and then start a new migration using a new version ", m.Database, m.Version)
			log.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return migrationHistory.ID, nil
			}
			return "", common.Wrap(err, common.MigrationFailed)
		}
	}

	largestSequence, err := store.GetLargestInstanceChangeHistorySequence(ctx, m.InstanceID, m.DatabaseID, false /* baseline */)
	if err != nil {
		return "", err
	}

	// Check if there is any higher version already been applied since the last baseline or branch.
	if version, err := store.GetLargestInstanceChangeHistoryVersionSinceBaseline(ctx, m.InstanceID, m.DatabaseID); err != nil {
		return "", err
	} else if version != nil && len(*version) > 0 && *version >= m.Version {
		return "", common.Errorf(common.MigrationOutOfOrder, "database %q has already applied version %s which >= %s", m.Database, *version, m.Version)
	}

	// Phase 2 - Record migration history as PENDING.
	// MySQL runs DDL in its own transaction, so we can't commit migration history together with DDL in a single transaction.
	// Thus we sort of doing a 2-phase commit, where we first write a PENDING migration record, and after migration completes, we then
	// update the record to DONE together with the updated schema.
	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	insertedID, err := store.CreatePendingInstanceChangeHistory(ctx, largestSequence+1, prevSchema, m, storedVersion, statementRecord)
	if err != nil {
		return "", err
	}

	return insertedID, nil
}

func endMigration(ctx context.Context, store *store.Store, startedNs int64, insertedID string, updatedSchema string, _ string, isDone bool) error {
	var err error
	migrationDurationNs := time.Now().UnixNano() - startedNs

	if isDone {
		err = store.UpdateInstanceChangeHistoryAsDone(ctx, migrationDurationNs, updatedSchema, insertedID)
		// Upon success, update the migration history as 'DONE', execution_duration_ns, updated schema.
	} else {
		// Otherwise, update the migration history as 'FAILED', execution_duration.
		err = store.UpdateInstanceChangeHistoryAsFailed(ctx, migrationDurationNs, insertedID)
	}
	return err
}
