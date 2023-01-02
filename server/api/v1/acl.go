package v1

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

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
	role, err := in.getWorkspaceRole(ctx, serverInfo.FullMethod)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	methodName := getShortMethodName(serverInfo.FullMethod)
	if role == api.Developer && isOwnerAndDBAMethod(methodName) {
		return nil, status.Errorf(codes.PermissionDenied, "only workspace owner and DBA can access method %q", methodName)
	}

	// TODO(d): implement authorization checks for project resources.

	// TODO(d): check ToProject for database transfer.
	if isTransferDatabaseMethods(methodName) {
		_, err := in.getTransferDatabaseToProjects(ctx, request)
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
	}

	// Stores principalID into context.
	childCtx := context.WithValue(ctx, common.RoleContextKey, role)
	return handler(childCtx, request)
}

func (in *ACLInterceptor) getWorkspaceRole(ctx context.Context, fullMethodName string) (api.Role, error) {
	// If RBAC feature is not enabled, all users are treated as OWNER.
	if !in.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
		return api.Owner, nil
	}

	principalPtr := ctx.Value(common.PrincipalIDContextKey)
	if principalPtr == nil {
		if auth.IsAuthenticationAllowed(fullMethodName) {
			return api.UnknownRole, nil
		}
		return api.UnknownRole, status.Errorf(codes.PermissionDenied, "principal key doesn't exist in the request context")
	}

	principalID := principalPtr.(int)
	user, err := in.store.GetUserByID(ctx, principalID)
	if err != nil {
		return api.UnknownRole, status.Errorf(codes.PermissionDenied, "failed to get member for user %v in processing authorize request.", principalID)
	}
	if user == nil {
		return api.UnknownRole, status.Errorf(codes.PermissionDenied, "member not found for user %v in processing authorize request.", principalID)
	}
	if user.MemberDeleted {
		return api.UnknownRole, status.Errorf(codes.PermissionDenied, "the user %v has been deactivated by the admin.", principalID)
	}
	return user.Role, nil
}

func (*ACLInterceptor) getTransferDatabaseToProjects(_ context.Context, request interface{}) ([]string, error) {
	// projectMap := make(map[string]bool)
	if updateDatabaseRequest, ok := request.(*v1pb.UpdateDatabaseRequest); ok {
		if !hasPath(updateDatabaseRequest.UpdateMask, "database.project") {
			return nil, nil
		}
	}
	return nil, nil
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
