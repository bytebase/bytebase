package v1

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ACLInterceptor is the v1 ACL interceptor for gRPC server.
type ACLInterceptor struct {
	store      *store.Store
	secret     string
	iamManager *iam.Manager
	profile    *config.Profile
}

// NewACLInterceptor returns a new v1 API ACL interceptor.
func NewACLInterceptor(store *store.Store, secret string, iamManager *iam.Manager, profile *config.Profile) *ACLInterceptor {
	return &ACLInterceptor{
		store:      store,
		secret:     secret,
		iamManager: iamManager,
		profile:    profile,
	}
}

// ACLInterceptor is the unary interceptor for gRPC API.
func (in *ACLInterceptor) ACLInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	user, err := in.getUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	if user != nil {
		// Store workspace role into context.
		role, err := in.iamManager.BackfillWorkspaceRoleForUser(ctx, user)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to backfill workspace role for user with error: %v", err.Error())
		}
		ctx = context.WithValue(ctx, common.RoleContextKey, role)
		ctx = context.WithValue(ctx, common.UserContextKey, user)
	}

	authContextAny := ctx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return nil, status.Errorf(codes.Internal, "auth context not found2")
	}

	if auth.IsAuthenticationAllowed(serverInfo.FullMethod, authContext) {
		return handler(ctx, request)
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated for method %q", serverInfo.FullMethod)
	}

	if err := in.checkIAMPermission(ctx, serverInfo.FullMethod, request, user, authContext); err != nil {
		return nil, err
	}

	return handler(ctx, request)
}

// ACLStreamInterceptor is the unary interceptor for gRPC API.
func (in *ACLInterceptor) ACLStreamInterceptor(request any, ss grpc.ServerStream, serverInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()

	user, err := in.getUser(ctx)
	if err != nil {
		return status.Errorf(codes.PermissionDenied, err.Error())
	}
	if user != nil {
		// Store workspace role into context.
		role, err := in.iamManager.BackfillWorkspaceRoleForUser(ctx, user)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to backfill workspace role for user with error: %v", err.Error())
		}
		ctx = context.WithValue(ctx, common.RoleContextKey, role)
		ctx = context.WithValue(ctx, common.UserContextKey, user)
		ss = overrideStream{ServerStream: ss, childCtx: ctx}
	}

	authContextAny := ctx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return status.Errorf(codes.Internal, "auth context not found3")
	}

	if auth.IsAuthenticationAllowed(serverInfo.FullMethod, authContext) {
		return handler(request, ss)
	}
	if user == nil {
		return status.Errorf(codes.Unauthenticated, "unauthenticated for method %q", serverInfo.FullMethod)
	}

	if err := in.checkIAMPermission(ctx, serverInfo.FullMethod, request, user, authContext); err != nil {
		return err
	}

	return handler(request, ss)
}

type overrideStream struct {
	childCtx context.Context
	grpc.ServerStream
}

func (s overrideStream) Context() context.Context {
	return s.childCtx
}

func (in *ACLInterceptor) getUser(ctx context.Context) (*store.UserMessage, error) {
	principalPtr := ctx.Value(common.PrincipalIDContextKey)
	if principalPtr == nil {
		return nil, nil
	}
	principalID, ok := principalPtr.(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := in.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "failed to get member for user %v in processing authorize request.", principalID)
	}
	if user == nil {
		return nil, status.Errorf(codes.PermissionDenied, "member not found for user %v in processing authorize request.", principalID)
	}
	if user.MemberDeleted {
		return nil, status.Errorf(codes.PermissionDenied, "the user %v has been deactivated by the admin.", principalID)
	}

	return user, nil
}

func hasPath(fieldMask *fieldmaskpb.FieldMask, want string) bool {
	if fieldMask == nil {
		return false
	}
	for _, path := range fieldMask.Paths {
		if path == want {
			return true
		}
	}
	return false
}

func (in *ACLInterceptor) checkIAMPermission(ctx context.Context, fullMethod string, req any, user *store.UserMessage, authContext *common.AuthContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			perr, ok := r.(error)
			if !ok {
				perr = errors.Errorf("%v", r)
			}
			err = errors.Errorf("iam check PANIC RECOVER, method: %v, err: %v", fullMethod, perr)

			slog.Error("iam check PANIC RECOVER", log.BBError(perr), log.BBStack("panic-stack"))
		}
	}()
	ok, extra, err := in.doIAMPermissionCheck(ctx, fullMethod, req, user, authContext)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check permission for method %q, extra %v, err: %v", fullMethod, extra, err)
	}
	if !ok {
		return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q, extra %v", fullMethod, authContext.Permission, extra)
	}

	return nil
}

