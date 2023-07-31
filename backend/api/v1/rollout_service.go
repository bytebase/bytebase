package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/runner/taskcheck"
	"github.com/bytebase/bytebase/backend/runner/taskrun"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RolloutService represents a service for managing rollout.
type RolloutService struct {
	v1pb.UnimplementedRolloutServiceServer
	store              *store.Store
	licenseService     enterpriseAPI.LicenseService
	dbFactory          *dbfactory.DBFactory
	taskScheduler      *taskrun.Scheduler
	taskCheckScheduler *taskcheck.Scheduler
	planCheckScheduler *plancheck.Scheduler
	stateCfg           *state.State
	activityManager    *activity.Manager
}

// NewRolloutService returns a rollout service instance.
func NewRolloutService(store *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, taskScheduler *taskrun.Scheduler, taskCheckScheduler *taskcheck.Scheduler, planCheckScheduler *plancheck.Scheduler, stateCfg *state.State, activityManager *activity.Manager) *RolloutService {
	return &RolloutService{
		store:              store,
		licenseService:     licenseService,
		dbFactory:          dbFactory,
		taskScheduler:      taskScheduler,
		taskCheckScheduler: taskCheckScheduler,
		planCheckScheduler: planCheckScheduler,
		stateCfg:           stateCfg,
		activityManager:    activityManager,
	}
}

// GetPlan gets a plan.
func (s *RolloutService) GetPlan(ctx context.Context, request *v1pb.GetPlanRequest) (*v1pb.Plan, error) {
	planID, err := common.GetPlanID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	plan, err := s.store.GetPlan(ctx, planID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found for id: %d", planID)
	}
	return convertToPlan(plan), nil
}

// ListPlans lists plans.
func (s *RolloutService) ListPlans(ctx context.Context, request *v1pb.ListPlansRequest) (*v1pb.ListPlansResponse, error) {
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var limit, offset int
	if request.PageToken != "" {
		var pageToken storepb.PageToken
		if err := unmarshalPageToken(request.PageToken, &pageToken); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page token: %v", err)
		}
		if pageToken.Limit < 0 {
			return nil, status.Errorf(codes.InvalidArgument, "page size cannot be negative")
		}
		limit = int(pageToken.Limit)
		offset = int(pageToken.Offset)
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	limitPlusOne := limit + 1

	find := &store.FindPlanMessage{
		Limit:  &limitPlusOne,
		Offset: &offset,
	}
	if projectID != "-" {
		find.ProjectID = &projectID
	}

	plans, err := s.store.ListPlans(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list plans, error: %v", err)
	}

	// has more pages
	if len(plans) == limitPlusOne {
		nextPageToken, err := getPageToken(limit, offset+limit)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		return &v1pb.ListPlansResponse{
			Plans:         convertToPlans(plans[:limit]),
			NextPageToken: nextPageToken,
		}, nil
	}

	// no subsequent pages
	return &v1pb.ListPlansResponse{
		Plans:         convertToPlans(plans),
		NextPageToken: "",
	}, nil
}

// CreatePlan creates a new plan.
func (s *RolloutService) CreatePlan(ctx context.Context, request *v1pb.CreatePlanRequest) (*v1pb.Plan, error) {
	creatorID := ctx.Value(common.PrincipalIDContextKey).(int)
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project not found for id: %v", projectID)
	}
	if err := validateSteps(request.Plan.Steps); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to validate plan steps, error: %v", err)
	}

	planMessage := &store.PlanMessage{
		ProjectID:   projectID,
		PipelineUID: nil,
		Name:        request.Plan.Title,
		Description: request.Plan.Description,
		Config: &storepb.PlanConfig{
			Steps: convertPlanSteps(request.Plan.Steps),
		},
	}

	plan, err := s.store.CreatePlan(ctx, planMessage, creatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan, error: %v", err)
	}
	return convertToPlan(plan), nil
}

// PreviewRollout previews the rollout for a plan.
func (s *RolloutService) PreviewRollout(ctx context.Context, request *v1pb.PreviewRolloutRequest) (*v1pb.Rollout, error) {
	projectID, err := common.GetProjectID(request.Project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}

	if err := validateSteps(request.Plan.Steps); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to validate plan steps, error: %v", err)
	}
	steps := convertPlanSteps(request.Plan.Steps)

	rollout, err := s.getPipelineCreate(ctx, steps, project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get pipeline create, error: %v", err)
	}
	if len(rollout.Stages) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "plan has no stage created")
	}

	rolloutV1, err := convertToRollout(ctx, s.store, project, rollout)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to rollout, error: %v", err)
	}
	return rolloutV1, nil
}

// GetRollout gets a rollout.
func (s *RolloutService) GetRollout(ctx context.Context, request *v1pb.GetRolloutRequest) (*v1pb.Rollout, error) {
	projectID, rolloutID, err := common.GetProjectIDRolloutID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	rollout, err := s.store.GetRollout(ctx, rolloutID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get pipeline, error: %v", err)
	}

	rolloutV1, err := convertToRollout(ctx, s.store, project, rollout)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to rollout, error: %v", err)
	}
	return rolloutV1, nil
}

// CreateRollout creates a rollout from plan.
func (s *RolloutService) CreateRollout(ctx context.Context, request *v1pb.CreateRolloutRequest) (*v1pb.Rollout, error) {
	creatorID := ctx.Value(common.PrincipalIDContextKey).(int)
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project not found for id: %v", projectID)
	}

	planID, err := common.GetPlanID(request.Plan)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	plan, err := s.store.GetPlan(ctx, planID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found for id: %d", planID)
	}

	pipelineCreate, err := s.getPipelineCreate(ctx, plan.Config.Steps, project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get pipeline create, error: %v", err)
	}
	if len(pipelineCreate.Stages) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no database matched for deployment")
	}
	pipeline, err := s.createPipeline(ctx, project, pipelineCreate, creatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create pipeline, error: %v", err)
	}

	// Update pipeline ID in the plan.
	if err := s.store.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:         planID,
		UpdaterID:   creatorID,
		PipelineUID: &pipeline.ID,
	}); err != nil {
		return nil, err
	}

	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PlanUID: &planID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue by plan id %v, error: %v", planID, err)
	}
	if issue != nil {
		if _, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
			PipelineUID: &pipeline.ID,
		}, creatorID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update issue by plan id %v, error: %v", planID, err)
		}
	}

	rollout, err := s.store.GetRollout(ctx, pipeline.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get pipeline, error: %v", err)
	}

	rolloutV1, err := convertToRollout(ctx, s.store, project, rollout)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to rollout, error: %v", err)
	}
	return rolloutV1, nil
}

