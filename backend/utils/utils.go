// Package utils is a utility library for server.
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-ost/go/base"
	ghostsql "github.com/github/gh-ost/go/sql"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/app/relay"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// GetLatestSchemaVersion gets the latest schema version for a database.
func GetLatestSchemaVersion(ctx context.Context, stores *store.Store, instanceID int, databaseID int, databaseName string) (model.Version, error) {
	// TODO(d): support semantic versioning.
	limit := 1
	history, err := stores.ListInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: &instanceID,
		DatabaseID: &databaseID,
		Limit:      &limit,
	})
	if err != nil {
		return model.Version{}, errors.Wrapf(err, "failed to get migration history for database %q", databaseName)
	}
	if len(history) == 0 {
		return model.Version{}, nil
	}
	return history[0].Version, nil
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
		cutoverLockTimeoutSeconds           = 60
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
	if err := migrationContext.SetConnectionConfig(""); err != nil {
		return nil, err
	}
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
		newMap := make(map[string]string)
		for k, v := range database.Metadata.Labels {
			newMap[k] = v
		}
		newMap[api.EnvironmentLabelKey] = database.EffectiveEnvironmentID

		idToLabels[database.UID] = newMap
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
func RefreshToken(ctx context.Context, s *store.Store, webURL string) common.TokenRefresher {
	return func(token, refreshToken string, expiresTs int64) error {
		_, err := s.PatchRepositoryV2(ctx, &store.PatchRepositoryMessage{
			WebURL:       &webURL,
			AccessToken:  &token,
			ExpiresTs:    &expiresTs,
			RefreshToken: &refreshToken,
		}, api.SystemBotID)
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

// GetTaskSheetID gets the sheetID of a task.
func GetTaskSheetID(taskPayload string) (int, error) {
	var taskSheetID struct {
		SheetID int `json:"sheetId"`
	}
	if err := json.Unmarshal([]byte(taskPayload), &taskSheetID); err != nil {
		return 0, err
	}
	return taskSheetID.SheetID, nil
}

// GetTaskSkipped gets skipped from a task.
func GetTaskSkipped(task *store.TaskMessage) (bool, error) {
	var payload struct {
		Skipped bool `json:"skipped,omitempty"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return false, err
	}
	return payload.Skipped, nil
}

// MergeTaskCreateLists merges a matrix of taskCreate and taskIndexDAG to a list of taskCreate and taskIndexDAG.
// The index of returned taskIndexDAG list is set regarding the merged taskCreate.
func MergeTaskCreateLists(taskCreateLists [][]*store.TaskMessage, taskIndexDAGLists [][]store.TaskIndexDAG) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	if len(taskCreateLists) != len(taskIndexDAGLists) {
		return nil, nil, errors.Errorf("expect taskCreateLists and taskIndexDAGLists to have the same length, get %d, %d respectively", len(taskCreateLists), len(taskIndexDAGLists))
	}
	var resTaskCreateList []*store.TaskMessage
	var resTaskIndexDAGList []store.TaskIndexDAG
	offset := 0
	for i := range taskCreateLists {
		taskCreateList := taskCreateLists[i]
		taskIndexDAGList := taskIndexDAGLists[i]

		resTaskCreateList = append(resTaskCreateList, taskCreateList...)
		for _, dag := range taskIndexDAGList {
			resTaskIndexDAGList = append(resTaskIndexDAGList, store.TaskIndexDAG{
				FromIndex: dag.FromIndex + offset,
				ToIndex:   dag.ToIndex + offset,
			})
		}
		offset += len(taskCreateList)
	}
	return resTaskCreateList, resTaskIndexDAGList, nil
}

// ExecuteMigrationDefault executes migration.
func ExecuteMigrationDefault(ctx context.Context, driverCtx context.Context, store *store.Store, stateCfg *state.State, taskRunUID int, driver db.Driver, mi *db.MigrationInfo, statement string, sheetID *int, opts db.ExecuteOptions) (migrationHistoryID string, updatedSchema string, resErr error) {
	execFunc := func(ctx context.Context, execStatement string) error {
		if _, err := driver.Execute(ctx, execStatement, false /* createDatabase */, opts); err != nil {
			return err
		}
		return nil
	}
	return ExecuteMigrationWithFunc(ctx, driverCtx, store, stateCfg, taskRunUID, driver, mi, statement, sheetID, execFunc)
}

// ExecuteMigrationWithFunc executes the migration with custom migration function.
func ExecuteMigrationWithFunc(ctx context.Context, driverCtx context.Context, s *store.Store, stateCfg *state.State, taskRunUID int, driver db.Driver, m *db.MigrationInfo, statement string, sheetID *int, execFunc func(ctx context.Context, execStatement string) error) (migrationHistoryID string, updatedSchema string, resErr error) {
	var prevSchemaBuf bytes.Buffer
	// Don't record schema if the database hasn't existed yet or is schemaless, e.g. MongoDB.
	// For baseline migration, we also record the live schema to detect the schema drift.
	// See https://bytebase.com/blog/what-is-database-schema-drift
	if _, err := driver.Dump(ctx, &prevSchemaBuf, true /* schemaOnly */); err != nil {
		return "", "", err
	}

	insertedID, err := BeginMigration(ctx, s, m, prevSchemaBuf.String(), statement, sheetID)
	if err != nil {
		if common.ErrorCode(err) == common.MigrationAlreadyApplied {
			return insertedID, prevSchemaBuf.String(), nil
		}
		msg := "failed to begin migration"
		if m.IssueUID != nil {
			msg += fmt.Sprintf(" for issue %d", *m.IssueUID)
		}
		return "", "", errors.Wrapf(err, msg)
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := EndMigration(ctx, s, startedNs, insertedID, updatedSchema, resErr == nil /* isDone */); err != nil {
			slog.Error("Failed to update migration history record",
				log.BBError(err),
				slog.String("migration_id", migrationHistoryID),
			)
		}
	}()

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could has non-empty sql but will not execute.
	// https://github.com/bytebase/bytebase/issues/394
	doMigrate := true
	if statement == "" || m.Type == db.Baseline {
		doMigrate = false
	}
	if doMigrate {
		var renderedStatement = statement
		// The m.DatabaseID is nil means the migration is a instance level migration
		if m.DatabaseID != nil {
			database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				UID: m.DatabaseID,
			})
			if err != nil {
				return "", "", err
			}
			if database == nil {
				return "", "", errors.Errorf("database %d not found", *m.DatabaseID)
			}
			materials := GetSecretMapFromDatabaseMessage(database)
			// To avoid leak the rendered statement, the error message should use the original statement and not the rendered statement.
			renderedStatement = RenderStatement(statement, materials)
		}

		if stateCfg != nil {
			stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
				state.TaskRunExecutionStatus{
					ExecutionStatus: v1pb.TaskRun_EXECUTING,
					UpdateTime:      time.Now(),
				})
		}

		if err := execFunc(driverCtx, renderedStatement); err != nil {
			return "", "", err
		}
	}

	if stateCfg != nil {
		stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
			state.TaskRunExecutionStatus{
				ExecutionStatus: v1pb.TaskRun_POST_EXECUTING,
				UpdateTime:      time.Now(),
			})
	}

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if _, err := driver.Dump(ctx, &afterSchemaBuf, true /* schemaOnly */); err != nil {
		// We will ignore the dump error if the database is dropped.
		if strings.Contains(err.Error(), "not found") {
			return insertedID, "", nil
		}
		return "", "", err
	}

	return insertedID, afterSchemaBuf.String(), nil
}

// BeginMigration checks before executing migration and inserts a migration history record with pending status.
func BeginMigration(ctx context.Context, stores *store.Store, m *db.MigrationInfo, prevSchema, statement string, sheetID *int) (string, error) {
	// Phase 1 - Pre-check before executing migration
	// Check if the same migration version has already been applied.
	if list, err := stores.ListInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: m.InstanceID,
		DatabaseID: m.DatabaseID,
		Version:    &m.Version,
	}); err != nil {
		return "", errors.Wrap(err, "failed to check duplicate version")
	} else if len(list) > 0 {
		migrationHistory := list[0]
		switch migrationHistory.Status {
		case db.Done:
			return migrationHistory.UID, common.Errorf(common.MigrationAlreadyApplied, "database %q has already applied version %s", m.Database, m.Version.Version)
		case db.Pending:
			err := errors.Errorf("database %q version %s migration is already in progress", m.Database, m.Version.Version)
			slog.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return migrationHistory.UID, nil
			}
			return "", common.Wrap(err, common.MigrationPending)
		case db.Failed:
			err := errors.Errorf("database %q version %s migration has failed, please check your database to make sure things are fine and then start a new migration using a new version ", m.Database, m.Version.Version)
			slog.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return migrationHistory.UID, nil
			}
			return "", common.Wrap(err, common.MigrationFailed)
		}
	}

	// Phase 2 - Record migration history as PENDING.
	// MySQL runs DDL in its own transaction, so we can't commit migration history together with DDL in a single transaction.
	// Thus we sort of doing a 2-phase commit, where we first write a PENDING migration record, and after migration completes, we then
	// update the record to DONE together with the updated schema.
	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	insertedID, err := stores.CreatePendingInstanceChangeHistory(ctx, prevSchema, m, statementRecord, sheetID)
	if err != nil {
		return "", err
	}

	return insertedID, nil
}

// EndMigration updates the migration history record to DONE or FAILED depending on migration is done or not.
func EndMigration(ctx context.Context, storeInstance *store.Store, startedNs int64, insertedID string, updatedSchema string, isDone bool) error {
	migrationDurationNs := time.Now().UnixNano() - startedNs
	update := &store.UpdateInstanceChangeHistoryMessage{
		ID:                  insertedID,
		ExecutionDurationNs: &migrationDurationNs,
	}
	if isDone {
		// Upon success, update the migration history as 'DONE', execution_duration_ns, updated schema.
		status := db.Done
		update.Status = &status
		update.Schema = &updatedSchema
	} else {
		// Otherwise, update the migration history as 'FAILED', execution_duration.
		status := db.Failed
		update.Status = &status
	}
	return storeInstance.UpdateInstanceChangeHistory(ctx, update)
}

// FindNextPendingStep finds the next pending step in the approval flow.
func FindNextPendingStep(template *storepb.ApprovalTemplate, approvers []*storepb.IssuePayloadApproval_Approver) *storepb.ApprovalStep {
	// We can do the finding like this for now because we are presuming that
	// one step is approved by one approver.
	// and the approver status is either
	// APPROVED or REJECTED.
	if len(approvers) >= len(template.Flow.Steps) {
		return nil
	}
	return template.Flow.Steps[len(approvers)]
}

// FindRejectedStep finds the rejected step in the approval flow.
func FindRejectedStep(template *storepb.ApprovalTemplate, approvers []*storepb.IssuePayloadApproval_Approver) *storepb.ApprovalStep {
	for i, approver := range approvers {
		if i >= len(template.Flow.Steps) {
			return nil
		}
		if approver.Status == storepb.IssuePayloadApproval_Approver_REJECTED {
			return template.Flow.Steps[i]
		}
	}
	return nil
}

// CheckApprovalApproved checks if the approval is approved.
func CheckApprovalApproved(approval *storepb.IssuePayloadApproval) (bool, error) {
	if approval == nil || !approval.ApprovalFindingDone {
		return false, nil
	}
	if approval.ApprovalFindingError != "" {
		return false, nil
	}
	if len(approval.ApprovalTemplates) == 0 {
		return true, nil
	}
	if len(approval.ApprovalTemplates) != 1 {
		return false, errors.Errorf("expecting one approval template but got %d", len(approval.ApprovalTemplates))
	}
	return FindRejectedStep(approval.ApprovalTemplates[0], approval.Approvers) == nil && FindNextPendingStep(approval.ApprovalTemplates[0], approval.Approvers) == nil, nil
}

// CheckIssueApproved checks if the issue is approved.
func CheckIssueApproved(issue *store.IssueMessage) (bool, error) {
	return CheckApprovalApproved(issue.Payload.Approval)
}

// HandleIncomingApprovalSteps handles incoming approval steps.
// - Blocks approval steps if no user can approve the step.
// - creates external approvals for external approval nodes.
func HandleIncomingApprovalSteps(ctx context.Context, s *store.Store, relayClient *relay.Client, issue *store.IssueMessage, approval *storepb.IssuePayloadApproval) ([]*storepb.IssuePayloadApproval_Approver, []*store.ActivityMessage, error) {
	if len(approval.ApprovalTemplates) == 0 {
		return nil, nil, nil
	}

	getActivityCreate := func(status storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_Status, comment string) (*store.ActivityMessage, error) {
		activityPayload, err := protojson.Marshal(&storepb.ActivityIssueCommentCreatePayload{
			Event: &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_{
				ApprovalEvent: &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent{
					Status: status,
				},
			},
			IssueName: issue.Title,
		})
		if err != nil {
			return nil, err
		}
		return &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.UID,
			Type:         api.ActivityIssueCommentCreate,
			Level:        api.ActivityInfo,
			Comment:      comment,
			Payload:      string(activityPayload),
		}, nil
	}

	var approvers []*storepb.IssuePayloadApproval_Approver
	var activities []*store.ActivityMessage

	step := FindNextPendingStep(approval.ApprovalTemplates[0], approval.Approvers)
	if step == nil {
		return nil, nil, nil
	}
	if len(step.Nodes) != 1 {
		return nil, nil, errors.Errorf("expecting one node but got %v", len(step.Nodes))
	}
	if step.Type != storepb.ApprovalStep_ANY {
		return nil, nil, errors.Errorf("expecting ANY step type but got %v", step.Type)
	}
	node := step.Nodes[0]
	if v, ok := node.GetPayload().(*storepb.ApprovalNode_ExternalNodeId); ok {
		if err := handleApprovalNodeExternalNode(ctx, s, relayClient, issue, v.ExternalNodeId); err != nil {
			approvers = append(approvers, &storepb.IssuePayloadApproval_Approver{
				Status:      storepb.IssuePayloadApproval_Approver_REJECTED,
				PrincipalId: api.SystemBotID,
			})
			activity, err := getActivityCreate(storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_REJECTED, fmt.Sprintf("failed to handle external node, err: %v", err))
			if err != nil {
				return nil, nil, err
			}
			activities = append(activities, activity)
		}
	}
	return approvers, activities, nil
}

func handleApprovalNodeExternalNode(ctx context.Context, s *store.Store, relayClient *relay.Client, issue *store.IssueMessage, externalNodeID string) error {
	getExternalApprovalByID := func(ctx context.Context, s *store.Store, externalApprovalID string) (*storepb.ExternalApprovalSetting_Node, error) {
		setting, err := s.GetWorkspaceExternalApprovalSetting(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get workspace external approval setting")
		}
		for _, node := range setting.Nodes {
			if node.Id == externalApprovalID {
				return node, nil
			}
		}
		return nil, nil
	}
	node, err := getExternalApprovalByID(ctx, s, externalNodeID)
	if err != nil {
		return errors.Wrapf(err, "failed to get external approval node %s", externalNodeID)
	}
	if node == nil {
		return errors.Errorf("external approval node %s not found", externalNodeID)
	}
	id, err := relayClient.Create(node.Endpoint, &relay.CreatePayload{
		IssueID:     fmt.Sprintf("%d", issue.UID),
		Title:       issue.Title,
		Description: issue.Description,
		Project:     issue.Project.ResourceID,
		CreateTime:  issue.CreatedTime,
		Creator:     issue.Creator.Email,
		Assignee:    issue.Assignee.Email,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create external approval")
	}
	payload, err := json.Marshal(&api.ExternalApprovalPayloadRelay{
		ExternalApprovalNodeID: node.Id,
		ID:                     id,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal external approval payload")
	}
	if _, err := s.CreateExternalApprovalV2(ctx, &store.ExternalApprovalMessage{
		IssueUID:     issue.UID,
		ApproverUID:  api.SystemBotID,
		Type:         api.ExternalApprovalTypeRelay,
		Payload:      string(payload),
		RequesterUID: api.SystemBotID,
	}); err != nil {
		return errors.Wrapf(err, "failed to create external approval")
	}
	return nil
}

// UpdateProjectPolicyFromGrantIssue updates the project policy from grant issue.
func UpdateProjectPolicyFromGrantIssue(ctx context.Context, stores *store.Store, issue *store.IssueMessage, grantRequest *storepb.GrantRequest) error {
	policy, err := stores.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &issue.Project.ResourceID})
	if err != nil {
		return err
	}
	var newConditionExpr string
	if grantRequest.Condition != nil {
		newConditionExpr = grantRequest.Condition.Expression
	}
	updated := false

	userID, err := strconv.Atoi(strings.TrimPrefix(grantRequest.User, "users/"))
	if err != nil {
		return err
	}
	newUser, err := stores.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if newUser == nil {
		return status.Errorf(codes.Internal, "user %v not found", userID)
	}
	for _, binding := range policy.Bindings {
		if binding.Role != api.Role(grantRequest.Role) {
			continue
		}
		var oldConditionExpr string
		if binding.Condition != nil {
			oldConditionExpr = binding.Condition.Expression
		}
		if oldConditionExpr != newConditionExpr {
			continue
		}
		// Append
		binding.Members = append(binding.Members, newUser)
		updated = true
		break
	}
	roleID := api.Role(strings.TrimPrefix(grantRequest.Role, "roles/"))
	if !updated {
		condition := grantRequest.Condition
		condition.Description = fmt.Sprintf("#%d", issue.UID)
		policy.Bindings = append(policy.Bindings, &store.PolicyBinding{
			Role:      roleID,
			Members:   []*store.UserMessage{newUser},
			Condition: condition,
		})
	}
	if _, err := stores.SetProjectIAMPolicy(ctx, policy, api.SystemBotID, issue.Project.UID); err != nil {
		return err
	}
	return nil
}

// RenderStatement renders the given template statement with the given key-value map.
func RenderStatement(templateStatement string, secrets map[string]string) string {
	// Happy path for empty template statement.
	if templateStatement == "" {
		return ""
	}
	// Optimizations for databases without secrets.
	if len(secrets) == 0 {
		return templateStatement
	}
	// Don't render statement larger than 1MB.
	if len(templateStatement) > 1024*1024 {
		return templateStatement
	}

	// The regular expression consists of:
	// \${{: matches the string ${{, where $ is escaped with a backslash.
	// \s*: matches zero or more whitespace characters.
	// secrets\.: matches the string secrets., where . is escaped with a backslash.
	// (?P<name>[A-Z0-9_]+): uses a named capture group name to match the secret name. The capture group is defined using the syntax (?P<name>) and matches one or more uppercase letters, digits, or underscores.
	re := regexp.MustCompile(`\${{\s*secrets\.(?P<name>[A-Z0-9_]+)\s*}}`)
	matches := re.FindAllStringSubmatch(templateStatement, -1)
	for _, match := range matches {
		name := match[1]
		if value, ok := secrets[name]; ok {
			templateStatement = strings.ReplaceAll(templateStatement, match[0], value)
		}
	}
	return templateStatement
}

// GetSecretMapFromDatabaseMessage extracts the secret map from the given database message.
func GetSecretMapFromDatabaseMessage(databaseMessage *store.DatabaseMessage) map[string]string {
	materials := make(map[string]string)
	if databaseMessage.Secrets == nil || len(databaseMessage.Secrets.Items) == 0 {
		return materials
	}

	for _, item := range databaseMessage.Secrets.Items {
		materials[item.Name] = item.Value
	}
	return materials
}

func convertVcsPushEventType(vcsType vcs.Type) storepb.VcsType {
	switch vcsType {
	case "GITLAB":
		return storepb.VcsType_GITLAB
	case "GITHUB":
		return storepb.VcsType_GITHUB
	case "BITBUCKET":
		return storepb.VcsType_BITBUCKET
	default:
		return storepb.VcsType_VCS_TYPE_UNSPECIFIED
	}
}

func convertVcsPushEventCommits(commits []vcs.Commit) []*storepb.Commit {
	var result []*storepb.Commit
	for i := range commits {
		commit := &commits[i]
		result = append(result, &storepb.Commit{
			Id:           commit.ID,
			Title:        commit.Title,
			Message:      commit.Message,
			CreatedTs:    commit.CreatedTs,
			Url:          commit.URL,
			AuthorName:   commit.AuthorName,
			AuthorEmail:  commit.AuthorEmail,
			AddedList:    commit.AddedList,
			ModifiedList: commit.ModifiedList,
		})
	}
	return result
}

func convertVcsPushEventFileCommit(c *vcs.FileCommit) *storepb.FileCommit {
	return &storepb.FileCommit{
		Id:          c.ID,
		Title:       c.Title,
		Message:     c.Message,
		CreatedTs:   c.CreatedTs,
		Url:         c.URL,
		AuthorName:  c.AuthorName,
		AuthorEmail: c.AuthorEmail,
		Added:       c.Added,
	}
}

// ConvertVcsPushEvent converts a vcs.pushEvent to a storepb.PushEvent.
func ConvertVcsPushEvent(pushEvent *vcs.PushEvent) *storepb.PushEvent {
	return &storepb.PushEvent{
		VcsType:            convertVcsPushEventType(pushEvent.VCSType),
		BaseDir:            pushEvent.BaseDirectory,
		Ref:                pushEvent.Ref,
		Before:             pushEvent.Before,
		After:              pushEvent.After,
		RepositoryId:       pushEvent.RepositoryID,
		RepositoryUrl:      pushEvent.RepositoryURL,
		RepositoryFullPath: pushEvent.RepositoryFullPath,
		AuthorName:         pushEvent.AuthorName,
		Commits:            convertVcsPushEventCommits(pushEvent.CommitList),
		FileCommit:         convertVcsPushEventFileCommit(&pushEvent.FileCommit),
	}
}

// GetMatchedAndUnmatchedDatabasesInDatabaseGroup returns the matched and unmatched databases in the given database group.
func GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx context.Context, databaseGroup *store.DatabaseGroupMessage, allDatabases []*store.DatabaseMessage) ([]*store.DatabaseMessage, []*store.DatabaseMessage, error) {
	prog, err := common.ValidateGroupCELExpr(databaseGroup.Expression.Expression)
	if err != nil {
		return nil, nil, err
	}
	var matches []*store.DatabaseMessage
	var unmatches []*store.DatabaseMessage

	// DONOT check bb.feature.database-grouping for instance. The API here is read-only in the frontend, we need to show if the instance is matched but missing required license.
	// The feature guard will works during issue creation.
	for _, database := range allDatabases {
		res, _, err := prog.ContextEval(ctx, map[string]any{
			"resource": map[string]any{
				"database_name":    database.DatabaseName,
				"environment_name": fmt.Sprintf("%s%s", common.EnvironmentNamePrefix, database.EffectiveEnvironmentID),
				"instance_id":      database.InstanceID,
			},
		})
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, err.Error())
		}

		val, err := res.ConvertToNative(reflect.TypeOf(false))
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "expect bool result")
		}
		if boolVal, ok := val.(bool); ok && boolVal {
			matches = append(matches, database)
		} else {
			unmatches = append(unmatches, database)
		}
	}
	return matches, unmatches, nil
}

// GetMatchedAndUnmatchedTablesInSchemaGroup returns the matched and unmatched tables in the given schema group.
func GetMatchedAndUnmatchedTablesInSchemaGroup(ctx context.Context, dbSchema *store.DBSchema, schemaGroup *store.SchemaGroupMessage) ([]string, []string, error) {
	prog, err := common.ValidateGroupCELExpr(schemaGroup.Expression.Expression)
	if err != nil {
		return nil, nil, err
	}

	var matched []string
	var unmatched []string

	for _, schema := range dbSchema.Metadata.Schemas {
		for _, table := range schema.Tables {
			res, _, err := prog.ContextEval(ctx, map[string]any{
				"resource": map[string]any{
					"table_name": table.Name,
				},
			})
			if err != nil {
				return nil, nil, status.Errorf(codes.Internal, err.Error())
			}

			val, err := res.ConvertToNative(reflect.TypeOf(false))
			if err != nil {
				return nil, nil, status.Errorf(codes.Internal, "expect bool result")
			}

			if boolVal, ok := val.(bool); ok && boolVal {
				matched = append(matched, table.Name)
			} else {
				unmatched = append(unmatched, table.Name)
			}
		}
	}
	return matched, unmatched, nil
}

// ChangeIssueStatus changes the status of an issue.
func ChangeIssueStatus(ctx context.Context, stores *store.Store, activityManager *activity.Manager, issue *store.IssueMessage, newStatus api.IssueStatus, updaterID int, comment string) error {
	updateIssueMessage := &store.UpdateIssueMessage{Status: &newStatus}
	updatedIssue, err := stores.UpdateIssueV2(ctx, issue.UID, updateIssueMessage, updaterID)
	if err != nil {
		return errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
	}

	payload, err := json.Marshal(api.ActivityIssueStatusUpdatePayload{
		OldStatus: issue.Status,
		NewStatus: newStatus,
		IssueName: updatedIssue.Title,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal activity after changing the issue status: %v", updatedIssue.Title)
	}
	activityCreate := &store.ActivityMessage{
		CreatorUID:   updaterID,
		ContainerUID: issue.UID,
		Type:         api.ActivityIssueStatusUpdate,
		Level:        api.ActivityInfo,
		Comment:      comment,
		Payload:      string(payload),
	}
	if _, err := activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
		Issue: updatedIssue,
	}); err != nil {
		return errors.Wrapf(err, "failed to create activity after changing the issue status: %v", updatedIssue.Title)
	}
	return nil
}
