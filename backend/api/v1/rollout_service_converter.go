package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/state"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	// emptyStageID is the placeholder used for stages without environment or with deleted environments.
	emptyStageID = "-"
)

// formatStageIDFromEnvironment returns the stage ID, using emptyStageID placeholder if environment is empty.
func formatStageIDFromEnvironment(environment string) string {
	if environment == "" {
		return emptyStageID
	}
	return environment
}

// formatEnvironmentFromStageID converts stage ID back to environment, handling the emptyStageID placeholder.
func formatEnvironmentFromStageID(stageID string) string {
	if stageID == emptyStageID {
		return ""
	}
	return stageID
}

func convertToTaskRuns(ctx context.Context, s *store.Store, stateCfg *state.State, taskRuns []*store.TaskRunMessage) ([]*v1pb.TaskRun, error) {
	var taskRunsV1 []*v1pb.TaskRun
	for _, taskRun := range taskRuns {
		taskRunV1, err := convertToTaskRun(ctx, s, stateCfg, taskRun)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert task run")
		}
		taskRunsV1 = append(taskRunsV1, taskRunV1)
	}
	return taskRunsV1, nil
}

func convertToTaskRun(ctx context.Context, s *store.Store, stateCfg *state.State, taskRun *store.TaskRunMessage) (*v1pb.TaskRun, error) {
	stageID := formatStageIDFromEnvironment(taskRun.Environment)
	t := &v1pb.TaskRun{
		Name:          common.FormatTaskRun(taskRun.ProjectID, taskRun.PipelineUID, stageID, taskRun.TaskUID, taskRun.ID),
		Creator:       common.FormatUserEmail(taskRun.Creator.Email),
		CreateTime:    timestamppb.New(taskRun.CreatedAt),
		UpdateTime:    timestamppb.New(taskRun.UpdatedAt),
		Status:        convertToTaskRunStatus(taskRun.Status),
		Detail:        taskRun.ResultProto.Detail,
		Changelog:     taskRun.ResultProto.Changelog,
		SchemaVersion: taskRun.ResultProto.Version,
		Sheet:         "",
	}
	if taskRun.StartedAt != nil {
		t.StartTime = timestamppb.New(*taskRun.StartedAt)
	}
	if taskRun.RunAt != nil {
		t.RunTime = timestamppb.New(*taskRun.RunAt)
	}

	if taskRun.SheetUID != nil && *taskRun.SheetUID != 0 {
		sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{UID: taskRun.SheetUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet")
		}
		if sheet == nil {
			return nil, errors.Errorf("sheet not found for uid %d", *taskRun.SheetUID)
		}
		t.Sheet = common.FormatSheet(taskRun.ProjectID, sheet.UID)
	}

	if v, ok := stateCfg.TaskRunSchedulerInfo.Load(taskRun.ID); ok {
		if info, ok := v.(*storepb.SchedulerInfo); ok {
			si, err := convertToSchedulerInfo(ctx, s, info)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to convert to scheduler info")
			}
			t.SchedulerInfo = si
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

	if taskRun.ResultProto.PriorBackupDetail != nil {
		t.PriorBackupDetail = convertToTaskRunPriorBackupDetail(taskRun.ResultProto.PriorBackupDetail)
	}

	return t, nil
}

func convertToSchedulerInfo(ctx context.Context, s *store.Store, si *storepb.SchedulerInfo) (*v1pb.TaskRun_SchedulerInfo, error) {
	if si == nil {
		return nil, nil
	}

	cause, err := convertToSchedulerInfoWaitingCause(ctx, s, si.WaitingCause)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert to scheduler info waiting cause")
	}

	return &v1pb.TaskRun_SchedulerInfo{
		ReportTime:   si.ReportTime,
		WaitingCause: cause,
	}, nil
}