// ListPlanCheckRuns lists plan check runs for the plan.
func (s *RolloutService) ListPlanCheckRuns(ctx context.Context, request *v1pb.ListPlanCheckRunsRequest) (*v1pb.ListPlanCheckRunsResponse, error) {
	planUID, err := common.GetPlanID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	planCheckRuns, err := s.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		PlanUID: &planUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list plan check runs, error: %v", err)
	}
	converted, err := convertToPlanCheckRuns(ctx, s.store, request.Parent, planCheckRuns)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert plan check runs, error: %v", err)
	}

	return &v1pb.ListPlanCheckRunsResponse{
		PlanCheckRuns: converted,
		NextPageToken: "",
	}, nil
}

// RunPlanChecks runs plan checks for a plan.
func (s *RolloutService) RunPlanChecks(ctx context.Context, request *v1pb.RunPlanChecksRequest) (*v1pb.RunPlanChecksResponse, error) {
	planUID, err := common.GetPlanID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	plan, err := s.store.GetPlan(ctx, planUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found")
	}
	if err := s.planCheckScheduler.SchedulePlanChecksForPlan(ctx, planUID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to run plan checks, error: %v", err)
	}
	return &v1pb.RunPlanChecksResponse{}, nil
}

// ListTaskRuns lists rollout task runs.
func (s *RolloutService) ListTaskRuns(ctx context.Context, request *v1pb.ListTaskRunsRequest) (*v1pb.ListTaskRunsResponse, error) {
	projectID, rolloutID, maybeStageID, maybeTaskID, err := common.GetProjectIDRolloutIDMaybeStageIDMaybeTaskID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %v not found", projectID)
	}

	taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
		PipelineUID: &rolloutID,
		StageUID:    maybeStageID,
		TaskUID:     maybeTaskID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list task runs, error: %v", err)
	}

	return &v1pb.ListTaskRunsResponse{
		TaskRuns:      convertToTaskRuns(taskRuns),
		NextPageToken: "",
	}, nil
}

