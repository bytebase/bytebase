package v1

import (
	"context"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/bus"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// formatEnvironmentFromStageID converts stage ID back to environment, handling the EmptyStageID placeholder.
func formatEnvironmentFromStageID(stageID string) string {
	if stageID == common.EmptyStageID {
		return ""
	}
	return stageID
}

func convertToTaskRuns(ctx context.Context, s *store.Store, bus *bus.Bus, taskRuns []*store.TaskRunMessage) ([]*v1pb.TaskRun, error) {
	var taskRunsV1 []*v1pb.TaskRun
	for _, taskRun := range taskRuns {
		taskRunV1, err := convertToTaskRun(ctx, s, bus, taskRun)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert task run")
		}
		taskRunsV1 = append(taskRunsV1, taskRunV1)
	}
	return taskRunsV1, nil
}

func convertToTaskRun(ctx context.Context, s *store.Store, bus *bus.Bus, taskRun *store.TaskRunMessage) (*v1pb.TaskRun, error) {
	stageID := common.FormatStageID(taskRun.Environment)
	t := &v1pb.TaskRun{
		Name:       common.FormatTaskRun(taskRun.ProjectID, taskRun.PlanUID, stageID, taskRun.TaskUID, taskRun.ID),
		Creator:    common.FormatUserEmail(taskRun.CreatorEmail),
		CreateTime: timestamppb.New(taskRun.CreatedAt),
		UpdateTime: timestamppb.New(taskRun.UpdatedAt),
		Status:     convertToTaskRunStatus(taskRun.Status),
		Detail:     taskRun.ResultProto.Detail,
	}
	if taskRun.StartedAt != nil {
		t.StartTime = timestamppb.New(*taskRun.StartedAt)
	}
	if taskRun.RunAt != nil {
		t.RunTime = timestamppb.New(*taskRun.RunAt)
	}

	if v, ok := bus.TaskRunSchedulerInfo.Load(taskRun.ID); ok {
		if info, ok := v.(*storepb.SchedulerInfo); ok {
			t.SchedulerInfo = convertToSchedulerInfo(info)
		}
	}

	if taskRun.ResultProto.ExportArchiveUid != 0 {
		t.ExportArchiveStatus = v1pb.TaskRun_EXPORTED
		exportArchiveUID := int(taskRun.ResultProto.ExportArchiveUid)
		exportArchive, err := s.GetExportArchive(ctx, &store.FindExportArchiveMessage{UID: &exportArchiveUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get export archive")
		}
		if exportArchive != nil {
			t.ExportArchiveStatus = v1pb.TaskRun_READY
		}
	}

	t.HasPriorBackup = taskRun.ResultProto.HasPriorBackup

	return t, nil
}

func convertToSchedulerInfo(si *storepb.SchedulerInfo) *v1pb.TaskRun_SchedulerInfo {
	if si == nil {
		return nil
	}

	return &v1pb.TaskRun_SchedulerInfo{
		ReportTime:   si.ReportTime,
		WaitingCause: convertToSchedulerInfoWaitingCause(si.WaitingCause),
	}
}

func convertToSchedulerInfoWaitingCause(c *storepb.SchedulerInfo_WaitingCause) *v1pb.TaskRun_SchedulerInfo_WaitingCause {
	if c == nil {
		return nil
	}
	switch cause := c.Cause.(type) {
	case *storepb.SchedulerInfo_WaitingCause_ParallelTasksLimit:
		return &v1pb.TaskRun_SchedulerInfo_WaitingCause{
			Cause: &v1pb.TaskRun_SchedulerInfo_WaitingCause_ParallelTasksLimit{
				ParallelTasksLimit: cause.ParallelTasksLimit,
			},
		}
	default:
		return nil
	}
}

func convertToTaskRunStatus(status storepb.TaskRun_Status) v1pb.TaskRun_Status {
	switch status {
	case storepb.TaskRun_STATUS_UNSPECIFIED:
		return v1pb.TaskRun_STATUS_UNSPECIFIED
	case storepb.TaskRun_PENDING:
		return v1pb.TaskRun_PENDING
	case storepb.TaskRun_RUNNING:
		return v1pb.TaskRun_RUNNING
	case storepb.TaskRun_DONE:
		return v1pb.TaskRun_DONE
	case storepb.TaskRun_FAILED:
		return v1pb.TaskRun_FAILED
	case storepb.TaskRun_CANCELED:
		return v1pb.TaskRun_CANCELED
	default:
		return v1pb.TaskRun_STATUS_UNSPECIFIED
	}
}

func convertToTaskRunLogPriorBackupDetail(priorBackupDetail *storepb.PriorBackupDetail) *v1pb.TaskRunLogEntry_PriorBackup_PriorBackupDetail {
	if priorBackupDetail == nil {
		return nil
	}
	convertTable := func(table *storepb.PriorBackupDetail_Item_Table) *v1pb.TaskRunLogEntry_PriorBackup_PriorBackupDetail_Item_Table {
		return &v1pb.TaskRunLogEntry_PriorBackup_PriorBackupDetail_Item_Table{
			Database: table.Database,
			Schema:   table.Schema,
			Table:    table.Table,
		}
	}

	items := []*v1pb.TaskRunLogEntry_PriorBackup_PriorBackupDetail_Item{}
	for _, item := range priorBackupDetail.Items {
		items = append(items, &v1pb.TaskRunLogEntry_PriorBackup_PriorBackupDetail_Item{
			SourceTable:   convertTable(item.SourceTable),
			TargetTable:   convertTable(item.TargetTable),
			StartPosition: convertToPosition(item.StartPosition),
			EndPosition:   convertToPosition(item.EndPosition),
		})
	}
	return &v1pb.TaskRunLogEntry_PriorBackup_PriorBackupDetail{
		Items: items,
	}
}

func convertToRollout(project *store.ProjectMessage, plan *store.PlanMessage, tasks []*store.TaskMessage, environmentOrderMap map[string]int) (*v1pb.Rollout, error) {
	// Calculate rollout times.
	// CreateTime: When the rollout/plan was created.
	// UpdateTime: Latest task update time (which reflects the latest task run update).
	createTime := plan.CreatedAt
	updateTime := plan.UpdatedAt
	for _, task := range tasks {
		// task.UpdatedAt is the updated_at of latest task run for this task.
		if task.UpdatedAt != nil && task.UpdatedAt.After(updateTime) {
			updateTime = *task.UpdatedAt
		}
	}

	rolloutV1 := &v1pb.Rollout{
		Name:       common.FormatRollout(project.ResourceID, plan.UID),
		Title:      plan.Name,
		Stages:     nil,
		CreateTime: timestamppb.New(createTime),
		UpdateTime: timestamppb.New(updateTime),
	}

	// Group tasks by environment.
	tasksByEnv := make(map[string][]*store.TaskMessage)
	for _, task := range tasks {
		tasksByEnv[task.Environment] = append(tasksByEnv[task.Environment], task)
	}

	// Collect environments that have tasks and are in the environment order map.
	var envs []string
	for env := range tasksByEnv {
		if _, exists := environmentOrderMap[env]; exists {
			envs = append(envs, env)
		}
	}
	// Sort environments by their order.
	slices.SortFunc(envs, func(a, b string) int {
		return environmentOrderMap[a] - environmentOrderMap[b]
	})

	// Create stages for known environments only.
	var stages []*v1pb.Stage
	for _, env := range envs {
		stageID := common.FormatStageID(env)
		envTasks := tasksByEnv[env]
		// Sort tasks by ID within each stage.
		slices.SortFunc(envTasks, func(a, b *store.TaskMessage) int {
			return a.ID - b.ID
		})

		// Convert tasks to v1pb.Task.
		var v1Tasks []*v1pb.Task
		for _, task := range envTasks {
			v1Task, err := convertToTask(project, task)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert task"))
			}
			v1Tasks = append(v1Tasks, v1Task)
		}

		stages = append(stages, &v1pb.Stage{
			Name:        common.FormatStage(project.ResourceID, plan.UID, stageID),
			Id:          stageID,
			Environment: common.FormatEnvironment(stageID),
			Tasks:       v1Tasks,
		})
	}

	rolloutV1.Stages = stages
	return rolloutV1, nil
}

