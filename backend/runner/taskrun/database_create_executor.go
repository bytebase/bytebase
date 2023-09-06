package taskrun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewDatabaseCreateExecutor creates a database create task executor.
func NewDatabaseCreateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &DatabaseCreateExecutor{
		store:        store,
		dbFactory:    dbFactory,
		schemaSyncer: schemaSyncer,
		profile:      profile,
	}
}

// DatabaseCreateExecutor is the database create task executor.
type DatabaseCreateExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	schemaSyncer *schemasync.Syncer
	profile      config.Profile
}

var cannotCreateDatabase = map[db.Type]bool{
	db.Redis:  true,
	db.Oracle: true,
	db.DM:     true,
}

// RunOnce will run the database create task executor once.
func (exec *DatabaseCreateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseCreatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid create database payload")
	}
	statement, err := exec.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet statement of sheet: %d", payload.SheetID)
	}
	sheet, err := exec.store.GetSheet(ctx, &store.FindSheetMessage{UID: &payload.SheetID}, api.SystemBotID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet: %d", payload.SheetID)
	}
	if sheet == nil {
		return true, nil, errors.Errorf("sheet not found: %d", payload.SheetID)
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

	project, err := exec.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &payload.ProjectID})
	if err != nil {
		return true, nil, errors.Errorf("failed to find project with ID %d", payload.ProjectID)
	}
	if project == nil {
		return true, nil, errors.Errorf("project not found with ID %d", payload.ProjectID)
	}

	// Create database.
	log.Debug("Start creating database...",
		zap.String("instance", instance.Title),
		zap.String("database", payload.DatabaseName),
		zap.String("statement", statement),
	)

	// Upsert first because we need database id in instance change history.
	// The sync status is NOT_FOUND, which will be updated to OK if succeeds.
	labels := make(map[string]string)
	if payload.Labels != "" {
		var databaseLabels []*api.DatabaseLabel
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
		EnvironmentID:        payload.EnvironmentID,
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
	case db.MongoDB:
		// For MongoDB, it allows us to connect to the non-existing database. So we pass the database name to driver to let us connect to the specific database.
		// And run the create collection statement later.
		// NOTE: we have to hack the database message.
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
		if err != nil {
			return true, nil, err
		}
	case db.Oracle:
		return true, nil, errors.Errorf("Do not support creating databases for Oracle")
	case db.DM:
		return true, nil, errors.Errorf("Do not support creating databases for DM")
	default:
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
		if err != nil {
			return true, nil, err
		}
	}
	defer defaultDBDriver.Close(ctx)
	if _, err := defaultDBDriver.Execute(driverCtx, statement, true /* createDatabase */, db.ExecuteOptions{}); err != nil {
		return true, nil, err
	}

	environmentID := instance.EnvironmentID
	if payload.EnvironmentID != "" {
		environmentID = payload.EnvironmentID
	}
	environment, err := exec.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &environmentID})
	if err != nil {
		return true, nil, err
	}
	// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
	peerSchemaVersion, peerSchema, err := exec.createInitialSchema(ctx, driverCtx, environment, instance, project, task, database)
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

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", instance.ResourceID),
			zap.String("databaseName", database.DatabaseName),
			zap.Error(err),
		)
	}

	return true, &api.TaskRunResultPayload{
		Detail:      fmt.Sprintf("Created database %q", payload.DatabaseName),
		MigrationID: "",
		Version:     peerSchemaVersion,
	}, nil
}