// BatchRunTasks runs tasks in batch.
func (s *RolloutService) BatchRunTasks(ctx context.Context, request *v1pb.BatchRunTasksRequest) (*v1pb.BatchRunTasksResponse, error) {
	if len(request.Tasks) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "The tasks in request cannot be empty")
	}
	projectID, rolloutID, _, err := common.GetProjectIDRolloutIDMaybeStageID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %v not found", projectID)
	}

	rollout, err := s.store.GetPipelineV2ByID(ctx, rolloutID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find rollout, error: %v", err)
	}
	if rollout == nil {
		return nil, status.Errorf(codes.NotFound, "rollout %v not found", rolloutID)
	}

	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &rolloutID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find issue, error: %v", err)
	}
	if issue == nil {
		return nil, status.Errorf(codes.NotFound, "issue not found for rollout %v", rolloutID)
	}

	stages, err := s.store.ListStageV2(ctx, rolloutID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list stages, error: %v", err)
	}
	if len(stages) == 0 {
		return nil, status.Errorf(codes.NotFound, "no stages found for rollout %v", rolloutID)
	}

	stageTasks := map[int][]int{}
	taskIDsToRunMap := map[int]bool{}
	taskIDsToRun := []int{}
	for _, task := range request.Tasks {
		_, _, stageID, taskID, err := common.GetProjectIDRolloutIDStageIDTaskID(task)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		stageTasks[stageID] = append(stageTasks[stageID], taskID)
		taskIDsToRun = append(taskIDsToRun, taskID)
		taskIDsToRunMap[taskID] = true
	}
	if len(stageTasks) > 1 {
		return nil, status.Errorf(codes.InvalidArgument, "tasks should be in the same stage")
	}
	var stageToRun *store.StageMessage
	for stageID := range stageTasks {
		for _, stage := range stages {
			if stage.ID == stageID {
				stageToRun = stage
				break
			}
			if stage.Active {
				return nil, status.Errorf(codes.InvalidArgument, "Tasks in a prior stage are not done yet")
			}
		}
		break
	}

	pendingApprovalStatus := []api.TaskStatus{api.TaskPendingApproval}
	stageToRunTasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: &rolloutID, StageID: &stageToRun.ID, StatusList: &pendingApprovalStatus})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tasks, error: %v", err)
	}
	if len(stageToRunTasks) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "No tasks to run in the stage")
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %v not found", principalID)
	}

	ok, err := s.taskScheduler.CanPrincipalChangeIssueStageTaskStatus(ctx, user, issue, stageToRun.EnvironmentID, api.TaskPending)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the user can run tasks, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "Not allowed to run tasks")
	}

	approved, err := utils.CheckIssueApproved(issue)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to check if the issue is approved").SetInternal(err)
	}
	if !approved {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Cannot patch task status because the issue is not approved")
	}

	var tasksToRun []*store.TaskMessage
	for _, task := range stageToRunTasks {
		if !taskIDsToRunMap[task.ID] {
			continue
		}
		tasksToRun = append(tasksToRun, task)
	}

	if err := s.store.BatchPatchTaskStatus(ctx, taskIDsToRun, api.TaskPending, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update task status, error: %v", err)
	}
	if err := s.activityManager.BatchCreateTaskStatusUpdateApprovalActivity(ctx, tasksToRun, principalID, issue, stageToRun.Name); err != nil {
		log.Error("failed to create task status update activity", zap.Error(err))
	}

	return &v1pb.BatchRunTasksResponse{}, nil
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
		Name:    fmt.Sprintf("%s/%s%d", parent, common.PlanCheckRunPrefix, run.UID),
		Uid:     fmt.Sprintf("%d", run.UID),
		Type:    convertToPlanCheckRunType(run.Type),
		Status:  convertToPlanCheckRunStatus(run.Status),
		Target:  "",
		Sheet:   "",
		Results: convertToPlanCheckRunResults(run.Result.Results),
		Error:   run.Result.Error,
	}
	if run.Config.DatabaseId != 0 {
		databaseUID := int(run.Config.DatabaseId)
		database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database")
		}
		converted.Target = fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName)
	}
	if run.Config.SheetId != 0 {
		sheetUID := int(run.Config.SheetId)
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
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_FAKE_ADVISE
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
				StatementType: report.SqlSummaryReport.StatementType,
				AffectedRows:  report.SqlSummaryReport.AffectedRows,
			},
		}
	case *storepb.PlanCheckRunResult_Result_SqlReviewReport_:
		resultV1.Report = &v1pb.PlanCheckRun_Result_SqlReviewReport_{
			SqlReviewReport: &v1pb.PlanCheckRun_Result_SqlReviewReport{
				Line:   report.SqlReviewReport.Line,
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
		Creator:       fmt.Sprintf("user/%s", taskRun.Creator.Email),
		Updater:       fmt.Sprintf("user/%s", taskRun.Updater.Email),
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
			rolloutStage.Tasks = append(rolloutStage.Tasks, rolloutTask)
		}

		rolloutV1.Stages = append(rolloutV1.Stages, rolloutStage)
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
		return convertToTaskFromDatabaseBackUp(ctx, s, project, task)
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
		Status:         convertToTaskStatus(task.Status, payload.Skipped),
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
		Status:         convertToTaskStatus(task.Status, payload.Skipped),
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
	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Type:           convertToTaskType(task.Type),
		Status:         convertToTaskStatus(task.Status, payload.Skipped),
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload: &v1pb.Task_DatabaseSchemaUpdate_{
			DatabaseSchemaUpdate: &v1pb.Task_DatabaseSchemaUpdate{
				Sheet:         getResourceNameForSheet(project, payload.SheetID),
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
		Status:         convertToTaskStatus(task.Status, payload.Skipped),
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
	v1pbTask := &v1pb.Task{
		Name:           fmt.Sprintf("%s%s/%s%d/%s%d/%s%d", common.ProjectNamePrefix, project.ResourceID, common.RolloutPrefix, task.PipelineID, common.StagePrefix, task.StageID, common.TaskPrefix, task.ID),
		Uid:            fmt.Sprintf("%d", task.ID),
		Title:          task.Name,
		SpecId:         payload.SpecID,
		Type:           convertToTaskType(task.Type),
		Status:         convertToTaskStatus(task.Status, payload.Skipped),
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload:        nil,
	}
	v1pbTaskPayload := &v1pb.Task_DatabaseDataUpdate_{
		DatabaseDataUpdate: &v1pb.Task_DatabaseDataUpdate{
			Sheet:             getResourceNameForSheet(project, payload.SheetID),
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

func convertToTaskFromDatabaseBackUp(ctx context.Context, s *store.Store, project *store.ProjectMessage, task *store.TaskMessage) (*v1pb.Task, error) {
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
		Status:         convertToTaskStatus(task.Status, payload.Skipped),
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
		Status:         convertToTaskStatus(task.Status, payload.Skipped),
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
		Status:         convertToTaskStatus(task.Status, payload.Skipped),
		BlockedByTasks: nil,
		Target:         fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Payload:        nil,
	}

	return v1pbTask, nil
}

func convertToTaskStatus(status api.TaskStatus, skipped bool) v1pb.Task_Status {
	switch status {
	case api.TaskPending:
		return v1pb.Task_PENDING
	case api.TaskPendingApproval:
		return v1pb.Task_PENDING_APPROVAL
	case api.TaskRunning:
		if skipped {
			return v1pb.Task_SKIPPED
		}
		return v1pb.Task_RUNNING
	case api.TaskDone:
		return v1pb.Task_DONE
	case api.TaskFailed:
		return v1pb.Task_FAILED
	case api.TaskCanceled:
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

// UpdatePlan updates a plan.
// Currently, only Spec.Config.Sheet can be updated.
func (s *RolloutService) UpdatePlan(ctx context.Context, request *v1pb.UpdatePlanRequest) (*v1pb.Plan, error) {
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "steps":
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid update_mask path %q", path)
		}
	}
	planID, err := common.GetPlanID(request.Plan.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	oldPlan, err := s.store.GetPlan(ctx, planID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan %q: %v", request.Plan.Name, err)
	}
	if oldPlan == nil {
		return nil, status.Errorf(codes.NotFound, "plan %q not found", request.Plan.Name)
	}
	oldSteps := convertToPlanSteps(oldPlan.Config.Steps)

	removed, added, updated := diffSpecs(oldSteps, request.Plan.Steps)
	if len(removed) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cannot remove specs from plan")
	}
	if len(added) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cannot add specs to plan")
	}
	if len(updated) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no specs updated")
	}

	updatedByID := make(map[string]*v1pb.Plan_Spec)
	for _, spec := range updated {
		updatedByID[spec.Id] = spec
	}

	updaterID := ctx.Value(common.PrincipalIDContextKey).(int)
	if oldPlan.PipelineUID != nil {
		tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: oldPlan.PipelineUID})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list tasks: %v", err)
		}
		for _, task := range tasks {
			switch task.Type {
			case api.TaskDatabaseSchemaUpdate, api.TaskDatabaseSchemaUpdateSDL, api.TaskDatabaseSchemaUpdateGhostSync, api.TaskDatabaseDataUpdate:
				var taskPayload struct {
					SpecID  string `json:"specId"`
					SheetID int    `json:"sheetId"`
				}
				if err := json.Unmarshal([]byte(task.Payload), &taskPayload); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
				}
				spec, ok := updatedByID[taskPayload.SpecID]
				if !ok {
					continue
				}
				config, ok := spec.Config.(*v1pb.Plan_Spec_ChangeDatabaseConfig)
				if !ok {
					continue
				}
				sheetID, _, err := common.GetProjectResourceIDSheetID(config.ChangeDatabaseConfig.Sheet)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get sheet id from %q, error: %v", config.ChangeDatabaseConfig.Sheet, err)
				}
				sheetIDInt, err := strconv.Atoi(sheetID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to convert sheet id %q to int, error: %v", sheetID, err)
				}
				sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
					UID: &sheetIDInt,
				}, api.SystemBotID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get sheet %q: %v", config.ChangeDatabaseConfig.Sheet, err)
				}
				if sheet == nil {
					return nil, status.Errorf(codes.NotFound, "sheet %q not found", config.ChangeDatabaseConfig.Sheet)
				}
				// TODO(p0ny): update schema version
				if _, err := s.store.UpdateTaskV2(ctx, &api.TaskPatch{
					ID:        task.ID,
					UpdaterID: updaterID,
					SheetID:   &sheet.UID,
				}); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to update task %q: %v", task.Name, err)
				}
			}
		}
	}

	if err := s.store.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       oldPlan.UID,
		UpdaterID: updaterID,
		Config: &storepb.PlanConfig{
			Steps: convertPlanSteps(request.Plan.Steps),
		},
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update plan %q: %v", request.Plan.Name, err)
	}

	updatedPlan, err := s.store.GetPlan(ctx, oldPlan.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated plan %q: %v", request.Plan.Name, err)
	}
	if updatedPlan == nil {
		return nil, status.Errorf(codes.NotFound, "updated plan %q not found", request.Plan.Name)
	}
	return convertToPlan(updatedPlan), nil
}