func isSkippedMethod(fullMethod string, authContext *common.AuthContext) bool {
	if auth.IsAuthenticationAllowed(fullMethod, authContext) {
		return true
	}

	// Below are the skipped.
	switch fullMethod {
	// skip methods that are not considered to be resource-related.
	case
		v1pb.ActuatorService_GetResourcePackage_FullMethodName,
		v1pb.ActuatorService_GetActuatorInfo_FullMethodName,
		v1pb.ActuatorService_UpdateActuatorInfo_FullMethodName,
		v1pb.ActuatorService_DeleteCache_FullMethodName,
		v1pb.ActuatorService_ListDebugLog_FullMethodName,
		v1pb.AnomalyService_SearchAnomalies_FullMethodName,
		v1pb.AuthService_GetUser_FullMethodName,
		v1pb.AuthService_ListUsers_FullMethodName,
		v1pb.AuthService_CreateUser_FullMethodName,
		v1pb.AuthService_UpdateUser_FullMethodName,
		v1pb.AuthService_DeleteUser_FullMethodName,
		v1pb.AuthService_UndeleteUser_FullMethodName,
		v1pb.AuthService_Login_FullMethodName,
		v1pb.AuthService_Logout_FullMethodName,
		v1pb.CelService_BatchParse_FullMethodName,
		v1pb.CelService_BatchDeparse_FullMethodName,
		v1pb.SQLService_Query_FullMethodName,
		// TODO(steven): maybe needs to add a permission to check.
		v1pb.SQLService_Execute_FullMethodName,
		v1pb.SQLService_SearchQueryHistories_FullMethodName,
		v1pb.SQLService_Export_FullMethodName,
		v1pb.SQLService_DifferPreview_FullMethodName,
		v1pb.SQLService_Check_FullMethodName,
		v1pb.SQLService_ParseMyBatisMapper_FullMethodName,
		v1pb.SQLService_Pretty_FullMethodName,
		v1pb.SQLService_StringifyMetadata_FullMethodName,
		v1pb.SQLService_GenerateRestoreSQL_FullMethodName,
		v1pb.SubscriptionService_GetSubscription_FullMethodName,
		v1pb.SubscriptionService_GetFeatureMatrix_FullMethodName,
		v1pb.SubscriptionService_UpdateSubscription_FullMethodName,
		// TODO(p0ny): permission for review config service.
		v1pb.ReviewConfigService_CreateReviewConfig_FullMethodName,
		v1pb.ReviewConfigService_ListReviewConfigs_FullMethodName,
		v1pb.ReviewConfigService_GetReviewConfig_FullMethodName,
		v1pb.ReviewConfigService_UpdateReviewConfig_FullMethodName,
		v1pb.ReviewConfigService_DeleteReviewConfig_FullMethodName:
		return true
	// skip checking for sheet service because we want to
	// discriminate bytebase artifact sheets and user sheets first.
	// TODO(p0ny): implement
	case
		v1pb.SheetService_CreateSheet_FullMethodName,
		v1pb.SheetService_GetSheet_FullMethodName,
		v1pb.SheetService_UpdateSheet_FullMethodName:
		return true
	// skip checking for sheet service because we want to
	// discriminate bytebase artifact sheets and user sheets first.
	// TODO(p0ny): implement
	case
		v1pb.WorksheetService_CreateWorksheet_FullMethodName,
		v1pb.WorksheetService_GetWorksheet_FullMethodName,
		v1pb.WorksheetService_SearchWorksheets_FullMethodName,
		v1pb.WorksheetService_UpdateWorksheet_FullMethodName,
		v1pb.WorksheetService_UpdateWorksheetOrganizer_FullMethodName,
		v1pb.WorksheetService_DeleteWorksheet_FullMethodName:
		return true
	// handled in the method because we need to consider branch.Creator.
	case
		v1pb.BranchService_UpdateBranch_FullMethodName,
		v1pb.BranchService_DeleteBranch_FullMethodName,
		v1pb.BranchService_MergeBranch_FullMethodName,
		v1pb.BranchService_RebaseBranch_FullMethodName:
		return true
	// handled in the method because we need to consider changelist.Creator.
	case
		v1pb.ChangelistService_UpdateChangelist_FullMethodName,
		v1pb.ChangelistService_DeleteChangelist_FullMethodName:
		return true
	// handled in the method because we need to consider plan.Creator.
	case
		v1pb.PlanService_UpdatePlan_FullMethodName,
		// TODO: maybe needs to add permission checks.
		v1pb.PlanService_BatchCancelPlanCheckRuns_FullMethodName:
		return true
	// handled in the method because we need to consider issue.Creator and issue type.
	// additional bb.plans.action and bb.rollouts.action permissions are required if the issue type is change database.
	case
		v1pb.IssueService_GetIssue_FullMethodName,
		v1pb.IssueService_ListIssueComments_FullMethodName,
		v1pb.IssueService_CreateIssue_FullMethodName,
		v1pb.IssueService_CreateIssueComment_FullMethodName,
		v1pb.IssueService_UpdateIssue_FullMethodName,
		v1pb.IssueService_BatchUpdateIssuesStatus_FullMethodName,
		v1pb.IssueService_UpdateIssueComment_FullMethodName:
		return true
	// skip checking for custom approval.
	case
		v1pb.IssueService_ApproveIssue_FullMethodName,
		v1pb.IssueService_RejectIssue_FullMethodName,
		v1pb.IssueService_RequestIssue_FullMethodName:
		return true
	// skip checking for the rollout-related.
	// these are determined by the rollout policy.
	case
		v1pb.RolloutService_BatchCancelTaskRuns_FullMethodName,
		v1pb.RolloutService_BatchSkipTasks_FullMethodName,
		v1pb.RolloutService_BatchRunTasks_FullMethodName:
		return true
	// handled in the method because checking is complex.
	case
		v1pb.AuditLogService_SearchAuditLogs_FullMethodName,
		v1pb.AuditLogService_ExportAuditLogs_FullMethodName,
		v1pb.InstanceService_SearchInstances_FullMethodName,
		v1pb.DatabaseService_ListSlowQueries_FullMethodName,
		v1pb.DatabaseService_ListDatabases_FullMethodName,
		v1pb.DatabaseService_SearchDatabases_FullMethodName,
		v1pb.IssueService_ListIssues_FullMethodName,
		v1pb.IssueService_SearchIssues_FullMethodName,

		v1pb.ProjectService_SearchProjects_FullMethodName,
		v1pb.PlanService_ListPlans_FullMethodName,
		v1pb.PlanService_SearchPlans_FullMethodName,
		v1pb.UserGroupService_DeleteUserGroup_FullMethodName,
		v1pb.UserGroupService_UpdateUserGroup_FullMethodName:
		return true
	}
	return false
}

