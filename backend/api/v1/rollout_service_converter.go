package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func convertToPlans(plans []*store.PlanMessage) []*v1pb.Plan {
	v1Plans := make([]*v1pb.Plan, len(plans))
	for i := range plans {
		v1Plans[i] = convertToPlan(plans[i])
	}
	return v1Plans
}

func convertToPlan(plan *store.PlanMessage) *v1pb.Plan {
	return &v1pb.Plan{
		Name:        fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, plan.ProjectID, common.PlanPrefix, plan.UID),
		Uid:         fmt.Sprintf("%d", plan.UID),
		Issue:       "",
		Title:       plan.Name,
		Description: plan.Description,
		Steps:       convertToPlanSteps(plan.Config.Steps),
	}
}

func convertToPlanSteps(steps []*storepb.PlanConfig_Step) []*v1pb.Plan_Step {
	v1Steps := make([]*v1pb.Plan_Step, len(steps))
	for i := range steps {
		v1Steps[i] = convertToPlanStep(steps[i])
	}
	return v1Steps
}

func convertToPlanStep(step *storepb.PlanConfig_Step) *v1pb.Plan_Step {
	return &v1pb.Plan_Step{
		Specs: convertToPlanSpecs(step.Specs),
	}
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
		EarliestAllowedTime: spec.EarliestAllowedTime,
		Id:                  spec.Id,
	}

	switch v := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		v1Spec.Config = convertToPlanSpecCreateDatabaseConfig(v)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		v1Spec.Config = convertToPlanSpecChangeDatabaseConfig(v)
	case *storepb.PlanConfig_Spec_RestoreDatabaseConfig:
		v1Spec.Config = convertToPlanSpecRestoreDatabaseConfig(v)
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
			Backup:       c.Backup,
			Environment:  c.Environment,
			Labels:       c.Labels,
		},
	}
}

func convertToPlanCreateDatabaseConfig(c *storepb.PlanConfig_CreateDatabaseConfig) *v1pb.Plan_CreateDatabaseConfig {
	// c.CreateDatabaseConfig is defined as optional in proto
	// so we need to test if it's nil
	if c == nil {
		return nil
	}
	return &v1pb.Plan_CreateDatabaseConfig{
		Target:       c.Target,
		Database:     c.Database,
		Table:        c.Table,
		CharacterSet: c.CharacterSet,
		Collation:    c.Collation,
		Cluster:      c.Cluster,
		Owner:        c.Owner,
		Backup:       c.Backup,
		Environment:  c.Environment,
		Labels:       c.Labels,
	}
}

func convertToPlanSpecChangeDatabaseConfig(config *storepb.PlanConfig_Spec_ChangeDatabaseConfig) *v1pb.Plan_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig
	return &v1pb.Plan_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
			Target:          c.Target,
			Sheet:           c.Sheet,
			Type:            convertToPlanSpecChangeDatabaseConfigType(c.Type),
			SchemaVersion:   c.SchemaVersion,
			RollbackEnabled: c.RollbackEnabled,
			RollbackDetail:  convertToPlanSpecChangeDatabaseConfigRollbackDetail(c.RollbackDetail),
		},
	}
}

func convertToPlanSpecChangeDatabaseConfigRollbackDetail(d *storepb.PlanConfig_ChangeDatabaseConfig_RollbackDetail) *v1pb.Plan_ChangeDatabaseConfig_RollbackDetail {
	if d == nil {
		return nil
	}
	return &v1pb.Plan_ChangeDatabaseConfig_RollbackDetail{
		RollbackFromIssue: d.RollbackFromIssue,
		RollbackFromTask:  d.RollbackFromIssue,
	}
}

