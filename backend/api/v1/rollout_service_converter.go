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

func convertToPlans(ctx context.Context, s *store.Store, plans []*store.PlanMessage) ([]*v1pb.Plan, error) {
	v1Plans := make([]*v1pb.Plan, len(plans))
	for i := range plans {
		p, err := convertToPlan(ctx, s, plans[i])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert plan")
		}
		v1Plans[i] = p
	}
	return v1Plans, nil
}

func convertToPlan(ctx context.Context, s *store.Store, plan *store.PlanMessage) (*v1pb.Plan, error) {
	p := &v1pb.Plan{
		Name:                    common.FormatPlan(plan.ProjectID, plan.UID),
		Title:                   plan.Name,
		Description:             plan.Description,
		Specs:                   convertToPlanSpecs(plan.Config.Specs), // Use specs field for output
		Deployment:              convertToPlanDeployment(plan.Config.Deployment),
		CreateTime:              timestamppb.New(plan.CreatedAt),
		UpdateTime:              timestamppb.New(plan.UpdatedAt),
		State:                   convertDeletedToState(plan.Deleted),
		PlanCheckRunStatusCount: map[string]int32{},
	}

	creator, err := s.GetUserByID(ctx, plan.CreatorUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan creator")
	}
	p.Creator = common.FormatUserEmail(creator.Email)

	issue, err := s.GetIssueV2(ctx, &store.FindIssueMessage{PlanUID: &plan.UID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get issue by plan uid %d", plan.UID)
	}
	if issue != nil {
		p.Issue = common.FormatIssue(issue.Project.ResourceID, issue.UID)
	}
	if plan.PipelineUID != nil {
		p.Rollout = common.FormatRollout(plan.ProjectID, *plan.PipelineUID)
	}
	planCheckRuns, err := s.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		PlanUID: &plan.UID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list plan check runs for plan uid %d", plan.UID)
	}
	for _, run := range planCheckRuns {
		p.PlanCheckRunStatusCount[string(run.Status)]++
		for _, result := range run.Result.Results {
			p.PlanCheckRunStatusCount[storepb.PlanCheckRunResult_Result_Status_name[int32(result.Status)]]++
		}
	}
	return p, nil
}

func convertToPlanSpecs(specs []*storepb.PlanConfig_Spec) []*v1pb.Plan_Spec {
	v1Specs := make([]*v1pb.Plan_Spec, len(specs))
	for i := range specs {
		v1Specs[i] = convertToPlanSpec(specs[i])
	}
	return v1Specs
}

func convertToPlanSpec(spec *storepb.PlanConfig_Spec) *v1pb.Plan_Spec {
	v1Spec := &v1pb.Plan_Spec{
		Id: spec.Id,
	}

	switch v := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		v1Spec.Config = convertToPlanSpecCreateDatabaseConfig(v)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		v1Spec.Config = convertToPlanSpecChangeDatabaseConfig(v)
	case *storepb.PlanConfig_Spec_ExportDataConfig:
		v1Spec.Config = convertToPlanSpecExportDataConfig(v)
	}

	return v1Spec
}

func convertToPlanSpecCreateDatabaseConfig(config *storepb.PlanConfig_Spec_CreateDatabaseConfig) *v1pb.Plan_Spec_CreateDatabaseConfig {
	c := config.CreateDatabaseConfig
	return &v1pb.Plan_Spec_CreateDatabaseConfig{
		CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
			Target:       c.Target,
			Database:     c.Database,
			Table:        c.Table,
			CharacterSet: c.CharacterSet,
			Collation:    c.Collation,
			Cluster:      c.Cluster,
			Owner:        c.Owner,
			Environment:  c.Environment,
		},
	}
}

func convertToPlanSpecChangeDatabaseConfig(config *storepb.PlanConfig_Spec_ChangeDatabaseConfig) *v1pb.Plan_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig
	return &v1pb.Plan_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
			Targets:           c.Targets,
			Sheet:             c.Sheet,
			Release:           c.Release,
			Type:              convertToPlanSpecChangeDatabaseConfigType(c.Type),
			GhostFlags:        c.GhostFlags,
			EnablePriorBackup: c.EnablePriorBackup,
		},
	}
}