func convertToTask(project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	//exhaustive:enforce
	switch task.Type {
	case storepb.Task_DATABASE_CREATE:
		return convertToTaskFromDatabaseCreate(project, task)
	case storepb.Task_DATABASE_MIGRATE:
		// All DATABASE_MIGRATE tasks are treated as schema updates (DDL or GHOST)
		return convertToTaskFromSchemaUpdate(project, task)
	case storepb.Task_DATABASE_EXPORT:
		return convertToTaskFromDatabaseDataExport(project, task)
	case storepb.Task_TASK_TYPE_UNSPECIFIED:
		return nil, errors.Errorf("task type %v is not supported", task.Type)
	default:
		return nil, errors.Errorf("task type %v is not supported", task.Type)
	}
}

func convertToTaskFromDatabaseCreate(project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	stageID := common.FormatStageID(task.Environment)
	v1pbTask := &v1pb.Task{
		Name:          common.FormatTask(project.ResourceID, task.PlanID, stageID, task.ID),
		SpecId:        task.Payload.GetSpecId(),
		Type:          convertToTaskType(task),
		Status:        convertToTaskStatus(task.LatestTaskRunStatus, task.Payload.GetSkipped()),
		SkippedReason: task.Payload.GetSkippedReason(),
		Target:        common.FormatInstance(task.InstanceID),
		Payload: &v1pb.Task_DatabaseCreate_{
			DatabaseCreate: &v1pb.Task_DatabaseCreate{
				Sheet: common.FormatSheet(project.ResourceID, task.Payload.GetSheetSha256()),
			},
		},
	}
	if task.UpdatedAt != nil {
		v1pbTask.UpdateTime = timestamppb.New(*task.UpdatedAt)
	}
	if task.RunAt != nil {
		v1pbTask.RunTime = timestamppb.New(*task.RunAt)
	}
	return v1pbTask, nil
}