func convertToPlanSpecChangeDatabaseConfigType(t storepb.PlanConfig_ChangeDatabaseConfig_Type) v1pb.Plan_ChangeDatabaseConfig_Type {
	switch t {
	case storepb.PlanConfig_ChangeDatabaseConfig_TYPE_UNSPECIFIED:
		return v1pb.Plan_ChangeDatabaseConfig_TYPE_UNSPECIFIED
	case storepb.PlanConfig_ChangeDatabaseConfig_BASELINE:
		return v1pb.Plan_ChangeDatabaseConfig_BASELINE
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
		return v1pb.Plan_ChangeDatabaseConfig_MIGRATE
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_SDL:
		return v1pb.Plan_ChangeDatabaseConfig_MIGRATE_SDL
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST:
		return v1pb.Plan_ChangeDatabaseConfig_MIGRATE_GHOST
	case storepb.PlanConfig_ChangeDatabaseConfig_BRANCH:
		return v1pb.Plan_ChangeDatabaseConfig_BRANCH
	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
		return v1pb.Plan_ChangeDatabaseConfig_DATA
	default:
		return v1pb.Plan_ChangeDatabaseConfig_TYPE_UNSPECIFIED
	}
}

func convertToPlanSpecRestoreDatabaseConfig(config *storepb.PlanConfig_Spec_RestoreDatabaseConfig) *v1pb.Plan_Spec_RestoreDatabaseConfig {
	c := config.RestoreDatabaseConfig
	v1Config := &v1pb.Plan_Spec_RestoreDatabaseConfig{
		RestoreDatabaseConfig: &v1pb.Plan_RestoreDatabaseConfig{
			Target: c.Target,
		},
	}
	switch source := c.Source.(type) {
	case *storepb.PlanConfig_RestoreDatabaseConfig_Backup:
		v1Config.RestoreDatabaseConfig.Source = &v1pb.Plan_RestoreDatabaseConfig_Backup{
			Backup: source.Backup,
		}
	case *storepb.PlanConfig_RestoreDatabaseConfig_PointInTime:
		v1Config.RestoreDatabaseConfig.Source = &v1pb.Plan_RestoreDatabaseConfig_PointInTime{
			PointInTime: source.PointInTime,
		}
	}

	v1Config.RestoreDatabaseConfig.CreateDatabaseConfig = convertToPlanCreateDatabaseConfig(c.CreateDatabaseConfig)
	return v1Config
}

func convertPlanSteps(steps []*v1pb.Plan_Step) []*storepb.PlanConfig_Step {
	storeSteps := make([]*storepb.PlanConfig_Step, len(steps))
	for i := range steps {
		storeSteps[i] = convertPlanStep(steps[i])
	}
	return storeSteps
}

func convertPlanStep(step *v1pb.Plan_Step) *storepb.PlanConfig_Step {
	return &storepb.PlanConfig_Step{
		Specs: convertPlanSpecs(step.Specs),
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
		EarliestAllowedTime: spec.EarliestAllowedTime,
		Id:                  spec.Id,
	}

	switch v := spec.Config.(type) {
	case *v1pb.Plan_Spec_CreateDatabaseConfig:
		storeSpec.Config = convertPlanSpecCreateDatabaseConfig(v)
	case *v1pb.Plan_Spec_ChangeDatabaseConfig:
		storeSpec.Config = convertPlanSpecChangeDatabaseConfig(v)
	case *v1pb.Plan_Spec_RestoreDatabaseConfig:
		storeSpec.Config = convertPlanSpecRestoreDatabaseConfig(v)
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
		Backup:       c.Backup,
		Environment:  c.Environment,
		Labels:       c.Labels,
	}
}

func convertPlanSpecChangeDatabaseConfig(config *v1pb.Plan_Spec_ChangeDatabaseConfig) *storepb.PlanConfig_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig
	return &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
			Target:          c.Target,
			Sheet:           c.Sheet,
			Type:            storepb.PlanConfig_ChangeDatabaseConfig_Type(c.Type),
			SchemaVersion:   c.SchemaVersion,
			RollbackEnabled: c.RollbackEnabled,
		},
	}
}