func convertToPlanSpecChangeDatabaseConfigType(t storepb.PlanConfig_ChangeDatabaseConfig_Type) v1pb.Plan_ChangeDatabaseConfig_Type {
	switch t {
	case storepb.PlanConfig_ChangeDatabaseConfig_TYPE_UNSPECIFIED:
		return v1pb.Plan_ChangeDatabaseConfig_TYPE_UNSPECIFIED
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
		return v1pb.Plan_ChangeDatabaseConfig_MIGRATE
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_SDL:
		return v1pb.Plan_ChangeDatabaseConfig_MIGRATE_SDL
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST:
		return v1pb.Plan_ChangeDatabaseConfig_MIGRATE_GHOST
	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
		return v1pb.Plan_ChangeDatabaseConfig_DATA
	default:
		return v1pb.Plan_ChangeDatabaseConfig_TYPE_UNSPECIFIED
	}
}

func convertToPlanSpecExportDataConfig(config *storepb.PlanConfig_Spec_ExportDataConfig) *v1pb.Plan_Spec_ExportDataConfig {
	c := config.ExportDataConfig
	return &v1pb.Plan_Spec_ExportDataConfig{
		ExportDataConfig: &v1pb.Plan_ExportDataConfig{
			Targets:  c.Targets,
			Sheet:    c.Sheet,
			Format:   convertExportFormat(c.Format),
			Password: c.Password,
		},
	}
}

func convertToPlanDeployment(deployment *storepb.PlanConfig_Deployment) *v1pb.Plan_Deployment {
	if deployment == nil {
		return nil
	}
	return &v1pb.Plan_Deployment{
		Environments:          deployment.Environments,
		DatabaseGroupMappings: convertToDatabaseGroupMappings(deployment.DatabaseGroupMappings),
	}
}

func convertToDatabaseGroupMappings(mappings []*storepb.PlanConfig_Deployment_DatabaseGroupMapping) []*v1pb.Plan_Deployment_DatabaseGroupMapping {
	v1Mappings := make([]*v1pb.Plan_Deployment_DatabaseGroupMapping, len(mappings))
	for i := range mappings {
		v1Mappings[i] = convertToDatabaseGroupMapping(mappings[i])
	}
	return v1Mappings
}

func convertToDatabaseGroupMapping(mapping *storepb.PlanConfig_Deployment_DatabaseGroupMapping) *v1pb.Plan_Deployment_DatabaseGroupMapping {
	if mapping == nil {
		return nil
	}
	return &v1pb.Plan_Deployment_DatabaseGroupMapping{
		DatabaseGroup: mapping.DatabaseGroup,
		Databases:     mapping.Databases,
	}
}

func convertPlan(plan *v1pb.Plan) *storepb.PlanConfig {
	if plan == nil {
		return nil
	}

	// At this point, plan.Specs should always be populated
	// (either originally or converted from steps at API entry point)
	return &storepb.PlanConfig{
		Specs:      convertPlanSpecs(plan.Specs),
		Deployment: nil,
	}
}

func convertPlanSpecs(specs []*v1pb.Plan_Spec) []*storepb.PlanConfig_Spec {
	storeSpecs := make([]*storepb.PlanConfig_Spec, len(specs))
	for i := range specs {
		storeSpecs[i] = convertPlanSpec(specs[i])
	}
	return storeSpecs
}

func convertPlanSpec(spec *v1pb.Plan_Spec) *storepb.PlanConfig_Spec {
	storeSpec := &storepb.PlanConfig_Spec{
		Id: spec.Id,
	}

	switch v := spec.Config.(type) {
	case *v1pb.Plan_Spec_CreateDatabaseConfig:
		storeSpec.Config = convertPlanSpecCreateDatabaseConfig(v)
	case *v1pb.Plan_Spec_ChangeDatabaseConfig:
		storeSpec.Config = convertPlanSpecChangeDatabaseConfig(v)
	case *v1pb.Plan_Spec_ExportDataConfig:
		storeSpec.Config = convertPlanSpecExportDataConfig(v)
	}
	return storeSpec
}

