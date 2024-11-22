package taskrun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewDatabaseCreateExecutor creates a database create task executor.
func NewDatabaseCreateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer, stateCfg *state.State, profile *config.Profile) Executor {
	return &DatabaseCreateExecutor{
		store:        store,
		dbFactory:    dbFactory,
		schemaSyncer: schemaSyncer,
		stateCfg:     stateCfg,
		profile:      profile,
	}
}

// DatabaseCreateExecutor is the database create task executor.
type DatabaseCreateExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	schemaSyncer *schemasync.Syncer
	stateCfg     *state.State
	profile      *config.Profile
}

var cannotCreateDatabase = map[storepb.Engine]bool{
	storepb.Engine_REDIS:            true,
	storepb.Engine_ORACLE:           true,
	storepb.Engine_DM:               true,
	storepb.Engine_OCEANBASE_ORACLE: true,
}

// RunOnce will run the database create task executor once.
func (exec *DatabaseCreateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *storepb.TaskRunResult, err error) {
	payload := &storepb.TaskDatabaseCreatePayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid create database payload")
	}
	sheetID := int(payload.SheetId)
	statement, err := exec.store.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet statement of sheet: %d", sheetID)
	}
	sheet, err := exec.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetID})
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet: %d", sheetID)
	}
	if sheet == nil {
		return true, nil, errors.Errorf("sheet not found: %d", sheetID)
	}

	statement = strings.TrimSpace(statement)
	if statement == "" {
		return true, nil, errors.Errorf("empty create database statement")
	}

	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}

	if cannotCreateDatabase[instance.Engine] {
		return true, nil, errors.Errorf("Creating database is not supported")
	}

	projectID := int(payload.ProjectId)
	project, err := exec.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
	if err != nil {
		return true, nil, errors.Errorf("failed to find project with ID %d", projectID)
	}
	if project == nil {
		return true, nil, errors.Errorf("project not found with ID %d", projectID)
	}

	// Create database.
	slog.Debug("Start creating database...",
		slog.String("instance", instance.Title),
		slog.String("database", payload.DatabaseName),
		slog.String("statement", statement),
	)

	// Upsert first because we need database id in instance change history.
	// The sync status is NOT_FOUND, which will be updated to OK if succeeds.
	labels := make(map[string]string)
	if payload.Labels != "" {
		var databaseLabels []*storepb.DatabaseLabel
		if err := json.Unmarshal([]byte(payload.Labels), &databaseLabels); err != nil {
			return true, nil, err
		}
		for _, databaseLabel := range databaseLabels {
			labels[databaseLabel.Key] = databaseLabel.Value
		}
	}
	database, err := exec.store.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:            project.ResourceID,
		InstanceID:           instance.ResourceID,
		DatabaseName:         payload.DatabaseName,
		EnvironmentID:        payload.EnvironmentId,
		SyncState:            api.NotFound,
		SuccessfulSyncTimeTs: time.Now().Unix(),
		Metadata: &storepb.DatabaseMetadata{
			Labels: labels,
		},
	})
	if err != nil {
		return true, nil, err
	}

	var defaultDBDriver db.Driver
	switch instance.Engine {
	case storepb.Engine_MONGODB:
		// For MongoDB, it allows us to connect to the non-existing database. So we pass the database name to driver to let us connect to the specific database.
		// And run the create collection statement later.
		// NOTE: we have to hack the database message.
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return true, nil, err
		}
	default:
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
		if err != nil {
			return true, nil, err
		}
	}
	defer defaultDBDriver.Close(ctx)

	if _, err := defaultDBDriver.Execute(driverCtx, statement, db.ExecuteOptions{CreateDatabase: true}); err != nil {
		return true, nil, err
	}

	environmentID := instance.EnvironmentID
	if payload.EnvironmentId != "" {
		environmentID = payload.EnvironmentId
	}
	environment, err := exec.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &environmentID})
	if err != nil {
		return true, nil, err
	}
	// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
	peerDatabase, peerSchemaVersion, peerSchema, err := exec.createInitialSchema(ctx, driverCtx, environment, instance, project, task, taskRunUID, database)
	if err != nil {
		return true, nil, err
	}

	syncStatus := api.OK
	if _, err := exec.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:    instance.ResourceID,
		DatabaseName:  payload.DatabaseName,
		SyncState:     &syncStatus,
		SchemaVersion: &peerSchemaVersion,
	}, api.SystemBotID); err != nil {
		return true, nil, err
	}

	// After the task related database entry created successfully,
	// we need to update task's database_id and statement with the newly created database immediately.
	// Here is the main reason:
	// The task database_id represents its related database entry both for creating and patching,
	// so we should sync its value right here when the related database entry created.
	// The new statement should include the schema from peer tenant database.
	taskDatabaseIDPatch := &api.TaskPatch{
		ID:         task.ID,
		UpdaterID:  api.SystemBotID,
		DatabaseID: &database.UID,
	}
	sheetPatch := &store.PatchSheetMessage{
		UID:       sheet.UID,
		UpdaterID: api.SystemBotID,
	}

	if peerSchema != "" {
		// Better displaying schema in the task.
		connectionStmt, err := getConnectionStatement(instance.Engine, payload.DatabaseName)
		if err != nil {
			return true, nil, err
		}
		fullSchema := fmt.Sprintf("%s\n%s\n%s", statement, connectionStmt, peerSchema)

		sheetPatch.Statement = &fullSchema
	}
	if _, err := exec.store.UpdateTaskV2(ctx, taskDatabaseIDPatch); err != nil {
		return true, nil, err
	}
	if _, err := exec.store.PatchSheet(ctx, sheetPatch); err != nil {
		return true, nil, errors.Wrapf(err, "failed to update sheet %d after executing the task", sheet.UID)
	}

	exec.reconcilePlan(ctx, project, database, peerDatabase)

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, false /* force */); err != nil {
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	}

	storedVersion, err := peerSchemaVersion.Marshal()
	if err != nil {
		slog.Error("failed to convert database schema version",
			slog.String("version", peerSchemaVersion.Version),
			log.BBError(err),
		)
	}
	return true, &storepb.TaskRunResult{
		Detail:  fmt.Sprintf("Created database %q", payload.DatabaseName),
		Version: storedVersion,
	}, nil
}

