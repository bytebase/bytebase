package v1

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgtype"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// RolloutService represents a service for managing rollout.
type RolloutService struct {
	v1connect.UnimplementedRolloutServiceHandler
	store          *store.Store
	sheetManager   *sheet.Manager
	licenseService *enterprise.LicenseService
	dbFactory      *dbfactory.DBFactory
	stateCfg       *state.State
	webhookManager *webhook.Manager
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewRolloutService returns a rollout service instance.
func NewRolloutService(store *store.Store, sheetManager *sheet.Manager, licenseService *enterprise.LicenseService, dbFactory *dbfactory.DBFactory, stateCfg *state.State, webhookManager *webhook.Manager, profile *config.Profile, iamManager *iam.Manager) *RolloutService {
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
func (s *RolloutService) PreviewRollout(ctx context.Context, req *connect.Request[v1pb.PreviewRolloutRequest]) (*connect.Response[v1pb.Rollout], error) {
	request := req.Msg
	projectID, err := common.GetProjectID(request.Project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
	}

	// Validate plan specs
	if err := validateSpecs(request.Plan.Specs); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to validate plan specs, error: %v", err))
	}

	specs := convertPlanSpecs(request.Plan.Specs)

	rollout, err := GetPipelineCreate(ctx, s.store, s.sheetManager, s.dbFactory, specs, nil /* snapshot */, project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get pipeline create, error: %v", err))
	}

	rolloutV1, err := convertToRollout(ctx, s.store, project, rollout)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to rollout, error: %v", err))
	}
	return connect.NewResponse(rolloutV1), nil
}

// GetRollout gets a rollout.
func (s *RolloutService) GetRollout(ctx context.Context, req *connect.Request[v1pb.GetRolloutRequest]) (*connect.Response[v1pb.Rollout], error) {
	request := req.Msg
	projectID, rolloutID, err := common.GetProjectIDRolloutID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
	}

	rollout, err := s.getRolloutWithTasks(ctx, projectID, rolloutID)
	if err != nil {
		return nil, err
	}

	rolloutV1, err := convertToRollout(ctx, s.store, project, rollout)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to rollout, error: %v", err))
	}
	return connect.NewResponse(rolloutV1), nil
}

// ListRollouts lists rollouts.
func (s *RolloutService) ListRollouts(ctx context.Context, req *connect.Request[v1pb.ListRolloutsRequest]) (*connect.Response[v1pb.ListRolloutsResponse], error) {
	request := req.Msg
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
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
	if err := s.buildRolloutFindWithFilter(ctx, find, request.Filter); err != nil {
		return nil, err
	}
	pipelines, err := s.store.ListPipelines(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list pipelines, error: %v", err))
	}

	var nextPageToken string
	// has more pages
	if len(pipelines) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get next page token, error: %v", err))
		}
		pipelines = pipelines[:offset.limit]
	}

	rollouts := []*v1pb.Rollout{}
	for _, pipeline := range pipelines {
		tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &pipeline.ID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get tasks for pipeline %d, error: %v", pipeline.ID, err))
		}
		rolloutMessage := *pipeline
		rolloutMessage.Tasks = tasks
		rollout, err := convertToRollout(ctx, s.store, project, &rolloutMessage)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to rollout, error: %v", err))
		}
		rollouts = append(rollouts, rollout)
	}

	return connect.NewResponse(&v1pb.ListRolloutsResponse{
		Rollouts:      rollouts,
		NextPageToken: nextPageToken,
	}), nil
}

// buildRolloutFindWithFilter builds the filter for rollout find.
func (s *RolloutService) buildRolloutFindWithFilter(ctx context.Context, pipelineFind *store.PipelineFind, filter string) error {
	if filter == "" {
		return nil
	}

	filterQ, err := s.store.GetListRolloutFilter(ctx, filter)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	pipelineFind.FilterQ = filterQ
	return nil
}