func convertPlanSpecRestoreDatabaseConfig(config *v1pb.Plan_Spec_RestoreDatabaseConfig) *storepb.PlanConfig_Spec_RestoreDatabaseConfig {
	c := config.RestoreDatabaseConfig
	storeConfig := &storepb.PlanConfig_Spec_RestoreDatabaseConfig{
		RestoreDatabaseConfig: &storepb.PlanConfig_RestoreDatabaseConfig{
			Target: c.Target,
		},
	}
	switch source := c.Source.(type) {
	case *v1pb.Plan_RestoreDatabaseConfig_Backup:
		storeConfig.RestoreDatabaseConfig.Source = &storepb.PlanConfig_RestoreDatabaseConfig_Backup{
			Backup: source.Backup,
		}
	case *v1pb.Plan_RestoreDatabaseConfig_PointInTime:
		storeConfig.RestoreDatabaseConfig.Source = &storepb.PlanConfig_RestoreDatabaseConfig_PointInTime{
			PointInTime: source.PointInTime,
		}
	}
	// c.CreateDatabaseConfig is defined as optional in proto
	// so we need to test if it's nil
	if c.CreateDatabaseConfig != nil {
		storeConfig.RestoreDatabaseConfig.CreateDatabaseConfig = convertPlanConfigCreateDatabaseConfig(c.CreateDatabaseConfig)
	}
	return storeConfig
}

// convertDatabaseLabels converts the map[string]string labels to []*api.DatabaseLabel JSON string.
func convertDatabaseLabels(labelsMap map[string]string) (string, error) {
	if len(labelsMap) == 0 {
		return "", nil
	}
	// For scalability, each database can have up to four labels for now.
	if len(labelsMap) > api.DatabaseLabelSizeMax {
		return "", errors.Errorf("database labels are up to a maximum of %d", api.DatabaseLabelSizeMax)
	}
	var labels []*api.DatabaseLabel
	for k, v := range labelsMap {
		labels = append(labels, &api.DatabaseLabel{
			Key:   k,
			Value: v,
		})
	}
	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal labels json")
	}
	return string(labelsJSON), nil
}

func convertToPlanCheckRuns(ctx context.Context, s *store.Store, parent string, runs []*store.PlanCheckRunMessage) ([]*v1pb.PlanCheckRun, error) {
	var planCheckRuns []*v1pb.PlanCheckRun
	for _, run := range runs {
		converted, err := convertToPlanCheckRun(ctx, s, parent, run)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert plan check run")
		}
		planCheckRuns = append(planCheckRuns, converted)
	}
	return planCheckRuns, nil
}

func convertToPlanCheckRun(ctx context.Context, s *store.Store, parent string, run *store.PlanCheckRunMessage) (*v1pb.PlanCheckRun, error) {
	converted := &v1pb.PlanCheckRun{
		Name:       fmt.Sprintf("%s/%s%d", parent, common.PlanCheckRunPrefix, run.UID),
		Uid:        fmt.Sprintf("%d", run.UID),
		CreateTime: timestamppb.New(time.Unix(run.CreatedTs, 0)),
		Type:       convertToPlanCheckRunType(run.Type),
		Status:     convertToPlanCheckRunStatus(run.Status),
		Target:     "",
		Sheet:      "",
		Results:    convertToPlanCheckRunResults(run.Result.Results),
		Error:      run.Result.Error,
	}

	instanceUID := int(run.Config.InstanceUid)
	databaseName := run.Config.DatabaseName
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance")
	}
	converted.Target = fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, instance.ResourceID, common.DatabaseIDPrefix, databaseName)

	if run.Config.SheetUid != 0 {
		sheetUID := int(run.Config.SheetUid)
		sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID}, api.SystemBotID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet")
		}
		sheetProject, err := s.GetProjectV2(ctx, &store.FindProjectMessage{UID: &sheet.ProjectUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet project")
		}
		converted.Sheet = fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, sheetProject.ResourceID, common.SheetIDPrefix, sheet.UID)
	}
	return converted, nil
}

