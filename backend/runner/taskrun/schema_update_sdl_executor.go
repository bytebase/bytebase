package taskrun

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/runner/utils"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	backendutils "github.com/bytebase/bytebase/backend/utils"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// NewSchemaUpdateSDLExecutor creates a schema update (SDL) task executor.
func NewSchemaUpdateSDLExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &SchemaUpdateSDLExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		license:         license,
		stateCfg:        stateCfg,
		schemaSyncer:    schemaSyncer,
		profile:         profile,
	}
}

// SchemaUpdateSDLExecutor is the schema update (SDL) task executor.
type SchemaUpdateSDLExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	license         enterprise.LicenseService
	stateCfg        *state.State
	schemaSyncer    *schemasync.Syncer
	profile         config.Profile
}

// RunOnce will run the schema update (SDL) task executor once.
func (exec *SchemaUpdateSDLExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (bool, *api.TaskRunResultPayload, error) {
	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_PRE_EXECUTING,
			UpdateTime:      time.Now(),
		})

	payload := &api.TaskDatabaseSchemaUpdateSDLPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update payload")
	}

	statement, err := exec.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return true, nil, err
	}

	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}

	materials := backendutils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := backendutils.RenderStatement(statement, materials)

	version := model.Version{Version: payload.SchemaVersion}
	ddl, err := utils.ComputeDatabaseSchemaDiff(ctx, instance, database, exec.dbFactory, renderedStatement)
	if err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema diff")
	}
	terminated, result, err := runMigration(ctx, driverCtx, exec.store, exec.dbFactory, exec.activityManager, exec.license, exec.stateCfg, exec.profile, task, taskRunUID, db.MigrateSDL, ddl, version, &payload.SheetID)

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	}

	return terminated, result, err
}
