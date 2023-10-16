package taskrun

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/runner/backuprun"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// NewPITRCutoverExecutor creates a PITR cutover task executor.
func NewPITRCutoverExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer, stateCfg *state.State, backupRunner *backuprun.Runner, activityManager *activity.Manager, profile config.Profile) Executor {
	return &PITRCutoverExecutor{
		store:           store,
		dbFactory:       dbFactory,
		schemaSyncer:    schemaSyncer,
		stateCfg:        stateCfg,
		backupRunner:    backupRunner,
		activityManager: activityManager,
		profile:         profile,
	}
}

// PITRCutoverExecutor is the PITR cutover task executor.
type PITRCutoverExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	schemaSyncer    *schemasync.Syncer
	stateCfg        *state.State
	backupRunner    *backuprun.Runner
	activityManager *activity.Manager
	profile         config.Profile
}

// RunOnce will run the PITR cutover task executor once.
// TODO: support cancellation.
func (exec *PITRCutoverExecutor) RunOnce(ctx context.Context, _ context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *api.TaskRunResultPayload, err error) {
	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_EXECUTING,
			UpdateTime:      time.Now(),
		})

	slog.Info("Run PITR cutover task", slog.String("task", task.Name))
	issue, err := exec.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		slog.Error("failed to fetch containing issue doing pitr cutover task", log.BBError(err))
		return true, nil, err
	}
	if issue == nil {
		return true, nil, errors.Errorf("issue not found for pipeline %v", task.PipelineID)
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}
	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}

	// Currently api.TaskDatabasePITRCutoverPayload is empty, so we do not need to unmarshal from task.Payload.
	terminated, result, err = exec.pitrCutover(ctx, exec.dbFactory, exec.backupRunner, exec.schemaSyncer, exec.profile, task, taskRunUID, database, issue)
	if err != nil {
		return terminated, result, err
	}

	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskID:    task.ID,
		OldStatus: api.TaskRunning,
		NewStatus: api.TaskDone,
		IssueName: issue.Title,
		TaskName:  task.Name,
	})
	if err != nil {
		slog.Error("failed to marshal activity", log.BBError(err))
		return terminated, result, nil
	}

	activityCreate := &store.ActivityMessage{
		CreatorUID:   task.UpdaterID,
		ContainerUID: issue.Project.UID,
		Type:         api.ActivityDatabaseRecoveryPITRDone,
		Level:        api.ActivityInfo,
		Payload:      string(payload),
		Comment:      fmt.Sprintf("Restore database %s in instance %s successfully.", database.DatabaseName, instance.Title),
	}
	if _, err = exec.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{Issue: issue}); err != nil {
		slog.Error("cannot create an pitr activity", log.BBError(err))
	}

	return terminated, result, nil
}

// pitrCutover performs the PITR cutover algorithm:
// 1. Swap the current and PITR database.
// 2. Create a backup with type PITR. The backup is scheduled asynchronously.
// We must check the possible failed/ongoing PITR type backup in the recovery process.
func (exec *PITRCutoverExecutor) pitrCutover(ctx context.Context, dbFactory *dbfactory.DBFactory, backupRunner *backuprun.Runner, schemaSyncer *schemasync.Syncer, profile config.Profile, task *store.TaskMessage, taskRunUID int, database *store.DatabaseMessage, issue *store.IssueMessage) (terminated bool, result *api.TaskRunResultPayload, err error) {
	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	environment, err := exec.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
	if err != nil {
		return true, nil, err
	}
	project, err := exec.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return true, nil, err
	}
	creator, err := exec.store.GetUserByID(ctx, task.CreatorID)
	if err != nil {
		return true, nil, err
	}

	if err := exec.doCutover(ctx, instance, issue, database.DatabaseName); err != nil {
		return true, nil, err
	}

	// RestorePITR will create the pitr database.
	// Since it's ephemeral and will be renamed to the original database soon, we will reuse the original
	// database's migration history, and append a new BRANCH migration.
	slog.Debug("Appending new migration history record")
	m := &db.MigrationInfo{
		InstanceID:     &task.InstanceID,
		IssueUID:       &issue.UID,
		ReleaseVersion: profile.Version,
		Version:        common.DefaultMigrationVersion(),
		Namespace:      database.DatabaseName,
		Database:       database.DatabaseName,
		DatabaseID:     &database.UID,
		Environment:    environment.ResourceID,
		Source:         db.MigrationSource(project.Workflow),
		Type:           db.Branch,
		Status:         db.Done,
		Description:    fmt.Sprintf("PITR: restoring database %s", database.DatabaseName),
		Creator:        creator.Name,
		CreatorID:      creator.ID,
	}

	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	if _, _, err := utils.ExecuteMigrationDefault(ctx, ctx, exec.store, exec.stateCfg, taskRunUID, driver, m, "" /* pitr cutover */, nil, db.ExecuteOptions{}); err != nil {
		slog.Error("Failed to add migration history record", log.BBError(err))
		return true, nil, errors.Wrap(err, "failed to add migration history record")
	}

	// TODO(dragonly): Only needed for in-place PITR.
	backupName := fmt.Sprintf("%s-%s-pitr-%d", slug.Make(project.Title), slug.Make(environment.Title), issue.CreatedTime.Unix())
	if _, err := backupRunner.ScheduleBackupTask(ctx, database, backupName, api.BackupTypePITR, api.SystemBotID); err != nil {
		return true, nil, errors.Wrapf(err, "failed to schedule backup task for database %q after PITR", database.DatabaseName)
	}

	// Sync database schema after restore is completed.
	if err := schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	}

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Swapped PITR database for target database %q", database.DatabaseName),
	}, nil
}

