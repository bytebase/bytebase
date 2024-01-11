package v1

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ACLInterceptor is the v1 ACL interceptor for gRPC server.
type ACLInterceptor struct {
	store          *store.Store
	secret         string
	licenseService enterprise.LicenseService
	iamManager     *iam.Manager
	profile        *config.Profile
}

// NewACLInterceptor returns a new v1 API ACL interceptor.
func NewACLInterceptor(store *store.Store, secret string, licenseService enterprise.LicenseService, iamManager *iam.Manager, profile *config.Profile) *ACLInterceptor {
	return &ACLInterceptor{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		iamManager:     iamManager,
		profile:        profile,
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
		ctx = context.WithValue(ctx, common.RoleContextKey, user.Role)
		ctx = context.WithValue(ctx, common.UserContextKey, user)
	}

	if auth.IsAuthenticationAllowed(serverInfo.FullMethod) {
		return handler(ctx, request)
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated for method %q", serverInfo.FullMethod)
	}
	if !in.profile.DevelopmentIAM && isOwnerOrDBA(user.Role) {
		return handler(ctx, request)
	}

	if err := in.aclInterceptorDo(ctx, serverInfo.FullMethod, request, user); err != nil {
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
		ctx = context.WithValue(ctx, common.RoleContextKey, user.Role)
		ctx = context.WithValue(ctx, common.UserContextKey, user)
		ss = overrideStream{ServerStream: ss, childCtx: ctx}
	}

	if auth.IsAuthenticationAllowed(serverInfo.FullMethod) {
		return handler(request, ss)
	}
	if user == nil {
		return status.Errorf(codes.Unauthenticated, "unauthenticated for method %q", serverInfo.FullMethod)
	}
	if !in.profile.DevelopmentIAM && isOwnerOrDBA(user.Role) {
		return handler(request, ss)
	}

	if err := in.aclInterceptorDo(ctx, serverInfo.FullMethod, request, user); err != nil {
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

func (in *ACLInterceptor) aclInterceptorDo(ctx context.Context, fullMethod string, request any, user *store.UserMessage) error {
	if isOwnerAndDBAMethod(fullMethod) {
		return status.Errorf(codes.PermissionDenied, "only workspace owner and DBA can access method %q", fullMethod)
	}

	if isProjectOwnerMethod(fullMethod) {
		projectIDs, err := getProjectIDs(request)
		if err != nil {
			return status.Errorf(codes.PermissionDenied, err.Error())
		}
		for _, projectID := range projectIDs {
			projectRoles, err := in.getProjectRoles(ctx, user, projectID)
			if err != nil {
				return status.Errorf(codes.PermissionDenied, err.Error())
			}
			if !projectRoles[api.ProjectOwner] {
				return status.Errorf(codes.PermissionDenied, "only the owner of project %q can access method %q", projectID, fullMethod)
			}
		}
	}

	if isTransferDatabaseMethods(fullMethod) {
		projectIDs, err := in.getTransferDatabaseToProjects(ctx, request)
		if err != nil {
			return status.Errorf(codes.PermissionDenied, err.Error())
		}
		for _, projectID := range projectIDs {
			projectRoles, err := in.getProjectRoles(ctx, user, projectID)
			if err != nil {
				return status.Errorf(codes.PermissionDenied, err.Error())
			}
			if !projectRoles[api.ProjectOwner] {
				return status.Errorf(codes.PermissionDenied, "only project owner can transfer database to project %q", projectID)
			}
		}
	}

	if in.profile.DevelopmentIAM {
		return in.checkIAMPermission(ctx, fullMethod, request, user)
	}

	return nil
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

	// If RBAC feature is not enabled, all users are treated as OWNER.
	if in.licenseService.IsFeatureEnabled(api.FeatureRBAC) != nil {
		user.Role = api.WorkspaceAdmin
		// TODO(p0ny): append projectOwner, projectQuerier, projectExporter as we will split workspaceAdmin into these roles.
		user.Roles = uniq(append(user.Roles, api.WorkspaceAdmin))
	}
	return user, nil
}

func (in *ACLInterceptor) getProjectRoles(ctx context.Context, user *store.UserMessage, projectID string) (map[api.Role]bool, error) {
	projectPolicy, err := in.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &projectID})
	if err != nil {
		return nil, err
	}
	roles := map[api.Role]bool{}
	for _, binding := range projectPolicy.Bindings {
		for _, member := range binding.Members {
			if member.ID == user.ID {
				roles[binding.Role] = true
				break
			}
		}
	}
	return roles, nil
}

func getProjectIDs(req any) ([]string, error) {
	switch request := req.(type) {
	case *v1pb.UpdateProjectRequest:
		if request.Project == nil {
			return nil, errors.Errorf("project not found")
		}
		projectID, err := common.GetProjectID(request.Project.Name)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	case *v1pb.DeleteProjectRequest:
		projectID, err := common.GetProjectID(request.Name)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	case *v1pb.UndeleteProjectRequest:
		projectID, err := common.GetProjectID(request.Name)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	case *v1pb.SetIamPolicyRequest:
		projectID, err := common.GetProjectID(request.Project)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	}

	return nil, nil
}

func (in *ACLInterceptor) getTransferDatabaseToProjects(ctx context.Context, req any) ([]string, error) {
	var requests []*v1pb.UpdateDatabaseRequest
	if request, ok := req.(*v1pb.UpdateDatabaseRequest); ok {
		requests = append(requests, request)
	}
	if request, ok := req.(*v1pb.BatchUpdateDatabasesRequest); ok {
		requests = request.Requests
	}

	projectIDMap := make(map[string]bool)
	for _, request := range requests {
		if !hasPath(request.UpdateMask, "project") || request.Database == nil {
			continue
		}
		instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Database.Name)
		if err != nil {
			return nil, err
		}
		instance, err := in.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
		}
		database, err := in.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instanceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return nil, err
		}
		if database == nil {
			return nil, errors.Errorf("database %q not found", request.Database.Name)
		}
		projectIDMap[database.ProjectID] = true
	}
	var projectIDs []string
	for projectID := range projectIDMap {
		projectIDs = append(projectIDs, projectID)
	}
	return projectIDs, nil
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