func convertPlanSpecCreateDatabaseConfig(config *v1pb.Plan_Spec_CreateDatabaseConfig) *storepb.PlanConfig_Spec_CreateDatabaseConfig {
	c := config.CreateDatabaseConfig
	return &storepb.PlanConfig_Spec_CreateDatabaseConfig{
		CreateDatabaseConfig: convertPlanConfigCreateDatabaseConfig(c),
	}
}

func convertPlanConfigCreateDatabaseConfig(c *v1pb.Plan_CreateDatabaseConfig) *storepb.PlanConfig_CreateDatabaseConfig {
	return &storepb.PlanConfig_CreateDatabaseConfig{
		Target:       c.Target,
		Database:     c.Database,
		Table:        c.Table,
		CharacterSet: c.CharacterSet,
		Collation:    c.Collation,
		Cluster:      c.Cluster,
		Owner:        c.Owner,
		Environment:  c.Environment,
	}
}

func convertPlanSpecChangeDatabaseConfig(config *v1pb.Plan_Spec_ChangeDatabaseConfig) *storepb.PlanConfig_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig
	return &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
			Targets:           c.Targets,
			Sheet:             c.Sheet,
			Release:           c.Release,
			Type:              storepb.PlanConfig_ChangeDatabaseConfig_Type(c.Type),
			GhostFlags:        c.GhostFlags,
			EnablePriorBackup: c.EnablePriorBackup,
		},
	}
}

func convertPlanSpecExportDataConfig(config *v1pb.Plan_Spec_ExportDataConfig) *storepb.PlanConfig_Spec_ExportDataConfig {
	c := config.ExportDataConfig
	return &storepb.PlanConfig_Spec_ExportDataConfig{
		ExportDataConfig: &storepb.PlanConfig_ExportDataConfig{
			Targets:  c.Targets,
			Sheet:    c.Sheet,
			Format:   convertToExportFormat(c.Format),
			Password: c.Password,
		},
	}
}

func convertToPlanCheckRuns(ctx context.Context, s *store.Store, projectID string, planUID int64, runs []*store.PlanCheckRunMessage) ([]*v1pb.PlanCheckRun, error) {
	var planCheckRuns []*v1pb.PlanCheckRun
	for _, run := range runs {
		converted, err := convertToPlanCheckRun(ctx, s, projectID, planUID, run)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert plan check run")
		}
		planCheckRuns = append(planCheckRuns, converted)
	}
	return planCheckRuns, nil
}

func convertToPlanCheckRun(ctx context.Context, s *store.Store, projectID string, planUID int64, run *store.PlanCheckRunMessage) (*v1pb.PlanCheckRun, error) {
	converted := &v1pb.PlanCheckRun{
		Name:       common.FormatPlanCheckRun(projectID, planUID, int64(run.UID)),
		CreateTime: timestamppb.New(run.CreatedAt),
		Type:       convertToPlanCheckRunType(run.Type),
		Status:     convertToPlanCheckRunStatus(run.Status),
		Target:     common.FormatDatabase(run.Config.InstanceId, run.Config.DatabaseName),
		Sheet:      "",
		Results:    convertToPlanCheckRunResults(run.Result.Results),
		Error:      run.Result.Error,
	}

	if sheetUID := int(run.Config.GetSheetUid()); sheetUID != 0 {
		sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet")
		}
		if sheet == nil {
			return nil, errors.Errorf("sheet not found for uid %d", sheetUID)
		}
		converted.Sheet = common.FormatSheet(projectID, sheet.UID)
	}

	return converted, nil
}