func convertToTaskFromSchemaUpdate(project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseName == nil {
		return nil, errors.Errorf("schema update task database is nil")
	}

	stageID := common.FormatStageID(task.Environment)

	// Build DatabaseUpdate payload
	databaseUpdate := &v1pb.Task_DatabaseUpdate{}

	// Set source: either sheet or release
	if releaseName := task.Payload.GetRelease(); releaseName != "" {
		databaseUpdate.Source = &v1pb.Task_DatabaseUpdate_Release{
			Release: releaseName,
		}
	} else if sheetSha256 := task.Payload.GetSheetSha256(); sheetSha256 != "" {
		databaseUpdate.Source = &v1pb.Task_DatabaseUpdate_Sheet{
			Sheet: common.FormatSheet(project.ResourceID, sheetSha256),
		}
	}

	v1pbTask := &v1pb.Task{
		Name:          common.FormatTask(project.ResourceID, task.PlanID, stageID, task.ID),
		SpecId:        task.Payload.GetSpecId(),
		Type:          convertToTaskType(task),
		Status:        convertToTaskStatus(task.LatestTaskRunStatus, task.Payload.GetSkipped()),
		SkippedReason: task.Payload.GetSkippedReason(),
		Target:        fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, task.InstanceID, common.DatabaseIDPrefix, *(task.DatabaseName)),
		Payload: &v1pb.Task_DatabaseUpdate_{
			DatabaseUpdate: databaseUpdate,
		},
	}
	if task.UpdatedAt != nil {
		v1pbTask.UpdateTime = timestamppb.New(*task.UpdatedAt)
	}
	if task.RunAt != nil {
		v1pbTask.RunTime = timestamppb.New(*task.RunAt)
	}
	return v1pbTask, nil
}