func convertToSchedulerInfoWaitingCause(ctx context.Context, s *store.Store, c *storepb.SchedulerInfo_WaitingCause) (*v1pb.TaskRun_SchedulerInfo_WaitingCause, error) {
	if c == nil {
		return nil, nil
	}
	switch cause := c.Cause.(type) {
	case *storepb.SchedulerInfo_WaitingCause_ConnectionLimit:
		return &v1pb.TaskRun_SchedulerInfo_WaitingCause{
			Cause: &v1pb.TaskRun_SchedulerInfo_WaitingCause_ConnectionLimit{
				ConnectionLimit: cause.ConnectionLimit,
			},
		}, nil
	case *storepb.SchedulerInfo_WaitingCause_TaskUid:
		taskUID := cause.TaskUid
		task, err := s.GetTaskV2ByID(ctx, int(taskUID))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get task %v", taskUID)
		}
		if task == nil {
			return nil, errors.Errorf("task %v not found", taskUID)
		}
		pipeline, err := s.GetPipelineV2ByID(ctx, task.PipelineID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get pipeline %v", task.PipelineID)
		}
		if pipeline == nil {
			return nil, errors.Errorf("pipeline %d not found", task.PipelineID)
		}
		issue, err := s.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get issue by pipeline %v", task.PipelineID)
		}
		var issueName string
		if issue != nil {
			issueName = common.FormatIssue(issue.Project.ResourceID, issue.UID)
		}
		stageID := formatStageIDFromEnvironment(task.Environment)
		return &v1pb.TaskRun_SchedulerInfo_WaitingCause{
			Cause: &v1pb.TaskRun_SchedulerInfo_WaitingCause_Task_{
				Task: &v1pb.TaskRun_SchedulerInfo_WaitingCause_Task{
					Task:  common.FormatTask(pipeline.ProjectID, task.PipelineID, stageID, task.ID),
					Issue: issueName,
				},
			},
		}, nil
	case *storepb.SchedulerInfo_WaitingCause_ParallelTasksLimit:
		return &v1pb.TaskRun_SchedulerInfo_WaitingCause{
			Cause: &v1pb.TaskRun_SchedulerInfo_WaitingCause_ParallelTasksLimit{
				ParallelTasksLimit: cause.ParallelTasksLimit,
			},
		}, nil
	default:
		return nil, nil
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

func convertToTaskRunPriorBackupDetail(priorBackupDetail *storepb.PriorBackupDetail) *v1pb.TaskRun_PriorBackupDetail {
	if priorBackupDetail == nil {
		return nil
	}
	convertTable := func(table *storepb.PriorBackupDetail_Item_Table) *v1pb.TaskRun_PriorBackupDetail_Item_Table {
		return &v1pb.TaskRun_PriorBackupDetail_Item_Table{
			Database: table.Database,
			Schema:   table.Schema,
			Table:    table.Table,
		}
	}

	items := []*v1pb.TaskRun_PriorBackupDetail_Item{}
	for _, item := range priorBackupDetail.Items {
		items = append(items, &v1pb.TaskRun_PriorBackupDetail_Item{
			SourceTable:   convertTable(item.SourceTable),
			TargetTable:   convertTable(item.TargetTable),
			StartPosition: convertToPosition(item.StartPosition),
			EndPosition:   convertToPosition(item.EndPosition),
		})
	}
	return &v1pb.TaskRun_PriorBackupDetail{
		Items: items,
	}
}

