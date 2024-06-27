package v1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RolloutService represents a service for managing rollout.
type RolloutService struct {
	v1pb.UnimplementedRolloutServiceServer
	store          *store.Store
	sheetManager   *sheet.Manager
	licenseService enterprise.LicenseService
	dbFactory      *dbfactory.DBFactory
	stateCfg       *state.State
	webhookManager *webhook.Manager
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewRolloutService returns a rollout service instance.
func NewRolloutService(store *store.Store, sheetManager *sheet.Manager, licenseService enterprise.LicenseService, dbFactory *dbfactory.DBFactory, stateCfg *state.State, webhookManager *webhook.Manager, profile *config.Profile, iamManager *iam.Manager) *RolloutService {
	return &RolloutService{
		store:          store,
		sheetManager:   sheetManager,
		licenseService: licenseService,
		dbFactory:      dbFactory,
		stateCfg:       stateCfg,
		webhookManager: webhookManager,
		profile:        profile,
		iamManager:     iamManager,
	}
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

	rollout, err := GetPipelineCreate(ctx, s.store, s.sheetManager, s.licenseService, s.dbFactory, steps, project)
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
	if rollout == nil {
		return nil, status.Errorf(codes.NotFound, "rollout not found for id: %d", rolloutID)
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

	planID, err := common.GetPlanID(request.GetRollout().GetPlan())
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

	pipelineCreate, err := GetPipelineCreate(ctx, s.store, s.sheetManager, s.licenseService, s.dbFactory, plan.Config.Steps, project)
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

	taskRunsV1, err := convertToTaskRuns(ctx, s.store, s.stateCfg, taskRuns)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to task runs, error: %v", err)
	}
	return &v1pb.ListTaskRunsResponse{
		TaskRuns:      taskRunsV1,
		NextPageToken: "",
	}, nil
}

func (s *RolloutService) GetTaskRunLog(ctx context.Context, request *v1pb.GetTaskRunLogRequest) (*v1pb.TaskRunLog, error) {
	_, _, _, _, taskRunUID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get task run uid, error: %v", err)
	}
	logs, err := s.store.ListTaskRunLogs(ctx, taskRunUID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to list task run logs, error: %v", err)
	}
	return convertToTaskRunLog(request.Parent, logs), nil
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

	stageToRunTasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: &rolloutID, StageID: &stageToRun.ID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tasks, error: %v", err)
	}
	if len(stageToRunTasks) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "No tasks to run in the stage")
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	role, ok := ctx.Value(common.RoleContextKey).(api.Role)
	if !ok {
		return nil, status.Errorf(codes.Internal, "role not found")
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
	if !approved && !(role == api.WorkspaceAdmin || role == api.WorkspaceDBA) {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot run the tasks because the issue is not approved")
	}

	var taskRunCreates []*store.TaskRunMessage
	for _, task := range stageToRunTasks {
		if !taskIDsToRunMap[task.ID] {
			continue
		}

		sheetUID, err := api.GetSheetUIDFromTaskPayload(task.Payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get sheet uid from task payload, error: %v", err)
		}
		create := &store.TaskRunMessage{
			TaskUID:   task.ID,
			SheetUID:  sheetUID,
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

	if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, request.Tasks, storepb.IssueCommentPayload_TaskUpdate_PENDING, user.ID); err != nil {
		slog.Warn("failed to create issue comment", "issueUID", issue.UID, log.BBError(err))
	}

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    webhook.EventTypeTaskRunStatusUpdate,
		Comment: request.Reason,
		Issue:   webhook.NewIssue(issue),
		Project: webhook.NewProject(issue.Project),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
			Title:  issue.Title,
			Status: api.TaskRunPending.String(),
		},
	})

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

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	var taskUIDs []int
	var tasksToSkip []*store.TaskMessage
	stageIDSet := map[int]struct{}{}
	for _, task := range request.Tasks {
		_, _, stageID, taskID, err := common.GetProjectIDRolloutIDStageIDTaskID(task)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if _, ok := taskByID[taskID]; !ok {
			return nil, status.Errorf(codes.NotFound, "task %v not found in the rollout", taskID)
		}
		taskUIDs = append(taskUIDs, taskID)
		tasksToSkip = append(tasksToSkip, taskByID[taskID])
		stageIDSet[stageID] = struct{}{}
	}

	stages, err := s.store.ListStageV2(ctx, rolloutID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list stages, error: %v", err)
	}
	stageMap := map[int]*store.StageMessage{}
	for _, stage := range stages {
		stageMap[stage.ID] = stage
	}

	for stageID := range stageIDSet {
		stage, ok := stageMap[stageID]
		if !ok {
			return nil, status.Errorf(codes.Internal, "stage ID %v not found in stages of rollout %v", stageID, rolloutID)
		}
		ok, err = canUserSkipStageTasks(ctx, s.store, user, issue, stage.EnvironmentID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check if the user can run tasks, error: %v", err)
		}
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "not allowed to skip tasks in stage %q", stage.Name)
		}
	}

	if err := s.store.BatchSkipTasks(ctx, taskUIDs, request.Reason, user.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to skip tasks, error: %v", err)
	}

	for _, task := range tasksToSkip {
		s.stateCfg.TaskSkippedOrDoneChan <- task.ID
	}

	if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, request.Tasks, storepb.IssueCommentPayload_TaskUpdate_SKIPPED, user.ID); err != nil {
		slog.Warn("failed to create issue comment", "issueUID", issue.UID, log.BBError(err))
	}

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    webhook.EventTypeTaskRunStatusUpdate,
		Comment: request.Reason,
		Issue:   webhook.NewIssue(issue),
		Project: webhook.NewProject(issue.Project),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
			Title:         issue.Title,
			Status:        api.TaskRunSkipped.String(),
			SkippedReason: request.Reason,
		},
	})

	return &v1pb.BatchSkipTasksResponse{}, nil
}

