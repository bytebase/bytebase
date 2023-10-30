package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RolloutService represents a service for managing rollout.
type RolloutService struct {
	v1pb.UnimplementedRolloutServiceServer
	store              *store.Store
	licenseService     enterprise.LicenseService
	dbFactory          *dbfactory.DBFactory
	planCheckScheduler *plancheck.Scheduler
	stateCfg           *state.State
	activityManager    *activity.Manager
}

// NewRolloutService returns a rollout service instance.
func NewRolloutService(store *store.Store, licenseService enterprise.LicenseService, dbFactory *dbfactory.DBFactory, planCheckScheduler *plancheck.Scheduler, stateCfg *state.State, activityManager *activity.Manager) *RolloutService {
	return &RolloutService{
		store:              store,
		licenseService:     licenseService,
		dbFactory:          dbFactory,
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
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
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
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
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

	plan, err := s.store.CreatePlan(ctx, planMessage, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan, error: %v", err)
	}

	planCheckRuns, err := getPlanCheckRunsFromPlan(ctx, s.store, plan)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan check runs for plan, error: %v", err)
	}
	if err := s.store.CreatePlanCheckRuns(ctx, planCheckRuns...); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan check runs, error: %v", err)
	}

	// Tickle plan check scheduler.
	s.stateCfg.PlanCheckTickleChan <- 0

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

	rollout, err := GetPipelineCreate(ctx, s.store, s.licenseService, s.dbFactory, steps, project)
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
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
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
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found for id: %d", planID)
	}

	pipelineCreate, err := GetPipelineCreate(ctx, s.store, s.licenseService, s.dbFactory, plan.Config.Steps, project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get pipeline create, error: %v", err)
	}
	if len(pipelineCreate.Stages) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no database matched for deployment")
	}
	pipeline, err := s.createPipeline(ctx, project, pipelineCreate, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create pipeline, error: %v", err)
	}

	// Update pipeline ID in the plan.
	if err := s.store.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:         planID,
		UpdaterID:   principalID,
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
		}, principalID); err != nil {
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

	// Tickle task run scheduler.
	s.stateCfg.TaskRunTickleChan <- 0

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
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found")
	}

	planCheckRuns, err := getPlanCheckRunsFromPlan(ctx, s.store, plan)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan check runs for plan, error: %v", err)
	}
	if err := s.store.CreatePlanCheckRuns(ctx, planCheckRuns...); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan check runs, error: %v", err)
	}

	// Tickle plan check scheduler.
	s.stateCfg.PlanCheckTickleChan <- 0

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
		TaskRuns:      convertToTaskRuns(s.stateCfg, taskRuns),
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

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %v not found", principalID)
	}

	ok, err = canUserRunStageTasks(ctx, s.store, user, issue, stageToRun.EnvironmentID)
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
	var tasksToRun []*store.TaskMessage
	for _, task := range stageToRunTasks {
		if !taskIDsToRunMap[task.ID] {
			continue
		}
		tasksToRun = append(tasksToRun, task)
		create := &store.TaskRunMessage{
			TaskUID:   task.ID,
			Name:      fmt.Sprintf("%s %d", task.Name, time.Now().Unix()),
			CreatorID: user.ID,
		}
		taskRunCreates = append(taskRunCreates, create)
	}
	sort.Slice(taskRunCreates, func(i, j int) bool {
		return taskRunCreates[i].TaskUID < taskRunCreates[j].TaskUID
	})

	if err := s.store.CreatePendingTaskRuns(ctx, taskRunCreates...); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create pending task runs, error %v", err)
	}

	if err := s.activityManager.BatchCreateActivitiesForRunTasks(ctx, tasksToRun, issue, request.Reason, user.ID); err != nil {
		slog.Error("failed to batch create activities for running tasks", log.BBError(err))
	}

	// Tickle task run scheduler.
	s.stateCfg.TaskRunTickleChan <- 0

	return &v1pb.BatchRunTasksResponse{}, nil
}

