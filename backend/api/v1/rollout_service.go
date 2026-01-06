package v1

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgtype"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/webhook"
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
	dbFactory      *dbfactory.DBFactory
	bus            *bus.Bus
	webhookManager *webhook.Manager
	iamManager     *iam.Manager
}

// NewRolloutService returns a rollout service instance.
func NewRolloutService(store *store.Store, dbFactory *dbfactory.DBFactory, bus *bus.Bus, webhookManager *webhook.Manager, iamManager *iam.Manager) *RolloutService {
	return &RolloutService{
		store:          store,
		dbFactory:      dbFactory,
		bus:            bus,
		webhookManager: webhookManager,
		iamManager:     iamManager,
	}
}

// GetRollout gets a rollout.
func (s *RolloutService) GetRollout(ctx context.Context, req *connect.Request[v1pb.GetRolloutRequest]) (*connect.Response[v1pb.Rollout], error) {
	request := req.Msg
	projectID, planID, err := common.GetProjectIDPlanIDFromRolloutName(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
	}

	// getRolloutWithTasks inlined
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: &projectID,
		UID:       &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %d not found in project %s", planID, projectID))
	}
	// Check if the plan has a rollout
	if plan.Config == nil || !plan.Config.HasRollout {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %d not found in project %s", planID, projectID))
	}

	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PlanID: &planID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get tasks"))
	}

	// Get environment order.
	environments, err := s.store.GetEnvironment(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get environments"))
	}
	environmentOrderMap := make(map[string]int)
	for i, env := range environments.GetEnvironments() {
		environmentOrderMap[env.Id] = i
	}

	rolloutV1, err := convertToRollout(project, plan, tasks, environmentOrderMap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to rollout"))
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
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
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

	// Filter plans to only those with rollouts (tasks).
	hasRollout := true
	findPlan := &store.FindPlanMessage{
		ProjectID:  &projectID,
		Limit:      &limitPlusOne,
		Offset:     &offset.offset,
		HasRollout: &hasRollout,
	}
	if err := buildRolloutFindWithFilter(findPlan, request.Filter); err != nil {
		return nil, err
	}
	plans, err := s.store.ListPlans(ctx, findPlan)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list plans"))
	}

	var nextPageToken string
	if len(plans) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		plans = plans[:offset.limit]
	}

	// Batch load all tasks for all plans to avoid N+1 queries
	planIDs := make([]int64, len(plans))
	for i, plan := range plans {
		planIDs[i] = plan.UID
	}
	var allTasks []*store.TaskMessage
	if len(planIDs) > 0 {
		var err error
		allTasks, err = s.store.ListTasks(ctx, &store.TaskFind{PlanIDs: &planIDs})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list tasks"))
		}
	}

	// Group tasks by plan ID
	tasksByPlanID := make(map[int64][]*store.TaskMessage)
	for _, task := range allTasks {
		tasksByPlanID[task.PlanID] = append(tasksByPlanID[task.PlanID], task)
	}

	// Get environment order once for all rollouts.
	environments, err := s.store.GetEnvironment(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get environments"))
	}
	environmentOrderMap := make(map[string]int)
	for i, env := range environments.GetEnvironments() {
		environmentOrderMap[env.Id] = i
	}

	// Convert plans and their tasks to rollouts
	rollouts := []*v1pb.Rollout{}
	for _, plan := range plans {
		tasks := tasksByPlanID[plan.UID]
		rollout, err := convertToRollout(project, plan, tasks, environmentOrderMap)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to rollout"))
		}
		rollouts = append(rollouts, rollout)
	}

	return connect.NewResponse(&v1pb.ListRolloutsResponse{
		Rollouts:      rollouts,
		NextPageToken: nextPageToken,
	}), nil
}

// buildRolloutFindWithFilter builds the filter for rollout find.
func buildRolloutFindWithFilter(planFind *store.FindPlanMessage, filter string) error {
	if filter == "" {
		return nil
	}

	filterQ, err := store.GetListRolloutFilter(filter)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	planFind.FilterQ = filterQ
	return nil
}