// reconcilePlan adds the created database to other plans,
// if the project has tenant mode enabled
// if the issue is open
// if the plan uses a deploymentConfig
// if peer database task is not done
// if the peer database task type is schemaUpdate.
func (exec *DatabaseCreateExecutor) reconcilePlan(ctx context.Context, project *store.ProjectMessage, createdDatabase *store.DatabaseMessage, peerDatabase *store.DatabaseMessage) {
	if peerDatabase == nil {
		return
	}

	issues, err := exec.store.ListIssueV2(ctx, &store.FindIssueMessage{
		ProjectID:   &project.ResourceID,
		DatabaseUID: &peerDatabase.UID,
		StatusList:  []api.IssueStatus{api.IssueOpen},
		TaskTypes:   &[]api.TaskType{api.TaskDatabaseSchemaUpdate},
	})
	if err != nil {
		slog.Debug("failed to list issues", log.BBError(err))
		return
	}
	for _, issue := range issues {
		err := func() error {
			plan, err := exec.store.GetPlan(ctx, &store.FindPlanMessage{PipelineID: issue.PipelineUID})
			if err != nil {
				return errors.Wrapf(err, "failed to get plan for issue %d, pipeline %d", issue.UID, issue.PipelineUID)
			}
			if plan == nil {
				return errors.Wrapf(err, "plan not found for issue %d, pipeline %d", issue.UID, issue.PipelineUID)
			}
			if len(plan.Config.GetSteps()) != 1 {
				return nil
			}
			if len(plan.Config.Steps[0].GetSpecs()) != 1 {
				return nil
			}
			spec := plan.Config.Steps[0].Specs[0]
			c := spec.GetChangeDatabaseConfig()
			if c == nil {
				return nil
			}
			_, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(c.Target)
			if err != nil {
				// continue because this is not a plan that uses database group.
				//nolint:nilerr
				return nil
			}
			databaseGroup, err := exec.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{ResourceID: &databaseGroupID})
			if err != nil {
				//nolint:nilerr
				return nil
			}
			if databaseGroup == nil || !databaseGroup.Payload.Multitenancy {
				return nil
			}
			isMatched, err := utils.CheckDatabaseGroupMatch(ctx, databaseGroup.Expression.Expression, createdDatabase)
			if err != nil || !isMatched {
				// continue if current database is not matched.
				//nolint:nilerr
				return nil
			}

			// We somehow reconciled the plan before, so we just return.
			tasks, err := exec.store.ListTasks(ctx, &api.TaskFind{
				PipelineID: issue.PipelineUID,
				DatabaseID: &createdDatabase.UID,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to list tasks for created database %q", createdDatabase.DatabaseName)
			}
			if len(tasks) > 0 {
				return nil
			}

			tasks, err = exec.store.ListTasks(ctx, &api.TaskFind{
				PipelineID: issue.PipelineUID,
				DatabaseID: &peerDatabase.UID,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to list tasks for peer database %q", peerDatabase.DatabaseName)
			}

			var creates []*store.TaskMessage
			for _, task := range tasks {
				if task.LatestTaskRunStatus == api.TaskRunDone {
					continue
				}
				switch task.Type {
				case api.TaskDatabaseSchemaUpdate:
				default:
					continue
				}
				taskCreate := &store.TaskMessage{
					CreatorID:         api.SystemBotID,
					UpdaterID:         api.SystemBotID,
					PipelineID:        task.PipelineID,
					StageID:           task.StageID,
					Name:              fmt.Sprintf("Copied task for database %q from %q", createdDatabase.DatabaseName, peerDatabase.DatabaseName),
					InstanceID:        task.InstanceID,
					DatabaseID:        &createdDatabase.UID,
					Type:              task.Type,
					EarliestAllowedTs: task.EarliestAllowedTs,
					Payload:           task.Payload,
				}
				creates = append(creates, taskCreate)
			}
			if _, err := exec.store.CreateTasksV2(ctx, creates...); err != nil {
				return errors.Wrapf(err, "failed to create tasks")
			}
			return nil
		}()
		if err != nil {
			slog.Error("failed to reconcile plan", log.BBError(err))
		}
	}
}

func (exec *DatabaseCreateExecutor) createInitialSchema(ctx context.Context, driverCtx context.Context, environment *store.EnvironmentMessage, instance *store.InstanceMessage, project *store.ProjectMessage, task *store.TaskMessage, taskRunUID int, database *store.DatabaseMessage) (*store.DatabaseMessage, model.Version, string, error) {
	peerDatabase, schemaVersion, schema, err := exec.getSchemaFromPeerTenantDatabase(ctx, instance, project, database)
	if err != nil {
		return nil, model.Version{}, "", err
	}
	if schema == "" {
		return nil, model.Version{}, "", nil
	}

	driver, err := exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
	if err != nil {
		return nil, model.Version{}, "", err
	}
	defer driver.Close(ctx)

	// TODO(d): support semantic versioning.
	mi := &db.MigrationInfo{
		InstanceID:     &task.InstanceID,
		CreatorID:      task.CreatorID,
		ReleaseVersion: exec.profile.Version,
		Version:        schemaVersion,
		Namespace:      database.DatabaseName,
		Database:       database.DatabaseName,
		DatabaseID:     &database.UID,
		Environment:    environment.ResourceID,
		Source:         db.UI,
		Type:           db.Migrate,
		Description:    "Create database",
	}
	creator, err := exec.store.GetUserByID(ctx, task.CreatorID)
	if err != nil {
		// If somehow we unable to find the principal, we just emit the error since it's not
		// critical enough to fail the entire operation.
		slog.Error("Failed to fetch creator for composing the migration info",
			slog.Int("task_id", task.ID),
			log.BBError(err),
		)
	} else {
		mi.Creator = creator.Name
	}
	issue, err := exec.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		// If somehow we unable to find the issue, we just emit the error since it's not
		// critical enough to fail the entire operation.
		slog.Error("Failed to fetch containing issue for composing the migration info",
			slog.Int("task_id", task.ID),
			log.BBError(err),
		)
	}
	mi.ProjectUID = &project.UID
	// TODO(d): how could issue be nil?
	if issue == nil {
		err := errors.Errorf("failed to fetch containing issue for composing the migration info, issue not found with pipeline ID %v", task.PipelineID)
		slog.Error(err.Error(),
			slog.Int("task_id", task.ID),
			log.BBError(err),
		)
	} else {
		mi.IssueUID = &issue.UID
	}

	mc := &migrateContext{
		instance:    instance,
		database:    database,
		sheet:       nil,
		task:        task,
		taskRunUID:  taskRunUID,
		taskRunName: common.FormatTaskRun(project.ResourceID, task.PipelineID, task.StageID, task.ID, taskRunUID),
		version:     schemaVersion.Version,
	}

	if _, _, err := executeMigrationDefault(ctx, driverCtx, exec.store, exec.stateCfg, driver, mi, mc, schema, db.ExecuteOptions{}); err != nil {
		return nil, model.Version{}, "", err
	}
	return peerDatabase, schemaVersion, schema, nil
}