func (exec *PITRCutoverExecutor) doCutover(ctx context.Context, instance *store.InstanceMessage, issue *store.IssueMessage, databaseName string) error {
	switch instance.Engine {
	case storepb.Engine_POSTGRES:
		// Retry so that if there are clients reconnecting to the related databases, we can potentially kill the connections and do the cutover successfully.
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		maxRetry := 3
		retry := 0
		for {
			select {
			case <-ticker.C:
				retry++
				if err := exec.pitrCutoverPostgres(ctx, instance, issue, databaseName); err != nil {
					if retry == maxRetry {
						return errors.Wrapf(err, "failed to do cutover for PostgreSQL after retried for %d times", maxRetry)
					}
					slog.Debug("Failed to do cutover for PostgreSQL. Retry later.", log.BBError(err))
				} else {
					return nil
				}
			case <-ctx.Done():
				return errors.Errorf("context is canceled when doing cutover for PostgreSQL")
			}
		}
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB:
		if err := exec.pitrCutoverMySQL(ctx, instance, issue, databaseName); err != nil {
			return errors.Wrap(err, "failed to do cutover for MySQL")
		}
		return nil
	default:
		return errors.Errorf("invalid database type %q for cutover task", instance.Engine)
	}
}

func (exec *PITRCutoverExecutor) pitrCutoverMySQL(ctx context.Context, instance *store.InstanceMessage, issue *store.IssueMessage, databaseName string) error {
	driver, err := exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)
	driverDB := driver.GetDB()
	conn, err := driverDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	slog.Debug("Swapping the original and PITR database", slog.String("originalDatabase", databaseName))
	pitrDatabaseName, pitrOldDatabaseName, err := mysql.SwapPITRDatabase(ctx, conn, databaseName, issue.CreatedTime.Unix())
	if err != nil {
		slog.Error("Failed to swap the original and PITR database", slog.String("originalDatabase", databaseName), slog.String("pitrDatabase", pitrDatabaseName), log.BBError(err))
		return errors.Wrap(err, "failed to swap the original and PITR database")
	}
	slog.Debug("Finished swapping the original and PITR database", slog.String("originalDatabase", databaseName), slog.String("pitrDatabase", pitrDatabaseName), slog.String("oldDatabase", pitrOldDatabaseName))
	return nil
}

func (exec *PITRCutoverExecutor) pitrCutoverPostgres(ctx context.Context, instance *store.InstanceMessage, issue *store.IssueMessage, databaseName string) error {
	pitrDatabaseName := util.GetPITRDatabaseName(databaseName, issue.CreatedTime.Unix())
	pitrOldDatabaseName := util.GetPITROldDatabaseName(databaseName, issue.CreatedTime.Unix())

	defaultDBDriver, err := exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return err
	}
	db := defaultDBDriver.GetDB()

	// The original database may not exist.
	// This is possible if there's a former task execution which successfully renamed original -> _del database and failed.
	// Now we have the _del and the _pitr database.
	existOriginal, err := pgDatabaseExist(ctx, db, databaseName)
	if err != nil {
		return errors.Wrapf(err, "failed to check existence of database %q", databaseName)
	}
	if existOriginal {
		// Kill connections to the original database in the beginning of cutover.
		if err := pgKillConnectionsToDatabase(ctx, db, databaseName); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s RENAME TO %s;", databaseName, pitrOldDatabaseName)); err != nil {
			return errors.Wrapf(err, "failed to rename database %q to %q", databaseName, pitrOldDatabaseName)
		}
		slog.Debug("Successfully renamed database", slog.String("from", databaseName), slog.String("to", pitrOldDatabaseName))
	}

	// The _pitr database may not exist.
	// This is possible if there's a former task execution which successfully renamed _pitr -> original database and failed.
	// Now we have the _del and the original database.
	existPITR, err := pgDatabaseExist(ctx, db, pitrDatabaseName)
	if err != nil {
		return errors.Wrapf(err, "failed to check existence of database %q", pitrDatabaseName)
	}
	if existPITR {
		// Kill connections to the original database again in case that the clients reconnect to the original database.
		if err := pgKillConnectionsToDatabase(ctx, db, databaseName); err != nil {
			return err
		}
		// Kill connection to the _pitr database in case there's someone connecting to all of the existing databases like postgres-exporter.
		if err := pgKillConnectionsToDatabase(ctx, db, pitrDatabaseName); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s RENAME TO %s;", pitrDatabaseName, databaseName)); err != nil {
			return errors.Wrapf(err, "failed to rename database %q to %q", pitrDatabaseName, databaseName)
		}
		slog.Debug("Successfully renamed database", slog.String("from", pitrDatabaseName), slog.String("to", databaseName))
	}

	return nil
}

func pgKillConnectionsToDatabase(ctx context.Context, db *sql.DB, database string) error {
	stmt := `
	SELECT pg_terminate_backend( pid )
	FROM pg_stat_activity
	WHERE pid <> pg_backend_pid( )
	  AND datname = $1;
`
	if _, err := db.ExecContext(ctx, stmt, database); err != nil {
		return errors.Wrapf(err, "failed to kill all connections to database %q", database)
	}
	return nil
}

func pgDatabaseExist(ctx context.Context, db *sql.DB, database string) (bool, error) {
	query := fmt.Sprintf("SELECT datname FROM pg_database WHERE datname='%s';", database)
	var unused string
	if err := db.QueryRowContext(ctx, query).Scan(&unused); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, util.FormatErrorWithQuery(err, query)
	}
	return true, nil
}
