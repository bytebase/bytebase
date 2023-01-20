// Package utils is a utility library for server.
package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-ost/go/base"
	ghostsql "github.com/github/gh-ost/go/sql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

// GetLatestSchemaVersion gets the latest schema version for a database.
func GetLatestSchemaVersion(ctx context.Context, driver db.Driver, databaseName string) (string, error) {
	// TODO(d): support semantic versioning.
	limit := 1
	history, err := driver.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{
		Database: &databaseName,
		Limit:    &limit,
	})
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
func GetGhostConfig(task *api.Task, dataSource *store.DataSourceMessage, instanceUsers []*store.InstanceUserMessage, tableName string, statement string, noop bool, serverIDOffset uint) GhostConfig {
	var isAWS bool
	for _, user := range instanceUsers {
		if user.Name == "'rdsadmin'@'localhost'" && strings.Contains(user.Grant, "SUPER") {
			isAWS = true
			break
		}
	}
	return GhostConfig{
		host:                 task.Instance.Host,
		port:                 task.Instance.Port,
		user:                 dataSource.Username,
		password:             dataSource.Password,
		database:             task.Database.Name,
		table:                tableName,
		alterStatement:       statement,
		socketFilename:       getSocketFilename(task.ID, task.Database.ID, task.Database.Name, tableName),
		postponeFlagFilename: GetPostponeFlagFilename(task.ID, task.Database.ID, task.Database.Name, tableName),
		noop:                 noop,
		// On the source and each replica, you must set the server_id system variable to establish a unique replication ID. For each server, you should pick a unique positive integer in the range from 1 to 2^32 − 1, and each ID must be different from every other ID in use by any other source or replica in the replication topology. Example: server-id=3.
		// https://dev.mysql.com/doc/refman/5.7/en/replication-options-source.html
		// Here we use serverID = offset + task.ID to avoid potential conflicts.
		serverID: serverIDOffset + uint(task.ID),
		// https://github.com/github/gh-ost/blob/master/doc/rds.md
		isAWS: isAWS,
	}
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
func GetActiveStage(pipeline *api.Pipeline) *api.Stage {
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status != api.TaskDone {
				return stage
			}
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
		if _, ok := idToLabels[database.UID]; !ok {
			idToLabels[database.UID] = make(map[string]string)
		}
		for key, value := range database.Labels {
			idToLabels[database.UID][key] = value
		}
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

			labels := idToLabels[database.UID]
			if isMatchExpressions(labels, deployment.Spec.Selector.MatchExpressions) {
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
func GetTaskStatement(task *api.Task) (string, error) {
	var taskStatement struct {
		Statement string `json:"statement"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &taskStatement); err != nil {
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