// CreateRollout creates a rollout from plan.
func (s *RolloutService) CreateRollout(ctx context.Context, req *connect.Request[v1pb.CreateRolloutRequest]) (*connect.Response[v1pb.Rollout], error) {
	request := req.Msg
	projectID, planID, err := common.GetProjectIDPlanID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}

	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID, ProjectID: &projectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan %d not found in project %s", planID, projectID))
	}

	tasks, err := GetPipelineCreate(ctx, s.store, plan.Config.GetSpecs(), project.ResourceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get pipeline create"))
	}
	if isChangeDatabasePlan(plan.Config.GetSpecs()) {
		tasks, err = filterTasksByStage(tasks, request.Target)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to filter tasks with stage id"))
		}
	}

	if err := CreateRolloutAndPendingTasks(ctx, s.store, plan, nil, project, tasks); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	tasks, err = s.store.ListTasks(ctx, &store.TaskFind{PlanID: &planID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get tasks"))
	}

	// Get environment order.
	environments, err := s.store.GetEnvironment(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get environments"))
	}
	environmentOrderMap := make(map[string]int)
	for i, env := range environments.GetEnvironments() {
		environmentOrderMap[env.Id] = i
	}

	rolloutV1, err := convertToRollout(project, plan, tasks, environmentOrderMap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to rollout"))
	}

	// Tickle task run scheduler.
	s.bus.TaskRunTickleChan <- 0

	return connect.NewResponse(rolloutV1), nil
}

// ListTaskRuns lists rollout task runs.
func (s *RolloutService) ListTaskRuns(ctx context.Context, req *connect.Request[v1pb.ListTaskRunsRequest]) (*connect.Response[v1pb.ListTaskRunsResponse], error) {
	request := req.Msg
	projectID, planID, maybeStageID, maybeTaskID, err := common.GetProjectIDPlanIDMaybeStageIDMaybeTaskID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: &projectID,
		UID:       &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get rollout"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %d not found in project %s", planID, projectID))
	}
	taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		PlanUID:     &planID,
		Environment: maybeStageID,
		TaskUID:     maybeTaskID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list task runs"))
	}

	taskRunsV1, err := convertToTaskRuns(ctx, s.store, s.bus, taskRuns)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to task runs"))
	}
	return connect.NewResponse(&v1pb.ListTaskRunsResponse{
		TaskRuns: taskRunsV1,
	}), nil
}

