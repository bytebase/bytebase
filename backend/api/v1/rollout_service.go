package v1

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgtype"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	rollout, err := GetPipelineCreate(ctx, s.store, s.sheetManager, s.licenseService, s.dbFactory, request.GetPlan().GetName(), steps, nil /* snapshot */, project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get pipeline create, error: %v", err)
	}
	if len(rollout.Stages) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "plan has no stage created, hint: check deployment config setting that the target database is in a stage")
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

// ListRollouts lists rollouts.
func (s *RolloutService) ListRollouts(ctx context.Context, request *v1pb.ListRolloutsRequest) (*v1pb.ListRolloutsResponse, error) {
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.PipelineFind{
		ProjectID: &projectID,
		Limit:     &limitPlusOne,
		Offset:    &offset.offset,
	}
	pipelines, err := s.store.ListPipelineV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list pipelines, error: %v", err)
	}

	var nextPageToken string
	// has more pages
	if len(pipelines) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		pipelines = pipelines[:offset.limit]
	}

	rollouts := []*v1pb.Rollout{}
	for _, pipeline := range pipelines {
		rolloutMessage, err := s.store.GetRollout(ctx, pipeline.ID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get rollout, error: %v", err)
		}
		if rolloutMessage == nil {
			return nil, status.Errorf(codes.Internal, "failed to get rollout %d", pipeline.ID)
		}
		rollout, err := convertToRollout(ctx, s.store, project, rolloutMessage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert to rollout, error: %v", err)
		}
		rollouts = append(rollouts, rollout)
	}

	return &v1pb.ListRolloutsResponse{
		Rollouts:      rollouts,
		NextPageToken: nextPageToken,
	}, nil
}

// CreateRollout creates a rollout from plan.
func (s *RolloutService) CreateRollout(ctx context.Context, request *v1pb.CreateRolloutRequest) (*v1pb.Rollout, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	_, planID, err := common.GetProjectIDPlanID(request.GetRollout().GetPlan())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found for id: %d", planID)
	}

	rolloutTitle := request.GetRollout().GetTitle()
	if rolloutTitle == "" {
		rolloutTitle = plan.Name
	}
	pipelineCreate, err := GetPipelineCreate(ctx, s.store, s.sheetManager, s.licenseService, s.dbFactory, rolloutTitle, plan.Config.GetSteps(), plan.Config.GetDeployment(), project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get pipeline create, error: %v", err)
	}
	if len(pipelineCreate.Stages) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no database matched for deployment, hint: check deployment config setting that the target database is in a stage")
	}
	if isChangeDatabasePlan(plan.Config.GetSteps()) {
		pipelineCreate, err = getPipelineCreateToTargetStage(ctx, s.store, plan.Config.GetDeployment().GetEnvironments(), pipelineCreate, request.Target)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to filter stages with stageId, error: %v", err)
		}
	}
	if request.ValidateOnly {
		rolloutV1, err := convertToRollout(ctx, s.store, project, pipelineCreate)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert to rollout, error: %v", err)
		}
		rolloutV1.Plan = request.Rollout.GetPlan()
		return rolloutV1, nil
	}
	pipelineUID, err := s.store.CreatePipelineAIO(ctx, planID, pipelineCreate, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create pipeline, error: %v", err)
	}

	rollout, err := s.store.GetRollout(ctx, pipelineUID)
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

// GetTaskRun gets a task run.
func (s *RolloutService) GetTaskRun(ctx context.Context, request *v1pb.GetTaskRunRequest) (*v1pb.TaskRun, error) {
	_, _, _, _, taskRunUID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	taskRun, err := s.store.GetTaskRun(ctx, taskRunUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get task run, error: %v", err)
	}
	taskRunV1, err := convertToTaskRun(ctx, s.store, s.stateCfg, taskRun)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to task run, error: %v", err)
	}
	return taskRunV1, nil
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

func (s *RolloutService) GetTaskRunSession(ctx context.Context, request *v1pb.GetTaskRunSessionRequest) (*v1pb.TaskRunSession, error) {
	_, _, _, taskUID, taskRunUID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get task run uid, error: %v", err)
	}
	connIDAny, ok := s.stateCfg.TaskRunConnectionID.Load(taskRunUID)
	if !ok {
		return nil, status.Errorf(codes.FailedPrecondition, "connection id not found for task run %d", taskRunUID)
	}
	connID, ok := connIDAny.(string)
	if !ok {
		return nil, status.Errorf(codes.Internal, "expect connection id to be of type string but found %T", connIDAny)
	}

	task, err := s.store.GetTaskV2ByID(ctx, taskUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get task, error: %v", err)
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get instance, error: %v", err)
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil, db.ConnectionContext{
		OperationalComponent: "get-taskrun-session",
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get driver, error: %v", err)
	}
	defer driver.Close(ctx)

	session, err := getSession(ctx, instance.Metadata.GetEngine(), driver.GetDB(), connID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get session, error: %v", err)
	}

	session.Name = request.Parent + "/session"

	return session, nil
}

