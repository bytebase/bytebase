package v1

import (
	"context"
	"log/slog"
	"regexp"
	"strings"

	"connectrpc.com/connect"
	annotationsproto "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
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

func (in *ACLInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		err := in.doACLCheck(ctx, req.Any(), req.Spec().Procedure)
		if err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (*ACLInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

func (in *ACLInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		wrappedConn := &aclStreamingConn{
			StreamingHandlerConn: conn,
			interceptor:          in,
			fullMethod:           conn.Spec().Procedure,
			ctx:                  ctx,
		}
		return next(ctx, wrappedConn)
	}
}

type aclStreamingConn struct {
	connect.StreamingHandlerConn
	interceptor *ACLInterceptor
	fullMethod  string
	ctx         context.Context
}

func (c *aclStreamingConn) Receive(msg any) error {
	err := c.interceptor.doACLCheck(c.ctx, msg, c.fullMethod)
	if err != nil {
		return err
	}
	return c.StreamingHandlerConn.Receive(msg)
}

// hasAllowMissingEnabled checks if the request has allow_missing field set to true.
// Uses proto reflection to handle different request types generically.
func hasAllowMissingEnabled(request any) bool {
	if request == nil {
		return false
	}

	pm, ok := request.(proto.Message)
	if !ok {
		return false
	}

	mr := pm.ProtoReflect()
	fd := mr.Descriptor().Fields().ByName("allow_missing")
	if fd == nil {
		return false
	}

	// Check if field is a bool and get its value
	if fd.Kind() != protoreflect.BoolKind {
		return false
	}

	return mr.Get(fd).Bool()
}

func (in *ACLInterceptor) doACLCheck(ctx context.Context, request any, fullMethod string) error {
	defer func() {
		if r := recover(); r != nil {
			perr, ok := r.(error)
			if !ok {
				perr = errors.Errorf("%v", r)
			}
			slog.Error("iam check PANIC RECOVER", log.BBError(perr), log.BBStack("panic-stack"))
		}
	}()

	authContextAny := ctx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return connect.NewError(connect.CodeInternal, errors.New("auth context not found"))
	}
	if err := populateRawResources(ctx, in.store, authContext, request, fullMethod); err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to populate raw resources %s", err))
	}

	if auth.IsAuthenticationAllowed(fullMethod, authContext) {
		return nil
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	if user == nil {
		return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("unauthenticated for method %q", fullMethod))
	}

	ok, extra, err := doIAMPermissionCheck(ctx, in.iamManager, fullMethod, user, authContext)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission for method %q, extra %v, err: %v", fullMethod, extra, err))
	}
	if !ok {
		err := connect.NewError(connect.CodePermissionDenied, errors.Errorf("permission denied for method %q, user does not have permission %q, extra %v", fullMethod, authContext.Permission, extra))
		if detail, detailErr := connect.NewErrorDetail(&v1pb.PermissionDeniedDetail{
			Method:              fullMethod,
			RequiredPermissions: []string{string(authContext.Permission)},
			Resources:           extra,
		}); detailErr == nil {
			err.AddDetail(detail)
		}
		return err
	}

	// Check allow_missing secondary permission if applicable
	// This handles Update methods that can create resources via allow_missing=true
	// When allow_missing is set, we additionally require create permission
	if hasAllowMissingEnabled(request) {
		// Derive create permission by replacing ".update" with ".create"
		// Example: "bb.roles.update" -> "bb.roles.create"
		createPerm := strings.Replace(string(authContext.Permission), ".update", ".create", 1)

		// Create a new auth context for create permission check
		createAuthContext := &common.AuthContext{
			Permission: iam.Permission(createPerm),
			AuthMethod: authContext.AuthMethod,
			Resources:  authContext.Resources,
		}
		ok, extra, err := doIAMPermissionCheck(ctx, in.iamManager, fullMethod, user, createAuthContext)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Errorf("failed to check create permission %q, extra %v, err: %v", createPerm, extra, err))
		}
		if !ok {
			err := connect.NewError(connect.CodePermissionDenied, errors.Errorf("permission denied: allow_missing=true requires both %s and %s, extra %v", authContext.Permission, createPerm, extra))
			if detail, detailErr := connect.NewErrorDetail(&v1pb.PermissionDeniedDetail{
				Method:              fullMethod,
				RequiredPermissions: []string{string(authContext.Permission), createPerm},
				Resources:           extra,
			}); detailErr == nil {
				err.AddDetail(detail)
			}
			return err
		}
	}

	return nil
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
			return false, nil, err
		}
		if ok {
			return true, nil, nil
		}
		projectResources := []string{}
		for _, id := range projectIDs {
			projectResources = append(projectResources, common.FormatProject(id))
		}
		return false, projectResources, nil
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
				instanceID, databaseName, err := common.GetInstanceDatabaseID(match)
				if err != nil {
					return errors.Wrapf(err, "failed to parse %q", match)
				}
				database, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{
					InstanceID:   &instanceID,
					DatabaseName: &databaseName,
				})
				if err != nil {
					return errors.Wrapf(err, "failed to get database")
				}
				if database == nil {
					return errors.Errorf("database %q not found", match)
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