// BatchSkipTasks skips tasks in batch.
func (s *RolloutService) BatchSkipTasks(ctx context.Context, request *v1pb.BatchSkipTasksRequest) (*v1pb.BatchSkipTasksResponse, error) {
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

	tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: &rolloutID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tasks, error: %v", err)
	}

	taskByID := make(map[int]*store.TaskMessage)
	for _, task := range tasks {
		taskByID[task.ID] = task
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	var taskUIDs []int
	var tasksToSkip []*store.TaskMessage
	for _, task := range request.Tasks {
		_, _, _, taskID, err := common.GetProjectIDRolloutIDStageIDTaskID(task)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if _, ok := taskByID[taskID]; !ok {
			return nil, status.Errorf(codes.NotFound, "task %v not found in the rollout", taskID)
		}
		taskUIDs = append(taskUIDs, taskID)
		tasksToSkip = append(tasksToSkip, taskByID[taskID])
	}

	if err := s.store.BatchSkipTasks(ctx, taskUIDs, request.Reason, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to skip tasks, error: %v", err)
	}

	for _, task := range tasksToSkip {
		s.stateCfg.TaskSkippedOrDoneChan <- task.ID
	}

	if err := s.activityManager.BatchCreateActivitiesForSkipTasks(ctx, tasksToSkip, issue, request.Reason, principalID); err != nil {
		slog.Error("failed to batch create activities for skipping tasks", log.BBError(err))
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

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %v not found", principalID)
	}

	ok, err = canUserCancelStageTaskRun(ctx, s.store, user, issue, stage.EnvironmentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the user can run tasks, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "Not allowed to run tasks")
	}

	var taskRunIDs []int
	var taskIDs []int
	for _, taskRun := range request.TaskRuns {
		_, _, _, taskID, taskRunID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(taskRun)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		taskIDs = append(taskIDs, taskID)
		taskRunIDs = append(taskRunIDs, taskRunID)
	}

	tasks, err := s.store.ListTasks(ctx, &api.TaskFind{IDs: &taskIDs})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tasks, error: %v", err)
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

	if err := s.store.BatchCancelTaskRuns(ctx, taskRunIDs, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to batch patch task run status to canceled, error: %v", err)
	}

	if err := s.activityManager.BatchCreateActivitiesForCancelTaskRuns(ctx, tasks, issue, request.Reason, principalID); err != nil {
		slog.Error("failed to batch create activities for cancel task runs", log.BBError(err))
	}

	return &v1pb.BatchCancelTaskRunsResponse{}, nil
}