func convertToTaskFromDatabaseDataExport(project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseName == nil {
		return nil, errors.Errorf("data export task database is nil")
	}

	targetDatabaseName := fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, task.InstanceID, common.DatabaseIDPrefix, *(task.DatabaseName))
	sheet := common.FormatSheet(project.ResourceID, task.Payload.GetSheetSha256())
	v1pbTaskPayload := v1pb.Task_DatabaseDataExport_{
		DatabaseDataExport: &v1pb.Task_DatabaseDataExport{
			Sheet: sheet,
		},
	}
	stageID := common.FormatStageID(task.Environment)
	v1pbTask := &v1pb.Task{
		Name:    common.FormatTask(project.ResourceID, task.PlanID, stageID, task.ID),
		SpecId:  task.Payload.GetSpecId(),
		Type:    convertToTaskType(task),
		Status:  convertToTaskStatus(task.LatestTaskRunStatus, false),
		Target:  targetDatabaseName,
		Payload: &v1pbTaskPayload,
	}
	if task.UpdatedAt != nil {
		v1pbTask.UpdateTime = timestamppb.New(*task.UpdatedAt)
	}
	if task.RunAt != nil {
		v1pbTask.RunTime = timestamppb.New(*task.RunAt)
	}
	return v1pbTask, nil
}

func convertToTaskStatus(latestTaskRunStatus storepb.TaskRun_Status, skipped bool) v1pb.Task_Status {
	if skipped {
		return v1pb.Task_SKIPPED
	}
	switch latestTaskRunStatus {
	case storepb.TaskRun_NOT_STARTED:
		return v1pb.Task_NOT_STARTED
	case storepb.TaskRun_PENDING:
		return v1pb.Task_PENDING
	case storepb.TaskRun_RUNNING:
		return v1pb.Task_RUNNING
	case storepb.TaskRun_DONE:
		return v1pb.Task_DONE
	case storepb.TaskRun_FAILED:
		return v1pb.Task_FAILED
	case storepb.TaskRun_CANCELED:
		return v1pb.Task_CANCELED
	default:
		return v1pb.Task_STATUS_UNSPECIFIED
	}
}

func convertToTaskType(task *store.TaskMessage) v1pb.Task_Type {
	//exhaustive:enforce
	switch task.Type {
	case storepb.Task_DATABASE_CREATE:
		return v1pb.Task_DATABASE_CREATE
	case storepb.Task_DATABASE_MIGRATE:
		return v1pb.Task_DATABASE_MIGRATE
	case storepb.Task_DATABASE_EXPORT:
		return v1pb.Task_DATABASE_EXPORT
	case storepb.Task_TASK_TYPE_UNSPECIFIED:
		return v1pb.Task_TYPE_UNSPECIFIED
	default:
		return v1pb.Task_TYPE_UNSPECIFIED
	}
}

func convertToTaskRunLog(parent string, logs []*store.TaskRunLog) *v1pb.TaskRunLog {
	return &v1pb.TaskRunLog{
		Name:    fmt.Sprintf("%s/log", parent),
		Entries: convertToTaskRunLogEntries(logs),
	}
}

