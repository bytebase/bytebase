package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// NewDatabaseCreateExecutor creates a database create task executor.
func NewDatabaseCreateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer, stateCfg *state.State, profile config.Profile) Executor {
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
	profile      config.Profile
}

var cannotCreateDatabase = map[storepb.Engine]bool{
	storepb.Engine_REDIS:            true,
	storepb.Engine_ORACLE:           true,
	storepb.Engine_DM:               true,
	storepb.Engine_OCEANBASE_ORACLE: true,
}

// RunOnce will run the database create task executor once.
func (exec *DatabaseCreateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *storepb.TaskRunResult, err error) {
	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_PRE_EXECUTING,
			UpdateTime:      time.Now(),
		})

	payload := &api.TaskDatabaseCreatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid create database payload")
	}
	statement, err := exec.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet statement of sheet: %d", payload.SheetID)
	}
	sheet, err := exec.store.GetSheet(ctx, &store.FindSheetMessage{UID: &payload.SheetID})
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
	slog.Debug("Start creating database...",
		slog.String("instance", instance.Title),
		slog.String("database", payload.DatabaseName),
		slog.String("statement", statement),
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

	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_EXECUTING,
			UpdateTime:      time.Now(),
		})

	if _, err := defaultDBDriver.Execute(driverCtx, statement, db.ExecuteOptions{CreateDatabase: true}); err != nil {
		return true, nil, err
	}

	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_POST_EXECUTING,
			UpdateTime:      time.Now(),
		})

	schemaVersion := &model.Version{}
	syncStatus := api.OK
	if _, err := exec.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:    instance.ResourceID,
		DatabaseName:  payload.DatabaseName,
		SyncState:     &syncStatus,
		SchemaVersion: schemaVersion,
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
	if _, err := exec.store.UpdateTaskV2(ctx, taskDatabaseIDPatch); err != nil {
		return true, nil, err
	}
	if _, err := exec.store.PatchSheet(ctx, sheetPatch); err != nil {
		return true, nil, errors.Wrapf(err, "failed to update sheet %d after executing the task", sheet.UID)
	}

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	}

	storedVersion, err := schemaVersion.Marshal()
	if err != nil {
		slog.Error("failed to convert database schema version",
			slog.String("version", schemaVersion.Version),
			log.BBError(err),
		)
	}
	return true, &storepb.TaskRunResult{
		Detail:  fmt.Sprintf("Created database %q", payload.DatabaseName),
		Version: storedVersion,
	}, nil
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