// CreateRollout creates a rollout from plan.
func (s *RolloutService) CreateRollout(ctx context.Context, req *connect.Request[v1pb.CreateRolloutRequest]) (*connect.Response[v1pb.Rollout], error) {
	request := req.Msg
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}

	_, planID, err := common.GetProjectIDPlanID(request.GetRollout().GetPlan())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID, ProjectID: &projectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan, error: %v", err))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan %d not found in project %s", planID, projectID))
	}

	pipelineCreate, err := GetPipelineCreate(ctx, s.store, s.sheetManager, s.dbFactory, plan.Config.GetSpecs(), plan.Config.GetDeployment(), project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get pipeline create, error: %v", err))
	}
	if isChangeDatabasePlan(plan.Config.GetSpecs()) {
		pipelineCreate, err = getPipelineCreateToTargetStage(ctx, s.store, plan.Config.GetDeployment(), pipelineCreate, request.Target)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to filter stages with stageId, error: %v", err))
		}
	}
	if request.ValidateOnly {
		rolloutV1, err := convertToRollout(ctx, s.store, project, pipelineCreate)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to rollout, error: %v", err))
		}
		rolloutV1.Plan = request.Rollout.GetPlan()
		return connect.NewResponse(rolloutV1), nil
	}
	pipelineUID, err := s.store.CreatePipelineAIO(ctx, planID, pipelineCreate, user.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create pipeline, error: %v", err))
	}

	rollout, err := s.getRolloutWithTasks(ctx, projectID, pipelineUID)
	if err != nil {
		return nil, err
	}

	rolloutV1, err := convertToRollout(ctx, s.store, project, rollout)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to rollout, error: %v", err))
	}

	// Tickle task run scheduler.
	s.stateCfg.TaskRunTickleChan <- 0

	return connect.NewResponse(rolloutV1), nil
}

// ListTaskRuns lists rollout task runs.
func (s *RolloutService) ListTaskRuns(ctx context.Context, req *connect.Request[v1pb.ListTaskRunsRequest]) (*connect.Response[v1pb.ListTaskRunsResponse], error) {
	request := req.Msg
	projectID, rolloutID, maybeStageID, maybeTaskID, err := common.GetProjectIDRolloutIDMaybeStageIDMaybeTaskID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	pipeline, err := s.store.GetPipeline(ctx, &store.PipelineFind{
		ID:        &rolloutID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get rollout, error: %v", err))
	}
	if pipeline == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("pipeline %d not found in project %s", rolloutID, projectID))
	}

	taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		PipelineUID: &rolloutID,
		Environment: maybeStageID,
		TaskUID:     maybeTaskID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list task runs, error: %v", err))
	}

	taskRunsV1, err := convertToTaskRuns(ctx, s.store, s.stateCfg, taskRuns)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to task runs, error: %v", err))
	}
	return connect.NewResponse(&v1pb.ListTaskRunsResponse{
		TaskRuns: taskRunsV1,
	}), nil
}

// GetTaskRun gets a task run.
func (s *RolloutService) GetTaskRun(ctx context.Context, req *connect.Request[v1pb.GetTaskRunRequest]) (*connect.Response[v1pb.TaskRun], error) {
	request := req.Msg
	projectID, rolloutID, _, _, taskRunUID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	pipeline, err := s.store.GetPipeline(ctx, &store.PipelineFind{
		ID:        &rolloutID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get rollout, error: %v", err))
	}
	if pipeline == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("pipeline %d not found in project %s", rolloutID, projectID))
	}

	taskRun, err := s.store.GetTaskRunV1(ctx, &store.FindTaskRunMessage{
		UID:         &taskRunUID,
		PipelineUID: &pipeline.ID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get task run, error: %v", err))
	}
	if taskRun == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("task run %d not found in pipeline %d", taskRunUID, pipeline.ID))
	}

	taskRunV1, err := convertToTaskRun(ctx, s.store, s.stateCfg, taskRun)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to task run, error: %v", err))
	}
	return connect.NewResponse(taskRunV1), nil
}

func (s *RolloutService) GetTaskRunLog(ctx context.Context, req *connect.Request[v1pb.GetTaskRunLogRequest]) (*connect.Response[v1pb.TaskRunLog], error) {
	request := req.Msg
	_, _, _, _, taskRunUID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get task run uid, error: %v", err))
	}
	logs, err := s.store.ListTaskRunLogs(ctx, taskRunUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to list task run logs, error: %v", err))
	}
	return connect.NewResponse(convertToTaskRunLog(request.Parent, logs)), nil
}