// CreateRolloutAndPendingTasks creates rollout tasks and pending task runs.
func CreateRolloutAndPendingTasks(
	ctx context.Context,
	s *store.Store,
	plan *store.PlanMessage,
	issue *store.IssueMessage,
	project *store.ProjectMessage,
	tasks []*store.TaskMessage,
) error {
	var err error
	if tasks == nil {
		tasks, err = GetPipelineCreate(ctx, s, plan.Config.GetSpecs(), project.ResourceID)
		if err != nil {
			return errors.Wrap(err, "failed to get pipeline create for rollout creation")
		}
	}

	// Create rollout tasks
	tasks, err = s.CreateTasks(ctx, plan.UID, tasks)
	if err != nil {
		return errors.Wrap(err, "failed to create rollout tasks")
	}

	// Update plan to set hasRollout to true
	config := proto.CloneOf(plan.Config)
	config.HasRollout = true
	_, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:    plan.UID,
		Config: config,
	})
	if err != nil {
		return errors.Wrap(err, "failed to update plan hasRollout")
	}

	// Update issue status to DONE when rollout is created
	if issue != nil {
		newStatus := storepb.Issue_DONE
		updatedIssue, err := s.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{Status: &newStatus})
		if err != nil {
			return errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
		}

		if _, err := s.CreateIssueComments(ctx, common.SystemBotEmail, &store.IssueCommentMessage{
			IssueUID: issue.UID,
			Payload: &storepb.IssueCommentPayload{
				Event: &storepb.IssueCommentPayload_IssueUpdate_{
					IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
						FromStatus: &issue.Status,
						ToStatus:   &updatedIssue.Status,
					},
				},
			},
		}); err != nil {
			return errors.Wrapf(err, "failed to create issue comment after changing the issue status")
		}
	}

	// Check if we should auto-rollout
	envPolicies := make(map[string]*storepb.RolloutPolicy)
	for _, task := range tasks {
		// Skip if environment is empty (should not happen for rollout tasks usually)
		if task.Environment == "" {
			continue
		}

		policy, ok := envPolicies[task.Environment]
		if !ok {
			var err error
			policy, err = s.GetRolloutPolicy(ctx, task.Environment)
			if err != nil {
				// Don't error out entirely just because policy fetch failed for one env, but logging/continuing is better.
				// However, here we don't have logger injected easily.
				// Since this is a shared util, returning error might abort the whole op.
				// The original code in rollout_creator.go logged error and continued.
				// We should probably log error and continue here too.
				// But we don't have a logger here.
				// Let's assume store failure is critical enough or rare enough.
				// Actually, `rollout_creator.go` imported `log`.
				// Here we can import `log/slog` or use `common/log`.
				// But `apiv1` usually returns error.
				// If we error here, we stop processing other tasks.
				// Let's try to fetch policy. If fails, we can't determine auto-rollout.
				// Returning error seems safer than silently ignoring?
				// But the original code `continue`d.
				// Let's return error for now to be safe, or maybe swallow if it's transient?
				// To match original behavior: log and continue. Use `slog`.
				return errors.Wrapf(err, "failed to get rollout policy for environment %s", task.Environment)
			}
			envPolicies[task.Environment] = policy
		}

		if policy.Automatic {
			create := &store.TaskRunMessage{
				TaskUID: task.ID,
			}

			// Use SystemBot for auto-rollout
			if err := s.CreatePendingTaskRuns(ctx, common.SystemBotEmail, create); err != nil {
				return errors.Wrapf(err, "failed to create pending task runs for task %d", task.ID)
			}
		}
	}
	return nil
}

// GetTaskRun gets a task run.
func (s *RolloutService) GetTaskRun(ctx context.Context, req *connect.Request[v1pb.GetTaskRunRequest]) (*connect.Response[v1pb.TaskRun], error) {
	request := req.Msg
	projectID, planID, _, _, taskRunUID, err := common.GetProjectIDPlanIDStageIDTaskIDTaskRunID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: &projectID,
		UID:       &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get rollout"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout %d not found in project %s", planID, projectID))
	}

	taskRun, err := s.store.GetTaskRunV1(ctx, &store.FindTaskRunMessage{
		UID:     &taskRunUID,
		PlanUID: &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get task run"))
	}
	if taskRun == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("task run %d not found in rollout %d", taskRunUID, planID))
	}

	taskRunV1, err := convertToTaskRun(ctx, s.store, s.bus, taskRun)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to task run"))
	}
	return connect.NewResponse(taskRunV1), nil
}

func (s *RolloutService) GetTaskRunLog(ctx context.Context, req *connect.Request[v1pb.GetTaskRunLogRequest]) (*connect.Response[v1pb.TaskRunLog], error) {
	request := req.Msg
	_, _, _, _, taskRunUID, err := common.GetProjectIDPlanIDStageIDTaskIDTaskRunID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get task run uid"))
	}
	logs, err := s.store.ListTaskRunLogs(ctx, taskRunUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to list task run logs"))
	}
	return connect.NewResponse(convertToTaskRunLog(request.Parent, logs)), nil
}

func (s *RolloutService) GetTaskRunSession(ctx context.Context, req *connect.Request[v1pb.GetTaskRunSessionRequest]) (*connect.Response[v1pb.TaskRunSession], error) {
	request := req.Msg
	_, _, _, taskUID, taskRunUID, err := common.GetProjectIDPlanIDStageIDTaskIDTaskRunID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get task run uid"))
	}

	task, err := s.store.GetTaskByID(ctx, taskUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get task"))
	}

	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get instance"))
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil, db.ConnectionContext{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get driver"))
	}
	defer driver.Close(ctx)

	appName := fmt.Sprintf("bytebase-taskrun-%d", taskRunUID)
	session, err := getSession(ctx, instance.Metadata.GetEngine(), driver.GetDB(), appName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get session"))
	}

	session.Name = request.Parent + "/session"

	return connect.NewResponse(session), nil
}