// diffSpecs check if there are any specs removed, added or updated in the new plan.
// Only updating sheet is taken into account.
func diffSpecs(oldSteps []*v1pb.Plan_Step, newSteps []*v1pb.Plan_Step) ([]*v1pb.Plan_Spec, []*v1pb.Plan_Spec, []*v1pb.Plan_Spec) {
	oldSpecs := make(map[string]*v1pb.Plan_Spec)
	newSpecs := make(map[string]*v1pb.Plan_Spec)
	var removed, added, updated []*v1pb.Plan_Spec
	for _, step := range oldSteps {
		for _, spec := range step.Specs {
			oldSpecs[spec.Id] = spec
		}
	}
	for _, step := range newSteps {
		for _, spec := range step.Specs {
			newSpecs[spec.Id] = spec
		}
	}
	for _, step := range oldSteps {
		for _, spec := range step.Specs {
			if _, ok := newSpecs[spec.Id]; !ok {
				removed = append(removed, spec)
				break
			}
		}
	}
	for _, step := range newSteps {
		for _, spec := range step.Specs {
			if _, ok := oldSpecs[spec.Id]; !ok {
				added = append(added, spec)
				break
			}
		}
	}
	for _, step := range newSteps {
		for _, spec := range step.Specs {
			if oldSpec, ok := oldSpecs[spec.Id]; ok {
				if isSpecSheetUpdated(oldSpec, spec) {
					updated = append(updated, spec)
					break
				}
			}
		}
	}
	return removed, added, updated
}

func isSpecSheetUpdated(specA *v1pb.Plan_Spec, specB *v1pb.Plan_Spec) bool {
	configA, ok := specA.Config.(*v1pb.Plan_Spec_ChangeDatabaseConfig)
	if !ok {
		return false
	}
	configB, ok := specB.Config.(*v1pb.Plan_Spec_ChangeDatabaseConfig)
	if !ok {
		return false
	}
	return configA.ChangeDatabaseConfig.Sheet != configB.ChangeDatabaseConfig.Sheet
}

func validateSteps(_ []*v1pb.Plan_Step) error {
	// FIXME: impl this func
	// targets should be unique
	// if deploymentConfig is used, only one spec is allowed.
	return nil
}

func (s *RolloutService) getPipelineCreate(ctx context.Context, steps []*storepb.PlanConfig_Step, project *store.ProjectMessage) (*store.PipelineMessage, error) {
	// FIXME: handle deploymentConfig
	pipelineCreate := &store.PipelineMessage{
		Name: "Rollout Pipeline",
	}
	for _, step := range steps {
		stageCreate := &store.StageMessage{}

		var stageEnvironmentID string
		registerEnvironmentID := func(environmentID string) error {
			if stageEnvironmentID == "" {
				stageEnvironmentID = environmentID
				return nil
			}
			if stageEnvironmentID != environmentID {
				return errors.Errorf("expect only one environment in a stage, got %s and %s", stageEnvironmentID, environmentID)
			}
			return nil
		}

		for _, spec := range step.Specs {
			taskCreates, taskIndexDAGCreates, err := s.getTaskCreatesFromSpec(ctx, spec, project, registerEnvironmentID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get task creates from spec")
			}

			stageCreate.TaskList = append(stageCreate.TaskList, taskCreates...)
			offset := len(stageCreate.TaskList)
			for i := range taskIndexDAGCreates {
				taskIndexDAGCreates[i].FromIndex += offset
				taskIndexDAGCreates[i].ToIndex += offset
			}
			stageCreate.TaskIndexDAGList = append(stageCreate.TaskIndexDAGList, taskIndexDAGCreates...)
		}

		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &stageEnvironmentID})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get environment")
		}
		stageCreate.EnvironmentID = environment.UID
		stageCreate.Name = fmt.Sprintf("%s Stage", environment.Title)

		pipelineCreate.Stages = append(pipelineCreate.Stages, stageCreate)
	}
	return pipelineCreate, nil
}

func (s *RolloutService) getTaskCreatesFromSpec(ctx context.Context, spec *storepb.PlanConfig_Spec, project *store.ProjectMessage, registerEnvironmentID func(string) error) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	if s.licenseService.IsFeatureEnabled(api.FeatureTaskScheduleTime) != nil {
		if spec.EarliestAllowedTime != nil && !spec.EarliestAllowedTime.AsTime().IsZero() {
			return nil, nil, errors.Errorf(api.FeatureTaskScheduleTime.AccessErrorMessage())
		}
	}

	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		return getTaskCreatesFromCreateDatabaseConfig(ctx, s.store, s.licenseService, s.dbFactory, spec, config.CreateDatabaseConfig, project, registerEnvironmentID)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		return getTaskCreatesFromChangeDatabaseConfig(ctx, s.store, spec, config.ChangeDatabaseConfig, project, registerEnvironmentID)
	case *storepb.PlanConfig_Spec_RestoreDatabaseConfig:
		return getTaskCreatesFromRestoreDatabaseConfig(ctx, s.store, s.licenseService, s.dbFactory, spec, config.RestoreDatabaseConfig, project, registerEnvironmentID)
	}

	return nil, nil, errors.Errorf("invalid spec config type %T", spec.Config)
}

