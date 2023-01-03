package v1

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/server/api/auth"
	"github.com/bytebase/bytebase/store"
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
	if isOwnerOrDBA(user) {
		return handler(childCtx, request)
	}

	methodName := getShortMethodName(serverInfo.FullMethod)
	if isOwnerAndDBAMethod(methodName) {
		return nil, status.Errorf(codes.PermissionDenied, "only workspace owner and DBA can access method %q", methodName)
	}

	// TODO(d): implement authorization checks for project resources.

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

func (*ACLInterceptor) getProjectMember(_ context.Context, _ *store.UserMessage, _ string) (api.Role, error) {
	return api.Developer, nil
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
		environmentID, instanceID, databaseID, err := getEnvironmentInstanceDatabaseID(request.Database.Name)
		if err != nil {
			return nil, err
		}
		database, err := in.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{EnvironmentID: &environmentID, InstanceID: &instanceID, DatabaseID: &databaseID})
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