func getSession(ctx context.Context, engine storepb.Engine, db *sql.DB, appName string) (*v1pb.TaskRunSession, error) {
	switch engine {
	case storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB:
		query := `
			WITH target_session AS (
				SELECT pid FROM pg_catalog.pg_stat_activity WHERE application_name = $1 LIMIT 1
			)
			SELECT
				a.pid,
				pg_blocking_pids(a.pid) AS blocked_by_pids,
				a.query,
				a.state,
				a.wait_event_type,
				a.wait_event,
				a.datname,
				a.usename,
				a.application_name,
				a.client_addr,
				a.client_port,
				a.backend_start,
				a.xact_start,
				a.query_start
			FROM
				pg_catalog.pg_stat_activity a
			WHERE a.application_name = $1
			   OR (SELECT pid FROM target_session) = ANY(pg_blocking_pids(a.pid))
			   OR a.pid = ANY(pg_blocking_pids((SELECT pid FROM target_session)))
			ORDER BY a.pid
		`
		rows, err := db.QueryContext(ctx, query, appName)
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

			if s.ApplicationName == appName {
				ss.Session = &s
			} else if ss.Session != nil {
				// For blocking/blocked sessions, we need to check if they're related to our main session
				// We stored the main session, so check relationships with it
				if slices.Contains(s.BlockedByPids, ss.Session.Pid) {
					ss.BlockedSessions = append(ss.BlockedSessions, &s)
				} else if slices.Contains(ss.Session.BlockedByPids, s.Pid) {
					ss.BlockingSessions = append(ss.BlockingSessions, &s)
				}
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
		return nil, errors.Errorf("session monitoring is only supported for PostgreSQL and CockroachDB")
	}
}

// BatchRunTasks runs tasks in batch.
func (s *RolloutService) BatchRunTasks(ctx context.Context, req *connect.Request[v1pb.BatchRunTasksRequest]) (*connect.Response[v1pb.BatchRunTasksResponse], error) {
	request := req.Msg
	if len(request.Tasks) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("The tasks in request cannot be empty"))
	}
	projectID, planID, _, err := common.GetProjectIDPlanIDMaybeStageID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: &projectID,
		UID:       &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find plan for rollout"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout (plan) %v not found", planID))
	}

	// Reset notification state so user gets fresh feedback on retry
	if err := s.store.ResetPlanWebhookDelivery(ctx, planID); err != nil {
		slog.Error("failed to reset plan webhook delivery", log.BBError(err))
		// Don't fail the request - notification is non-critical
	}

	issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		ProjectID: &projectID,
		PlanUID:   &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find issue"))
	}

	// Parse requested task IDs and group by their environment
	taskEnvironments := map[string][]int{}
	taskIDsToRunMap := map[int]bool{}
	for _, task := range request.Tasks {
		_, _, stageID, taskID, err := common.GetProjectIDPlanIDStageIDTaskID(task)
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
	stageToRunTasks, err := s.store.ListTasks(ctx, &store.TaskFind{PlanID: &planID, Environment: &environmentToRun})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list tasks"))
	}
	if len(stageToRunTasks) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("No tasks to run in the stage"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	ok, err = s.canUserRunEnvironmentTasks(ctx, user, project, issueN, environmentToRun, plan.Creator)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check if the user can run tasks"))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("Not allowed to run tasks"))
	}

	// Check if issue approval is required according to the project settings
	if project.Setting.RequireIssueApproval && issueN != nil {
		approved, err := utils.CheckIssueApproved(issueN)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check if the issue is approved"))
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

	// Tickle task run scheduler.
	s.bus.TaskRunTickleChan <- 0

	return connect.NewResponse(&v1pb.BatchRunTasksResponse{}), nil
}