func getTaskCreatesFromCreateDatabaseConfig(ctx context.Context, s *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_CreateDatabaseConfig, project *store.ProjectMessage, registerEnvironmentID func(string) error) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	if c.Database == "" {
		return nil, nil, errors.Errorf("database name is required")
	}
	instanceID, err := common.GetInstanceID(c.Target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance id from %q", c.Target)
	}
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, nil, err
	}
	if instance == nil {
		return nil, nil, errors.Errorf("instance ID not found %v", instanceID)
	}
	if instance.Engine == db.Oracle {
		return nil, nil, errors.Errorf("creating Oracle database is not supported")
	}
	environment, err := s.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return nil, nil, err
	}
	if environment == nil {
		return nil, nil, errors.Errorf("environment ID not found %v", instance.EnvironmentID)
	}

	if err := registerEnvironmentID(environment.ResourceID); err != nil {
		return nil, nil, err
	}

	if instance.Engine == db.MongoDB && c.Table == "" {
		return nil, nil, errors.Errorf("collection name is required for MongoDB")
	}

	taskCreates, err := func() ([]*store.TaskMessage, error) {
		if err := checkCharacterSetCollationOwner(instance.Engine, c.CharacterSet, c.Collation, c.Owner); err != nil {
			return nil, err
		}
		if c.Database == "" {
			return nil, errors.Errorf("database name is required")
		}
		if instance.Engine == db.Snowflake {
			// Snowflake needs to use upper case of DatabaseName.
			c.Database = strings.ToUpper(c.Database)
		}
		if instance.Engine == db.MongoDB && c.Table == "" {
			return nil, common.Errorf(common.Invalid, "Failed to create issue, collection name missing for MongoDB")
		}
		// Validate the labels. Labels are set upon task completion.
		labelsJSON, err := convertDatabaseLabels(c.Labels)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid database label %q", c.Labels)
		}

		// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
		if project.TenantMode == api.TenantModeTenant {
			if err := licenseService.IsFeatureEnabled(api.FeatureMultiTenancy); err != nil {
				return nil, err
			}
		}

		// Get admin data source username.
		adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
		if adminDataSource == nil {
			return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.Title)
		}
		databaseName := c.Database
		switch instance.Engine {
		case db.Snowflake:
			// Snowflake needs to use upper case of DatabaseName.
			databaseName = strings.ToUpper(databaseName)
		case db.MySQL, db.MariaDB, db.OceanBase:
			// For MySQL, we need to use different case of DatabaseName depends on the variable `lower_case_table_names`.
			// https://dev.mysql.com/doc/refman/8.0/en/identifier-case-sensitivity.html
			// And also, meet an error in here is not a big deal, we will just use the original DatabaseName.
			driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
			if err != nil {
				log.Warn("failed to get admin database driver for instance %q, please check the connection for admin data source", zap.Error(err), zap.String("instance", instance.Title))
				break
			}
			defer driver.Close(ctx)
			var lowerCaseTableNames int
			var unused any
			db := driver.GetDB()
			if err := db.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'lower_case_table_names'").Scan(&unused, &lowerCaseTableNames); err != nil {
				log.Warn("failed to get lower_case_table_names for instance %q", zap.Error(err), zap.String("instance", instance.Title))
				break
			}
			if lowerCaseTableNames == 1 {
				databaseName = strings.ToLower(databaseName)
			}
		}

		statement, err := getCreateDatabaseStatement(instance.Engine, c, databaseName, adminDataSource.Username)
		if err != nil {
			return nil, err
		}
		sheet, err := s.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  api.SystemBotID,
			ProjectUID: project.UID,
			Name:       fmt.Sprintf("Sheet for creating database %v", databaseName),
			Statement:  statement,
			Visibility: store.ProjectSheet,
			Source:     store.SheetFromBytebaseArtifact,
			Type:       store.SheetForSQL,
			Payload:    "{}",
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database creation sheet")
		}

		payload := api.TaskDatabaseCreatePayload{
			SpecID:       spec.Id,
			ProjectID:    project.UID,
			CharacterSet: c.CharacterSet,
			TableName:    c.Table,
			Collation:    c.Collation,
			Labels:       labelsJSON,
			DatabaseName: databaseName,
			SheetID:      sheet.UID,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database creation task, unable to marshal payload")
		}

		return []*store.TaskMessage{
			{
				InstanceID:        instance.UID,
				DatabaseID:        nil,
				Name:              fmt.Sprintf("Create database %v", payload.DatabaseName),
				Status:            api.TaskPendingApproval,
				Type:              api.TaskDatabaseCreate,
				DatabaseName:      payload.DatabaseName,
				Payload:           string(bytes),
				EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			},
		}, nil
	}()
	if err != nil {
		return nil, nil, err
	}

	return taskCreates, nil, nil
}