// UpdatePlan updates a plan.
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
	oldPlan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan %q: %v", request.Plan.Name, err)
	}
	if oldPlan == nil {
		return nil, status.Errorf(codes.NotFound, "plan %q not found", request.Plan.Name)
	}
	oldSteps := convertToPlanSteps(oldPlan.Config.Steps)

	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PlanUID: &oldPlan.UID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue: %v", err)
	}

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

	tasksMap := map[int]*store.TaskMessage{}
	var taskPatchList []*api.TaskPatch
	var statementUpdates []api.ActivityPipelineTaskStatementUpdatePayload
	var earliestUpdates []api.ActivityPipelineTaskEarliestAllowedTimeUpdatePayload
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if oldPlan.PipelineUID != nil {
		tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: oldPlan.PipelineUID})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list tasks: %v", err)
		}
		for _, task := range tasks {
			doUpdate := false
			taskPatch := &api.TaskPatch{
				ID:        task.ID,
				UpdaterID: principalID,
			}
			tasksMap[task.ID] = task

			var taskSpecID struct {
				SpecID string `json:"specId"`
			}
			if err := json.Unmarshal([]byte(task.Payload), &taskSpecID); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
			}
			spec, ok := updatedByID[taskSpecID.SpecID]
			if !ok {
				continue
			}

			// EarliestAllowedTs
			if spec.EarliestAllowedTime.GetSeconds() != task.EarliestAllowedTs {
				seconds := spec.EarliestAllowedTime.GetSeconds()
				taskPatch.EarliestAllowedTs = &seconds
				doUpdate = true
				earliestUpdates = append(earliestUpdates, api.ActivityPipelineTaskEarliestAllowedTimeUpdatePayload{
					TaskID:               task.ID,
					OldEarliestAllowedTs: task.EarliestAllowedTs,
					NewEarliestAllowedTs: seconds,
					IssueName:            issue.Title,
					TaskName:             task.Name,
				})
			}

			// RollbackEnabled
			if err := func() error {
				if task.Type != api.TaskDatabaseDataUpdate {
					return nil
				}
				payload := &api.TaskDatabaseDataUpdatePayload{}
				if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
					return status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
				}
				config, ok := spec.Config.(*v1pb.Plan_Spec_ChangeDatabaseConfig)
				if !ok {
					return nil
				}
				if config.ChangeDatabaseConfig.RollbackEnabled != payload.RollbackEnabled {
					taskPatch.RollbackEnabled = &config.ChangeDatabaseConfig.RollbackEnabled
					doUpdate = true
				}
				return nil
			}(); err != nil {
				return nil, err
			}

			// Sheet
			if err := func() error {
				switch task.Type {
				case api.TaskDatabaseSchemaUpdate, api.TaskDatabaseSchemaUpdateSDL, api.TaskDatabaseSchemaUpdateGhostSync, api.TaskDatabaseDataUpdate:
					var taskPayload struct {
						SpecID  string `json:"specId"`
						SheetID int    `json:"sheetId"`
					}
					if err := json.Unmarshal([]byte(task.Payload), &taskPayload); err != nil {
						return status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
					}
					config, ok := spec.Config.(*v1pb.Plan_Spec_ChangeDatabaseConfig)
					if !ok {
						return nil
					}
					_, sheetUID, err := common.GetProjectResourceIDSheetUID(config.ChangeDatabaseConfig.Sheet)
					if err != nil {
						return status.Errorf(codes.Internal, "failed to get sheet id from %q, error: %v", config.ChangeDatabaseConfig.Sheet, err)
					}
					if taskPayload.SheetID == sheetUID {
						return nil
					}

					sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
						UID: &sheetUID,
					}, api.SystemBotID)
					if err != nil {
						return status.Errorf(codes.Internal, "failed to get sheet %q: %v", config.ChangeDatabaseConfig.Sheet, err)
					}
					if sheet == nil {
						return status.Errorf(codes.NotFound, "sheet %q not found", config.ChangeDatabaseConfig.Sheet)
					}
					doUpdate = true
					// TODO(p0ny): update schema version
					taskPatch.SheetID = &sheet.UID
					statementUpdates = append(statementUpdates, api.ActivityPipelineTaskStatementUpdatePayload{
						TaskID:     task.ID,
						OldSheetID: taskPayload.SheetID,
						NewSheetID: sheet.UID,
						TaskName:   task.Name,
						IssueName:  issue.Title,
					})
				}
				return nil
			}(); err != nil {
				return nil, err
			}

			if !doUpdate {
				continue
			}

			taskPatchList = append(taskPatchList, taskPatch)
		}
	}

	for _, taskPatch := range taskPatchList {
		if taskPatch.SheetID != nil || taskPatch.EarliestAllowedTs != nil {
			task := tasksMap[taskPatch.ID]
			if task.LatestTaskRunStatus == api.TaskRunPending || task.LatestTaskRunStatus == api.TaskRunRunning {
				return nil, status.Errorf(codes.FailedPrecondition, "cannot update plan because task %q is %s", task.Name, task.LatestTaskRunStatus)
			}
		}
	}

	var doUpdateSheet bool
	for _, taskPatch := range taskPatchList {
		if taskPatch.SheetID != nil {
			doUpdateSheet = true
			break
		}
	}

	for _, taskPatch := range taskPatchList {
		task := tasksMap[taskPatch.ID]
		if _, err := s.store.UpdateTaskV2(ctx, taskPatch); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update task %q: %v", task.Name, err)
		}

		taskPatched, err := s.store.GetTaskV2ByID(ctx, task.ID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get updated task %q: %v", task.Name, err)
		}

		// enqueue or cancel after it's written to the database.
		if taskPatch.RollbackEnabled != nil {
			// Enqueue the rollback sql generation if the task done.
			if *taskPatch.RollbackEnabled && taskPatched.LatestTaskRunStatus == api.TaskRunDone {
				s.stateCfg.RollbackGenerate.Store(taskPatched.ID, taskPatched)
			} else if !*taskPatch.RollbackEnabled {
				// Cancel running rollback sql generation.
				if v, ok := s.stateCfg.RollbackCancel.Load(taskPatched.ID); ok {
					if cancel, ok := v.(context.CancelFunc); ok {
						cancel()
					}
					// We don't erase the keys for RollbackCancel and RollbackGenerate here because they will eventually be erased by the rollback runner.
				}
			}
		}
	}

	if err := s.store.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       oldPlan.UID,
		UpdaterID: principalID,
		Config: &storepb.PlanConfig{
			Steps: convertPlanSteps(request.Plan.Steps),
		},
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update plan %q: %v", request.Plan.Name, err)
	}

	updatedPlan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &oldPlan.UID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated plan %q: %v", request.Plan.Name, err)
	}
	if updatedPlan == nil {
		return nil, status.Errorf(codes.NotFound, "updated plan %q not found", request.Plan.Name)
	}

	if doUpdateSheet {
		planCheckRuns, err := getPlanCheckRunsFromPlan(ctx, s.store, updatedPlan)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get plan check runs for plan, error: %v", err)
		}
		if err := s.store.CreatePlanCheckRuns(ctx, planCheckRuns...); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create plan check runs, error: %v", err)
		}
	}

	for _, statementUpdate := range statementUpdates {
		task := tasksMap[statementUpdate.TaskID]
		if err := func() error {
			payload, err := json.Marshal(statementUpdate)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal payload")
			}
			_, err = s.activityManager.CreateActivity(ctx, &store.ActivityMessage{
				CreatorUID:   principalID,
				ContainerUID: task.PipelineID,
				Type:         api.ActivityPipelineTaskStatementUpdate,
				Payload:      string(payload),
				Level:        api.ActivityInfo,
			}, &activity.Metadata{
				Issue: issue,
			})
			return errors.Wrapf(err, "failed to create activity")
		}(); err != nil {
			slog.Error("failed to create statement update activity after updating plan", log.BBError(err))
		}
	}
	for _, earliestUpdate := range earliestUpdates {
		task := tasksMap[earliestUpdate.TaskID]
		if err := func() error {
			payload, err := json.Marshal(earliestUpdate)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal payload")
			}
			_, err = s.activityManager.CreateActivity(ctx, &store.ActivityMessage{
				CreatorUID:   principalID,
				ContainerUID: task.PipelineID,
				Type:         api.ActivityPipelineTaskEarliestAllowedTimeUpdate,
				Payload:      string(payload),
				Level:        api.ActivityInfo,
			}, &activity.Metadata{
				Issue: issue,
			})
			return errors.Wrapf(err, "failed to create activity")
		}(); err != nil {
			slog.Error("failed to create earliest update activity after updating plan", log.BBError(err))
		}
	}

	if issue != nil && doUpdateSheet {
		if err := func() error {
			issue, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
				PayloadUpsert: &storepb.IssuePayload{
					Approval: &storepb.IssuePayloadApproval{
						ApprovalFindingDone: false,
					},
				},
			}, api.SystemBotID)
			if err != nil {
				return errors.Errorf("failed to update issue: %v", err)
			}
			s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
			return nil
		}(); err != nil {
			slog.Error("failed to update issue to refind approval", log.BBError(err))
		}
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
				if !cmp.Equal(oldSpec, spec, protocmp.Transform()) {
					updated = append(updated, spec)
				}
			}
		}
	}
	return removed, added, updated
}