func getSession(ctx context.Context, engine storepb.Engine, db *sql.DB, connID string) (*v1pb.TaskRunSession, error) {
	switch engine {
	case storepb.Engine_POSTGRES:
		query := `
			SELECT
				pid,
				pg_blocking_pids(pid) AS blocked_by_pids,
				query,
				state,
				wait_event_type,
				wait_event,
				datname,
				usename,
				application_name,
				client_addr,
				client_port,
				backend_start,
				xact_start,
				query_start
			FROM
				pg_catalog.pg_stat_activity
			WHERE pid = $1
			OR pid = ANY(pg_blocking_pids($1))
			OR $1 = ANY(pg_blocking_pids(pid))
			ORDER BY pid
		`
		rows, err := db.QueryContext(ctx, query, connID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to query rows")
		}
		defer rows.Close()

		ss := &v1pb.TaskRunSession_Postgres{}
		for rows.Next() {
			var s v1pb.TaskRunSession_Postgres_Session

			var blockedByPids pgtype.TextArray

			var bs time.Time
			var xs, qs *time.Time
			if err := rows.Scan(
				&s.Pid,
				&blockedByPids,
				&s.Query,
				&s.State,
				&s.WaitEventType,
				&s.WaitEvent,
				&s.Datname,
				&s.Usename,
				&s.ApplicationName,
				&s.ClientAddr,
				&s.ClientPort,
				&bs,
				&xs,
				&qs,
			); err != nil {
				return nil, errors.Wrapf(err, "failed to scan")
			}

			if err := blockedByPids.AssignTo(&s.BlockedByPids); err != nil {
				return nil, errors.Wrapf(err, "failed to assign blocking pids")
			}

			s.BackendStart = timestamppb.New(bs)
			if xs != nil {
				s.XactStart = timestamppb.New(*xs)
			}
			if qs != nil {
				s.QueryStart = timestamppb.New(*qs)
			}

			if s.Pid == connID {
				ss.Session = &s
			} else if slices.Contains(s.BlockedByPids, connID) {
				ss.BlockedSessions = append(ss.BlockedSessions, &s)
			} else {
				ss.BlockingSessions = append(ss.BlockingSessions, &s)
			}
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrapf(err, "failed to scan rows")
		}

		return &v1pb.TaskRunSession{
			Session: &v1pb.TaskRunSession_Postgres_{
				Postgres: ss,
			},
		}, nil
	default:
		return nil, errors.Errorf("unsupported engine type %v", engine.String())
	}
}