func (s *RolloutService) GetTaskRunSession(ctx context.Context, req *connect.Request[v1pb.GetTaskRunSessionRequest]) (*connect.Response[v1pb.TaskRunSession], error) {
	request := req.Msg
	_, _, _, taskUID, taskRunUID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get task run uid, error: %v", err))
	}
	connIDAny, ok := s.stateCfg.TaskRunConnectionID.Load(taskRunUID)
	if !ok {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("connection id not found for task run %d", taskRunUID))
	}
	connID, ok := connIDAny.(string)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("expect connection id to be of type string but found %T", connIDAny))
	}

	task, err := s.store.GetTaskByID(ctx, taskUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get task, error: %v", err))
	}

	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get instance, error: %v", err))
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil, db.ConnectionContext{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get driver, error: %v", err))
	}
	defer driver.Close(ctx)

	session, err := getSession(ctx, instance.Metadata.GetEngine(), driver.GetDB(), connID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get session, error: %v", err))
	}

	session.Name = request.Parent + "/session"

	return connect.NewResponse(session), nil
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
func (s *RolloutService) BatchRunTasks(ctx context.Context, req *connect.Request[v1pb.BatchRunTasksRequest]) (*connect.Response[v1pb.BatchRunTasksResponse], error) {
	request := req.Msg
	if len(request.Tasks) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("The tasks in request cannot be empty"))
	}
	projectID, rolloutID, _, err := common.GetProjectIDRolloutIDMaybeStageID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	rollout, err := s.store.GetPipeline(ctx, &store.PipelineFind{
		ID:        &rolloutID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find rollout, error: %v", err))
	}
	if rollout == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %v not found", rolloutID))
	}

	issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		PipelineID: &rolloutID,
		ProjectID:  &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find issue, error: %v", err))
	}

	// Parse requested task IDs and group by their environment
	taskEnvironments := map[string][]int{}
	taskIDsToRunMap := map[int]bool{}
	for _, task := range request.Tasks {
		_, _, stageID, taskID, err := common.GetProjectIDRolloutIDStageIDTaskID(task)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		environment := formatEnvironmentFromStageID(stageID)
		taskEnvironments[environment] = append(taskEnvironments[environment], taskID)
		taskIDsToRunMap[taskID] = true
	}
	if len(taskEnvironments) > 1 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("tasks should be in the same environment"))
	}

	// Get the environment for the tasks to run
	var environmentToRun string
	for env := range taskEnvironments {
		environmentToRun = env
		break
	}

	// Get all tasks in the same environment
	stageToRunTasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &rolloutID, Environment: &environmentToRun})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list tasks, error: %v", err))
	}
	if len(stageToRunTasks) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("No tasks to run in the stage"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	ok, err = s.canUserRunEnvironmentTasks(ctx, user, project, issueN, environmentToRun, rollout.Creator)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check if the user can run tasks, error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("Not allowed to run tasks"))
	}

	// Check if issue approval is required according to the project settings
	if project.Setting.RequireIssueApproval && issueN != nil {
		approved, err := utils.CheckIssueApproved(issueN)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check if the issue is approved, error: %v", err))
		}
		if !approved {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot run the tasks because issue approval is required but the issue is not approved"))
		}
	}

	var taskRunCreates []*store.TaskRunMessage
	for _, task := range stageToRunTasks {
		if !taskIDsToRunMap[task.ID] {
			continue
		}

		create := &store.TaskRunMessage{
			TaskUID: task.ID,
		}
		if task.Payload.GetSheetId() != 0 {
			sheetUID := int(task.Payload.GetSheetId())
			create.SheetUID = &sheetUID
		}
		if request.GetRunTime() != nil {
			t := request.GetRunTime().AsTime()
			create.RunAt = &t
		}
		taskRunCreates = append(taskRunCreates, create)
	}
	slices.SortFunc(taskRunCreates, func(a, b *store.TaskRunMessage) int {
		return a.TaskUID - b.TaskUID
	})

	if err := s.store.CreatePendingTaskRuns(ctx, user.Email, taskRunCreates...); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create pending task runs, error %v", err))
	}

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    storepb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE,
		Issue:   webhook.NewIssue(issueN),
		Project: webhook.NewProject(project),
		Rollout: webhook.NewRollout(rollout),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
			Status: storepb.TaskRun_PENDING.String(),
		},
	})
	// Tickle task run scheduler.
	s.stateCfg.TaskRunTickleChan <- 0

	return connect.NewResponse(&v1pb.BatchRunTasksResponse{}), nil
}