func getConnectionStatement(dbType storepb.Engine, databaseName string) (string, error) {
	switch dbType {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return fmt.Sprintf("USE `%s`;\n", databaseName), nil
	case storepb.Engine_MSSQL:
		return fmt.Sprintf(`USE "%s";\n`, databaseName), nil
	case storepb.Engine_POSTGRES, storepb.Engine_RISINGWAVE:
		return fmt.Sprintf("\\connect \"%s\";\n", databaseName), nil
	case storepb.Engine_CLICKHOUSE:
		return fmt.Sprintf("USE `%s`;\n", databaseName), nil
	case storepb.Engine_SNOWFLAKE:
		return fmt.Sprintf("USE DATABASE %s;\n", databaseName), nil
	case storepb.Engine_SQLITE:
		return fmt.Sprintf("USE `%s`;\n", databaseName), nil
	case storepb.Engine_MONGODB:
		// We embed mongosh to execute the mongodb statement, and `use` statement is not effective in mongosh.
		// We will connect to the specified database by specifying the database name in the connection string.
		return "", nil
	case storepb.Engine_REDSHIFT:
		return fmt.Sprintf("\\connect \"%s\";\n", databaseName), nil
	case storepb.Engine_SPANNER:
		return "", nil
	}

	return "", errors.Errorf("unsupported database type %s", dbType)
}