// BatchRunTasks runs tasks in batch.
func (s *RolloutService) BatchRunTasks(ctx context.Context, request *v1pb.BatchRunTasksRequest) (*v1pb.BatchRunTasksResponse, error) {
	if len(request.Tasks) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "The tasks in request cannot be empty")
	}
	projectID, rolloutID, _, err := common.GetProjectIDRolloutIDMaybeStageID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	issueN, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &rolloutID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find issue, error: %v", err)
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
			return nil, status.Error(codes.InvalidArgument, err.Error())
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

	stageToRunTasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &rolloutID, StageID: &stageToRun.ID})
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

	ok, err = s.canUserRunStageTasks(ctx, user, project, issueN, stageToRun, rollout.CreatorUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the user can run tasks, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "Not allowed to run tasks")
	}

	// Don't need to check if issue is approved if
	// the user has bb.taskruns.create permission.
	ok, err = s.iamManager.CheckPermission(ctx, iam.PermissionTaskRunsCreate, user)
	if err != nil {
		return nil, err
	}
	if !ok {
		if issueN != nil {
			approved, err := utils.CheckIssueApproved(issueN)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to check if the issue is approved, error: %v", err)
			}
			if !approved {
				return nil, status.Errorf(codes.FailedPrecondition, "cannot run the tasks because the issue is not approved")
			}
		}
	}

	var taskRunCreates []*store.TaskRunMessage
	for _, task := range stageToRunTasks {
		if !taskIDsToRunMap[task.ID] {
			continue
		}

		create := &store.TaskRunMessage{
			TaskUID:   task.ID,
			CreatorID: user.ID,
		}
		if task.Payload.GetSheetId() != 0 {
			sheetUID := int(task.Payload.GetSheetId())
			create.SheetUID = &sheetUID
		}
		taskRunCreates = append(taskRunCreates, create)
	}
	sort.Slice(taskRunCreates, func(i, j int) bool {
		return taskRunCreates[i].TaskUID < taskRunCreates[j].TaskUID
	})

	if err := s.store.CreatePendingTaskRuns(ctx, taskRunCreates...); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create pending task runs, error %v", err)
	}

	if issueN != nil {
		if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issueN.UID, request.Tasks, storepb.IssueCommentPayload_TaskUpdate_PENDING, user.ID, request.Reason); err != nil {
			slog.Warn("failed to create issue comment", "issueUID", issueN.UID, log.BBError(err))
		}
	}
	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    webhook.EventTypeTaskRunStatusUpdate,
		Comment: request.Reason,
		Issue:   webhook.NewIssue(issueN),
		Project: webhook.NewProject(project),
		Rollout: webhook.NewRollout(rollout),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	issueN, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &rolloutID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find issue, error: %v", err)
	}

	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &rolloutID})
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
			return nil, status.Error(codes.InvalidArgument, err.Error())
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
		ok, err = s.canUserSkipStageTasks(ctx, user, project, issueN, stage, rollout.CreatorUID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check if the user can run tasks, error: %v", err)
		}
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "not allowed to skip tasks in stage %q", stage.Environment)
		}
	}

	if err := s.store.BatchSkipTasks(ctx, taskUIDs, request.Reason); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to skip tasks, error: %v", err)
	}

	for _, task := range tasksToSkip {
		s.stateCfg.TaskSkippedOrDoneChan <- task.ID
	}

	if issueN != nil {
		if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issueN.UID, request.Tasks, storepb.IssueCommentPayload_TaskUpdate_SKIPPED, user.ID, request.Reason); err != nil {
			slog.Warn("failed to create issue comment", "issueUID", issueN.UID, log.BBError(err))
		}
	}
	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    webhook.EventTypeTaskRunStatusUpdate,
		Comment: request.Reason,
		Issue:   webhook.NewIssue(issueN),
		Project: webhook.NewProject(project),
		Rollout: webhook.NewRollout(rollout),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	issueN, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &rolloutID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find issue, error: %v", err)
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

	ok, err = s.canUserCancelStageTaskRun(ctx, user, project, issueN, stage, rollout.CreatorUID)
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
			return nil, status.Error(codes.InvalidArgument, err.Error())
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
			return nil, status.Errorf(codes.InvalidArgument, "taskRun %v is not pending or running", taskRun.ID)
		}
	}

	for _, taskRun := range taskRuns {
		if taskRun.Status == api.TaskRunRunning {
			if cancelFunc, ok := s.stateCfg.RunningTaskRunsCancelFunc.Load(taskRun.ID); ok {
				cancelFunc.(context.CancelFunc)()
			}
		}
	}

	if err := s.store.BatchCancelTaskRuns(ctx, taskRunIDs); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to batch patch task run status to canceled, error: %v", err)
	}

	if issueN != nil {
		if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issueN.UID, taskNames, storepb.IssueCommentPayload_TaskUpdate_CANCELED, user.ID, request.Reason); err != nil {
			slog.Warn("failed to create issue comment", "issueUID", issueN.UID, log.BBError(err))
		}
	}
	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    webhook.EventTypeTaskRunStatusUpdate,
		Comment: request.Reason,
		Issue:   webhook.NewIssue(issueN),
		Rollout: webhook.NewRollout(rollout),
		Project: webhook.NewProject(project),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
			Status: api.TaskRunCanceled.String(),
		},
	})

	return &v1pb.BatchCancelTaskRunsResponse{}, nil
}