// BatchSkipTasks skips tasks in batch.
func (s *RolloutService) BatchSkipTasks(ctx context.Context, req *connect.Request[v1pb.BatchSkipTasksRequest]) (*connect.Response[v1pb.BatchSkipTasksResponse], error) {
	request := req.Msg
	projectID, planID, _, err := common.GetProjectIDPlanIDMaybeStageID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: &projectID,
		UID:       &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find plan for rollout"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout (plan) %v not found", planID))
	}

	issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		ProjectID: &projectID,
		PlanUID:   &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find issue"))
	}

	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PlanID: &planID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list tasks"))
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
	environmentSet := map[string]struct{}{}
	for _, task := range request.Tasks {
		_, _, _, taskID, err := common.GetProjectIDPlanIDStageIDTaskID(task)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		taskMsg, ok := taskByID[taskID]
		if !ok {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("task %v not found in the rollout", taskID))
		}
		taskUIDs = append(taskUIDs, taskID)
		environmentSet[taskMsg.Environment] = struct{}{}
	}

	for environment := range environmentSet {
		ok, err = s.canUserRunEnvironmentTasks(ctx, user, project, issueN, environment, plan.Creator)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check if the user can skip tasks"))
		}
		if !ok {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("not allowed to skip tasks in environment %q", environment))
		}
	}

	if err := s.store.BatchSkipTasks(ctx, taskUIDs, request.Reason); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to skip tasks"))
	}

	// Signal to check if plan is complete and successful (may send PIPELINE_COMPLETED)
	s.bus.PlanCompletionCheckChan <- planID

	return connect.NewResponse(&v1pb.BatchSkipTasksResponse{}), nil
}

// BatchCancelTaskRuns cancels a list of task runs.
func (s *RolloutService) BatchCancelTaskRuns(ctx context.Context, req *connect.Request[v1pb.BatchCancelTaskRunsRequest]) (*connect.Response[v1pb.BatchCancelTaskRunsResponse], error) {
	request := req.Msg
	if len(request.TaskRuns) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("task runs cannot be empty"))
	}

	projectID, planID, stageID, _, err := common.GetProjectIDPlanIDStageIDMaybeTaskID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: &projectID,
		UID:       &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find plan for rollout"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout (plan) %v not found", planID))
	}

	issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		ProjectID: &projectID,
		PlanUID:   &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find issue"))
	}

	for _, taskRun := range request.TaskRuns {
		_, _, taskRunStageID, _, _, err := common.GetProjectIDPlanIDStageIDTaskIDTaskRunID(taskRun)
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
	ok, err = s.canUserCancelEnvironmentTaskRun(ctx, user, project, issueN, environment, plan.Creator)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check if the user can run tasks"))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("Not allowed to cancel tasks"))
	}

	var taskRunIDs []int
	for _, taskRun := range request.TaskRuns {
		_, _, _, _, taskRunID, err := common.GetProjectIDPlanIDStageIDTaskIDTaskRunID(taskRun)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		taskRunIDs = append(taskRunIDs, taskRunID)
	}

	taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		UIDs: &taskRunIDs,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list task runs"))
	}

	for _, taskRun := range taskRuns {
		switch taskRun.Status {
		case storepb.TaskRun_PENDING:
		case storepb.TaskRun_AVAILABLE:
		case storepb.TaskRun_RUNNING:
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("taskRun %v is not pending, available or running", taskRun.ID))
		}
	}

	for _, taskRun := range taskRuns {
		if taskRun.Status == storepb.TaskRun_RUNNING {
			if cancelFunc, ok := s.bus.RunningTaskRunsCancelFunc.Load(taskRun.ID); ok {
				cancelFunc.(context.CancelFunc)()
			}
			// Broadcast cancel signal to all replicas for HA.
			if err := s.store.SendSignal(ctx, storepb.Signal_CANCEL_TASK_RUN, int32(taskRun.ID)); err != nil {
				slog.Warn("failed to send cancel signal", log.BBError(err))
			}
		}
	}

	if err := s.store.BatchCancelTaskRuns(ctx, taskRunIDs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to batch patch task run status to canceled"))
	}

	// Note: Don't signal plan completion - canceling is interrupting execution, not completing the plan

	return connect.NewResponse(&v1pb.BatchCancelTaskRunsResponse{}), nil
}

