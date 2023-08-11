package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/runner/taskcheck"
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
	taskCheckScheduler *taskcheck.Scheduler
	planCheckScheduler *plancheck.Scheduler
	stateCfg           *state.State
	activityManager    *activity.Manager
}

// NewRolloutService returns a rollout service instance.
func NewRolloutService(store *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, taskCheckScheduler *taskcheck.Scheduler, planCheckScheduler *plancheck.Scheduler, stateCfg *state.State, activityManager *activity.Manager) *RolloutService {
	return &RolloutService{
		store:              store,
		licenseService:     licenseService,
		dbFactory:          dbFactory,
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

	rollout, err := getPipelineCreate(ctx, s.store, s.licenseService, s.dbFactory, steps, project)
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

	pipelineCreate, err := getPipelineCreate(ctx, s.store, s.licenseService, s.dbFactory, plan.Config.Steps, project)
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

	planCheckRuns, err := getPlanCheckRunsForPlan(ctx, s.store, plan)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan check runs for plan, error: %v", err)
	}
	if err := s.store.CreatePlanCheckRuns(ctx, planCheckRuns...); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan check runs, error: %v", err)
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
	for _, task := range request.Tasks {
		_, _, stageID, taskID, err := common.GetProjectIDRolloutIDStageIDTaskID(task)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		stageTasks[stageID] = append(stageTasks[stageID], taskID)
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
		}
		break
	}
	if stageToRun == nil {
		return nil, status.Errorf(codes.Internal, "failed to find the stage to run")
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

	ok, err := canUserRunStageTasks(ctx, s.store, user, issue, stageToRun.EnvironmentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the user can run tasks, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "Not allowed to run tasks")
	}

	approved, err := utils.CheckIssueApproved(issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the issue is approved, error: %v", err)
	}
	if !approved {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot run the tasks because the issue is not approved")
	}

	var taskRunCreates []*store.TaskRunMessage
	for _, task := range stageToRunTasks {
		if !taskIDsToRunMap[task.ID] {
			continue
		}
		create := &store.TaskRunMessage{
			TaskUID:   task.ID,
			Name:      fmt.Sprintf("%s %d", task.Name, time.Now().Unix()),
			CreatorID: user.ID,
		}
		taskRunCreates = append(taskRunCreates, create)
	}

	if err := s.store.CreatePendingTaskRuns(ctx, taskRunCreates...); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create pending task runs")
	}

	return &v1pb.BatchRunTasksResponse{}, nil
}

func (s *RolloutService) BatchSkipTasks(ctx context.Context, request *v1pb.BatchSkipTasksRequest) (*v1pb.BatchSkipTasksResponse, error) {
	updaterID := ctx.Value(common.PrincipalIDContextKey).(int)
	var taskUIDs []int
	for _, task := range request.Tasks {
		_, _, _, taskID, err := common.GetProjectIDRolloutIDStageIDTaskID(task)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		taskUIDs = append(taskUIDs, taskID)
	}

	if err := s.store.BatchSkipTasks(ctx, taskUIDs, "", updaterID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to skip tasks, error: %v", err)
	}
	return &v1pb.BatchSkipTasksResponse{}, nil
}