func (s *RolloutService) PreviewTaskRunRollback(ctx context.Context, request *v1pb.PreviewTaskRunRollbackRequest) (*v1pb.PreviewTaskRunRollbackResponse, error) {
	_, _, _, taskUID, taskRunUID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get task run uid, error: %v", err)
	}

	taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
		UID: &taskRunUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list task runs, error: %v", err)
	}
	if len(taskRuns) == 0 {
		return nil, status.Errorf(codes.NotFound, "task run %v not found", taskRunUID)
	}
	if len(taskRuns) > 1 {
		return nil, status.Errorf(codes.Internal, "found multiple task runs with the same uid %v", taskRunUID)
	}

	taskRun := taskRuns[0]

	if taskRun.Status != api.TaskRunDone {
		return nil, status.Errorf(codes.InvalidArgument, "task run %v is not done", taskRun.ID)
	}

	if taskRun.ResultProto == nil {
		return nil, status.Errorf(codes.InvalidArgument, "task run %v has no result", taskRun.ID)
	}

	backupDetail := taskRun.ResultProto.PriorBackupDetail
	if backupDetail == nil {
		return nil, status.Errorf(codes.InvalidArgument, "task run %v has no rollback", taskRun.ID)
	}

	task, err := s.store.GetTaskV2ByID(ctx, taskUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get task, error: %v", err)
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get instance, error: %v", err)
	}

	if taskRun.SheetUID == nil {
		return nil, status.Errorf(codes.InvalidArgument, "task run %v has no sheet", taskRun.ID)
	}
	statements, err := s.store.GetSheetStatementByID(ctx, *taskRun.SheetUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get sheet statements, error: %v", err)
	}

	var results []string
	for _, item := range backupDetail.Items {
		restore, err := base.GenerateRestoreSQL(ctx, instance.Metadata.GetEngine(), base.RestoreContext{
			InstanceID:              instance.ResourceID,
			GetDatabaseMetadataFunc: BuildGetDatabaseMetadataFunc(s.store),
			ListDatabaseNamesFunc:   BuildListDatabaseNamesFunc(s.store),
			IsCaseSensitive:         store.IsObjectCaseSensitive(instance),
		}, statements, item)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate restore sql, error: %v", err)
		}
		results = append(results, restore)
	}

	return &v1pb.PreviewTaskRunRollbackResponse{
		Statement: strings.Join(results, "\n"),
	}, nil
}

func isChangeDatabasePlan(steps []*storepb.PlanConfig_Step) bool {
	for _, step := range steps {
		for _, spec := range step.GetSpecs() {
			if spec.GetChangeDatabaseConfig() != nil {
				return true
			}
		}
	}
	return false
}

// GetPipelineCreate gets a pipeline create message from a plan.
func GetPipelineCreate(ctx context.Context, s *store.Store, sheetManager *sheet.Manager, licenseService enterprise.LicenseService, dbFactory *dbfactory.DBFactory, rolloutTitle string, steps []*storepb.PlanConfig_Step, deployment *storepb.PlanConfig_Deployment /* nullable */, project *store.ProjectMessage) (*store.PipelineMessage, error) {
	// Flatten all specs from steps.
	var specs []*storepb.PlanConfig_Spec
	for _, step := range steps {
		specs = append(specs, step.Specs...)
	}

	// Step 1 - transform database group specs.
	// Others are untouched.
	transformSpecs, err := transformDatabaseGroupSpecs(ctx, s, project, specs, deployment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to transform database group specs")
	}

	// Step 2 - list snapshot environments.
	snapshotEnvironments := deployment.GetEnvironments()
	if len(snapshotEnvironments) == 0 {
		environments, err := s.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list environments")
		}
		for _, e := range environments {
			snapshotEnvironments = append(snapshotEnvironments, e.ResourceID)
		}
	}
	environmentIndex := make(map[string]int)
	for i, e := range snapshotEnvironments {
		environmentIndex[e] = i
	}

	// Step 3 - convert all task creates.
	var taskCreates []*store.TaskMessage
	for _, spec := range transformSpecs {
		tcs, err := getTaskCreatesFromSpec(ctx, s, sheetManager, licenseService, dbFactory, spec, project)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get task creates from spec")
		}
		taskCreates = append(taskCreates, tcs...)
	}
	if len(taskCreates) == 0 {
		return nil, errors.Errorf("there is no tasks created from the plan")
	}

	// Step 4 - construct all environment stages.
	var stages []*store.StageMessage
	for _, environmentID := range snapshotEnvironments {
		stages = append(stages, &store.StageMessage{
			Environment: environmentID,
		})
	}

	// Step 5 - build tasks for each stage.
	for _, spec := range transformSpecs {
		tc, err := getTaskCreatesFromSpec(ctx, s, sheetManager, licenseService, dbFactory, spec, project)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get task creates from spec")
		}
		for _, t := range tc {
			e, err := s.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &t.EnvironmentID})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			environmentIndex := environmentIndex[e.ResourceID]
			stages[environmentIndex].TaskList = append(stages[environmentIndex].TaskList, t)
		}
	}
	return &store.PipelineMessage{
		Name:      rolloutTitle,
		ProjectID: project.ResourceID,
		Stages: slices.DeleteFunc(stages, func(stage *store.StageMessage) bool {
			return len(stage.TaskList) == 0
		}),
	}, nil
}