// BatchSkipTasks skips tasks in batch.
func (s *RolloutService) BatchSkipTasks(ctx context.Context, req *connect.Request[v1pb.BatchSkipTasksRequest]) (*connect.Response[v1pb.BatchSkipTasksResponse], error) {
	request := req.Msg
	projectID, rolloutID, _, err := common.GetProjectIDRolloutIDMaybeStageID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	rollout, err := s.store.GetPipeline(ctx, &store.PipelineFind{
		ID:        &rolloutID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find rollout, error: %v", err))
	}
	if rollout == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %v not found", rolloutID))
	}

	issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		PipelineID: &rolloutID,
		ProjectID:  &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find issue, error: %v", err))
	}

	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &rolloutID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list tasks, error: %v", err))
	}

	taskByID := make(map[int]*store.TaskMessage)
	for _, task := range tasks {
		taskByID[task.ID] = task
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}
	var taskUIDs []int
	var tasksToSkip []*store.TaskMessage
	environmentSet := map[string]struct{}{}
	for _, task := range request.Tasks {
		_, _, _, taskID, err := common.GetProjectIDRolloutIDStageIDTaskID(task)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		taskMsg, ok := taskByID[taskID]
		if !ok {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("task %v not found in the rollout", taskID))
		}
		taskUIDs = append(taskUIDs, taskID)
		tasksToSkip = append(tasksToSkip, taskMsg)
		environmentSet[taskMsg.Environment] = struct{}{}
	}

	for environment := range environmentSet {
		ok, err = s.canUserRunEnvironmentTasks(ctx, user, project, issueN, environment, rollout.Creator)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check if the user can skip tasks, error: %v", err))
		}
		if !ok {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("not allowed to skip tasks in environment %q", environment))
		}
	}

	if err := s.store.BatchSkipTasks(ctx, taskUIDs, request.Reason); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to skip tasks, error: %v", err))
	}

	for _, task := range tasksToSkip {
		s.stateCfg.TaskSkippedOrDoneChan <- task.ID
	}

	if issueN != nil {
		if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issueN.UID, request.Tasks, storepb.TaskRun_SKIPPED, user.Email, request.Reason); err != nil {
			slog.Warn("failed to create issue comment", "issueUID", issueN.UID, log.BBError(err))
		}
	}
	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    storepb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE,
		Comment: request.Reason,
		Issue:   webhook.NewIssue(issueN),
		Project: webhook.NewProject(project),
		Rollout: webhook.NewRollout(rollout),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
			Status:        storepb.TaskRun_SKIPPED.String(),
			SkippedReason: request.Reason,
		},
	})

	return connect.NewResponse(&v1pb.BatchSkipTasksResponse{}), nil
}

// BatchCancelTaskRuns cancels a list of task runs.
func (s *RolloutService) BatchCancelTaskRuns(ctx context.Context, req *connect.Request[v1pb.BatchCancelTaskRunsRequest]) (*connect.Response[v1pb.BatchCancelTaskRunsResponse], error) {
	request := req.Msg
	if len(request.TaskRuns) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("task runs cannot be empty"))
	}

	projectID, rolloutID, stageID, _, err := common.GetProjectIDRolloutIDStageIDMaybeTaskID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	rollout, err := s.store.GetPipeline(ctx, &store.PipelineFind{
		ID:        &rolloutID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find rollout, error: %v", err))
	}
	if rollout == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %v not found", rolloutID))
	}

	issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		PipelineID: &rolloutID,
		ProjectID:  &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find issue, error: %v", err))
	}

	for _, taskRun := range request.TaskRuns {
		_, _, taskRunStageID, _, _, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(taskRun)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if taskRunStageID != stageID {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("task run %v is not in the specified stage %v", taskRun, stageID))
		}
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	environment := formatEnvironmentFromStageID(stageID)
	ok, err = s.canUserCancelEnvironmentTaskRun(ctx, user, project, issueN, environment, rollout.Creator)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check if the user can run tasks, error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("Not allowed to cancel tasks"))
	}

	var taskRunIDs []int
	for _, taskRun := range request.TaskRuns {
		_, _, _, _, taskRunID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(taskRun)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		taskRunIDs = append(taskRunIDs, taskRunID)
	}

	taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		UIDs: &taskRunIDs,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list task runs, error: %v", err))
	}

	for _, taskRun := range taskRuns {
		switch taskRun.Status {
		case storepb.TaskRun_PENDING:
		case storepb.TaskRun_RUNNING:
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("taskRun %v is not pending or running", taskRun.ID))
		}
	}

	for _, taskRun := range taskRuns {
		if taskRun.Status == storepb.TaskRun_RUNNING {
			if cancelFunc, ok := s.stateCfg.RunningTaskRunsCancelFunc.Load(taskRun.ID); ok {
				cancelFunc.(context.CancelFunc)()
			}
		}
	}

	if err := s.store.BatchCancelTaskRuns(ctx, taskRunIDs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to batch patch task run status to canceled, error: %v", err))
	}

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    storepb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE,
		Issue:   webhook.NewIssue(issueN),
		Rollout: webhook.NewRollout(rollout),
		Project: webhook.NewProject(project),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
			Status: storepb.TaskRun_CANCELED.String(),
		},
	})

	return connect.NewResponse(&v1pb.BatchCancelTaskRunsResponse{}), nil
}