func getTaskCreatesFromChangeDatabaseConfig(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ChangeDatabaseConfig, _ *store.ProjectMessage, registerEnvironmentID func(string) error) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	// possible target:
	// 1. instances/{instance}/databases/{database}
	instanceID, databaseName, err := common.GetInstanceDatabaseID(c.Target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance database id from target %q", c.Target)
	}
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance %q", instanceID)
	}
	if instance == nil {
		return nil, nil, errors.Errorf("instance %q not found", instanceID)
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, nil, errors.Errorf("database %q not found", databaseName)
	}

	if err := registerEnvironmentID(database.EffectiveEnvironmentID); err != nil {
		return nil, nil, err
	}

	switch c.Type {
	case storepb.PlanConfig_ChangeDatabaseConfig_BASELINE:
		payload := api.TaskDatabaseSchemaBaselinePayload{
			SpecID:        spec.Id,
			SchemaVersion: getOrDefaultSchemaVersion(c.SchemaVersion),
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal task database schema baseline payload")
		}
		payloadString := string(bytes)
		taskCreate := &store.TaskMessage{
			Name:              fmt.Sprintf("Establish baseline for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaBaseline,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []*store.TaskMessage{taskCreate}, nil, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
		_, sheetIDStr, err := common.GetProjectResourceIDSheetID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		sheetID, err := strconv.Atoi(sheetIDStr)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert sheet id %q to int", sheetIDStr)
		}
		payload := api.TaskDatabaseSchemaUpdatePayload{
			SpecID:        spec.Id,
			SheetID:       sheetID,
			SchemaVersion: getOrDefaultSchemaVersion(c.SchemaVersion),
			VCSPushEvent:  nil,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal task database schema update payload")
		}
		payloadString := string(bytes)
		taskCreate := &store.TaskMessage{
			Name:              fmt.Sprintf("DDL(schema) for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaUpdate,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []*store.TaskMessage{taskCreate}, nil, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_SDL:
		_, sheetIDStr, err := common.GetProjectResourceIDSheetID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		sheetID, err := strconv.Atoi(sheetIDStr)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert sheet id %q to int", sheetIDStr)
		}
		payload := api.TaskDatabaseSchemaUpdateSDLPayload{
			SpecID:        spec.Id,
			SheetID:       sheetID,
			SchemaVersion: getOrDefaultSchemaVersion(c.SchemaVersion),
			VCSPushEvent:  nil,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal database schema update SDL payload")
		}
		payloadString := string(bytes)
		taskCreate := &store.TaskMessage{
			Name:              fmt.Sprintf("SDL for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaUpdateSDL,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []*store.TaskMessage{taskCreate}, nil, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST:
		_, sheetIDStr, err := common.GetProjectResourceIDSheetID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		sheetID, err := strconv.Atoi(sheetIDStr)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert sheet id %q to int", sheetIDStr)
		}
		var taskCreateList []*store.TaskMessage
		// task "sync"
		payloadSync := api.TaskDatabaseSchemaUpdateGhostSyncPayload{
			SpecID:        spec.Id,
			SheetID:       sheetID,
			SchemaVersion: c.SchemaVersion,
			VCSPushEvent:  nil,
		}
		bytesSync, err := json.Marshal(payloadSync)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal database schema update gh-ost sync payload")
		}
		taskCreateList = append(taskCreateList, &store.TaskMessage{
			Name:              fmt.Sprintf("Update schema gh-ost sync for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaUpdateGhostSync,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           string(bytesSync),
		})

		// task "cutover"
		payloadCutover := api.TaskDatabaseSchemaUpdateGhostCutoverPayload{
			SpecID: spec.Id,
		}
		bytesCutover, err := json.Marshal(payloadCutover)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal database schema update ghost cutover payload")
		}
		taskCreateList = append(taskCreateList, &store.TaskMessage{
			Name:              fmt.Sprintf("Update schema gh-ost cutover for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaUpdateGhostCutover,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           string(bytesCutover),
		})

		// The below list means that taskCreateList[0] blocks taskCreateList[1].
		// In other words, task "sync" blocks task "cutover".
		taskIndexDAGList := []store.TaskIndexDAG{
			{FromIndex: 0, ToIndex: 1},
		}
		return taskCreateList, taskIndexDAGList, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
		_, sheetIDStr, err := common.GetProjectResourceIDSheetID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		sheetID, err := strconv.Atoi(sheetIDStr)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert sheet id %q to int", sheetIDStr)
		}
		payload := api.TaskDatabaseDataUpdatePayload{
			SpecID:            spec.Id,
			SheetID:           sheetID,
			SchemaVersion:     getOrDefaultSchemaVersion(c.SchemaVersion),
			VCSPushEvent:      nil,
			RollbackEnabled:   c.RollbackEnabled,
			RollbackSQLStatus: api.RollbackSQLStatusPending,
		}
		if c.RollbackDetail != nil {
			issueID, err := common.GetIssueID(c.RollbackDetail.RollbackFromIssue)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get issue id from issue %q", c.RollbackDetail.RollbackFromIssue)
			}
			payload.RollbackFromIssueID = issueID
			taskID, err := common.GetTaskID(c.RollbackDetail.RollbackFromTask)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get task id from task %q", c.RollbackDetail.RollbackFromTask)
			}
			payload.RollbackFromTaskID = taskID
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Failed to marshal database data update payload")
		}
		payloadString := string(bytes)
		taskCreate := &store.TaskMessage{
			Name:              fmt.Sprintf("DML(data) for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseDataUpdate,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []*store.TaskMessage{taskCreate}, nil, nil
	default:
		return nil, nil, errors.Errorf("unsupported change database config type %q", c.Type)
	}
}

func getTaskCreatesFromRestoreDatabaseConfig(ctx context.Context, s *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_RestoreDatabaseConfig, project *store.ProjectMessage, registerEnvironmentID func(string) error) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	if c.Source == nil {
		return nil, nil, errors.Errorf("missing source in restore database config")
	}
	instanceID, databaseName, err := common.GetInstanceDatabaseID(c.Target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance and database id from target %q", c.Target)
	}
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance %q", instanceID)
	}
	if instance == nil {
		return nil, nil, errors.Errorf("instance %q not found", instanceID)
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, nil, errors.Errorf("database %q not found", databaseName)
	}
	if database.ProjectID != project.ResourceID {
		return nil, nil, errors.Errorf("database %q is not in project %q", databaseName, project.ResourceID)
	}

	if err := registerEnvironmentID(database.EffectiveEnvironmentID); err != nil {
		return nil, nil, err
	}

	var taskCreates []*store.TaskMessage

	if c.CreateDatabaseConfig != nil {
		restorePayload := api.TaskDatabasePITRRestorePayload{
			SpecID:    spec.Id,
			ProjectID: project.UID,
		}
		// restore to a new database
		targetInstanceID, err := common.GetInstanceID(c.CreateDatabaseConfig.Target)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get instance id from %q", c.CreateDatabaseConfig.Target)
		}
		targetInstance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &targetInstanceID})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find the instance with ID %q", targetInstanceID)
		}

		// task 1: create the database
		createDatabaseTasks, _, err := getTaskCreatesFromCreateDatabaseConfig(ctx, s, licenseService, dbFactory, spec, c.CreateDatabaseConfig, project, registerEnvironmentID)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create the database create task list")
		}
		if len(createDatabaseTasks) != 1 {
			return nil, nil, errors.Errorf("expected 1 task to create the database, got %d", len(createDatabaseTasks))
		}
		taskCreates = append(taskCreates, createDatabaseTasks[0])

		// task 2: restore the database
		switch source := c.Source.(type) {
		case *storepb.PlanConfig_RestoreDatabaseConfig_Backup:
			backupInstanceID, backupDatabaseName, backupName, err := common.GetInstanceDatabaseIDBackupName(source.Backup)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to parse backup name %q", source.Backup)
			}
			backupInstance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &backupInstanceID})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get instance %q", backupInstanceID)
			}
			backupDatabase, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:          &backupInstanceID,
				DatabaseName:        &backupDatabaseName,
				IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(backupInstance),
			})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get database %q", backupDatabaseName)
			}
			if backupDatabase == nil {
				return nil, nil, errors.Errorf("failed to find database %q where backup %q is created", backupDatabaseName, source.Backup)
			}
			backup, err := s.GetBackupV2(ctx, &store.FindBackupMessage{
				DatabaseUID: &backupDatabase.UID,
				Name:        &backupName,
			})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get backup %q", backupName)
			}
			if backup == nil {
				return nil, nil, errors.Errorf("failed to find backup %q", backupName)
			}
			restorePayload.BackupID = &backup.UID
		case *storepb.PlanConfig_RestoreDatabaseConfig_PointInTime:
			ts := source.PointInTime.GetSeconds()
			restorePayload.PointInTimeTs = &ts
		}
		restorePayload.TargetInstanceID = &targetInstance.UID
		restorePayload.DatabaseName = &c.CreateDatabaseConfig.Database

		restorePayloadBytes, err := json.Marshal(restorePayload)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create PITR restore task, unable to marshal payload")
		}

		restoreTaskCreate := &store.TaskMessage{
			Name:       fmt.Sprintf("Restore to new database %q", *restorePayload.DatabaseName),
			Status:     api.TaskPendingApproval,
			Type:       api.TaskDatabaseRestorePITRRestore,
			InstanceID: instance.UID,
			DatabaseID: &database.UID,
			Payload:    string(restorePayloadBytes),
		}
		taskCreates = append(taskCreates, restoreTaskCreate)
	} else {
		// in-place restore

		// task 1: restore
		restorePayload := api.TaskDatabasePITRRestorePayload{
			SpecID:    spec.Id,
			ProjectID: project.UID,
		}
		switch source := c.Source.(type) {
		case *storepb.PlanConfig_RestoreDatabaseConfig_Backup:
			backupInstanceID, backupDatabaseName, backupName, err := common.GetInstanceDatabaseIDBackupName(source.Backup)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to parse backup name %q", source.Backup)
			}
			backupInstance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &backupInstanceID})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get instance %q", backupInstanceID)
			}
			backupDatabase, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:          &backupInstanceID,
				DatabaseName:        &backupDatabaseName,
				IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(backupInstance),
			})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get database %q", backupDatabaseName)
			}
			if backupDatabase == nil {
				return nil, nil, errors.Errorf("failed to find database %q where backup %q is created", backupDatabaseName, source.Backup)
			}
			backup, err := s.GetBackupV2(ctx, &store.FindBackupMessage{
				DatabaseUID: &backupDatabase.UID,
				Name:        &backupName,
			})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get backup %q", backupName)
			}
			if backup == nil {
				return nil, nil, errors.Errorf("failed to find backup %q", backupName)
			}
			restorePayload.BackupID = &backup.UID
		case *storepb.PlanConfig_RestoreDatabaseConfig_PointInTime:
			ts := source.PointInTime.GetSeconds()
			restorePayload.PointInTimeTs = &ts
		}
		restorePayloadBytes, err := json.Marshal(restorePayload)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create PITR restore task, unable to marshal payload")
		}

		restoreTaskCreate := &store.TaskMessage{
			Name:       fmt.Sprintf("Restore to PITR database %q", database.DatabaseName),
			Status:     api.TaskPendingApproval,
			Type:       api.TaskDatabaseRestorePITRRestore,
			InstanceID: instance.UID,
			DatabaseID: &database.UID,
			Payload:    string(restorePayloadBytes),
		}

		taskCreates = append(taskCreates, restoreTaskCreate)

		// task 2: cutover
		cutoverPayload := api.TaskDatabasePITRCutoverPayload{
			SpecID: spec.Id,
		}
		cutoverPayloadBytes, err := json.Marshal(cutoverPayload)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create PITR cutover task, unable to marshal payload")
		}
		taskCreates = append(taskCreates, &store.TaskMessage{
			Name:       fmt.Sprintf("Swap PITR and the original database %q", database.DatabaseName),
			InstanceID: instance.UID,
			DatabaseID: &database.UID,
			Status:     api.TaskPendingApproval,
			Type:       api.TaskDatabaseRestorePITRCutover,
			Payload:    string(cutoverPayloadBytes),
		})
	}

	// We make sure that we will always return 2 tasks.
	taskIndexDAGs := []store.TaskIndexDAG{
		{
			FromIndex: 0,
			ToIndex:   1,
		},
	}
	return taskCreates, taskIndexDAGs, nil
}

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