func convertToRollout(ctx context.Context, s *store.Store, project *store.ProjectMessage, rollout *store.PipelineMessage) (*v1pb.Rollout, error) {
	rolloutV1 := &v1pb.Rollout{
		Name:       common.FormatRollout(project.ResourceID, rollout.ID),
		Plan:       "",
		Title:      "",
		Stages:     nil,
		CreateTime: timestamppb.New(rollout.CreatedAt),
		UpdateTime: timestamppb.New(rollout.UpdatedAt),
	}

	creator, err := s.GetUserByID(ctx, rollout.CreatorUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rollout creator")
	}
	// For preview rollout, creator could be nil.
	if creator != nil {
		rolloutV1.Creator = common.FormatUserEmail(creator.Email)
	}

	plan, err := s.GetPlan(ctx, &store.FindPlanMessage{PipelineID: &rollout.ID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan")
	}
	if plan != nil {
		rolloutV1.Plan = common.FormatPlan(project.ResourceID, plan.UID)
		rolloutV1.Title = plan.Name
	}

	if rollout.IssueID != nil {
		rolloutV1.Issue = common.FormatIssue(project.ResourceID, *rollout.IssueID)
	}

	// Get environment order from plan deployment config or global settings
	var environmentOrder []string
	if plan != nil && len(plan.Config.GetDeployment().GetEnvironments()) > 0 {
		environmentOrder = plan.Config.Deployment.GetEnvironments()
	} else {
		// Use global environment setting order
		var err error
		environmentOrder, err = getAllEnvironmentIDs(ctx, s)
		if err != nil {
			return nil, err
		}
	}

	// Group tasks by environment.
	tasksByEnv := map[string][]*v1pb.Task{}
	for _, task := range rollout.Tasks {
		rolloutTask, err := convertToTask(ctx, s, project, task)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert task"))
		}

		tasksByEnv[task.Environment] = append(tasksByEnv[task.Environment], rolloutTask)
	}

	var stages []*v1pb.Stage
	for _, environment := range environmentOrder {
		tasks := tasksByEnv[environment]
		if len(tasks) > 0 {
			stageID := formatStageIDFromEnvironment(environment)
			stages = append(stages, &v1pb.Stage{
				Name:        common.FormatStage(project.ResourceID, rollout.ID, stageID),
				Id:          stageID,
				Environment: common.FormatEnvironment(stageID),
				Tasks:       tasks,
			})
		}
		delete(tasksByEnv, environment)
	}

	for environment, tasks := range tasksByEnv {
		if len(tasks) > 0 {
			stageID := formatStageIDFromEnvironment(environment)
			stages = append([]*v1pb.Stage{
				{
					Name:        common.FormatStage(project.ResourceID, rollout.ID, stageID),
					Id:          stageID,
					Environment: common.FormatEnvironment(stageID),
					Tasks:       tasks,
				},
			}, stages...)
		}
	}

	rolloutV1.Stages = stages
	return rolloutV1, nil
}

func convertToTask(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	//exhaustive:enforce
	switch task.Type {
	case storepb.Task_DATABASE_CREATE:
		return convertToTaskFromDatabaseCreate(ctx, s, project, task)
	case storepb.Task_DATABASE_MIGRATE:
		// Handle DATABASE_MIGRATE based on migrate_type
		switch task.Payload.GetMigrateType() {
		case storepb.MigrationType_DDL, storepb.MigrationType_GHOST, storepb.MigrationType_MIGRATION_TYPE_UNSPECIFIED:
			return convertToTaskFromSchemaUpdate(ctx, s, project, task)
		case storepb.MigrationType_DML:
			return convertToTaskFromDataUpdate(ctx, s, project, task)
		default:
			return nil, errors.Errorf("unsupported migrate type %v", task.Payload.GetMigrateType())
		}
	case storepb.Task_DATABASE_SDL:
		return convertToTaskFromSchemaUpdate(ctx, s, project, task)
	case storepb.Task_DATABASE_EXPORT:
		return convertToTaskFromDatabaseDataExport(ctx, s, project, task)
	case storepb.Task_TASK_TYPE_UNSPECIFIED:
		return nil, errors.Errorf("task type %v is not supported", task.Type)
	default:
		return nil, errors.Errorf("task type %v is not supported", task.Type)
	}
}