func convertToPlanCheckRunType(t store.PlanCheckRunType) v1pb.PlanCheckRun_Type {
	switch t {
	case store.PlanCheckDatabaseStatementFakeAdvise:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_FAKE_ADVISE
	case store.PlanCheckDatabaseStatementCompatibility:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_COMPATIBILITY
	case store.PlanCheckDatabaseStatementAdvise:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_ADVISE
	case store.PlanCheckDatabaseStatementType:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_TYPE
	case store.PlanCheckDatabaseStatementSummaryReport:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_SUMMARY_REPORT
	case store.PlanCheckDatabaseConnect:
		return v1pb.PlanCheckRun_DATABASE_CONNECT
	case store.PlanCheckDatabaseGhostSync:
		return v1pb.PlanCheckRun_DATABASE_GHOST_SYNC
	case store.PlanCheckDatabasePITRMySQL:
		return v1pb.PlanCheckRun_DATABASE_PITR_MYSQL
	}
	return v1pb.PlanCheckRun_TYPE_UNSPECIFIED
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
	}
	return v1pb.PlanCheckRun_STATUS_UNSPECIFIED
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
				Code:             report.SqlSummaryReport.Code,
				StatementTypes:   report.SqlSummaryReport.StatementTypes,
				AffectedRows:     report.SqlSummaryReport.AffectedRows,
				ChangedResources: convertToChangedResources(report.SqlSummaryReport.ChangedResources),
			},
		}
	case *storepb.PlanCheckRunResult_Result_SqlReviewReport_:
		resultV1.Report = &v1pb.PlanCheckRun_Result_SqlReviewReport_{
			SqlReviewReport: &v1pb.PlanCheckRun_Result_SqlReviewReport{
				Line:   report.SqlReviewReport.Line,
				Column: report.SqlReviewReport.Column,
				Detail: report.SqlReviewReport.Detail,
				Code:   report.SqlReviewReport.Code,
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
	}
	return v1pb.PlanCheckRun_Result_STATUS_UNSPECIFIED
}

func convertToTaskRuns(taskRuns []*store.TaskRunMessage) []*v1pb.TaskRun {
	var taskRunsV1 []*v1pb.TaskRun
	for _, taskRun := range taskRuns {
		taskRunsV1 = append(taskRunsV1, convertToTaskRun(taskRun))
	}
	return taskRunsV1
}

func convertToTaskRunStatus(status api.TaskRunStatus) v1pb.TaskRun_Status {
	switch status {
	case api.TaskRunUnknown:
		return v1pb.TaskRun_STATUS_UNSPECIFIED
	case api.TaskRunPending:
		return v1pb.TaskRun_PENDING
	case api.TaskRunRunning:
		return v1pb.TaskRun_RUNNING
	case api.TaskRunDone:
		return v1pb.TaskRun_DONE
	case api.TaskRunFailed:
		return v1pb.TaskRun_FAILED
	case api.TaskRunCanceled:
		return v1pb.TaskRun_CANCELED
	default:
		return v1pb.TaskRun_STATUS_UNSPECIFIED
	}
}

func convertToTaskRun(taskRun *store.TaskRunMessage) *v1pb.TaskRun {
	return &v1pb.TaskRun{
		Name:          fmt.Sprintf("%s%s/%s%d/%s%d/%s%d/%s%d", common.ProjectNamePrefix, taskRun.ProjectID, common.RolloutPrefix, taskRun.PipelineUID, common.StagePrefix, taskRun.StageUID, common.TaskPrefix, taskRun.TaskUID, common.TaskRunPrefix, taskRun.ID),
		Uid:           fmt.Sprintf("%d", taskRun.ID),
		Creator:       fmt.Sprintf("users/%s", taskRun.Creator.Email),
		Updater:       fmt.Sprintf("users/%s", taskRun.Updater.Email),
		CreateTime:    timestamppb.New(time.Unix(taskRun.CreatedTs, 0)),
		UpdateTime:    timestamppb.New(time.Unix(taskRun.UpdatedTs, 0)),
		Title:         taskRun.Name,
		Status:        convertToTaskRunStatus(taskRun.Status),
		Detail:        taskRun.ResultProto.Detail,
		ChangeHistory: taskRun.ResultProto.ChangeHistory,
		SchemaVersion: taskRun.ResultProto.Version,
	}
}