func (in *ACLInterceptor) doIAMPermissionCheck(ctx context.Context, fullMethod string, req any, user *store.UserMessage, authContext *common.AuthContext) (bool, []string, error) {
	if isSkippedMethod(fullMethod, authContext) {
		return true, nil, nil
	}

	p := authContext.Permission
	switch fullMethod {
	// special cases for bb.instance.get permission check.
	// we permit users to get instances (and all the related info) if they can get any database in the instance, even if they don't have bb.instance.get permission.
	case
		v1pb.InstanceService_GetInstance_FullMethodName,
		v1pb.InstanceRoleService_GetInstanceRole_FullMethodName,
		v1pb.InstanceRoleService_ListInstanceRoles_FullMethodName:
		var instanceID string
		var err error
		switch r := req.(type) {
		case *v1pb.GetInstanceRequest:
			instanceID, err = common.GetInstanceID(r.GetName())
		case *v1pb.GetInstanceRoleRequest:
			instanceID, _, err = common.GetInstanceRoleID(r.GetName())
		case *v1pb.ListInstanceRolesRequest:
			instanceID, err = common.GetInstanceID(r.GetParent())
		}
		if err != nil {
			return false, []string{instanceID}, err
		}
		ok, err := in.checkIAMPermissionInstancesGet(ctx, user, instanceID)
		return ok, []string{instanceID}, err
	}

	projectIDs, ok := common.GetProjectIDsFromContext(ctx)
	if !ok {
		return false, projectIDs, errors.Errorf("failed to get project ids")
	}
	ok, err := in.iamManager.CheckPermission(ctx, p, user, projectIDs...)
	return ok, projectIDs, err
}

func (in *ACLInterceptor) checkIAMPermissionInstancesGet(ctx context.Context, user *store.UserMessage, instanceID string) (bool, error) {
	// fast path for Admins and DBAs.
	ok, err := in.iamManager.CheckPermission(ctx, iam.PermissionInstancesGet, user)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}

	databaseFind := &store.FindDatabaseMessage{
		InstanceID:  &instanceID,
		ShowDeleted: true,
	}
	databases, err := searchDatabases(ctx, in.store, in.iamManager, databaseFind)
	if err != nil {
		return false, errors.Wrapf(err, "failed to search databases")
	}
	return len(databases) > 0, nil
}