// checkCharacterSetCollationOwner checks if the character set, collation and owner are legal according to the dbType.
func checkCharacterSetCollationOwner(dbType db.Type, characterSet, collation, owner string) error {
	switch dbType {
	case db.Spanner:
		// Spanner does not support character set and collation at the database level.
		if characterSet != "" {
			return errors.Errorf("Spanner does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("Spanner does not support collation, but got %s", collation)
		}
	case db.ClickHouse:
		// ClickHouse does not support character set and collation at the database level.
		if characterSet != "" {
			return errors.Errorf("ClickHouse does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("ClickHouse does not support collation, but got %s", collation)
		}
	case db.Snowflake:
		if characterSet != "" {
			return errors.Errorf("Snowflake does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("Snowflake does not support collation, but got %s", collation)
		}
	case db.Postgres:
		if owner == "" {
			return errors.Errorf("database owner is required for PostgreSQL")
		}
	case db.Redshift:
		if owner == "" {
			return errors.Errorf("database owner is required for Redshift")
		}
	case db.SQLite, db.MongoDB, db.MSSQL:
		// no-op.
	default:
		if characterSet == "" {
			return errors.Errorf("character set missing for %s", string(dbType))
		}
		// For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
		// If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
		// install it.
		if collation == "" {
			return errors.Errorf("collation missing for %s", string(dbType))
		}
	}
	return nil
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

func getCreateDatabaseStatement(dbType db.Type, c *storepb.PlanConfig_CreateDatabaseConfig, databaseName, adminDatasourceUser string) (string, error) {
	var stmt string
	switch dbType {
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		return fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s;", databaseName, c.CharacterSet, c.Collation), nil
	case db.MSSQL:
		return fmt.Sprintf(`CREATE DATABASE "%s";`, databaseName), nil
	case db.Postgres:
		// On Cloud RDS, the data source role isn't the actual superuser with sudo privilege.
		// We need to grant the database owner role to the data source admin so that Bytebase can have permission for the database using the data source admin.
		if adminDatasourceUser != "" && c.Owner != adminDatasourceUser {
			stmt = fmt.Sprintf("GRANT \"%s\" TO \"%s\";\n", c.Owner, adminDatasourceUser)
		}
		if c.Collation == "" {
			stmt = fmt.Sprintf("%sCREATE DATABASE \"%s\" ENCODING %q;", stmt, databaseName, c.CharacterSet)
		} else {
			stmt = fmt.Sprintf("%sCREATE DATABASE \"%s\" ENCODING %q LC_COLLATE %q;", stmt, databaseName, c.CharacterSet, c.Collation)
		}
		// Set the database owner.
		// We didn't use CREATE DATABASE WITH OWNER because RDS requires the current role to be a member of the database owner.
		// However, people can still use ALTER DATABASE to change the owner afterwards.
		// Error string below:
		// query: CREATE DATABASE h1 WITH OWNER hello;
		// ERROR:  must be member of role "hello"
		//
		// For tenant project, the schema for the newly created database will belong to the same owner.
		// TODO(d): alter schema "public" owner to the database owner.
		return fmt.Sprintf("%s\nALTER DATABASE \"%s\" OWNER TO \"%s\";", stmt, databaseName, c.Owner), nil
	case db.ClickHouse:
		clusterPart := ""
		if c.Cluster != "" {
			clusterPart = fmt.Sprintf(" ON CLUSTER `%s`", c.Cluster)
		}
		return fmt.Sprintf("CREATE DATABASE `%s`%s;", databaseName, clusterPart), nil
	case db.Snowflake:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case db.SQLite:
		// This is a fake CREATE DATABASE and USE statement since a single SQLite file represents a database. Engine driver will recognize it and establish a connection to create the sqlite file representing the database.
		return fmt.Sprintf("CREATE DATABASE '%s';", databaseName), nil
	case db.MongoDB:
		// We just run createCollection in mongosh instead of execute `use <database>` first, because we execute the
		// mongodb statement in mongosh with --file flag, and it doesn't support `use <database>` statement in the file.
		// And we pass the database name to Bytebase engine driver, which will be used to build the connection string.
		return fmt.Sprintf(`db.createCollection("%s");`, c.Table), nil
	case db.Spanner:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case db.Oracle:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case db.Redshift:
		options := make(map[string]string)
		if adminDatasourceUser != "" && c.Owner != adminDatasourceUser {
			options["OWNER"] = fmt.Sprintf("%q", c.Owner)
		}
		stmt := fmt.Sprintf("CREATE DATABASE \"%s\"", databaseName)
		if len(options) > 0 {
			list := make([]string, 0, len(options))
			for k, v := range options {
				list = append(list, fmt.Sprintf("%s=%s", k, v))
			}
			stmt = fmt.Sprintf("%s WITH\n\t%s", stmt, strings.Join(list, "\n\t"))
		}
		return fmt.Sprintf("%s;", stmt), nil
	}
	return "", errors.Errorf("unsupported database type %s", dbType)
}

func (s *RolloutService) createPipeline(ctx context.Context, project *store.ProjectMessage, pipelineCreate *store.PipelineMessage, creatorID int) (*store.PipelineMessage, error) {
	pipelineCreated, err := s.store.CreatePipelineV2(ctx, &store.PipelineMessage{
		Name:      pipelineCreate.Name,
		ProjectID: project.ResourceID,
	}, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pipeline for issue")
	}

	var stageCreates []*store.StageMessage
	for _, stage := range pipelineCreate.Stages {
		stageCreates = append(stageCreates, &store.StageMessage{
			Name:          stage.Name,
			EnvironmentID: stage.EnvironmentID,
			PipelineID:    pipelineCreated.ID,
		})
	}
	createdStages, err := s.store.CreateStageV2(ctx, stageCreates, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create stages for issue")
	}
	if len(createdStages) != len(stageCreates) {
		return nil, errors.Errorf("failed to create stages, expect to have created %d stages, got %d", len(stageCreates), len(createdStages))
	}

	for i, stageCreate := range pipelineCreate.Stages {
		createdStage := createdStages[i]

		var taskCreateList []*store.TaskMessage
		for _, taskCreate := range stageCreate.TaskList {
			c := taskCreate
			c.CreatorID = creatorID
			c.PipelineID = pipelineCreated.ID
			c.StageID = createdStage.ID
			taskCreateList = append(taskCreateList, c)
		}
		tasks, err := s.store.CreateTasksV2(ctx, taskCreateList...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create tasks for issue")
		}

		// TODO(p0ny): create task dags in batch.
		for _, indexDAG := range stageCreate.TaskIndexDAGList {
			if err := s.store.CreateTaskDAGV2(ctx, &store.TaskDAGMessage{
				FromTaskID: tasks[indexDAG.FromIndex].ID,
				ToTaskID:   tasks[indexDAG.ToIndex].ID,
			}); err != nil {
				return nil, errors.Wrap(err, "failed to create task DAG for issue")
			}
		}
	}

	return pipelineCreated, nil
}

func getResourceNameForSheet(project *store.ProjectMessage, sheetUID int) string {
	return fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, project.ResourceID, common.SheetIDPrefix, sheetUID)
}

func getOrDefaultSchemaVersion(v string) string {
	if v != "" {
		return v
	}
	return common.DefaultMigrationVersion()
}