func convertToTaskRunLogEntries(logs []*store.TaskRunLog) []*v1pb.TaskRunLogEntry {
	var entries []*v1pb.TaskRunLogEntry
	for _, l := range logs {
		switch l.Payload.Type {
		case storepb.TaskRunLog_SCHEMA_DUMP_START:
			e := &v1pb.TaskRunLogEntry{
				Type:     v1pb.TaskRunLogEntry_SCHEMA_DUMP,
				LogTime:  timestamppb.New(l.T),
				DeployId: l.Payload.DeployId,
				SchemaDump: &v1pb.TaskRunLogEntry_SchemaDump{
					StartTime: timestamppb.New(l.T),
				},
			}
			entries = append(entries, e)

		case storepb.TaskRunLog_SCHEMA_DUMP_END:
			if len(entries) == 0 {
				continue
			}
			prev := entries[len(entries)-1]
			if prev == nil || prev.Type != v1pb.TaskRunLogEntry_SCHEMA_DUMP {
				continue
			}
			prev.SchemaDump.EndTime = timestamppb.New(l.T)
			prev.SchemaDump.Error = l.Payload.SchemaDumpEnd.Error

		case storepb.TaskRunLog_COMMAND_EXECUTE:
			e := &v1pb.TaskRunLogEntry{
				Type:     v1pb.TaskRunLogEntry_COMMAND_EXECUTE,
				LogTime:  timestamppb.New(l.T),
				DeployId: l.Payload.DeployId,
				CommandExecute: &v1pb.TaskRunLogEntry_CommandExecute{
					LogTime:   timestamppb.New(l.T),
					Range:     convertToRange(l.Payload.CommandExecute.Range),
					Statement: l.Payload.CommandExecute.Statement,
				},
			}
			entries = append(entries, e)

		case storepb.TaskRunLog_COMMAND_RESPONSE:
			if len(entries) == 0 {
				continue
			}
			prev := entries[len(entries)-1]
			if prev == nil || prev.Type != v1pb.TaskRunLogEntry_COMMAND_EXECUTE {
				continue
			}
			prev.CommandExecute.Response = &v1pb.TaskRunLogEntry_CommandExecute_CommandResponse{
				LogTime:         timestamppb.New(l.T),
				Error:           l.Payload.CommandResponse.Error,
				AffectedRows:    l.Payload.CommandResponse.AffectedRows,
				AllAffectedRows: l.Payload.CommandResponse.AllAffectedRows,
			}

		case storepb.TaskRunLog_DATABASE_SYNC_START:
			e := &v1pb.TaskRunLogEntry{
				Type:     v1pb.TaskRunLogEntry_DATABASE_SYNC,
				LogTime:  timestamppb.New(l.T),
				DeployId: l.Payload.DeployId,
				DatabaseSync: &v1pb.TaskRunLogEntry_DatabaseSync{
					StartTime: timestamppb.New(l.T),
				},
			}
			entries = append(entries, e)

		case storepb.TaskRunLog_DATABASE_SYNC_END:
			if len(entries) == 0 {
				continue
			}
			prev := entries[len(entries)-1]
			if prev == nil || prev.Type != v1pb.TaskRunLogEntry_DATABASE_SYNC {
				continue
			}
			prev.DatabaseSync.EndTime = timestamppb.New(l.T)
			prev.DatabaseSync.Error = l.Payload.DatabaseSyncEnd.Error

		case storepb.TaskRunLog_TRANSACTION_CONTROL:
			e := &v1pb.TaskRunLogEntry{
				Type:     v1pb.TaskRunLogEntry_TRANSACTION_CONTROL,
				LogTime:  timestamppb.New(l.T),
				DeployId: l.Payload.DeployId,
				TransactionControl: &v1pb.TaskRunLogEntry_TransactionControl{
					Type:  convertTaskRunLogTransactionControlType(l.Payload.TransactionControl.Type),
					Error: l.Payload.TransactionControl.Error,
				},
			}
			entries = append(entries, e)

		case storepb.TaskRunLog_PRIOR_BACKUP_START:
			e := &v1pb.TaskRunLogEntry{
				Type:     v1pb.TaskRunLogEntry_PRIOR_BACKUP,
				LogTime:  timestamppb.New(l.T),
				DeployId: l.Payload.DeployId,
				PriorBackup: &v1pb.TaskRunLogEntry_PriorBackup{
					StartTime: timestamppb.New(l.T),
				},
			}
			entries = append(entries, e)

		case storepb.TaskRunLog_PRIOR_BACKUP_END:
			if len(entries) == 0 {
				continue
			}
			prev := entries[len(entries)-1]
			if prev == nil || prev.Type != v1pb.TaskRunLogEntry_PRIOR_BACKUP {
				continue
			}
			prev.PriorBackup.EndTime = timestamppb.New(l.T)
			prev.PriorBackup.Error = l.Payload.PriorBackupEnd.Error
			prev.PriorBackup.PriorBackupDetail = convertToTaskRunLogPriorBackupDetail(l.Payload.PriorBackupEnd.PriorBackupDetail)

		case storepb.TaskRunLog_COMPUTE_DIFF_START:
			e := &v1pb.TaskRunLogEntry{
				Type:     v1pb.TaskRunLogEntry_COMPUTE_DIFF,
				LogTime:  timestamppb.New(l.T),
				DeployId: l.Payload.DeployId,
				ComputeDiff: &v1pb.TaskRunLogEntry_ComputeDiff{
					StartTime: timestamppb.New(l.T),
				},
			}
			entries = append(entries, e)

		case storepb.TaskRunLog_COMPUTE_DIFF_END:
			if len(entries) == 0 {
				continue
			}
			prev := entries[len(entries)-1]
			if prev == nil || prev.Type != v1pb.TaskRunLogEntry_COMPUTE_DIFF {
				continue
			}
			prev.ComputeDiff.EndTime = timestamppb.New(l.T)
			prev.ComputeDiff.Error = l.Payload.ComputeDiffEnd.Error

		case storepb.TaskRunLog_RETRY_INFO:
			e := &v1pb.TaskRunLogEntry{
				Type:     v1pb.TaskRunLogEntry_RETRY_INFO,
				LogTime:  timestamppb.New(l.T),
				DeployId: l.Payload.DeployId,
				RetryInfo: &v1pb.TaskRunLogEntry_RetryInfo{
					RetryCount:     l.Payload.RetryInfo.RetryCount,
					MaximumRetries: l.Payload.RetryInfo.MaximumRetries,
					Error:          l.Payload.RetryInfo.Error,
				},
			}
			entries = append(entries, e)

		case storepb.TaskRunLog_RELEASE_FILE_EXECUTE:
			e := &v1pb.TaskRunLogEntry{
				Type:     v1pb.TaskRunLogEntry_RELEASE_FILE_EXECUTE,
				LogTime:  timestamppb.New(l.T),
				DeployId: l.Payload.DeployId,
				ReleaseFileExecute: &v1pb.TaskRunLogEntry_ReleaseFileExecute{
					Version:  l.Payload.ReleaseFileExecute.Version,
					FilePath: l.Payload.ReleaseFileExecute.FilePath,
				},
			}
			entries = append(entries, e)
		default:
		}
	}

	return entries
}

func convertTaskRunLogTransactionControlType(t storepb.TaskRunLog_TransactionControl_Type) v1pb.TaskRunLogEntry_TransactionControl_Type {
	switch t {
	case storepb.TaskRunLog_TransactionControl_BEGIN:
		return v1pb.TaskRunLogEntry_TransactionControl_BEGIN
	case storepb.TaskRunLog_TransactionControl_COMMIT:
		return v1pb.TaskRunLogEntry_TransactionControl_COMMIT
	case storepb.TaskRunLog_TransactionControl_ROLLBACK:
		return v1pb.TaskRunLogEntry_TransactionControl_ROLLBACK
	default:
		return v1pb.TaskRunLogEntry_TransactionControl_TYPE_UNSPECIFIED
	}
}

func convertToRange(r *storepb.Range) *v1pb.Range {
	if r == nil {
		return nil
	}
	return &v1pb.Range{
		Start: r.Start,
		End:   r.End,
	}
}