func convertToRollout(ctx context.Context, s *store.Store, project *store.ProjectMessage, rollout *store.PipelineMessage) (*v1pb.Rollout, error) {
	rolloutV1 := &v1pb.Rollout{
		Name:   fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, rollout.ID),
		Uid:    fmt.Sprintf("%d", rollout.ID),
		Plan:   "",
		Title:  rollout.Name,
		Stages: nil,
	}

	taskIDToName := map[int]string{}
	for _, stage := range rollout.Stages {
		environment, err := s.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
			UID: &stage.EnvironmentID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get environment %d", stage.EnvironmentID)
		}
		if environment == nil {
			return nil, errors.Errorf("environment %d not found", stage.EnvironmentID)
		}
		rolloutStage := &v1pb.Stage{
			Name:        fmt.Sprintf("%s%s/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, rollout.ID, common.StagePrefix, stage.ID),
			Uid:         fmt.Sprintf("%d", stage.ID),
			Environment: fmt.Sprintf("%s%s", common.EnvironmentNamePrefix, environment.ResourceID),
			Title:       stage.Name,
		}
		for _, task := range stage.TaskList {
			rolloutTask, err := convertToTask(ctx, s, project, task)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert task, error: %v", err)
			}
			taskIDToName[task.ID] = rolloutTask.Name
			rolloutStage.Tasks = append(rolloutStage.Tasks, rolloutTask)
		}

		rolloutV1.Stages = append(rolloutV1.Stages, rolloutStage)
	}

	for i, rolloutStage := range rolloutV1.Stages {
		for j, rolloutTask := range rolloutStage.Tasks {
			task := rollout.Stages[i].TaskList[j]
			for _, blockingTask := range task.BlockedBy {
				rolloutTask.BlockedByTasks = append(rolloutTask.BlockedByTasks, taskIDToName[blockingTask])
			}
		}
	}

	return rolloutV1, nil
}

func convertToTask(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	switch task.Type {
	case api.TaskDatabaseCreate:
		return convertToTaskFromDatabaseCreate(ctx, s, project, task)
	case api.TaskDatabaseSchemaBaseline:
		return convertToTaskFromSchemaBaseline(ctx, s, project, task)
	case api.TaskDatabaseSchemaUpdate, api.TaskDatabaseSchemaUpdateSDL, api.TaskDatabaseSchemaUpdateGhostSync:
		return convertToTaskFromSchemaUpdate(ctx, s, project, task)
	case api.TaskDatabaseSchemaUpdateGhostCutover:
		return convertToTaskFromSchemaUpdateGhostCutover(ctx, s, project, task)
	case api.TaskDatabaseDataUpdate:
		return convertToTaskFromDataUpdate(ctx, s, project, task)
	case api.TaskDatabaseBackup:
		return convertToTaskFromDatabaseBackup(ctx, s, project, task)
	case api.TaskDatabaseRestorePITRRestore:
		return convertToTaskFromDatabaseRestoreRestore(ctx, s, project, task)
	case api.TaskDatabaseRestorePITRCutover:
		return convertToTaskFromDatabaseRestoreCutOver(ctx, s, project, task)
	case api.TaskGeneral:
		fallthrough
	default:
		return nil, errors.Errorf("task type %v is not supported", task.Type)
	}
}

func convertToDatabaseLabels(labelsJSON string) (map[string]string, error) {
	if labelsJSON == "" {
		return nil, nil
	}
	var labels []*api.DatabaseLabel
	if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
		return nil, err
	}
	labelsMap := make(map[string]string)
	for _, label := range labels {
		labelsMap[label.Key] = label.Value
	}
	return labelsMap, nil
}

func convertToTaskFromDatabaseCreate(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	payload := &api.TaskDatabaseCreatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task payload")
	}
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		UID: &task.InstanceID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %d", task.InstanceID)
	}
	labels, err := convertToDatabaseLabels(payload.Labels)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert database labels %v", payload.Labels)
	}
	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Type:           convertToTaskType(task.Type),
		Status:         convertToTaskStatus(task.LatestTaskRunStatus, payload.Skipped),
		SkippedReason:  payload.SkippedReason,
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s", common.InstanceNamePrefix, instance.ResourceID),
		Payload: &v1pb.Task_DatabaseCreate_{
			DatabaseCreate: &v1pb.Task_DatabaseCreate{
				Project:      "",
				Database:     payload.DatabaseName,
				Table:        payload.TableName,
				Sheet:        getResourceNameForSheet(project, payload.SheetID),
				CharacterSet: payload.CharacterSet,
				Collation:    payload.Collation,
				Environment:  payload.Environment,
				Labels:       labels,
			},
		},
	}

	return v1pbTask, nil
}