func convertToPlanCheckRunType(t store.PlanCheckRunType) v1pb.PlanCheckRun_Type {
	switch t {
	case store.PlanCheckDatabaseStatementFakeAdvise:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_FAKE_ADVISE
	case store.PlanCheckDatabaseStatementAdvise:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_ADVISE
	case store.PlanCheckDatabaseStatementSummaryReport:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_SUMMARY_REPORT
	case store.PlanCheckDatabaseConnect:
		return v1pb.PlanCheckRun_DATABASE_CONNECT
	case store.PlanCheckDatabaseGhostSync:
		return v1pb.PlanCheckRun_DATABASE_GHOST_SYNC
	default:
		return v1pb.PlanCheckRun_TYPE_UNSPECIFIED
	}
}

func convertToPlanCheckRunStatus(status store.PlanCheckRunStatus) v1pb.PlanCheckRun_Status {
	switch status {
	case store.PlanCheckRunStatusCanceled:
		return v1pb.PlanCheckRun_CANCELED
	case store.PlanCheckRunStatusDone:
		return v1pb.PlanCheckRun_DONE
	case store.PlanCheckRunStatusFailed:
		return v1pb.PlanCheckRun_FAILED
	case store.PlanCheckRunStatusRunning:
		return v1pb.PlanCheckRun_RUNNING
	default:
		return v1pb.PlanCheckRun_STATUS_UNSPECIFIED
	}
}

func convertToPlanCheckRunResults(results []*storepb.PlanCheckRunResult_Result) []*v1pb.PlanCheckRun_Result {
	var resultsV1 []*v1pb.PlanCheckRun_Result
	for _, result := range results {
		resultsV1 = append(resultsV1, convertToPlanCheckRunResult(result))
	}
	return resultsV1
}

func convertToPlanCheckRunResult(result *storepb.PlanCheckRunResult_Result) *v1pb.PlanCheckRun_Result {
	resultV1 := &v1pb.PlanCheckRun_Result{
		Status:  convertToPlanCheckRunResultStatus(result.Status),
		Title:   result.Title,
		Content: result.Content,
		Code:    result.Code,
		Report:  nil,
	}
	switch report := result.Report.(type) {
	case *storepb.PlanCheckRunResult_Result_SqlSummaryReport_:
		resultV1.Report = &v1pb.PlanCheckRun_Result_SqlSummaryReport_{
			SqlSummaryReport: &v1pb.PlanCheckRun_Result_SqlSummaryReport{
				StatementTypes:   report.SqlSummaryReport.StatementTypes,
				AffectedRows:     report.SqlSummaryReport.AffectedRows,
				ChangedResources: convertToChangedResources(report.SqlSummaryReport.ChangedResources),
			},
		}
	case *storepb.PlanCheckRunResult_Result_SqlReviewReport_:
		resultV1.Report = &v1pb.PlanCheckRun_Result_SqlReviewReport_{
			SqlReviewReport: &v1pb.PlanCheckRun_Result_SqlReviewReport{
				Line:          report.SqlReviewReport.Line,
				Column:        report.SqlReviewReport.Column,
				StartPosition: convertToPosition(report.SqlReviewReport.StartPosition),
				EndPosition:   convertToPosition(report.SqlReviewReport.EndPosition),
			},
		}
	}
	return resultV1
}