// getSchemaFromPeerTenantDatabase gets the schema version and schema from a peer tenant database.
// It's used for creating a database in a tenant mode project.
// When a peer tenant database doesn't exist, we will return an error if there are databases in the project with the same name.
// Otherwise, we will create a blank database without schema.
func (exec *DatabaseCreateExecutor) getSchemaFromPeerTenantDatabase(ctx context.Context, instance *store.InstanceMessage, project *store.ProjectMessage, database *store.DatabaseMessage) (*store.DatabaseMessage, model.Version, string, error) {
	// Try to find a peer tenant database from database groups.
	matchedDatabases, err := exec.getPeerTenantDatabasesFromDatabaseGroup(ctx, instance, project, database)
	if err != nil {
		return nil, model.Version{}, "", errors.Wrapf(err, "Failed to fetch database groups in project ID: %v", project.UID)
	}

	// Filter out the database itself.
	var databases []*store.DatabaseMessage
	for _, d := range matchedDatabases {
		if d.UID != database.UID {
			databases = append(databases, d)
		}
	}
	matchedDatabases = databases
	if len(matchedDatabases) == 0 {
		return nil, model.Version{}, "", nil
	}

	// Then we will try to find a peer tenant database from deployment schedule with the matched databases.
	deploymentConfig, err := exec.store.GetDeploymentConfigV2(ctx, project.UID)
	if err != nil {
		return nil, model.Version{}, "", errors.Wrapf(err, "Failed to fetch deployment config for project ID: %v", project.UID)
	}
	if err := utils.ValidateDeploymentSchedule(deploymentConfig.Schedule); err != nil {
		return nil, model.Version{}, "", errors.Errorf("Failed to get deployment schedule")
	}
	matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploymentConfig.Schedule, matchedDatabases)
	if err != nil {
		return nil, model.Version{}, "", errors.Errorf("Failed to create deployment pipeline")
	}
	similarDB := getPeerTenantDatabase(matrix, instance.EnvironmentID)
	if similarDB == nil {
		return nil, model.Version{}, "", nil
	}
	similarDBInstance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &similarDB.InstanceID})
	if err != nil {
		return nil, model.Version{}, "", err
	}

	driver, err := exec.dbFactory.GetAdminDatabaseDriver(ctx, similarDBInstance, similarDB, db.ConnectionContext{})
	if err != nil {
		return nil, model.Version{}, "", err
	}
	defer driver.Close(ctx)
	schemaVersion, err := getLatestDoneSchemaVersion(ctx, exec.store, similarDBInstance.UID, similarDB.UID, similarDB.DatabaseName)
	if err != nil {
		return nil, model.Version{}, "", errors.Wrapf(err, "failed to get migration history for database %q", similarDB.DatabaseName)
	}

	dbSchema := (*storepb.DatabaseSchemaMetadata)(nil)
	if instance.Engine == storepb.Engine_MYSQL {
		dbSchema, err = driver.SyncDBSchema(ctx)
		if err != nil {
			return nil, model.Version{}, "", errors.Wrapf(err, "failed to get schema for database %q", similarDB.DatabaseName)
		}
	}

	var schemaBuf bytes.Buffer
	if err := driver.Dump(ctx, &schemaBuf, dbSchema); err != nil {
		return nil, model.Version{}, "", err
	}
	return similarDB, schemaVersion, schemaBuf.String(), nil
}

