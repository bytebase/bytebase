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
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ACLInterceptor is the v1 ACL interceptor for gRPC server.
type ACLInterceptor struct {
	store          *store.Store
	secret         string
	licenseService enterpriseAPI.LicenseService
	mode           common.ReleaseMode
}

// NewACLInterceptor returns a new v1 API ACL interceptor.
func NewACLInterceptor(store *store.Store, secret string, licenseService enterpriseAPI.LicenseService, mode common.ReleaseMode) *ACLInterceptor {
	return &ACLInterceptor{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		mode:           mode,
	}
}

// ACLInterceptor is the unary interceptor for gRPC API.
func (in *ACLInterceptor) ACLInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	user, err := in.getUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	if auth.IsAuthenticationAllowed(serverInfo.FullMethod) {
		return handler(ctx, request)
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated for method %q", serverInfo.FullMethod)
	}
	// Store workspace role into context.
	childCtx := context.WithValue(ctx, common.RoleContextKey, user.Role)
	if isOwnerOrDBA(user.Role) {
		return handler(childCtx, request)
	}

	methodName := getShortMethodName(serverInfo.FullMethod)
	if isOwnerAndDBAMethod(methodName) {
		return nil, status.Errorf(codes.PermissionDenied, "only workspace owner and DBA can access method %q", methodName)
	}

	if isProjectOwnerMethod(methodName) {
		projectIDs, err := getProjectIDs(request)
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
		for _, projectID := range projectIDs {
			projectRoles, err := in.getProjectRoles(ctx, user, projectID)
			if err != nil {
				return nil, status.Errorf(codes.PermissionDenied, err.Error())
			}
			if !projectRoles[api.Owner] {
				return nil, status.Errorf(codes.PermissionDenied, "only the owner of project %q can access method %q", projectID, methodName)
			}
		}
	}

	if isTransferDatabaseMethods(methodName) {
		projectIDs, err := in.getTransferDatabaseToProjects(ctx, request)
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
		for _, projectID := range projectIDs {
			projectRoles, err := in.getProjectRoles(ctx, user, projectID)
			if err != nil {
				return nil, status.Errorf(codes.PermissionDenied, err.Error())
			}
			if !projectRoles[api.Owner] {
				return nil, status.Errorf(codes.PermissionDenied, "only project owner can transfer database to project %q", projectID)
			}
		}
	}

	return handler(childCtx, request)
}

func (in *ACLInterceptor) getUser(ctx context.Context) (*store.UserMessage, error) {
	principalPtr := ctx.Value(common.PrincipalIDContextKey)
	if principalPtr == nil {
		return nil, nil
	}
	principalID := principalPtr.(int)
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
	if !in.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
		user.Role = api.Owner
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
		projectID, err := getProjectID(request.Project.Name)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	case *v1pb.DeleteProjectRequest:
		projectID, err := getProjectID(request.Name)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	case *v1pb.UndeleteProjectRequest:
		projectID, err := getProjectID(request.Name)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	case *v1pb.SetIamPolicyRequest:
		projectID, err := getProjectID(request.Project)
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
		instanceID, databaseName, err := getInstanceDatabaseID(request.Database.Name)
		if err != nil {
			return nil, err
		}
		database, err := in.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instanceID, DatabaseName: &databaseName})
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
