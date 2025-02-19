package taskrun

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/github/gh-ost/go/logic"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewSchemaUpdateGhostExecutor creates a schema update (gh-ost) task executor.
func NewSchemaUpdateGhostExecutor(s *store.Store, secret string, dbFactory *dbfactory.DBFactory, license enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile *config.Profile) Executor {
	return &SchemaUpdateGhostExecutor{
		s:            s,
		secret:       secret,
		dbFactory:    dbFactory,
		license:      license,
		stateCfg:     stateCfg,
		schemaSyncer: schemaSyncer,
		profile:      profile,
	}
}

// SchemaUpdateGhostExecutor is the schema update (gh-ost) task executor.
type SchemaUpdateGhostExecutor struct {
	s            *store.Store
	secret       string
	dbFactory    *dbfactory.DBFactory
	license      enterprise.LicenseService
	stateCfg     *state.State
	schemaSyncer *schemasync.Syncer
	profile      *config.Profile
}

func (exec *SchemaUpdateGhostExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (bool, *storepb.TaskRunResult, error) {
	payload := &storepb.TaskDatabaseUpdatePayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update gh-ost sync payload")
	}
	sheetID := int(payload.SheetId)
	statement, err := exec.s.GetSheetStatementByID(ctx, int(payload.SheetId))
	if err != nil {
		return true, nil, err
	}
	flags := payload.Flags

	instance, err := exec.s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	if instance == nil {
		return true, nil, errors.Errorf("instance %s not found", task.InstanceID)
	}
	database, err := exec.s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return true, nil, err
	}
	if database == nil {
		return true, nil, errors.Errorf("database not found")
	}

	execFunc := func(execCtx context.Context, execStatement string) error {
		// set buffer size to 1 to unblock the sender because there is no listener if the task is canceled.
		migrationError := make(chan error, 1)

		statement := strings.TrimSpace(execStatement)

		tableName, err := ghost.GetTableNameFromStatement(statement)
		if err != nil {
			return err
		}

		adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
		if adminDataSource == nil {
			return common.Errorf(common.Internal, "admin data source not found for instance %s", instance.ResourceID)
		}

		migrationContext, err := ghost.NewMigrationContext(ctx, task.ID, database, adminDataSource, exec.secret, tableName, "", execStatement, false, flags, 10000000)
		if err != nil {
			return errors.Wrap(err, "failed to init migrationContext for gh-ost")
		}
		defer func() {
			// Use migrationContext.Uuid as the tls_config_key by convention.
			// We need to deregister it when gh-ost exits.
			// https://github.com/bytebase/gh-ost2/pull/4
			gomysql.DeregisterTLSConfig(migrationContext.Uuid)
		}()
		// TODO(p0ny): unset in NewMigrationContext.
		migrationContext.PostponeCutOverFlagFile = ""

		migrator := logic.NewMigrator(migrationContext, "bb")

		go func() {
			if err := migrator.Migrate(); err != nil {
				slog.Error("failed to run gh-ost migration", log.BBError(err))
				migrationError <- err
				return
			}
			migrationError <- nil
		}()

		select {
		case err := <-migrationError:
			if err != nil {
				return err
			}
			return nil
		case <-execCtx.Done():
			migrationContext.PanicAbort <- errors.New("task canceled")
			return errors.New("task canceled")
		}
	}

	terminated, result, err := runMigrationWithFunc(ctx, driverCtx, exec.s, exec.dbFactory, exec.stateCfg, exec.schemaSyncer, exec.profile, task, taskRunUID, db.Migrate, statement, payload.SchemaVersion, &sheetID, execFunc)
	// sync database schema anyways
	exec.s.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
		Type:              storepb.TaskRunLog_DATABASE_SYNC_START,
		DatabaseSyncStart: &storepb.TaskRunLog_DatabaseSyncStart{},
	})
	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, false /* force */); err != nil {
		exec.s.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
			Type: storepb.TaskRunLog_DATABASE_SYNC_END,
			DatabaseSyncEnd: &storepb.TaskRunLog_DatabaseSyncEnd{
				Error: err.Error(),
			},
		})
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	} else {
		exec.s.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
			Type: storepb.TaskRunLog_DATABASE_SYNC_END,
			DatabaseSyncEnd: &storepb.TaskRunLog_DatabaseSyncEnd{
				Error: "",
			},
		})
	}

	return terminated, result, err
}