// BatchCancelTaskRuns cancels a list of task runs.
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
		return nil, status.Errorf(codes.PermissionDenied, "Not allowed to cancel tasks")
	}

	var taskRunIDs []int
	var taskNames []string
	for _, taskRun := range request.TaskRuns {
		projectID, rolloutID, stageID, taskID, taskRunID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(taskRun)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		taskRunIDs = append(taskRunIDs, taskRunID)
		taskNames = append(taskNames, common.FormatTask(projectID, rolloutID, stageID, taskID))
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

	if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, taskNames, storepb.IssueCommentPayload_TaskUpdate_CANCELED, user.ID); err != nil {
		slog.Warn("failed to create issue comment", "issueUID", issue.UID, log.BBError(err))
	}

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    webhook.EventTypeTaskRunStatusUpdate,
		Comment: request.Reason,
		Issue:   webhook.NewIssue(issue),
		Project: webhook.NewProject(issue.Project),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
			Title:  issue.Title,
			Status: api.TaskRunCanceled.String(),
		},
	})

	return &v1pb.BatchCancelTaskRunsResponse{}, nil
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
	if len(steps) == 0 {
		return errors.Errorf("the plan has zero step")
	}
	var databaseTarget, databaseGroupTarget, deploymentConfigTarget int
	seenID := map[string]bool{}
	for _, step := range steps {
		if len(step.Specs) == 0 {
			return errors.Errorf("the plan step has zero spec")
		}
		seenIDInStep := map[string]bool{}
		for _, spec := range step.Specs {
			id := spec.GetId()
			if id == "" {
				return errors.Errorf("spec id cannot be empty")
			}
			if seenID[id] {
				return errors.Errorf("found duplicate spec id %q", spec.GetId())
			}
			seenID[id] = true
			seenIDInStep[id] = true
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
		for _, spec := range step.Specs {
			for _, dependOnSpec := range spec.DependsOnSpecs {
				if !seenIDInStep[dependOnSpec] {
					return errors.Errorf("spec %q depends on spec %q, but spec %q is not found in the step", spec.Id, dependOnSpec, dependOnSpec)
				}
				if dependOnSpec == spec.Id {
					return errors.Errorf("spec %q depends on itself", spec.Id)
				}
			}
		}
	}
	if deploymentConfigTarget > 1 {
		return errors.Errorf("expect at most 1 deploymentConfig target, got %d", deploymentConfigTarget)
	}
	if deploymentConfigTarget != 0 && (databaseTarget > 0 || databaseGroupTarget > 0) {
		return errors.Errorf("expect no other kinds of targets if there is deploymentConfig target")
	}
	if databaseGroupTarget > 1 {
		return errors.Errorf("expect at most 1 databaseGroup target, got %d", databaseGroupTarget)
	}
	if databaseGroupTarget > 0 && (databaseTarget > 0 || deploymentConfigTarget > 0) {
		return errors.Errorf("expect no other kinds of targets if there is databaseGroup target")
	}
	return nil
}

