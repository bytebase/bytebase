package v1

import (
	"context"
	"log/slog"
	"regexp"
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
func (in *ACLInterceptor) ACLInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
	defer func() {
		if r := recover(); r != nil {
			perr, ok := r.(error)
			if !ok {
				perr = errors.Errorf("%v", r)
			}
			err = errors.Errorf("iam check PANIC RECOVER, method: %v, err: %v", serverInfo.FullMethod, perr)

			slog.Error("iam check PANIC RECOVER", log.BBError(perr), log.BBStack("panic-stack"))
		}
	}()
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
	if err := in.populateRawResources(ctx, authContext, request, serverInfo.FullMethod); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to populate raw resources %s", err)
	}

	if auth.IsAuthenticationAllowed(serverInfo.FullMethod, authContext) {
		return handler(ctx, request)
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated for method %q", serverInfo.FullMethod)
	}

	ok, extra, err := in.doIAMPermissionCheck(ctx, serverInfo.FullMethod, request, user, authContext)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission for method %q, extra %v, err: %v", serverInfo.FullMethod, extra, err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q, extra %v", serverInfo.FullMethod, authContext.Permission, extra)
	}

	return handler(ctx, request)
}

// ACLStreamInterceptor is the unary interceptor for gRPC API.
func (in *ACLInterceptor) ACLStreamInterceptor(request any, ss grpc.ServerStream, serverInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			perr, ok := r.(error)
			if !ok {
				perr = errors.Errorf("%v", r)
			}
			err = errors.Errorf("iam check PANIC RECOVER, method: %v, err: %v", serverInfo.FullMethod, perr)

			slog.Error("iam check PANIC RECOVER", log.BBError(perr), log.BBStack("panic-stack"))
		}
	}()

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
	if err := in.populateRawResources(ctx, authContext, request, serverInfo.FullMethod); err != nil {
		return status.Errorf(codes.Internal, "failed to populate raw resources %s", err)
	}

	if auth.IsAuthenticationAllowed(serverInfo.FullMethod, authContext) {
		return handler(request, ss)
	}
	if user == nil {
		return status.Errorf(codes.Unauthenticated, "unauthenticated for method %q", serverInfo.FullMethod)
	}

	ok, extra, err := in.doIAMPermissionCheck(ctx, serverInfo.FullMethod, request, user, authContext)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check permission for method %q, extra %v, err: %v", serverInfo.FullMethod, extra, err)
	}
	if !ok {
		return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q, extra %v", serverInfo.FullMethod, authContext.Permission, extra)
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

func (in *ACLInterceptor) doIAMPermissionCheck(ctx context.Context, fullMethod string, req any, user *store.UserMessage, authContext *common.AuthContext) (bool, []string, error) {
	if auth.IsAuthenticationAllowed(fullMethod, authContext) {
		return true, nil, nil
	}
	if authContext.AuthMethod == common.AuthMethodCustom {
		return true, nil, nil
	}
	if authContext.AuthMethod == common.AuthMethodIAM {
		// Handle GetProject() error status.
		if len(authContext.Resources) == 0 {
			return false, nil, errors.Errorf("no resource found for IAM auth method")
		}
		for _, resource := range authContext.Resources {
			slog.Debug("IAM auth method", slog.String("method", fullMethod), slog.String("permission", authContext.Permission), slog.String("project", resource.ProjectID), slog.Bool("workspace", resource.Workspace))
			if resource.Workspace {
				ok, err := in.iamManager.CheckPermission(ctx, authContext.Permission, user)
				if err != nil {
					return false, nil, err
				}
				if !ok {
					return false, nil, nil
				}
			} else {
				ok, err := in.iamManager.CheckPermission(ctx, authContext.Permission, user, resource.ProjectID)
				if err != nil {
					return false, []string{resource.ProjectID}, err
				}
				if !ok {
					return false, []string{resource.ProjectID}, nil
				}
			}
		}
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

var projectRegex = regexp.MustCompile(`^projects/[^/]+`)
var databaseRegex = regexp.MustCompile(`^instances/[^/]+/databases/[^/]+`)

func (in *ACLInterceptor) populateRawResources(ctx context.Context, authContext *common.AuthContext, request any, method string) error {
	if authContext.AllowWithoutCredential {
		return nil
	}
	if authContext.AuthMethod != common.AuthMethodIAM {
		return nil
	}
	resources := getResourceFromRequest(request, method)
	for _, resource := range resources {
		switch {
		case strings.HasPrefix(resource.Name, "projects/"):
			project := projectRegex.FindString(resource.Name)
			if project == "" {
				return errors.Errorf("invalid project resource %q", resource.Name)
			}
			projectID, err := common.GetProjectID(project)
			if err != nil {
				return err
			}
			resource.ProjectID = projectID
		case strings.HasPrefix(resource.Name, "instances/") && strings.Contains(resource.Name, "/databases/"):
			match := databaseRegex.FindString(resource.Name)
			if match != "" {
				database, err := getDatabaseMessage(ctx, in.store, match)
				if err != nil {
					return errors.Wrapf(err, "failed to get database %q", match)
				}
				resource.ProjectID = database.ProjectID
			}
		default:
			resource.Workspace = true
		}
	}
	authContext.Resources = resources
	return nil
}

func getResourceFromRequest(request any, method string) []*common.Resource {
	pm, ok := request.(proto.Message)
	if !ok {
		return nil
	}
	mr := pm.ProtoReflect()

	methodTokens := strings.Split(method, "/")
	if len(methodTokens) != 3 {
		return nil
	}
	shortMethod := methodTokens[2]

	var resources []*common.Resource
	if strings.HasPrefix(shortMethod, "Batch") {
		requestsDesc := mr.Descriptor().Fields().ByName("requests")
		if requestsDesc != nil {
			requestsValue := mr.Get(requestsDesc)
			requestsValueList := requestsValue.List()
			shortMethodWithoutBatch := strings.TrimSuffix(strings.TrimPrefix(shortMethod, "Batch"), "s")
			for i := 0; i < requestsValueList.Len(); i++ {
				r := requestsValueList.Get(i).Message()
				resource := getResourceFromSingleRequest(r, shortMethodWithoutBatch)
				if resource != nil {
					resources = append(resources, resource)
				}
			}
			return resources
		}
	}
	resource := getResourceFromSingleRequest(mr, shortMethod)
	if resource != nil {
		resources = append(resources, resource)
	}
	return resources
}

func getResourceFromSingleRequest(mr protoreflect.Message, shortMethod string) *common.Resource {
	parentDesc := mr.Descriptor().Fields().ByName("parent")
	if parentDesc != nil && proto.HasExtension(parentDesc.Options(), annotationsproto.E_ResourceReference) {
		v := mr.Get(parentDesc)
		return &common.Resource{Name: v.String()}
	}
	nameDesc := mr.Descriptor().Fields().ByName("name")
	if nameDesc != nil && proto.HasExtension(nameDesc.Options(), annotationsproto.E_ResourceReference) {
		v := mr.Get(nameDesc)
		return &common.Resource{Name: v.String()}
	}
	// This is primarily used by Get/SetIAMPolicy().
	resourceFieldDesc := mr.Descriptor().Fields().ByName("resource")
	if resourceFieldDesc != nil && proto.HasExtension(resourceFieldDesc.Options(), annotationsproto.E_ResourceReference) {
		v := mr.Get(resourceFieldDesc)
		return &common.Resource{Name: v.String()}
	}
	// This is primarily used by AddWebhook().
	projectFieldDesc := mr.Descriptor().Fields().ByName("project")
	if projectFieldDesc != nil && proto.HasExtension(projectFieldDesc.Options(), annotationsproto.E_ResourceReference) {
		v := mr.Get(projectFieldDesc)
		return &common.Resource{Name: v.String()}
	}

	// Listing top-level resources.
	if strings.HasPrefix(shortMethod, "List") {
		return &common.Resource{Workspace: true}
	}

	isCreate := strings.HasPrefix(shortMethod, "Create")
	isUpdate := strings.HasPrefix(shortMethod, "Update")
	isRemove := strings.HasPrefix(shortMethod, "Remove")
	if !isCreate && !isUpdate && !isRemove {
		return nil
	}
	var resourceName string
	if isCreate {
		resourceName = strings.TrimPrefix(shortMethod, "Create")
	}
	if isUpdate {
		resourceName = strings.TrimPrefix(shortMethod, "Update")
	}
	// RemoveWebhook.
	if isRemove {
		resourceName = strings.TrimPrefix(shortMethod, "Remove")
	}
	resourceName = toSnakeCase(resourceName)
	resourceDesc := mr.Descriptor().Fields().ByName(protoreflect.Name(resourceName))
	if resourceDesc == nil {
		return &common.Resource{Workspace: true}
	}
	if proto.HasExtension(resourceDesc.Message().Options(), annotationsproto.E_Resource) {
		// Parent-less resource. Return workspace resource for Create() method.
		if isCreate {
			return &common.Resource{Workspace: true}
		}
		resourceValue := mr.Get(resourceDesc)
		resourceNameDesc := resourceDesc.Message().Fields().ByName("name")
		if resourceNameDesc != nil {
			v := resourceValue.Message().Get(resourceNameDesc)
			return &common.Resource{Name: v.String()}
		}
	}
	return &common.Resource{Workspace: true}
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
