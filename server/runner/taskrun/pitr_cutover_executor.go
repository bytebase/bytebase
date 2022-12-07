package taskrun

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/server/runner/backuprun"
	"github.com/bytebase/bytebase/server/runner/schemasync"
	"github.com/bytebase/bytebase/store"
)

// NewPITRCutoverExecutor creates a PITR cutover task executor.
func NewPITRCutoverExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer, backupRunner *backuprun.Runner, activityManager *activity.Manager, profile config.Profile) Executor {
	return &PITRCutoverExecutor{
		store:           store,
		dbFactory:       dbFactory,
		schemaSyncer:    schemaSyncer,
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
	backupRunner    *backuprun.Runner
	activityManager *activity.Manager
	profile         config.Profile
}

// RunOnce will run the PITR cutover task executor once.
func (exec *PITRCutoverExecutor) RunOnce(ctx context.Context, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run PITR cutover task", zap.String("task", task.Name))
	issue, err := getIssueByPipelineID(ctx, exec.store, task.PipelineID)
	if err != nil {
		log.Error("failed to fetch containing issue doing pitr cutover task", zap.Error(err))
		return true, nil, err
	}

	// Currently api.TaskDatabasePITRCutoverPayload is empty, so we do not need to unmarshal from task.Payload.

	terminated, result, err = exec.pitrCutover(ctx, exec.dbFactory, exec.backupRunner, exec.schemaSyncer, exec.profile, task, issue)
	if err != nil {
		return terminated, result, err
	}

	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskID:    task.ID,
		OldStatus: task.Status,
		NewStatus: api.TaskDone,
		IssueName: issue.Name,
		TaskName:  task.Name,
	})
	if err != nil {
		log.Error("failed to marshal activity", zap.Error(err))
		return terminated, result, nil
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   task.UpdaterID,
		ContainerID: issue.ProjectID,
		Type:        api.ActivityDatabaseRecoveryPITRDone,
		Level:       api.ActivityInfo,
		Payload:     string(payload),
		Comment:     fmt.Sprintf("Restore database %s in instance %s successfully.", task.Database.Name, task.Instance.Name),
	}
	if _, err = exec.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{Issue: issue}); err != nil {
		log.Error("cannot create an pitr activity", zap.Error(err))
	}

	return terminated, result, nil
}

// pitrCutover performs the PITR cutover algorithm:
// 1. Swap the current and PITR database.
// 2. Create a backup with type PITR. The backup is scheduled asynchronously.
// We must check the possible failed/ongoing PITR type backup in the recovery process.
func (exec *PITRCutoverExecutor) pitrCutover(ctx context.Context, dbFactory *dbfactory.DBFactory, backupRunner *backuprun.Runner, schemaSyncer *schemasync.Syncer, profile config.Profile, task *api.Task, issue *api.Issue) (terminated bool, result *api.TaskRunResultPayload, err error) {
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, task.Instance, "" /* databaseName */)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	if err := exec.doCutover(ctx, driver, task, issue); err != nil {
		return true, nil, err
	}

	// RestorePITR will create the pitr database.
	// Since it's ephemeral and will be renamed to the original database soon, we will reuse the original
	// database's migration history, and append a new BRANCH migration.
	log.Debug("Appending new migration history record")
	m := &db.MigrationInfo{
		ReleaseVersion: profile.Version,
		Version:        common.DefaultMigrationVersion(),
		Namespace:      task.Database.Name,
		Database:       task.Database.Name,
		Environment:    task.Database.Instance.Environment.Name,
		Source:         db.MigrationSource(task.Database.Project.WorkflowType),
		Type:           db.Branch,
		Status:         db.Done,
		Description:    fmt.Sprintf("PITR: restoring database %s", task.Database.Name),
		Creator:        task.Creator.Name,
		IssueID:        strconv.Itoa(issue.ID),
	}

	if _, _, err := driver.ExecuteMigration(ctx, m, "/* pitr cutover */"); err != nil {
		log.Error("Failed to add migration history record", zap.Error(err))
		return true, nil, errors.Wrap(err, "failed to add migration history record")
	}

	// TODO(dragonly): Only needed for in-place PITR.
	backupName := fmt.Sprintf("%s-%s-pitr-%d", api.ProjectShortSlug(task.Database.Project), api.EnvSlug(task.Database.Instance.Environment), issue.CreatedTs)
	if _, err := backupRunner.ScheduleBackupTask(ctx, task.Database, backupName, api.BackupTypePITR, api.SystemBotID); err != nil {
		return true, nil, errors.Wrapf(err, "failed to schedule backup task for database %q after PITR", task.Database.Name)
	}

	// Sync database schema after restore is completed.
	if err := schemaSyncer.SyncDatabaseSchema(ctx, task.Database.Instance, task.Database.Name); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", task.Database.Instance.Name),
			zap.String("databaseName", task.Database.Name),
			zap.Error(err),
		)
	}

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Swapped PITR database for target database %q", task.Database.Name),
	}, nil
}

