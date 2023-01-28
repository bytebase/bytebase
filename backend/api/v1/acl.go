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
func (in *ACLInterceptor) ACLInterceptor(ctx context.Context, request interface{}, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
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
			projectRole, err := in.getProjectMember(ctx, user, projectID)
			if err != nil {
				return nil, status.Errorf(codes.PermissionDenied, err.Error())
			}
			if projectRole != api.Owner {
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
			projectRole, err := in.getProjectMember(ctx, user, projectID)
			if err != nil {
				return nil, status.Errorf(codes.PermissionDenied, err.Error())
			}
			if projectRole != api.Owner {
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
	if in.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
		user.Role = api.Owner
	}
	return user, nil
}

func (in *ACLInterceptor) getProjectMember(ctx context.Context, user *store.UserMessage, projectID string) (api.Role, error) {
	projectPolicy, err := in.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &projectID})
	if err != nil {
		return api.UnknownRole, err
	}
	for _, binding := range projectPolicy.Bindings {
		for _, member := range binding.Members {
			if member.ID == user.ID {
				return binding.Role, nil
			}
		}
	}
	return api.UnknownRole, nil
}

func getProjectIDs(req interface{}) ([]string, error) {
	if request, ok := req.(*v1pb.UpdateProjectRequest); ok {
		if request.Project == nil {
			return nil, errors.Errorf("project not found")
		}
		projectID, err := getProjectID(request.Project.Name)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	}
	if request, ok := req.(*v1pb.DeleteProjectRequest); ok {
		projectID, err := getProjectID(request.Name)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	}
	if request, ok := req.(*v1pb.UndeleteProjectRequest); ok {
		projectID, err := getProjectID(request.Name)
		if err != nil {
			return nil, err
		}
		return []string{projectID}, nil
	}

	return nil, nil
}

func (in *ACLInterceptor) getTransferDatabaseToProjects(ctx context.Context, req interface{}) ([]string, error) {
	var requests []*v1pb.UpdateDatabaseRequest
	if request, ok := req.(*v1pb.UpdateDatabaseRequest); ok {
		requests = append(requests, request)
	}
	if request, ok := req.(*v1pb.BatchUpdateDatabasesRequest); ok {
		requests = request.Requests
	}

	projectIDMap := make(map[string]bool)
	for _, request := range requests {
		if !hasPath(request.UpdateMask, "database.project") || request.Database == nil {
			continue
		}
		environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(request.Database.Name)
		if err != nil {
			return nil, err
		}
		database, err := in.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{EnvironmentID: &environmentID, InstanceID: &instanceID, DatabaseName: &databaseName})
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
