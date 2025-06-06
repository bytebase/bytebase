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
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if user != nil {
		ctx = context.WithValue(ctx, common.UserContextKey, user)
	}

	authContextAny := ctx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return nil, status.Errorf(codes.Internal, "auth context not found")
	}
	if err := populateRawResources(ctx, in.store, authContext, request, serverInfo.FullMethod); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to populate raw resources %s", err)
	}

	if auth.IsAuthenticationAllowed(serverInfo.FullMethod, authContext) {
		return handler(ctx, request)
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated for method %q", serverInfo.FullMethod)
	}

	ok, extra, err := doIAMPermissionCheck(ctx, in.iamManager, serverInfo.FullMethod, user, authContext)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission for method %q, extra %v, err: %v", serverInfo.FullMethod, extra, err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q, extra %v", serverInfo.FullMethod, authContext.Permission, extra)
	}

	return handler(ctx, request)
}

// ACLStreamInterceptor is the unary interceptor for gRPC API.
func (in *ACLInterceptor) ACLStreamInterceptor(srv any, ss grpc.ServerStream, serverInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
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
		return status.Error(codes.PermissionDenied, err.Error())
	}
	if user != nil {
		ctx = context.WithValue(ctx, common.UserContextKey, user)
		ss = &overrideStream{ServerStream: ss, childCtx: ctx, iamManager: in.iamManager, store: in.store, user: user, fullMethod: serverInfo.FullMethod}
	}

	return handler(srv, ss)
}

type overrideStream struct {
	grpc.ServerStream

	childCtx   context.Context
	iamManager *iam.Manager
	store      *store.Store
	user       *store.UserMessage
	fullMethod string
}

func (o overrideStream) Context() context.Context {
	return o.childCtx
}

func (o *overrideStream) RecvMsg(request any) error {
	authContextAny := o.childCtx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return status.Errorf(codes.Internal, "auth context not found")
	}
	if err := populateRawResources(o.childCtx, o.store, authContext, request, o.fullMethod); err != nil {
		return status.Errorf(codes.Internal, "failed to populate raw resources %s", err)
	}

	if auth.IsAuthenticationAllowed(o.fullMethod, authContext) {
		return o.ServerStream.RecvMsg(request)
	}
	if o.user == nil {
		return status.Errorf(codes.Unauthenticated, "unauthenticated for method %q", o.fullMethod)
	}

	ok, extra, err := doIAMPermissionCheck(o.childCtx, o.iamManager, o.fullMethod, o.user, authContext)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check permission for method %q, extra %v, err: %v", o.fullMethod, extra, err)
	}
	if !ok {
		return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q, extra %v", o.fullMethod, authContext.Permission, extra)
	}

	return o.ServerStream.RecvMsg(request)
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

func doIAMPermissionCheck(ctx context.Context, iamManager *iam.Manager, fullMethod string, user *store.UserMessage, authContext *common.AuthContext) (bool, []string, error) {
	if auth.IsAuthenticationAllowed(fullMethod, authContext) {
		return true, nil, nil
	}
	if authContext.AuthMethod != common.AuthMethodIAM {
		return true, nil, nil
	}
	// Handle GetProject() error status.
	if len(authContext.Resources) == 0 {
		return false, nil, errors.Errorf("no resource found for IAM auth method")
	}
	if authContext.HasWorkspaceResource() {
		ok, err := iamManager.CheckPermission(ctx, authContext.Permission, user)
		if err != nil {
			return false, nil, err
		}
		if !ok {
			return false, nil, nil
		}
	}
	projectIDs := authContext.GetProjectResources()
	if len(projectIDs) > 0 {
		ok, err := iamManager.CheckPermission(ctx, authContext.Permission, user, projectIDs...)
		if err != nil {
			return false, projectIDs, err
		}
		if !ok {
			return false, projectIDs, nil
		}
	}
	return true, nil, nil
}

var projectRegex = regexp.MustCompile(`^projects/[^/]+`)
var databaseRegex = regexp.MustCompile(`^instances/[^/]+/databases/[^/]+`)

func populateRawResources(ctx context.Context, stores *store.Store, authContext *common.AuthContext, request any, method string) error {
	resources, err := getResourceFromRequest(request, method)
	if err != nil {
		return err
	}
	for _, resource := range resources {
		switch {
		// TODO(d): remove "projects/-" hack later.
		case strings.HasPrefix(resource.Name, "projects/") && resource.Name != "projects/-":
			project := projectRegex.FindString(resource.Name)
			if project == "" {
				return errors.Errorf("invalid project resource %q", resource.Name)
			}
			projectID, err := common.GetProjectID(project)
			if err != nil {
				return err
			}
			resource.ProjectID = projectID
		case strings.HasPrefix(resource.Name, "instances/") && strings.Contains(resource.Name, "/databases/") && !strings.HasPrefix(resource.Name, "instances/-/databases/"):
			match := databaseRegex.FindString(resource.Name)
			if match != "" {
				database, err := getDatabaseMessage(ctx, stores, match)
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

func getResourceFromRequest(request any, method string) ([]*common.Resource, error) {
	pm, ok := request.(proto.Message)
	if !ok {
		return nil, errors.Errorf("invalid request for method %q", method)
	}
	mr := pm.ProtoReflect()

	methodTokens := strings.Split(method, "/")
	if len(methodTokens) != 3 {
		return nil, errors.Errorf("invalid method %q", method)
	}
	shortMethod := methodTokens[2]

	var resources []*common.Resource

	// Transferring database projects needs to check both projects.
	var updateDatabaseRequests []*v1pb.UpdateDatabaseRequest
	switch r := request.(type) {
	case *v1pb.UpdateDatabaseRequest:
		updateDatabaseRequests = append(updateDatabaseRequests, r)
	case *v1pb.BatchUpdateDatabasesRequest:
		updateDatabaseRequests = append(updateDatabaseRequests, r.Requests...)
	}
	for _, r := range updateDatabaseRequests {
		if hasPath(r.GetUpdateMask(), "project") {
			projectID, err := common.GetProjectID(r.GetDatabase().GetProject())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get projectID from %q", r.GetDatabase().GetProject())
			}
			// Allow to transfer databases to the default project.
			if projectID == common.DefaultProjectID {
				continue
			}
			resources = append(resources, &common.Resource{Name: r.GetDatabase().GetProject()})
		}
	}

	// HACK(p0ny): unfortunately, BatchUpdateIssuesStatus doesn't comply to aip.
	if r, ok := request.(*v1pb.BatchUpdateIssuesStatusRequest); ok {
		for _, issue := range r.Issues {
			resources = append(resources, &common.Resource{
				Name: issue,
			})
		}
		return resources, nil
	}

	if strings.HasPrefix(shortMethod, "Batch") {
		// Handle batch get requests.
		if strings.HasPrefix(shortMethod, "BatchGet") {
			namesDesc := mr.Descriptor().Fields().ByName("names")
			if namesDesc != nil {
				namesValue := mr.Get(namesDesc)
				namesValueList := namesValue.List()
				for i := 0; i < namesValueList.Len(); i++ {
					v := namesValueList.Get(i)
					resources = append(resources, &common.Resource{Name: v.String()})
				}
				return resources, nil
			}
		}

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
			return resources, nil
		}
	}
	resource := getResourceFromSingleRequest(mr, shortMethod)
	if resource != nil {
		resources = append(resources, resource)
	}
	return resources, nil
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
	isTest := strings.HasPrefix(shortMethod, "Test")
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
	if isTest {
		resourceName = strings.TrimPrefix(shortMethod, "Test")
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
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
		ShowDeleted:     true,
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