func convertToTaskFromSchemaBaseline(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseID == nil {
		return nil, errors.Errorf("database id is nil")
	}
	payload := &api.TaskDatabaseSchemaBaselinePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task payload")
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}
	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Type:           convertToTaskType(task.Type),
		Status:         convertToTaskStatus(task.LatestTaskRunStatus, payload.Skipped),
		SkippedReason:  payload.SkippedReason,
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload: &v1pb.Task_DatabaseSchemaBaseline_{
			DatabaseSchemaBaseline: &v1pb.Task_DatabaseSchemaBaseline{
				SchemaVersion: payload.SchemaVersion,
			},
		},
	}
	return v1pbTask, nil
}

func convertToTaskFromSchemaUpdate(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseID == nil {
		return nil, errors.Errorf("database id is nil")
	}
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task payload")
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}

	// HACK: task.Statement is not empty means that the statement comes from a database group target.
	// we don't want to create new sheets every time so we pass the statement as sheet.
	sheet := getResourceNameForSheet(project, payload.SheetID)
	if task.Statement != "" {
		sheet = task.Statement
	}

	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Type:           convertToTaskType(task.Type),
		Status:         convertToTaskStatus(task.LatestTaskRunStatus, payload.Skipped),
		SkippedReason:  payload.SkippedReason,
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload: &v1pb.Task_DatabaseSchemaUpdate_{
			DatabaseSchemaUpdate: &v1pb.Task_DatabaseSchemaUpdate{
				Sheet:         sheet,
				SchemaVersion: payload.SchemaVersion,
			},
		},
	}
	return v1pbTask, nil
}

func convertToTaskFromSchemaUpdateGhostCutover(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseID == nil {
		return nil, errors.Errorf("database id is nil")
	}
	payload := &api.TaskDatabaseSchemaUpdateGhostCutoverPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task payload")
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}
	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Status:         convertToTaskStatus(task.LatestTaskRunStatus, payload.Skipped),
		SkippedReason:  payload.SkippedReason,
		Type:           convertToTaskType(task.Type),
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload:        nil,
	}
	return v1pbTask, nil
}

func convertToTaskFromDataUpdate(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseID == nil {
		return nil, errors.Errorf("database id is nil")
	}
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task payload")
	}
	var rollbackSheetName string
	if payload.RollbackSheetID != 0 {
		rollbackSheetName = getResourceNameForSheet(project, payload.RollbackSheetID)
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}

	// HACK: task.Statement is not empty means that the statement comes from a database group target.
	// we don't want to create new sheets every time so we pass the statement as sheet.
	sheet := getResourceNameForSheet(project, payload.SheetID)
	if task.Statement != "" {
		sheet = task.Statement
	}

	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Type:           convertToTaskType(task.Type),
		Status:         convertToTaskStatus(task.LatestTaskRunStatus, payload.Skipped),
		SkippedReason:  payload.SkippedReason,
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload:        nil,
	}
	v1pbTaskPayload := &v1pb.Task_DatabaseDataUpdate_{
		DatabaseDataUpdate: &v1pb.Task_DatabaseDataUpdate{
			Sheet:             sheet,
			SchemaVersion:     payload.SchemaVersion,
			RollbackEnabled:   payload.RollbackEnabled,
			RollbackSqlStatus: convertToRollbackSQLStatus(payload.RollbackSQLStatus),
			RollbackError:     payload.RollbackError,
			RollbackSheet:     rollbackSheetName,
			RollbackFromIssue: "",
			RollbackFromTask:  "",
		},
	}
	if payload.RollbackFromIssueID != 0 && payload.RollbackFromTaskID != 0 {
		rollbackFromIssue, err := s.GetIssueV2(ctx, &store.FindIssueMessage{
			UID: &payload.RollbackFromIssueID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get rollback issue %q", payload.RollbackFromIssueID)
		}
		rollbackFromTask, err := s.GetTaskV2ByID(ctx, payload.RollbackFromTaskID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get rollback task %q", payload.RollbackFromTaskID)
		}
		v1pbTaskPayload.DatabaseDataUpdate.RollbackFromIssue = fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, project.ResourceID, common.IssuePrefix, rollbackFromIssue.UID)
		v1pbTaskPayload.DatabaseDataUpdate.RollbackFromTask = fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, rollbackFromIssue.Project.ResourceID, common.RolloutPrefix, rollbackFromTask.PipelineID, common.StagePrefix, rollbackFromTask.StageID, common.TaskPrefix, rollbackFromTask.ID)
	}

	v1pbTask.Payload = v1pbTaskPayload
	return v1pbTask, nil
}