// BatchCancelTaskRuns cancels a list of task runs.
// TODO(p0ny): forbid cancel noncancellable task runs.
func (s *RolloutService) BatchCancelTaskRuns(ctx context.Context, request *v1pb.BatchCancelTaskRunsRequest) (*v1pb.BatchCancelTaskRunsResponse, error) {
	if len(request.TaskRuns) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "task runs cannot be empty")
	}

	projectID, rolloutID, stageID, _, err := common.GetProjectIDRolloutIDStageIDMaybeTaskID(request.Parent)
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

	var stage *store.StageMessage
	for i := range stages {
		if stages[i].ID == stageID {
			stage = stages[i]
			break
		}
	}
	if stage == nil {
		return nil, status.Errorf(codes.NotFound, "stage %v not found in rollout %v", stageID, rolloutID)
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %v not found", principalID)
	}

	ok, err := canUserCancelStageTaskRun(ctx, s.store, user, issue, stage.EnvironmentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the user can run tasks, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "Not allowed to run tasks")
	}

	var taskRunIDs []int
	for _, taskRun := range request.TaskRuns {
		_, _, _, _, taskRunID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(taskRun)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		taskRunIDs = append(taskRunIDs, taskRunID)
	}

	taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
		UIDs: &taskRunIDs,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list task runs, error: %v", err)
	}

	for _, taskRun := range taskRuns {
		switch taskRun.Status {
		case api.TaskRunPending:
		case api.TaskRunRunning:
		default:
			return nil, status.Errorf(codes.InvalidArgument, "taskRun %v is not pending or running", taskRun.Name)
		}
	}

	for _, taskRun := range taskRuns {
		if taskRun.Status == api.TaskRunRunning {
			if cancelFunc, ok := s.stateCfg.RunningTaskRunsCancelFunc.Load(taskRun.ID); ok {
				cancelFunc.(context.CancelFunc)()
			}
		}
	}

	if err := s.store.BatchPatchTaskRunStatus(ctx, taskRunIDs, api.TaskRunCanceled, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to batch patch task run status to canceled, error: %v", err)
	}

	return &v1pb.BatchCancelTaskRunsResponse{}, nil
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
				_, sheetIDStr, err := common.GetProjectResourceIDSheetID(config.ChangeDatabaseConfig.Sheet)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get sheet id from %q, error: %v", config.ChangeDatabaseConfig.Sheet, err)
				}
				sheetID, err := strconv.Atoi(sheetIDStr)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to convert sheet id %q to int, error: %v", sheetID, err)
				}
				sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
					UID: &sheetID,
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
			}
		}
	}
	for _, step := range newSteps {
		for _, spec := range step.Specs {
			if _, ok := oldSpecs[spec.Id]; !ok {
				added = append(added, spec)
			}
		}
	}
	for _, step := range newSteps {
		for _, spec := range step.Specs {
			if oldSpec, ok := oldSpecs[spec.Id]; ok {
				if isSpecSheetUpdated(oldSpec, spec) {
					updated = append(updated, spec)
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
	return nil
}

func getPipelineCreate(ctx context.Context, s *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, steps []*storepb.PlanConfig_Step, project *store.ProjectMessage) (*store.PipelineMessage, error) {
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
			taskCreates, taskIndexDAGCreates, err := getTaskCreatesFromSpec(ctx, s, licenseService, dbFactory, spec, project, registerEnvironmentID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get task creates from spec")
			}

			offset := len(stageCreate.TaskList)
			for i := range taskIndexDAGCreates {
				taskIndexDAGCreates[i].FromIndex += offset
				taskIndexDAGCreates[i].ToIndex += offset
			}
			stageCreate.TaskList = append(stageCreate.TaskList, taskCreates...)
			stageCreate.TaskIndexDAGList = append(stageCreate.TaskIndexDAGList, taskIndexDAGCreates...)
		}

		environment, err := s.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &stageEnvironmentID})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get environment")
		}
		stageCreate.EnvironmentID = environment.UID
		stageCreate.Name = fmt.Sprintf("%s Stage", environment.Title)

		pipelineCreate.Stages = append(pipelineCreate.Stages, stageCreate)
	}
	return pipelineCreate, nil
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

			// HACK: statement is present because the task came from a database group target plan spec.
			// we need to create the sheet and update payload.SheetID.
			if c.Statement != "" {
				sheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
					CreatorID:  api.SystemBotID,
					ProjectUID: project.UID,
					Name:       fmt.Sprintf("Sheet for task %v", c.Name),
					Statement:  c.Statement,
					Visibility: store.ProjectSheet,
					Source:     store.SheetFromBytebaseArtifact,
					Type:       store.SheetForSQL,
					Payload:    "{}",
				})
				if err != nil {
					return nil, errors.Wrapf(err, "failed to create sheet for task %v", c.Name)
				}
				switch c.Type {
				case api.TaskDatabaseSchemaUpdate:
					payload := &api.TaskDatabaseSchemaUpdatePayload{}
					if err := json.Unmarshal([]byte(c.Payload), payload); err != nil {
						return nil, errors.Wrapf(err, "failed to unmarshal payload for task %v", c.Name)
					}
					payload.SheetID = sheet.UID
					payloadBytes, err := json.Marshal(payload)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to marshal payload for task %v", c.Name)
					}
					c.Payload = string(payloadBytes)
				case api.TaskDatabaseDataUpdate:
					payload := &api.TaskDatabaseDataUpdatePayload{}
					if err := json.Unmarshal([]byte(c.Payload), payload); err != nil {
						return nil, errors.Wrapf(err, "failed to unmarshal payload for task %v", c.Name)
					}
					payload.SheetID = sheet.UID
					payloadBytes, err := json.Marshal(payload)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to marshal payload for task %v", c.Name)
					}
					c.Payload = string(payloadBytes)
				}
			}

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

// canUserRunStageTasks returns if a user can run the tasks in a stage.
func canUserRunStageTasks(ctx context.Context, s *store.Store, user *store.UserMessage, issue *store.IssueMessage, stageEnvironmentID int) (bool, error) {
	// the workspace owner and DBA roles can always run tasks.
	if user.Role == api.Owner || user.Role == api.DBA {
		return true, nil
	}
	groupValue, err := getGroupValueForIssueTypeEnvironment(ctx, s, issue.Type, stageEnvironmentID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get assignee group value for issueID %d", issue.UID)
	}
	// as the policy says, the project owner has the privilege to run.
	if groupValue == api.AssigneeGroupValueProjectOwner {
		policy, err := s.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &issue.Project.UID})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", issue.Project.UID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Owner {
				continue
			}
			for _, member := range binding.Members {
				if member.ID == user.ID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// canUserCancelStageTaskRun returns if a user can cancel the task runs in a stage.
func canUserCancelStageTaskRun(ctx context.Context, s *store.Store, user *store.UserMessage, issue *store.IssueMessage, stageEnvironmentID int) (bool, error) {
	// the workspace owner and DBA roles can always cancel task runs.
	if user.Role == api.Owner || user.Role == api.DBA {
		return true, nil
	}
	// The creator can cancel task runs.
	if user.ID == issue.Creator.ID {
		return true, nil
	}
	groupValue, err := getGroupValueForIssueTypeEnvironment(ctx, s, issue.Type, stageEnvironmentID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get assignee group value for issueID %d", issue.UID)
	}
	// as the policy says, the project owner has the privilege to cancel.
	if groupValue == api.AssigneeGroupValueProjectOwner {
		policy, err := s.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &issue.Project.UID})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", issue.Project.UID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Owner {
				continue
			}
			for _, member := range binding.Members {
				if member.ID == user.ID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func getGroupValueForIssueTypeEnvironment(ctx context.Context, s *store.Store, issueType api.IssueType, environmentID int) (api.AssigneeGroupValue, error) {
	defaultGroupValue := api.AssigneeGroupValueWorkspaceOwnerOrDBA
	policy, err := s.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return defaultGroupValue, errors.Wrapf(err, "failed to get pipeline approval policy by environmentID %d", environmentID)
	}
	if policy == nil {
		return defaultGroupValue, nil
	}

	for _, assigneeGroup := range policy.AssigneeGroupList {
		if assigneeGroup.IssueType == issueType {
			return assigneeGroup.Value, nil
		}
	}
	return defaultGroupValue, nil
}