// GetPipelineCreate gets a pipeline create message from a plan.
func GetPipelineCreate(ctx context.Context, s *store.Store, sheetManager *sheet.Manager, licenseService enterprise.LicenseService, dbFactory *dbfactory.DBFactory, steps []*storepb.PlanConfig_Step, project *store.ProjectMessage) (*store.PipelineMessage, error) {
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
			if _, _, err := common.GetProjectIDDatabaseGroupID(config.Target); err == nil {
				stepsFromDatabaseGroup, err := transformDatabaseGroupTargetToSteps(ctx, s, spec, config, project)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to transform databaseGroup target to steps")
				}
				transformedSteps = stepsFromDatabaseGroup
			}
		}
	}

	pipelineCreate := &store.PipelineMessage{
		Name: "Rollout Pipeline",
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

		taskIndexesBySpecID := map[string][]int{}
		for _, spec := range step.Specs {
			taskCreates, taskIndexDAGCreates, err := getTaskCreatesFromSpec(ctx, s, sheetManager, licenseService, dbFactory, spec, project, registerEnvironmentID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get task creates from spec")
			}

			offset := len(stageCreate.TaskList)
			for i := range taskCreates {
				taskIndexesBySpecID[spec.Id] = append(taskIndexesBySpecID[spec.Id], i+offset)
			}
			for i := range taskIndexDAGCreates {
				taskIndexDAGCreates[i].FromIndex += offset
				taskIndexDAGCreates[i].ToIndex += offset
			}
			stageCreate.TaskList = append(stageCreate.TaskList, taskCreates...)
			stageCreate.TaskIndexDAGList = append(stageCreate.TaskIndexDAGList, taskIndexDAGCreates...)
		}
		stageCreate.TaskIndexDAGList = append(stageCreate.TaskIndexDAGList, getTaskIndexDAGs(step.Specs, func(specID string) []int {
			return taskIndexesBySpecID[specID]
		})...)

		environment, err := s.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &stageEnvironmentID})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get environment")
		}
		if environment == nil {
			return nil, errors.Errorf("environment %q not found", stageEnvironmentID)
		}
		stageCreate.EnvironmentID = environment.UID
		stageCreate.Name = fmt.Sprintf("%s Stage", environment.Title)
		if step.Title != "" {
			stageCreate.Name = step.Title
		}

		pipelineCreate.Stages = append(pipelineCreate.Stages, stageCreate)
	}
	return pipelineCreate, nil
}

func getTaskIndexDAGs(specs []*storepb.PlanConfig_Spec, getTaskIndexes func(specID string) []int) []store.TaskIndexDAG {
	var taskIndexDAGs []store.TaskIndexDAG
	for _, spec := range specs {
		for _, dependsOnSpec := range spec.DependsOnSpecs {
			for _, dependsOnTask := range getTaskIndexes(dependsOnSpec) {
				for _, task := range getTaskIndexes(spec.Id) {
					taskIndexDAGs = append(taskIndexDAGs, store.TaskIndexDAG{
						FromIndex: dependsOnTask,
						ToIndex:   task,
					})
				}
			}
		}
	}
	return taskIndexDAGs
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

// canUserRunStageTasks returns if a user can run the tasks in a stage.
func canUserRunStageTasks(ctx context.Context, s *store.Store, user *store.UserMessage, issue *store.IssueMessage, stageEnvironmentID int) (bool, error) {
	// For data export issues, only the creator can run tasks.
	if issue.Type == api.IssueDatabaseDataExport {
		return issue.Creator.ID == user.ID, nil
	}

	// The workspace owner and DBA roles can always run tasks.
	if slices.Contains(user.Roles, api.WorkspaceAdmin) || slices.Contains(user.Roles, api.WorkspaceDBA) {
		return true, nil
	}

	p, err := s.GetRolloutPolicy(ctx, stageEnvironmentID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get rollout policy for stageEnvironmentID %d", stageEnvironmentID)
	}

	policy, err := s.GetProjectIamPolicy(ctx, issue.Project.UID)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", issue.Project.UID)
	}

	roles, err := utils.GetUserFormattedRolesMap(ctx, s, user, policy)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get roles")
	}

	if p.Automatic {
		return true, nil
	}

	for _, role := range p.WorkspaceRoles {
		if roles[role] {
			return true, nil
		}
	}
	for _, role := range p.ProjectRoles {
		if roles[role] {
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
	return canUserRunStageTasks(ctx, s, user, issue, stageEnvironmentID)
}

func canUserSkipStageTasks(ctx context.Context, s *store.Store, user *store.UserMessage, issue *store.IssueMessage, stageEnvironmentID int) (bool, error) {
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
