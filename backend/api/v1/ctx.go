package v1

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type ContextProvider struct {
	s *store.Store
}

func NewContextProvider(s *store.Store) *ContextProvider {
	return &ContextProvider{
		s: s,
	}
}

// ContextProvider is the unary interceptor for gRPC API.
func (p *ContextProvider) UnaryInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	projectIDs, err := p.do(ctx, serverInfo.FullMethod, request)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project ids for method %q, err: %v", serverInfo.FullMethod, err)
	}

	ctx = common.WithProjectIDs(ctx, projectIDs)

	return handler(ctx, request)
}

func (p *ContextProvider) do(ctx context.Context, fullMethod string, req any) ([]string, error) {
	switch fullMethod {
	case
		v1pb.RolloutService_GetRollout_FullMethodName,
		v1pb.RolloutService_CreateRollout_FullMethodName,
		v1pb.RolloutService_PreviewRollout_FullMethodName,
		v1pb.RolloutService_ListTaskRuns_FullMethodName,
		v1pb.RolloutService_GetTaskRunLog_FullMethodName:
		return p.getProjectIDsForRolloutService(ctx, req)

	case
		v1pb.PlanService_GetPlan_FullMethodName,
		v1pb.PlanService_CreatePlan_FullMethodName,
		v1pb.PlanService_ListPlanCheckRuns_FullMethodName,
		v1pb.PlanService_RunPlanChecks_FullMethodName:
		return p.getProjectIDsForPlanService(ctx, req)
	}

	return nil, nil
}

func (*ContextProvider) getProjectIDsForRolloutService(_ context.Context, req any) ([]string, error) {
	var projects, rollouts, plans, tasks, taskRuns []string
	switch r := req.(type) {
	case *v1pb.GetRolloutRequest:
		rollouts = append(rollouts, r.GetName())
	case *v1pb.CreateRolloutRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.PreviewRolloutRequest:
		projects = append(projects, r.GetProject())
	case *v1pb.GetPlanRequest:
		plans = append(plans, r.GetName())
	case *v1pb.CreatePlanRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.ListTaskRunsRequest:
		tasks = append(tasks, r.GetParent())
	case *v1pb.GetTaskRunLogRequest:
		taskRuns = append(taskRuns, r.GetParent())
	case *v1pb.ListPlanCheckRunsRequest:
		plans = append(plans, r.GetParent())
	case *v1pb.RunPlanChecksRequest:
		plans = append(plans, r.GetName())
	}

	var projectIDs []string
	for _, plan := range plans {
		projectID, _, err := common.GetProjectIDPlanID(plan)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse plan %q", plan)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, project := range projects {
		projectID, err := common.GetProjectID(project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse project %q", project)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, rollout := range rollouts {
		projectID, _, err := common.GetProjectIDRolloutID(rollout)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rollout %q", rollout)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, task := range tasks {
		projectID, _, _, _, err := common.GetProjectIDRolloutIDMaybeStageIDMaybeTaskID(task)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse task %q", task)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, taskRun := range taskRuns {
		projectID, _, _, _, _, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(taskRun)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse taskRun %q", taskRun)
		}
		projectIDs = append(projectIDs, projectID)
	}

	return utils.Uniq(projectIDs), nil
}

func (*ContextProvider) getProjectIDsForPlanService(_ context.Context, req any) ([]string, error) {
	var projects, plans []string
	switch r := req.(type) {
	case *v1pb.GetPlanRequest:
		plans = append(plans, r.GetName())
	case *v1pb.CreatePlanRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.ListPlanCheckRunsRequest:
		plans = append(plans, r.GetParent())
	case *v1pb.RunPlanChecksRequest:
		plans = append(plans, r.GetName())
	}

	var projectIDs []string
	for _, plan := range plans {
		projectID, _, err := common.GetProjectIDPlanID(plan)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse plan %q", plan)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, project := range projects {
		projectID, err := common.GetProjectID(project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse project %q", project)
		}
		projectIDs = append(projectIDs, projectID)
	}

	return utils.Uniq(projectIDs), nil
}

func getDatabaseMessage(ctx context.Context, s *store.Store, databaseResourceName string) (*store.DatabaseMessage, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(databaseResourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %q", databaseResourceName)
	}

	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found")
	}

	find := &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		ShowDeleted:         true,
	}
	database, err := s.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database %q not found", databaseResourceName)
	}
	return database, nil
}