func (exec *DatabaseCreateExecutor) createInitialSchema(ctx context.Context, driverCtx context.Context, environment *store.EnvironmentMessage, instance *store.InstanceMessage, project *store.ProjectMessage, task *store.TaskMessage, database *store.DatabaseMessage) (string, string, error) {
	if project.TenantMode != api.TenantModeTenant {
		return "", "", nil
	}

	schemaVersion, schema, err := exec.getSchemaFromPeerTenantDatabase(ctx, exec.store, exec.dbFactory, instance, project, database)
	if err != nil {
		return "", "", err
	}
	if schema == "" {
		return "", "", nil
	}

	driver, err := exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return "", "", err
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
		Force:          true,
	}
	creator, err := exec.store.GetUserByID(ctx, task.CreatorID)
	if err != nil {
		// If somehow we unable to find the principal, we just emit the error since it's not
		// critical enough to fail the entire operation.
		log.Error("Failed to fetch creator for composing the migration info",
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	} else {
		mi.Creator = creator.Name
	}
	issue, err := exec.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		// If somehow we unable to find the issue, we just emit the error since it's not
		// critical enough to fail the entire operation.
		log.Error("Failed to fetch containing issue for composing the migration info",
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	}
	if issue == nil {
		err := errors.Errorf("failed to fetch containing issue for composing the migration info, issue not found with pipeline ID %v", task.PipelineID)
		log.Error(err.Error(),
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	} else {
		mi.IssueID = strconv.Itoa(issue.UID)
		mi.IssueIDInt = &issue.UID
	}

	if _, _, err := utils.ExecuteMigrationDefault(ctx, driverCtx, exec.store, driver, mi, schema, nil, db.ExecuteOptions{}); err != nil {
		return "", "", err
	}
	return schemaVersion, schema, nil
}

func getConnectionStatement(dbType db.Type, databaseName string) (string, error) {
	switch dbType {
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		return fmt.Sprintf("USE `%s`;\n", databaseName), nil
	case db.MSSQL:
		return fmt.Sprintf(`USE "%s";\n`, databaseName), nil
	case db.Postgres, db.RisingWave:
		return fmt.Sprintf("\\connect \"%s\";\n", databaseName), nil
	case db.ClickHouse:
		return fmt.Sprintf("USE `%s`;\n", databaseName), nil
	case db.Snowflake:
		return fmt.Sprintf("USE DATABASE %s;\n", databaseName), nil
	case db.SQLite:
		return fmt.Sprintf("USE `%s`;\n", databaseName), nil
	case db.MongoDB:
		// We embed mongosh to execute the mongodb statement, and `use` statement is not effective in mongosh.
		// We will connect to the specified database by specifying the database name in the connection string.
		return "", nil
	case db.Redshift:
		return fmt.Sprintf("\\connect \"%s\";\n", databaseName), nil
	case db.Spanner:
		return "", nil
	}

	return "", errors.Errorf("unsupported database type %s", dbType)
}

// getSchemaFromPeerTenantDatabase gets the schema version and schema from a peer tenant database.
// It's used for creating a database in a tenant mode project.
// When a peer tenant database doesn't exist, we will return an error if there are databases in the project with the same name.
// Otherwise, we will create a blank database without schema.
func (*DatabaseCreateExecutor) getSchemaFromPeerTenantDatabase(ctx context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, instance *store.InstanceMessage, project *store.ProjectMessage, database *store.DatabaseMessage) (string, string, error) {
	allDatabases, err := stores.ListDatabases(ctx, &store.FindDatabaseMessage{
		ProjectID: &project.ResourceID,
	})
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to fetch databases in project ID: %v", project.UID)
	}
	var databases []*store.DatabaseMessage
	for _, d := range allDatabases {
		if d.UID != database.UID {
			databases = append(databases, d)
		}
	}

	deploymentConfig, err := stores.GetDeploymentConfigV2(ctx, project.UID)
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to fetch deployment config for project ID: %v", project.UID)
	}
	apiDeploymentConfig, err := deploymentConfig.ToAPIDeploymentConfig()
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to convert deployment config for project ID: %v", project.UID)
	}

	deploySchedule, err := api.ValidateAndGetDeploymentSchedule(apiDeploymentConfig.Payload)
	if err != nil {
		return "", "", errors.Errorf("Failed to get deployment schedule")
	}
	matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploySchedule, databases)
	if err != nil {
		return "", "", errors.Errorf("Failed to create deployment pipeline")
	}
	similarDB := getPeerTenantDatabase(matrix, instance.EnvironmentID)
	if similarDB == nil {
		return "", "", nil
	}
	similarDBInstance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &similarDB.InstanceID})
	if err != nil {
		return "", "", err
	}

	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, similarDBInstance, similarDB)
	if err != nil {
		return "", "", err
	}
	defer driver.Close(ctx)
	schemaVersion, err := utils.GetLatestSchemaVersion(ctx, stores, similarDBInstance.UID, similarDB.UID, similarDB.DatabaseName)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to get migration history for database %q", similarDB.DatabaseName)
	}

	var schemaBuf bytes.Buffer
	if _, err := driver.Dump(ctx, &schemaBuf, true /* schemaOnly */); err != nil {
		return "", "", err
	}
	return schemaVersion, schemaBuf.String(), nil
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