func convertToPlanCheckRunResultStatus(status storepb.PlanCheckRunResult_Result_Status) v1pb.PlanCheckRun_Result_Status {
	switch status {
	case storepb.PlanCheckRunResult_Result_STATUS_UNSPECIFIED:
		return v1pb.PlanCheckRun_Result_STATUS_UNSPECIFIED
	case storepb.PlanCheckRunResult_Result_SUCCESS:
		return v1pb.PlanCheckRun_Result_SUCCESS
	case storepb.PlanCheckRunResult_Result_WARNING:
		return v1pb.PlanCheckRun_Result_WARNING
	case storepb.PlanCheckRunResult_Result_ERROR:
		return v1pb.PlanCheckRun_Result_ERROR
	default:
		return v1pb.PlanCheckRun_Result_STATUS_UNSPECIFIED
	}
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
	case storepb.Task_DATABASE_SCHEMA_UPDATE, storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST, storepb.Task_DATABASE_SCHEMA_UPDATE_SDL:
		return convertToTaskFromSchemaUpdate(ctx, s, project, task)
	case storepb.Task_DATABASE_DATA_UPDATE:
		return convertToTaskFromDataUpdate(ctx, s, project, task)
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
		Type:          convertToTaskType(task.Type),
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

	stageID := formatStageIDFromEnvironment(task.Environment)
	v1pbTask := &v1pb.Task{
		Name:          common.FormatTask(project.ResourceID, task.PipelineID, stageID, task.ID),
		SpecId:        task.Payload.GetSpecId(),
		Type:          convertToTaskType(task.Type),
		Status:        convertToTaskStatus(task.LatestTaskRunStatus, task.Payload.GetSkipped()),
		SkippedReason: task.Payload.GetSkippedReason(),
		Target:        fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload: &v1pb.Task_DatabaseSchemaUpdate_{
			DatabaseSchemaUpdate: &v1pb.Task_DatabaseSchemaUpdate{
				Sheet:         common.FormatSheet(project.ResourceID, int(task.Payload.GetSheetId())),
				SchemaVersion: task.Payload.GetSchemaVersion(),
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
		Type:          convertToTaskType(task.Type),
		Status:        convertToTaskStatus(task.LatestTaskRunStatus, task.Payload.GetSkipped()),
		SkippedReason: task.Payload.GetSkippedReason(),
		Target:        fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload:       nil,
	}
	v1pbTaskPayload := &v1pb.Task_DatabaseDataUpdate_{
		DatabaseDataUpdate: &v1pb.Task_DatabaseDataUpdate{
			Sheet:         common.FormatSheet(project.ResourceID, int(task.Payload.GetSheetId())),
			SchemaVersion: task.Payload.GetSchemaVersion(),
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
		Type:    convertToTaskType(task.Type),
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

func convertToTaskType(taskType storepb.Task_Type) v1pb.Task_Type {
	//exhaustive:enforce
	switch taskType {
	case storepb.Task_DATABASE_CREATE:
		return v1pb.Task_DATABASE_CREATE
	case storepb.Task_DATABASE_SCHEMA_UPDATE:
		return v1pb.Task_DATABASE_SCHEMA_UPDATE
	case storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST:
		return v1pb.Task_DATABASE_SCHEMA_UPDATE_GHOST
	case storepb.Task_DATABASE_SCHEMA_UPDATE_SDL:
		return v1pb.Task_DATABASE_SCHEMA_UPDATE_SDL
	case storepb.Task_DATABASE_DATA_UPDATE:
		return v1pb.Task_DATABASE_DATA_UPDATE
	case storepb.Task_DATABASE_EXPORT:
		return v1pb.Task_DATABASE_EXPORT
	case storepb.Task_TASK_TYPE_UNSPECIFIED:
		return v1pb.Task_TYPE_UNSPECIFIED
	default:
		return v1pb.Task_TYPE_UNSPECIFIED
	}
}

func convertToStoreTaskType(taskType v1pb.Task_Type) storepb.Task_Type {
	//exhaustive:enforce
	switch taskType {
	case v1pb.Task_DATABASE_CREATE:
		return storepb.Task_DATABASE_CREATE
	case v1pb.Task_DATABASE_SCHEMA_UPDATE:
		return storepb.Task_DATABASE_SCHEMA_UPDATE
	case v1pb.Task_DATABASE_SCHEMA_UPDATE_GHOST:
		return storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST
	case v1pb.Task_DATABASE_SCHEMA_UPDATE_SDL:
		return storepb.Task_DATABASE_SCHEMA_UPDATE_SDL
	case v1pb.Task_DATABASE_DATA_UPDATE:
		return storepb.Task_DATABASE_DATA_UPDATE
	case v1pb.Task_DATABASE_EXPORT:
		return storepb.Task_DATABASE_EXPORT
	case v1pb.Task_TYPE_UNSPECIFIED, v1pb.Task_GENERAL:
		return storepb.Task_TASK_TYPE_UNSPECIFIED
	default:
		return storepb.Task_TASK_TYPE_UNSPECIFIED
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