func convertToTaskFromDatabaseBackup(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseID == nil {
		return nil, errors.Errorf("database id is nil")
	}
	payload := &api.TaskDatabaseBackupPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task payload")
	}
	backup, err := s.GetBackupByUID(ctx, payload.BackupID)
	if err != nil {
		return nil, errors.Errorf("failed to get backup by uid: %v", err)
	}
	if backup == nil {
		return nil, errors.Errorf("backup not found")
	}
	databaseBackup, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		UID: &backup.DatabaseUID,
	})
	if err != nil {
		return nil, errors.Errorf("failed to get database: %v", err)
	}
	if databaseBackup == nil {
		return nil, errors.Errorf("database not found")
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}
	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Type:           convertToTaskType(task.Type),
		Status:         convertToTaskStatus(task.LatestTaskRunStatus, payload.Skipped),
		SkippedReason:  payload.SkippedReason,
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload: &v1pb.Task_DatabaseBackup_{
			DatabaseBackup: &v1pb.Task_DatabaseBackup{
				Backup: fmt.Sprintf("%s%s/%s%s/%s%d", common.InstanceNamePrefix, databaseBackup.InstanceID, common.DatabaseIDPrefix, databaseBackup.DatabaseName, common.BackupPrefix, backup.UID),
			},
		},
	}
	return v1pbTask, nil
}

func convertToTaskFromDatabaseRestoreRestore(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseID == nil {
		return nil, errors.Errorf("database id is nil")
	}
	payload := &api.TaskDatabasePITRRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task payload")
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}
	v1pbTaskPayload := v1pb.Task_DatabaseRestoreRestore_{
		DatabaseRestoreRestore: &v1pb.Task_DatabaseRestoreRestore{},
	}
	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Type:           convertToTaskType(task.Type),
		Status:         convertToTaskStatus(task.LatestTaskRunStatus, payload.Skipped),
		SkippedReason:  payload.SkippedReason,
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload:        nil,
	}
	if (payload.BackupID == nil) == (payload.PointInTimeTs == nil) {
		return nil, errors.Errorf("payload.BackupID and payload.PointInTimeTs cannot be both nil or both not nil")
	}
	if (payload.TargetInstanceID == nil) != (payload.DatabaseName == nil) {
		return nil, errors.Errorf("payload.TargetInstanceID and payload.DatabaseName must be both nil or both not nil")
	}

	if payload.TargetInstanceID != nil {
		targetInstance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
			UID: payload.TargetInstanceID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get target instance")
		}
		if targetInstance == nil {
			return nil, errors.Errorf("target instance not found")
		}
		v1pbTaskPayload.DatabaseRestoreRestore.Target = fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, targetInstance.ResourceID, common.DatabaseIDPrefix, *payload.DatabaseName)
	}

	if payload.BackupID != nil {
		backup, err := s.GetBackupByUID(ctx, *payload.BackupID)
		if err != nil {
			return nil, errors.Errorf("failed to get backup by uid: %v", err)
		}
		if backup == nil {
			return nil, errors.Errorf("backup not found")
		}
		databaseBackup, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			UID: &backup.DatabaseUID,
		})
		if err != nil {
			return nil, errors.Errorf("failed to get database: %v", err)
		}
		if databaseBackup == nil {
			return nil, errors.Errorf("database not found")
		}
		v1pbTaskPayload.DatabaseRestoreRestore.Source = &v1pb.Task_DatabaseRestoreRestore_Backup{
			Backup: fmt.Sprintf("%s%s/%s%s/%s%d", common.InstanceNamePrefix, databaseBackup.InstanceID, common.DatabaseIDPrefix, databaseBackup.DatabaseName, common.BackupPrefix, backup.UID),
		}
	}
	if payload.PointInTimeTs != nil {
		v1pbTaskPayload.DatabaseRestoreRestore.Source = &v1pb.Task_DatabaseRestoreRestore_PointInTime{
			PointInTime: timestamppb.New(time.Unix(*payload.PointInTimeTs, 0)),
		}
	}
	v1pbTask.Payload = &v1pbTaskPayload

	return v1pbTask, nil
}