func (exec *DatabaseCreateExecutor) getPeerTenantDatabasesFromDatabaseGroup(ctx context.Context, instance *store.InstanceMessage, project *store.ProjectMessage, database *store.DatabaseMessage) ([]*store.DatabaseMessage, error) {
	dbGroups, err := exec.store.ListDatabaseGroups(ctx, &store.FindDatabaseGroupMessage{ProjectUID: &project.UID})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch database groups in project ID: %v", project.UID)
	}
	allDatabases, err := exec.store.ListDatabases(ctx, &store.FindDatabaseMessage{
		ProjectID: &project.ResourceID,
		Engine:    &instance.Engine,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch databases in project ID: %v", project.UID)
	}

	var matchedDatabases []*store.DatabaseMessage
	for _, dbGroup := range dbGroups {
		// TODO(steven): move this filter into FindDatabaseGroupMessage.
		if !dbGroup.Payload.Multitenancy {
			continue
		}

		isMatched, err := utils.CheckDatabaseGroupMatch(ctx, dbGroup.Expression.Expression, database)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get matched and unmatched databases in database group %q", dbGroup.Placeholder)
		}
		// If current database is not matched, continue to the next database group.
		if !isMatched {
			continue
		}

		matchedDb, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, dbGroup, allDatabases)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get matched and unmatched databases in database group %q", dbGroup.Placeholder)
		}
		if len(matchedDb) > 0 {
			matchedDatabases = matchedDb
		}
	}
	return matchedDatabases, nil
}

// GetLatestDoneSchemaVersion gets the latest schema version for a database.
func getLatestDoneSchemaVersion(ctx context.Context, stores *store.Store, instanceID int, databaseID int, databaseName string) (model.Version, error) {
	// TODO(d): support semantic versioning.
	limit := 1
	done := db.Done
	history, err := stores.ListInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: &instanceID,
		DatabaseID: &databaseID,
		Status:     &done,
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

func getPeerTenantDatabase(databaseMatrix [][]*store.DatabaseMessage, environmentID string) *store.DatabaseMessage {
	var similarDB *store.DatabaseMessage
	// We try to use an existing tenant with the same environment, if possible.
	for _, databaseList := range databaseMatrix {
		for _, db := range databaseList {
			if db.EffectiveEnvironmentID == environmentID {
				similarDB = db
				break
			}
		}
		if similarDB != nil {
			break
		}
	}
	if similarDB == nil {
		for _, stage := range databaseMatrix {
			if len(stage) > 0 {
				similarDB = stage[0]
				break
			}
		}
	}

	return similarDB
}