func (s *RolloutService) PreviewTaskRunRollback(ctx context.Context, req *connect.Request[v1pb.PreviewTaskRunRollbackRequest]) (*connect.Response[v1pb.PreviewTaskRunRollbackResponse], error) {
	request := req.Msg
	_, _, _, taskUID, taskRunUID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get task run uid, error: %v", err))
	}

	taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		UID: &taskRunUID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list task runs, error: %v", err))
	}
	if len(taskRuns) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("task run %v not found", taskRunUID))
	}
	if len(taskRuns) > 1 {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("found multiple task runs with the same uid %v", taskRunUID))
	}

	taskRun := taskRuns[0]

	if taskRun.Status != storepb.TaskRun_DONE {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("task run %v is not done", taskRun.ID))
	}

	if taskRun.ResultProto == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("task run %v has no result", taskRun.ID))
	}

	backupDetail := taskRun.ResultProto.PriorBackupDetail
	if backupDetail == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("task run %v has no rollback", taskRun.ID))
	}

	task, err := s.store.GetTaskByID(ctx, taskUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get task, error: %v", err))
	}

	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get instance, error: %v", err))
	}

	if taskRun.SheetUID == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("task run %v has no sheet", taskRun.ID))
	}
	statements, err := s.store.GetSheetStatementByID(ctx, *taskRun.SheetUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get sheet statements, error: %v", err))
	}

	var results []string
	for _, item := range backupDetail.Items {
		restore, err := parserbase.GenerateRestoreSQL(ctx, instance.Metadata.GetEngine(), parserbase.RestoreContext{
			InstanceID:              instance.ResourceID,
			GetDatabaseMetadataFunc: BuildGetDatabaseMetadataFunc(s.store),
			ListDatabaseNamesFunc:   BuildListDatabaseNamesFunc(s.store),
			IsCaseSensitive:         store.IsObjectCaseSensitive(instance),
		}, statements, item)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate restore sql, error: %v", err))
		}
		results = append(results, restore)
	}

	return connect.NewResponse(&v1pb.PreviewTaskRunRollbackResponse{
		Statement: strings.Join(results, "\n"),
	}), nil
}

func isChangeDatabasePlan(specs []*storepb.PlanConfig_Spec) bool {
	for _, spec := range specs {
		if spec.GetChangeDatabaseConfig() != nil {
			return true
		}
	}
	return false
}

// getPlanEnvironmentSnapshots returns the environment snapshots and environment index map.
func getPlanEnvironmentSnapshots(ctx context.Context, s *store.Store, deployment *storepb.PlanConfig_Deployment) ([]string, map[string]int, error) {
	snapshotEnvironments := deployment.GetEnvironments()
	if len(snapshotEnvironments) == 0 {
		var err error
		snapshotEnvironments, err = getAllEnvironmentIDs(ctx, s)
		if err != nil {
			return nil, nil, err
		}
	}
	environmentIndex := make(map[string]int)
	for i, e := range snapshotEnvironments {
		environmentIndex[e] = i
	}
	return snapshotEnvironments, environmentIndex, nil
}