func convertToTaskFromDatabaseRestoreCutOver(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseID == nil {
		return nil, errors.Errorf("database id is nil")
	}
	payload := &api.TaskDatabasePITRCutoverPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task payload")
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}
	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Type:           convertToTaskType(task.Type),
		Status:         convertToTaskStatus(task.LatestTaskRunStatus, payload.Skipped),
		SkippedReason:  payload.SkippedReason,
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload:        nil,
	}

	return v1pbTask, nil
}

func convertToTaskStatus(latestTaskRunStatus api.TaskRunStatus, skipped bool) v1pb.Task_Status {
	if skipped {
		return v1pb.Task_SKIPPED
	}
	switch latestTaskRunStatus {
	case api.TaskRunNotStarted:
		return v1pb.Task_NOT_STARTED
	case api.TaskRunPending:
		return v1pb.Task_PENDING
	case api.TaskRunRunning:
		return v1pb.Task_RUNNING
	case api.TaskRunDone:
		return v1pb.Task_DONE
	case api.TaskRunFailed:
		return v1pb.Task_FAILED
	case api.TaskRunCanceled:
		return v1pb.Task_CANCELED
	default:
		return v1pb.Task_STATUS_UNSPECIFIED
	}
}

func convertToTaskType(taskType api.TaskType) v1pb.Task_Type {
	switch taskType {
	case api.TaskGeneral:
		return v1pb.Task_GENERAL
	case api.TaskDatabaseCreate:
		return v1pb.Task_DATABASE_CREATE
	case api.TaskDatabaseSchemaBaseline:
		return v1pb.Task_DATABASE_SCHEMA_BASELINE
	case api.TaskDatabaseSchemaUpdate:
		return v1pb.Task_DATABASE_SCHEMA_UPDATE
	case api.TaskDatabaseSchemaUpdateSDL:
		return v1pb.Task_DATABASE_SCHEMA_UPDATE_SDL
	case api.TaskDatabaseSchemaUpdateGhostSync:
		return v1pb.Task_DATABASE_SCHEMA_UPDATE_GHOST_SYNC
	case api.TaskDatabaseSchemaUpdateGhostCutover:
		return v1pb.Task_DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER
	case api.TaskDatabaseDataUpdate:
		return v1pb.Task_DATABASE_DATA_UPDATE
	case api.TaskDatabaseBackup:
		return v1pb.Task_DATABASE_BACKUP
	case api.TaskDatabaseRestorePITRRestore:
		return v1pb.Task_DATABASE_RESTORE_RESTORE
	case api.TaskDatabaseRestorePITRCutover:
		return v1pb.Task_DATABASE_RESTORE_CUTOVER
	default:
		return v1pb.Task_TYPE_UNSPECIFIED
	}
}

func convertToRollbackSQLStatus(status api.RollbackSQLStatus) v1pb.Task_DatabaseDataUpdate_RollbackSqlStatus {
	switch status {
	case api.RollbackSQLStatusPending:
		return v1pb.Task_DatabaseDataUpdate_PENDING
	case api.RollbackSQLStatusDone:
		return v1pb.Task_DatabaseDataUpdate_DONE
	case api.RollbackSQLStatusFailed:
		return v1pb.Task_DatabaseDataUpdate_FAILED

	default:
		return v1pb.Task_DatabaseDataUpdate_ROLLBACK_SQL_STATUS_UNSPECIFIED
	}
}

func getResourceNameForSheet(project *store.ProjectMessage, sheetUID int) string {
	return fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, project.ResourceID, common.SheetIDPrefix, sheetUID)
}