// filter pipelineCreate.Stages using targetEnvironmentID.
func getPipelineCreateToTargetStage(ctx context.Context, s *store.Store, snapshotEnvironments []string, pipelineCreate *store.PipelineMessage, targetEnvironment *string) (*store.PipelineMessage, error) {
	if targetEnvironment == nil {
		return pipelineCreate, nil
	}
	if *targetEnvironment == "" {
		pipelineCreate.Stages = nil
		return pipelineCreate, nil
	}
	targetEnvironmentID, err := common.GetEnvironmentID(*targetEnvironment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get environment id from %q", *targetEnvironment)
	}
	if len(snapshotEnvironments) == 0 {
		environments, err := s.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list environments")
		}
		for _, e := range environments {
			snapshotEnvironments = append(snapshotEnvironments, e.ResourceID)
		}
	}

	foundID := false
	var stageCreates []*store.StageMessage
	i := 0
	for _, environmentID := range snapshotEnvironments {
		if i < len(pipelineCreate.Stages) && pipelineCreate.Stages[i].Environment == environmentID {
			stageCreates = append(stageCreates, pipelineCreate.Stages[i])
			i++
		}
		if environmentID == targetEnvironmentID {
			foundID = true
			break
		}
	}
	if !foundID {
		return nil, errors.Errorf("environment %q not found", targetEnvironmentID)
	}
	pipelineCreate.Stages = stageCreates
	return pipelineCreate, nil
}

func GetValidRolloutPolicyForStage(ctx context.Context, stores *store.Store, stage *store.StageMessage) (*storepb.RolloutPolicy, error) {
	policy, err := stores.GetRolloutPolicy(ctx, stage.Environment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rollout policy for stageEnvironmentID %s", stage.Environment)
	}
	return policy, nil
}

// canUserRunStageTasks returns if a user can run the tasks in a stage.
func (s *RolloutService) canUserRunStageTasks(ctx context.Context, user *store.UserMessage, project *store.ProjectMessage, issue *store.IssueMessage, stage *store.StageMessage, creatorUID int) (bool, error) {
	// For data export issues, only the creator can run tasks.
	if issue != nil && issue.Type == api.IssueDatabaseDataExport {
		return issue.Creator.ID == user.ID, nil
	}

	// Users with bb.taskRuns.create can always create task runs.
	ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionTaskRunsCreate, user, project.ResourceID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check workspace role")
	}
	if ok {
		return true, nil
	}

	p, err := GetValidRolloutPolicyForStage(ctx, s.store, stage)
	if err != nil {
		return false, err
	}

	policy, err := s.store.GetProjectIamPolicy(ctx, project.ResourceID)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get project %s IAM policy", project.ResourceID)
	}
	workspacePolicy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get workspace IAM policy")
	}
	roles := utils.GetUserFormattedRolesMap(ctx, s.store, user, policy.Policy)
	workspaceRoles := utils.GetUserFormattedRolesMap(ctx, s.store, user, workspacePolicy.Policy)
	for k := range workspaceRoles {
		roles[k] = true
	}

	for _, role := range p.Roles {
		if roles[role] {
			return true, nil
		}
	}

	if user.ID == creatorUID {
		for _, issueRole := range p.IssueRoles {
			if issueRole == "roles/CREATOR" {
				return true, nil
			}
		}
	}

	if issue != nil {
		if lastApproverUID := getLastApproverUID(issue.Payload.GetApproval()); lastApproverUID != nil && *lastApproverUID == user.ID {
			for _, issueRole := range p.IssueRoles {
				if issueRole == "roles/LAST_APPROVER" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// canUserCancelStageTaskRun returns if a user can cancel the task runs in a stage.
func (s *RolloutService) canUserCancelStageTaskRun(ctx context.Context, user *store.UserMessage, project *store.ProjectMessage, issue *store.IssueMessage, stage *store.StageMessage, creatorUID int) (bool, error) {
	return s.canUserRunStageTasks(ctx, user, project, issue, stage, creatorUID)
}

func (s *RolloutService) canUserSkipStageTasks(ctx context.Context, user *store.UserMessage, project *store.ProjectMessage, issue *store.IssueMessage, stage *store.StageMessage, creatorUID int) (bool, error) {
	return s.canUserRunStageTasks(ctx, user, project, issue, stage, creatorUID)
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
