package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"go.uber.org/zap"
)

// NewPITRCutoverTaskExecutor creates a PITR cutover task executor.
func NewPITRCutoverTaskExecutor(instance mysqlutil.Instance) TaskExecutor {
	return &PITRCutoverTaskExecutor{
		mysqlutil: instance,
	}
}

// PITRCutoverTaskExecutor is the PITR cutover task executor.
type PITRCutoverTaskExecutor struct {
	mysqlutil mysqlutil.Instance
}

// RunOnce will run the PITR cutover task executor once.
func (exec *PITRCutoverTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run PITR cutover task", zap.String("task", task.Name))

	issue, err := getIssueByPipelineID(ctx, server.store, task.PipelineID)
	if err != nil {
		log.Error("failed to fetch containing issue doing pitr cutover task", zap.Error(err))
		return true, nil, err
	}

	// Currently api.TaskDatabasePITRCutoverPayload is empty, so we do not need to unmarshal from task.Payload.

	terminated, result, err = exec.pitrCutover(ctx, task, server, issue)
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
	activityMeta := ActivityMeta{}
	activityMeta.issue = issue
	if _, err = server.ActivityManager.CreateActivity(ctx, activityCreate, &activityMeta); err != nil {
		log.Error("cannot create an pitr activity", zap.Error(err))
	}

	return terminated, result, nil
}

// pitrCutover performs the PITR cutover algorithm:
// 1. Swap the current and PITR database.
// 2. Create a backup with type PITR. The backup is scheduled asynchronously.
// We must check the possible failed/ongoing PITR type backup in the recovery process.
func (exec *PITRCutoverTaskExecutor) pitrCutover(ctx context.Context, task *api.Task, server *Server, issue *api.Issue) (terminated bool, result *api.TaskRunResultPayload, err error) {
	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", "" /* pgInstanceDir */)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	driverDB, err := driver.GetDbConnection(ctx, "")
	if err != nil {
		return true, nil, err
	}
	conn, err := driverDB.Conn(ctx)
	if err != nil {
		return true, nil, err
	}
	defer conn.Close()

	log.Debug("Swapping the original and PITR database", zap.String("originalDatabase", task.Database.Name))
	pitrDatabaseName, pitrOldDatabaseName, err := mysql.SwapPITRDatabase(ctx, conn, task.Database.Name, issue.CreatedTs)
	if err != nil {
		log.Error("Failed to swap the original and PITR database", zap.String("originalDatabase", task.Database.Name), zap.String("pitrDatabase", pitrDatabaseName), zap.Error(err))
		return true, nil, fmt.Errorf("failed to swap the original and PITR database, error: %w", err)
	}
	log.Debug("Finished swapping the original and PITR database", zap.String("originalDatabase", task.Database.Name), zap.String("pitrDatabase", pitrDatabaseName), zap.String("oldDatabase", pitrOldDatabaseName))

	if server.profile.Mode == common.ReleaseModeDev {
		backupName := fmt.Sprintf("%s-%s-pitr-%d", api.ProjectShortSlug(task.Database.Project), api.EnvSlug(task.Database.Instance.Environment), issue.CreatedTs)
		if _, err := server.scheduleBackupTask(ctx, task.Database, backupName, api.BackupTypePITR, api.BackupStorageBackendLocal, api.SystemBotID); err != nil {
			return true, nil, fmt.Errorf("failed to schedule backup task for database %q after PITR, error: %w", task.Database.Name, err)
		}
	}

	log.Debug("Appending new migration history record")
	m := &db.MigrationInfo{
		ReleaseVersion: server.profile.Version,
		Version:        common.DefaultMigrationVersion(),
		Namespace:      task.Database.Name,
		Database:       task.Database.Name,
		Environment:    task.Database.Instance.Environment.Name,
		Source:         db.MigrationSource(task.Database.Project.WorkflowType),
		Type:           db.Baseline,
		Status:         db.Done,
		Description:    fmt.Sprintf("PITR: restoring database %s", task.Database.Name),
		Creator:        task.Creator.Name,
		IssueID:        strconv.Itoa(issue.ID),
	}

	if _, _, err := driver.ExecuteMigration(ctx, m, "/* pitr cutover */"); err != nil {
		log.Error("Failed to add migration history record", zap.Error(err))
		return true, nil, fmt.Errorf("failed to add migration history record, error: %w", err)
	}

	// Sync database schema after restore is completed.
	if err := server.syncDatabaseSchema(ctx, task.Database.Instance, task.Database.Name); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instance", task.Database.Instance.Name),
			zap.String("databaseName", task.Database.Name),
		)
	}

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Swapped PITR database for target database %q", task.Database.Name),
	}, nil
}