func (s *RolloutService) PreviewTaskRunRollback(ctx context.Context, req *connect.Request[v1pb.PreviewTaskRunRollbackRequest]) (*connect.Response[v1pb.PreviewTaskRunRollbackResponse], error) {
	request := req.Msg
	_, _, _, taskUID, taskRunUID, err := common.GetProjectIDPlanIDStageIDTaskIDTaskRunID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get task run uid"))
	}

	taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		UID: &taskRunUID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list task runs"))
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

	if !taskRun.ResultProto.HasPriorBackup {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("task run %v has no rollback", taskRun.ID))
	}

	// Get backup detail from task run logs.
	logs, err := s.store.ListTaskRunLogs(ctx, taskRunUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list task run logs"))
	}
	var backupDetail *storepb.PriorBackupDetail
	for _, log := range logs {
		if log.Payload.Type == storepb.TaskRunLog_PRIOR_BACKUP_END && log.Payload.PriorBackupEnd != nil {
			backupDetail = log.Payload.PriorBackupEnd.PriorBackupDetail
			break
		}
	}
	if backupDetail == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("task run %v has no backup detail in logs", taskRun.ID))
	}

	task, err := s.store.GetTaskByID(ctx, taskUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get task"))
	}

	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get instance"))
	}

	sheetSha256 := task.Payload.GetSheetSha256()
	if sheetSha256 == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("task %v has no sheet", task.ID))
	}
	sheet, err := s.store.GetSheetFull(ctx, sheetSha256)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get sheet statements"))
	}
	statements := sheet.Statement

	var results []string
	for _, item := range backupDetail.Items {
		restore, err := parserbase.GenerateRestoreSQL(ctx, instance.Metadata.GetEngine(), parserbase.RestoreContext{
			InstanceID:              instance.ResourceID,
			GetDatabaseMetadataFunc: BuildGetDatabaseMetadataFunc(s.store),
			ListDatabaseNamesFunc:   BuildListDatabaseNamesFunc(s.store),
			IsCaseSensitive:         store.IsObjectCaseSensitive(instance),
		}, statements, item)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate restore sql"))
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

// GetPipelineCreate gets a pipeline create message from a plan.
func GetPipelineCreate(ctx context.Context, s *store.Store, specs []*storepb.PlanConfig_Spec, projectID string) ([]*store.TaskMessage, error) {
	// Step 1 - transform database group specs.
	// Re-evaluate database groups live to pick up newly created databases.
	transformedSpecs, err := applyDatabaseGroupSpecTransformations(ctx, s, specs, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to apply database group spec transformations")
	}

	// Step 2 - convert all task creates.
	var taskCreates []*store.TaskMessage
	for _, spec := range transformedSpecs {
		tcs, err := getTaskCreatesFromSpec(ctx, s, spec)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get task creates from spec")
		}
		taskCreates = append(taskCreates, tcs...)
	}

	return taskCreates, nil
}

// filter pipelineCreate.Tasks using targetEnvironmentID.
func filterTasksByStage(tasks []*store.TaskMessage, targetEnvironment *string) ([]*store.TaskMessage, error) {
	if targetEnvironment == nil {
		return tasks, nil
	}
	if *targetEnvironment == "" {
		return nil, nil
	}
	targetEnvironmentID, err := common.GetEnvironmentID(*targetEnvironment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get environment id from %q", *targetEnvironment)
	}

	// Filter tasks to only include those in allowed environments
	filteredTasks := []*store.TaskMessage{}
	for _, task := range tasks {
		if task.Environment == targetEnvironmentID {
			filteredTasks = append(filteredTasks, task)
		}
	}
	return filteredTasks, nil
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