func validateSteps(steps []*v1pb.Plan_Step) error {
	var databaseTarget, databaseGroupTarget, deploymentConfigTarget int
	for _, step := range steps {
		for _, spec := range step.Specs {
			if config := spec.GetChangeDatabaseConfig(); config != nil {
				if _, _, err := common.GetInstanceDatabaseID(config.Target); err == nil {
					databaseTarget++
				} else if _, _, err := common.GetProjectIDDatabaseGroupID(config.Target); err == nil {
					databaseGroupTarget++
				} else if _, _, err := common.GetProjectIDDeploymentConfigID(config.Target); err == nil {
					deploymentConfigTarget++
				} else {
					return errors.Errorf("unknown target %q", config.Target)
				}
			}
		}
	}
	if deploymentConfigTarget > 1 {
		return errors.Errorf("expect at most on deploymentConfig target, got %d", deploymentConfigTarget)
	}
	if deploymentConfigTarget != 0 && (databaseTarget > 0 || databaseGroupTarget > 0) {
		return errors.Errorf("expect no database or databaseGroup target when deploymentConfig target is set")
	}
	return nil
}

// GetPipelineCreate gets a pipeline create message from a plan.
func GetPipelineCreate(ctx context.Context, s *store.Store, licenseService enterprise.LicenseService, dbFactory *dbfactory.DBFactory, steps []*storepb.PlanConfig_Step, project *store.ProjectMessage) (*store.PipelineMessage, error) {
	pipelineCreate := &store.PipelineMessage{
		Name: "Rollout Pipeline",
	}

	transformedSteps := steps
	if len(steps) == 1 && len(steps[0].Specs) == 1 {
		spec := steps[0].Specs[0]
		if config := spec.GetChangeDatabaseConfig(); config != nil {
			if _, _, err := common.GetProjectIDDeploymentConfigID(config.Target); err == nil {
				stepsFromDeploymentConfig, err := transformDeploymentConfigTargetToSteps(ctx, s, spec, config, project)
				if err != nil {
					return nil, errors.Wrap(err, "failed to transform deploymentConfig target to steps")
				}
				transformedSteps = stepsFromDeploymentConfig
			}
		}
	}

	for _, step := range transformedSteps {
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
		if environment == nil {
			return nil, errors.Errorf("environment %q not found", stageEnvironmentID)
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

	p, err := s.GetRolloutPolicy(ctx, stageEnvironmentID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get rollout policy for stageEnvironmentID %d", stageEnvironmentID)
	}

	policy, err := s.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &issue.Project.UID})
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", issue.Project.UID)
	}
	userProjectRoles := map[api.Role]bool{}
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member.ID == user.ID {
				userProjectRoles[binding.Role] = true
				break
			}
		}
	}

	if p.Automatic && len(userProjectRoles) > 0 {
		return true, nil
	}

	for _, role := range p.ProjectRoles {
		apiRole := api.Role(strings.TrimPrefix(role, "roles/"))
		if userProjectRoles[apiRole] {
			return true, nil
		}
	}

	if user.ID == issue.Creator.ID {
		for _, issueRole := range p.IssueRoles {
			if issueRole == "roles/CREATOR" {
				return true, nil
			}
		}
	}

	if lastApproverUID := getLastApproverUID(issue.Payload.GetApproval()); lastApproverUID != nil && *lastApproverUID == user.ID {
		for _, issueRole := range p.IssueRoles {
			if issueRole == "roles/LAST_APPROVER" {
				return true, nil
			}
		}
	}

	return false, nil
}

// canUserCancelStageTaskRun returns if a user can cancel the task runs in a stage.
func canUserCancelStageTaskRun(ctx context.Context, s *store.Store, user *store.UserMessage, issue *store.IssueMessage, stageEnvironmentID int) (bool, error) {
	// The creator can cancel task runs.
	if user.ID == issue.Creator.ID {
		return true, nil
	}
	return canUserRunStageTasks(ctx, s, user, issue, stageEnvironmentID)
}

func getLastApproverUID(approval *storepb.IssuePayloadApproval) *int {
	if approval == nil {
		return nil
	}
	if !approval.ApprovalFindingDone {
		return nil
	}
	if approval.ApprovalFindingError != "" {
		return nil
	}
	if len(approval.Approvers) > 0 {
		id := int(approval.Approvers[len(approval.Approvers)-1].PrincipalId)
		return &id
	}
	return nil
}