func convertToTaskFromDatabaseCreate(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &task.InstanceID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", task.InstanceID)
	}
	stageID := formatStageIDFromEnvironment(task.Environment)
	v1pbTask := &v1pb.Task{
		Name:          common.FormatTask(project.ResourceID, task.PipelineID, stageID, task.ID),
		SpecId:        task.Payload.GetSpecId(),
		Type:          convertToTaskType(task),
		Status:        convertToTaskStatus(task.LatestTaskRunStatus, task.Payload.GetSkipped()),
		SkippedReason: task.Payload.GetSkippedReason(),
		Target:        common.FormatInstance(instance.ResourceID),
		Payload: &v1pb.Task_DatabaseCreate_{
			DatabaseCreate: &v1pb.Task_DatabaseCreate{
				Project:      "",
				Database:     task.Payload.GetDatabaseName(),
				Table:        task.Payload.GetTableName(),
				Sheet:        common.FormatSheet(project.ResourceID, int(task.Payload.GetSheetId())),
				CharacterSet: task.Payload.GetCharacterSet(),
				Collation:    task.Payload.GetCollation(),
				Environment:  common.FormatEnvironment(task.Payload.GetEnvironmentId()),
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

func convertToTaskFromSchemaUpdate(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseName == nil {
		return nil, errors.Errorf("schema update task database is nil")
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName, ShowDeleted: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}

	// Determine DatabaseChangeType and MigrationType based on task type
	var databaseChangeType v1pb.DatabaseChangeType
	var migrationType v1pb.MigrationType
	switch task.Type {
	case storepb.Task_DATABASE_MIGRATE:
		databaseChangeType = v1pb.DatabaseChangeType_MIGRATE
		switch task.Payload.GetMigrateType() {
		case storepb.MigrationType_DDL:
			migrationType = v1pb.MigrationType_DDL
		case storepb.MigrationType_GHOST:
			migrationType = v1pb.MigrationType_GHOST
		default:
			migrationType = v1pb.MigrationType_DDL
		}
	case storepb.Task_DATABASE_SDL:
		databaseChangeType = v1pb.DatabaseChangeType_SDL
	default:
		databaseChangeType = v1pb.DatabaseChangeType_DATABASE_CHANGE_TYPE_UNSPECIFIED
	}

	stageID := formatStageIDFromEnvironment(task.Environment)
	v1pbTask := &v1pb.Task{
		Name:          common.FormatTask(project.ResourceID, task.PipelineID, stageID, task.ID),
		SpecId:        task.Payload.GetSpecId(),
		Type:          convertToTaskType(task),
		Status:        convertToTaskStatus(task.LatestTaskRunStatus, task.Payload.GetSkipped()),
		SkippedReason: task.Payload.GetSkippedReason(),
		Target:        fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload: &v1pb.Task_DatabaseUpdate_{
			DatabaseUpdate: &v1pb.Task_DatabaseUpdate{
				Sheet:              common.FormatSheet(project.ResourceID, int(task.Payload.GetSheetId())),
				SchemaVersion:      task.Payload.GetSchemaVersion(),
				DatabaseChangeType: databaseChangeType,
				MigrationType:      migrationType,
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

func convertToTaskFromDataUpdate(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseName == nil {
		return nil, errors.Errorf("data update task database is nil")
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName, ShowDeleted: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}

	stageID := formatStageIDFromEnvironment(task.Environment)
	v1pbTask := &v1pb.Task{
		Name:          common.FormatTask(project.ResourceID, task.PipelineID, stageID, task.ID),
		SpecId:        task.Payload.GetSpecId(),
		Type:          convertToTaskType(task),
		Status:        convertToTaskStatus(task.LatestTaskRunStatus, task.Payload.GetSkipped()),
		SkippedReason: task.Payload.GetSkippedReason(),
		Target:        fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload:       nil,
	}
	v1pbTaskPayload := &v1pb.Task_DatabaseUpdate_{
		DatabaseUpdate: &v1pb.Task_DatabaseUpdate{
			Sheet:              common.FormatSheet(project.ResourceID, int(task.Payload.GetSheetId())),
			SchemaVersion:      task.Payload.GetSchemaVersion(),
			DatabaseChangeType: v1pb.DatabaseChangeType_MIGRATE,
			MigrationType:      v1pb.MigrationType_DML,
		},
	}

	v1pbTask.Payload = v1pbTaskPayload
	if task.UpdatedAt != nil {
		v1pbTask.UpdateTime = timestamppb.New(*task.UpdatedAt)
	}
	if task.RunAt != nil {
		v1pbTask.RunTime = timestamppb.New(*task.RunAt)
	}
	return v1pbTask, nil
}

func convertToTaskFromDatabaseDataExport(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseName == nil {
		return nil, errors.Errorf("data export task database is nil")
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName, ShowDeleted: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}
	targetDatabaseName := fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName)
	sheet := common.FormatSheet(project.ResourceID, int(task.Payload.GetSheetId()))
	v1pbTaskPayload := v1pb.Task_DatabaseDataExport_{
		DatabaseDataExport: &v1pb.Task_DatabaseDataExport{
			Target:   targetDatabaseName,
			Sheet:    sheet,
			Format:   convertExportFormat(task.Payload.GetFormat()),
			Password: &task.Payload.Password,
		},
	}
	stageID := formatStageIDFromEnvironment(task.Environment)
	v1pbTask := &v1pb.Task{
		Name:    common.FormatTask(project.ResourceID, task.PipelineID, stageID, task.ID),
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
	case storepb.Task_DATABASE_SDL:
		return v1pb.Task_DATABASE_SDL
	case storepb.Task_DATABASE_EXPORT:
		return v1pb.Task_DATABASE_EXPORT
	case storepb.Task_TASK_TYPE_UNSPECIFIED:
		return v1pb.Task_TYPE_UNSPECIFIED
	default:
		return v1pb.Task_TYPE_UNSPECIFIED
	}
}

// getMigrateTypeFromMigrationType converts v1pb.MigrationType to storepb.MigrationType
func getMigrateTypeFromMigrationType(migrationType v1pb.MigrationType) storepb.MigrationType {
	switch migrationType {
	case v1pb.MigrationType_DDL:
		return storepb.MigrationType_DDL
	case v1pb.MigrationType_GHOST:
		return storepb.MigrationType_GHOST
	case v1pb.MigrationType_DML:
		return storepb.MigrationType_DML
	default:
		return storepb.MigrationType_MIGRATION_TYPE_UNSPECIFIED
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
					LogTime:        timestamppb.New(l.T),
					CommandIndexes: l.Payload.CommandExecute.CommandIndexes,
					Statement:      l.Payload.CommandExecute.Statement,
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

		case storepb.TaskRunLog_TASK_RUN_STATUS_UPDATE:
			e := &v1pb.TaskRunLogEntry{
				Type:     v1pb.TaskRunLogEntry_TASK_RUN_STATUS_UPDATE,
				LogTime:  timestamppb.New(l.T),
				DeployId: l.Payload.DeployId,
				TaskRunStatusUpdate: &v1pb.TaskRunLogEntry_TaskRunStatusUpdate{
					Status: convertTaskRunLogTaskRunStatus(l.Payload.TaskRunStatusUpdate.Status),
				},
			}
			entries = append(entries, e)

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
			prev.PriorBackup.PriorBackupDetail = convertToTaskRunPriorBackupDetail(l.Payload.PriorBackupEnd.PriorBackupDetail)

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
		default:
		}
	}

	return entries
}

func convertTaskRunLogTaskRunStatus(s storepb.TaskRunLog_TaskRunStatusUpdate_Status) v1pb.TaskRunLogEntry_TaskRunStatusUpdate_Status {
	switch s {
	case storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_WAITING:
		return v1pb.TaskRunLogEntry_TaskRunStatusUpdate_RUNNING_WAITING
	case storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_RUNNING:
		return v1pb.TaskRunLogEntry_TaskRunStatusUpdate_RUNNING_RUNNING
	default:
		return v1pb.TaskRunLogEntry_TaskRunStatusUpdate_STATUS_UNSPECIFIED
	}
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