// GetPipelineCreate gets a pipeline create message from a plan.
func GetPipelineCreate(ctx context.Context, s *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory, specs []*storepb.PlanConfig_Spec, deployment *storepb.PlanConfig_Deployment /* nullable */, project *store.ProjectMessage) (*store.PipelineMessage, error) {
	// Step 1 - transform database group specs.
	// Others are untouched.
	transformedSpecs := applyDatabaseGroupSpecTransformations(specs, deployment)

	// Step 2 - convert all task creates.
	var taskCreates []*store.TaskMessage
	for _, spec := range transformedSpecs {
		tcs, err := getTaskCreatesFromSpec(ctx, s, sheetManager, dbFactory, spec, project)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get task creates from spec")
		}
		taskCreates = append(taskCreates, tcs...)
	}

	return &store.PipelineMessage{
		ProjectID: project.ResourceID,
		Tasks:     taskCreates,
	}, nil
}

// filter pipelineCreate.Tasks using targetEnvironmentID.
func getPipelineCreateToTargetStage(ctx context.Context, s *store.Store, deployment *storepb.PlanConfig_Deployment, pipelineCreate *store.PipelineMessage, targetEnvironment *string) (*store.PipelineMessage, error) {
	if targetEnvironment == nil {
		return pipelineCreate, nil
	}
	if *targetEnvironment == "" {
		pipelineCreate.Tasks = nil
		return pipelineCreate, nil
	}
	targetEnvironmentID, err := common.GetEnvironmentID(*targetEnvironment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get environment id from %q", *targetEnvironment)
	}

	snapshotEnvironments, _, err := getPlanEnvironmentSnapshots(ctx, s, deployment)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get environment snapshots")
	}

	// Build a set of allowed environments up to and including the target
	allowedEnvironments := make(map[string]bool)
	for _, environmentID := range snapshotEnvironments {
		allowedEnvironments[environmentID] = true
		if environmentID == targetEnvironmentID {
			break
		}
	}

	if !allowedEnvironments[targetEnvironmentID] {
		return nil, errors.Errorf("environment %q not found", targetEnvironmentID)
	}

	// Filter tasks to only include those in allowed environments
	filteredTasks := []*store.TaskMessage{}
	for _, task := range pipelineCreate.Tasks {
		if allowedEnvironments[task.Environment] {
			filteredTasks = append(filteredTasks, task)
		}
	}
	pipelineCreate.Tasks = filteredTasks
	return pipelineCreate, nil
}

func GetValidRolloutPolicyForEnvironment(ctx context.Context, stores *store.Store, environment string) (*storepb.RolloutPolicy, error) {
	policy, err := stores.GetRolloutPolicy(ctx, environment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rollout policy for environment %s", environment)
	}
	return policy, nil
}

// canUserRunEnvironmentTasks returns if a user can run the tasks in an environment.
func (s *RolloutService) canUserRunEnvironmentTasks(ctx context.Context, user *store.UserMessage, project *store.ProjectMessage, issue *store.IssueMessage, environment string, _ string) (bool, error) {
	// For data export issues, only the creator can run tasks.
	if issue != nil && issue.Type == storepb.Issue_DATABASE_EXPORT {
		return issue.CreatorEmail == user.Email, nil
	}

	// Users with bb.taskRuns.create can always create task runs.
	ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionTaskRunsCreate, user, project.ResourceID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check workspace role")
	}
	if ok {
		return true, nil
	}

	p, err := GetValidRolloutPolicyForEnvironment(ctx, s.store, environment)
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

	return false, nil
}

func (s *RolloutService) canUserCancelEnvironmentTaskRun(ctx context.Context, user *store.UserMessage, project *store.ProjectMessage, issue *store.IssueMessage, environment string, creator string) (bool, error) {
	return s.canUserRunEnvironmentTasks(ctx, user, project, issue, environment, creator)
}

// getRolloutWithTasks retrieves a pipeline by ID with its tasks populated.
func (s *RolloutService) getRolloutWithTasks(ctx context.Context, projectID string, rolloutID int) (*store.PipelineMessage, error) {
	pipeline, err := s.store.GetPipeline(ctx, &store.PipelineFind{
		ID:        &rolloutID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get pipeline, error: %v", err))
	}
	if pipeline == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %d not found in project %s", rolloutID, projectID))
	}

	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &rolloutID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get tasks, error: %v", err))
	}

	rollout := *pipeline
	rollout.Tasks = tasks
	return &rollout, nil
}