func (exec *PITRCutoverExecutor) doCutover(ctx context.Context, driver db.Driver, task *api.Task, issue *api.Issue) error {
	switch task.Instance.Engine {
	case db.Postgres:
		// Retry so that if there are clients reconnecting to the related databases, we can potentially kill the connections and do the cutover successfully.
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		maxRetry := 3
		retry := 0
		for {
			select {
			case <-ticker.C:
				retry++
				if err := exec.pitrCutoverPostgres(ctx, driver, task, issue); err != nil {
					if retry == maxRetry {
						return errors.Wrapf(err, "failed to do cutover for PostgreSQL after retried for %d times", maxRetry)
					}
					log.Debug("Failed to do cutover for PostgreSQL. Retry later.", zap.Error(err))
				} else {
					return nil
				}
			case <-ctx.Done():
				return errors.Errorf("context is canceled when doing cutover for PostgreSQL")
			}
		}
	case db.MySQL:
		if err := exec.pitrCutoverMySQL(ctx, driver, task, issue); err != nil {
			return errors.Wrap(err, "failed to do cutover for MySQL")
		}
		return nil
	default:
		return errors.Errorf("invalid database type %q for cutover task", task.Instance.Engine)
	}
}

func (*PITRCutoverExecutor) pitrCutoverMySQL(ctx context.Context, driver db.Driver, task *api.Task, issue *api.Issue) error {
	driverDB, err := driver.GetDBConnection(ctx, "")
	if err != nil {
		return err
	}
	conn, err := driverDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Debug("Swapping the original and PITR database", zap.String("originalDatabase", task.Database.Name))
	pitrDatabaseName, pitrOldDatabaseName, err := mysql.SwapPITRDatabase(ctx, conn, task.Database.Name, issue.CreatedTs)
	if err != nil {
		log.Error("Failed to swap the original and PITR database", zap.String("originalDatabase", task.Database.Name), zap.String("pitrDatabase", pitrDatabaseName), zap.Error(err))
		return errors.Wrap(err, "failed to swap the original and PITR database")
	}
	log.Debug("Finished swapping the original and PITR database", zap.String("originalDatabase", task.Database.Name), zap.String("pitrDatabase", pitrDatabaseName), zap.String("oldDatabase", pitrOldDatabaseName))
	return nil
}

func (*PITRCutoverExecutor) pitrCutoverPostgres(ctx context.Context, driver db.Driver, task *api.Task, issue *api.Issue) error {
	pitrDatabaseName := util.GetPITRDatabaseName(task.Database.Name, issue.CreatedTs)
	pitrOldDatabaseName := util.GetPITROldDatabaseName(task.Database.Name, issue.CreatedTs)
	db, err := driver.GetDBConnection(ctx, db.BytebaseDatabase)
	if err != nil {
		return errors.Wrap(err, "failed to get connection for PostgreSQL")
	}

	// The original database may not exist.
	// This is possible if there's a former task execution which successfully renamed original -> _del database and failed.
	// Now we have the _del and the _pitr database.
	existOriginal, err := pgDatabaseExist(ctx, db, task.Database.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to check existence of database %q", task.Database.Name)
	}
	if existOriginal {
		// Kill connections to the original database in the beginning of cutover.
		if err := pgKillConnectionsToDatabase(ctx, db, task.Database.Name); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s RENAME TO %s;", task.Database.Name, pitrOldDatabaseName)); err != nil {
			return errors.Wrapf(err, "failed to rename database %q to %q", task.Database.Name, pitrOldDatabaseName)
		}
		log.Debug("Successfully renamed database", zap.String("from", task.Database.Name), zap.String("to", pitrOldDatabaseName))
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
		if err := pgKillConnectionsToDatabase(ctx, db, task.Database.Name); err != nil {
			return err
		}
		// Kill connection to the _pitr database in case there's someone connecting to all of the existing databases like postgres-exporter.
		if err := pgKillConnectionsToDatabase(ctx, db, pitrDatabaseName); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s RENAME TO %s;", pitrDatabaseName, task.Database.Name)); err != nil {
			return errors.Wrapf(err, "failed to rename database %q to %q", pitrDatabaseName, task.Database.Name)
		}
		log.Debug("Successfully renamed database", zap.String("from", pitrDatabaseName), zap.String("to", task.Database.Name))
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
