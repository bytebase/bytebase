package v1

import (
	"context"
	"log/slog"
	"strings"

	annotationsproto "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
		return nil, status.Errorf(codes.Internal, "auth context not found")
	}
	if err := in.populateRawResources(authContext, request, serverInfo.FullMethod); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to populate raw resources %s", err)
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
		return status.Errorf(codes.Internal, "auth context not found")
	}
	if err := in.populateRawResources(authContext, request, serverInfo.FullMethod); err != nil {
		return status.Errorf(codes.Internal, "failed to populate raw resources %s", err)
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

func (in *ACLInterceptor) doIAMPermissionCheck(ctx context.Context, fullMethod string, req any, user *store.UserMessage, authContext *common.AuthContext) (bool, []string, error) {
	if auth.IsAuthenticationAllowed(fullMethod, authContext) {
		return true, nil, nil
	}
	if authContext.AuthMethod == common.AuthMethodCustom {
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

func (*ACLInterceptor) populateRawResources(authContext *common.AuthContext, request any, method string) error {
	name := getResourceFromRequest(request, method)
	if name != "" {
		authContext.Resources = append(authContext.Resources, &common.Resource{
			Name: name,
		})
	}
	return nil
}

func getResourceFromRequest(request any, method string) string {
	var resource string
	pm, ok := request.(proto.Message)
	if !ok {
		return ""
	}
	mr := pm.ProtoReflect()

	parentDesc := mr.Descriptor().Fields().ByName("parent")
	if parentDesc != nil && proto.HasExtension(parentDesc.Options(), annotationsproto.E_ResourceReference) {
		v := mr.Get(parentDesc)
		return v.String()
	}
	nameDesc := mr.Descriptor().Fields().ByName("name")
	if nameDesc != nil && proto.HasExtension(nameDesc.Options(), annotationsproto.E_ResourceReference) {
		v := mr.Get(nameDesc)
		return v.String()
	}
	// This is primarily used by Get/SetIAMPolicy().
	resourceDesc := mr.Descriptor().Fields().ByName("resource")
	if resourceDesc != nil && proto.HasExtension(resourceDesc.Options(), annotationsproto.E_ResourceReference) {
		v := mr.Get(resourceDesc)
		return v.String()
	}
	methodTokens := strings.Split(method, "/")
	if len(methodTokens) != 3 {
		return ""
	}
	if strings.HasPrefix(methodTokens[2], "Create") {
		return ""
	}

	mr.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Name() == "name" {
			resource = v.String()
			return false
		}
		fdm := fd.Message()
		if fdm == nil {
			return true
		}
		if !proto.HasExtension(fdm.Options(), annotationsproto.E_Resource) {
			return true
		}

		nameDesc := fdm.Fields().ByName("name")
		if nameDesc != nil {
			rn := v.Message().Get(nameDesc)
			resource = rn.String()
			return false
		}
		return false
	})
	return resource
}
