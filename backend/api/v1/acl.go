package v1

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"connectrpc.com/connect"
	annotationsproto "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
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
	if err := c.StreamingHandlerConn.Receive(msg); err != nil {
		return err
	}
	return c.interceptor.doACLCheck(c.ctx, msg, c.fullMethod)
}

// hasAllowMissingEnabled checks if the request has allow_missing field set to true.
// Uses proto reflection to handle different request types generically.
func hasAllowMissingEnabled(request any) bool {
	if request == nil {
		return false
	}
	if r, ok := request.(*v1pb.BatchUpdateInstancesRequest); ok {
		for _, updateRequest := range r.GetRequests() {
			if updateRequest.GetAllowMissing() {
				return true
			}
		}
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
	authContextAny := ctx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return connect.NewError(connect.CodeInternal, errors.New("auth context not found"))
	}
	authContext.Permission = getPermissionForRequest(request, authContext.Permission)

	if auth.IsAuthenticationSkipped(fullMethod, authContext) {
		return nil
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	if user == nil {
		return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("unauthenticated for method %q", fullMethod))
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	if workspaceID == "" {
		return connect.NewError(connect.CodeUnauthenticated, errors.Errorf("empty workspace id"))
	}
	resources, err := populateRawResources(ctx, in.store, request, fullMethod)
	if err != nil {
		return resourceResolutionConnectError(err)
	}

	// Workspace isolation: verify all resources belong to the caller's
	// workspace BEFORE publishing them on the AuthContext. Project, instance,
	// and database ownership is already validated in populateRawResources via
	// workspace-filtered store lookups; here we validate workspace resources.
	// Publishing only validated entries matters on the internal MCP chain,
	// where the audit interceptor runs outside ACL and derives a denied row's
	// parents from Resources — an entry that failed this check must never
	// become an audit parent (the denial would be filed under the foreign
	// workspace the request named, not under the caller).
	// Runs after authentication so unauthenticated requests get 401 first,
	// preventing resource existence probing.
	for _, resource := range resources {
		switch resource.Type {
		case common.ResourceTypeWorkspace:
			if resource.ID != workspaceID {
				return connect.NewError(connect.CodePermissionDenied, errors.Errorf("workspace mismatch"))
			}
		default:
		}
	}
	authContext.Resources = resources

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

		// Create a new auth context for create permission check. It must keep
		// carrying the delegated grant state: this secondary check is the same
		// request, and shedding the marker would let a future grant-keyed gate
		// evaluate an MCP-originated request as public-chain.
		createAuthContext := &common.AuthContext{
			Permission:     permission.Permission(createPerm),
			AuthMethod:     authContext.AuthMethod,
			Resources:      authContext.Resources,
			DelegatedGrant: authContext.DelegatedGrant,
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

func resourceResolutionConnectError(err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return err
	}
	return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to populate raw resources"))
}

func getPermissionForRequest(request any, defaultPermission permission.Permission) permission.Permission {
	if r, ok := request.(*v1pb.ListInstanceDatabaseRequest); ok && r.GetInstance() != nil {
		return permission.InstancesCreate
	}
	return defaultPermission
}

func hasPath(fieldMask *fieldmaskpb.FieldMask, want string) bool {
	if fieldMask == nil {
		return false
	}
	return slices.Contains(fieldMask.Paths, want)
}

func doIAMPermissionCheck(ctx context.Context, iamManager *iam.Manager, fullMethod string, user *store.UserMessage, authContext *common.AuthContext) (bool, []string, error) {
	if auth.IsAuthenticationSkipped(fullMethod, authContext) {
		return true, nil, nil
	}
	if authContext.AuthMethod != common.AuthMethodIAM {
		return true, nil, nil
	}
	// Handle GetProject() error status.
	if len(authContext.Resources) == 0 {
		return false, nil, errors.Errorf("no resource found for IAM auth method")
	}

	var hasWorkspaceResource bool
	projectIDMap := make(map[string]bool)
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	for _, resource := range authContext.Resources {
		switch resource.Type {
		case common.ResourceTypeWorkspace:
			hasWorkspaceResource = true
		case common.ResourceTypeProject:
			projectIDMap[resource.ID] = true
		default:
			return false, nil, errors.Errorf("unknown resource type %v", resource.Type)
		}
	}

	if hasWorkspaceResource {
		ok, err := iamManager.CheckPermission(ctx, authContext.Permission, user, workspaceID)
		if err != nil {
			return false, nil, err
		}
		if !ok {
			return false, nil, nil
		}
	}
	if len(projectIDMap) > 0 {
		var projectIDs []string
		for projectID := range projectIDMap {
			projectIDs = append(projectIDs, projectID)
		}
		ok, err := iamManager.CheckPermission(ctx, authContext.Permission, user, workspaceID, projectIDs...)
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

const (
	maxResourceRouteDepth = 3

	workspaceCollection = "workspaces"
	projectCollection   = "projects"
	instanceCollection  = "instances"
	databaseCollection  = "databases"
)

type resourceRoute [maxResourceRouteDepth]string

type resourceResolver func(context.Context, *store.Store, []string) (*common.Resource, error)

var resourceResolvers = map[resourceRoute]resourceResolver{
	{workspaceCollection}:                                       resolveWorkspaceResource,
	{projectCollection}:                                         resolveProjectResource,
	{projectCollection, instanceCollection}:                     resolveProjectInstanceResource,
	{projectCollection, instanceCollection, databaseCollection}: resolveProjectDatabaseResource,
	{instanceCollection}:                                        resolveWorkspaceInstanceResource,
	{instanceCollection, databaseCollection}:                    resolveWorkspaceDatabaseResource,
}

func getResourceRoute(parts []string) resourceRoute {
	var route resourceRoute
	for i := range route {
		partIndex := 2 * i
		if partIndex >= len(parts) {
			break
		}
		route[i] = parts[partIndex]
	}
	return route
}

func findResourceResolver(route resourceRoute) (resourceRoute, resourceResolver, bool) {
	for {
		if resolver, ok := resourceResolvers[route]; ok {
			return route, resolver, true
		}
		cleared := false
		for i := len(route) - 1; i >= 0; i-- {
			if route[i] != "" {
				route[i] = ""
				cleared = true
				break
			}
		}
		if !cleared {
			return resourceRoute{}, nil, false
		}
	}
}

func resolveRawResource(ctx context.Context, stores *store.Store, name string) (*common.Resource, error) {
	return resolveRawResourceWithArchivedProject(ctx, stores, name, false)
}

func resolveRawResourceWithArchivedProject(ctx context.Context, stores *store.Store, name string, allowArchivedProject bool) (*common.Resource, error) {
	if name == "projects/-" || strings.HasPrefix(name, "instances/-/databases/") {
		return workspaceFallback(ctx), nil
	}

	parts := strings.Split(name, "/")
	route := getResourceRoute(parts)
	if allowArchivedProject && route == (resourceRoute{projectCollection}) {
		projectID, err := requiredResourceIdentifier(parts, 1, projectCollection)
		if err != nil {
			return nil, err
		}
		return &common.Resource{Type: common.ResourceTypeProject, ID: projectID}, nil
	}
	if allowArchivedProject && route == (resourceRoute{projectCollection, instanceCollection}) {
		return resolveProjectInstanceResourceForLifecycle(ctx, stores, parts)
	}
	_, resolver, ok := findResourceResolver(route)
	if !ok {
		return workspaceFallback(ctx), nil
	}
	return resolver(ctx, stores, parts)
}

func resolveWorkspaceResource(_ context.Context, _ *store.Store, parts []string) (*common.Resource, error) {
	workspaceID, err := requiredResourceIdentifier(parts, 1, workspaceCollection)
	if err != nil {
		return nil, err
	}
	return &common.Resource{Type: common.ResourceTypeWorkspace, ID: workspaceID}, nil
}

func resolveProjectResource(ctx context.Context, stores *store.Store, parts []string) (*common.Resource, error) {
	projectID, err := requiredResourceIdentifier(parts, 1, projectCollection)
	if err != nil {
		return nil, err
	}
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	project, err := stores.GetProject(ctx, &store.FindProjectMessage{
		Workspace:  workspaceID,
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get project"))
	}
	if project == nil || project.Workspace != workspaceID {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
	}
	return &common.Resource{Type: common.ResourceTypeProject, ID: projectID}, nil
}

func resolveProjectInstanceResource(ctx context.Context, stores *store.Store, parts []string) (*common.Resource, error) {
	projectID, err := requiredResourceIdentifier(parts, 1, projectCollection)
	if err != nil {
		return nil, err
	}
	instanceID, err := requiredResourceIdentifier(parts, 3, instanceCollection)
	if err != nil {
		return nil, err
	}
	if _, err := getProjectScopedInstance(ctx, stores, projectID, instanceID); err != nil {
		return nil, err
	}
	return &common.Resource{Type: common.ResourceTypeProject, ID: projectID}, nil
}

func resolveProjectDatabaseResource(ctx context.Context, stores *store.Store, parts []string) (*common.Resource, error) {
	projectID, err := requiredResourceIdentifier(parts, 1, projectCollection)
	if err != nil {
		return nil, err
	}
	instanceID, err := requiredResourceIdentifier(parts, 3, instanceCollection)
	if err != nil {
		return nil, err
	}
	databaseName, err := requiredResourceIdentifier(parts, 5, databaseCollection)
	if err != nil {
		return nil, err
	}
	instance, err := getProjectScopedInstance(ctx, stores, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	database, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    common.GetWorkspaceIDFromContext(ctx),
		InstanceID:   &instance.ResourceID,
		DatabaseName: &databaseName,
		ShowDeleted:  true,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get database"))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", strings.Join(parts[:6], "/")))
	}
	return &common.Resource{Type: common.ResourceTypeProject, ID: projectID}, nil
}

func resolveWorkspaceInstanceResource(ctx context.Context, stores *store.Store, parts []string) (*common.Resource, error) {
	instanceID, err := requiredResourceIdentifier(parts, 1, instanceCollection)
	if err != nil {
		return nil, err
	}
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:     workspaceID,
		WorkspaceOnly: true,
		ResourceID:    &instanceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get instance"))
	}
	if instance == nil || instance.Workspace != workspaceID {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", strings.Join(parts[:2], "/")))
	}
	return &common.Resource{Type: common.ResourceTypeWorkspace, ID: workspaceID}, nil
}

func resolveWorkspaceDatabaseResource(ctx context.Context, stores *store.Store, parts []string) (*common.Resource, error) {
	instanceID, err := requiredResourceIdentifier(parts, 1, instanceCollection)
	if err != nil {
		return nil, err
	}
	databaseName, err := requiredResourceIdentifier(parts, 3, databaseCollection)
	if err != nil {
		return nil, err
	}
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:     workspaceID,
		WorkspaceOnly: true,
		ResourceID:    &instanceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get instance"))
	}
	if instance == nil || instance.Workspace != workspaceID {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", instanceID))
	}
	database, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    workspaceID,
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
		ShowDeleted:  true,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get database"))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", strings.Join(parts[:4], "/")))
	}
	return &common.Resource{Type: common.ResourceTypeProject, ID: database.ProjectID}, nil
}

func requiredResourceIdentifier(parts []string, index int, collection string) (string, error) {
	if len(parts) <= index || parts[index] == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid %s resource %q", collection, strings.Join(parts, "/")))
	}
	return parts[index], nil
}

func workspaceFallback(ctx context.Context) *common.Resource {
	if workspaceID := common.GetWorkspaceIDFromContext(ctx); workspaceID != "" {
		return &common.Resource{Type: common.ResourceTypeWorkspace, ID: workspaceID}
	}
	return nil
}

// populateRawResources extracts authorization resources from the request.
func populateRawResources(ctx context.Context, stores *store.Store, request any, method string) ([]*common.Resource, error) {
	rawNames, err := getResourceFromRequest(ctx, request, method)
	if err != nil {
		return nil, err
	}

	var resources []*common.Resource
	for _, name := range rawNames {
		resource, err := resolveRawResourceWithArchivedProject(ctx, stores, name, allowsArchivedProjectResourceResolution(method))
		if err != nil {
			return nil, err
		}
		if resource != nil {
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func getProjectScopedInstance(ctx context.Context, stores *store.Store, projectID, instanceID string) (*store.InstanceMessage, error) {
	return getProjectScopedInstanceWithArchivedProject(ctx, stores, projectID, instanceID, false)
}

func getProjectScopedInstanceWithArchivedProject(ctx context.Context, stores *store.Store, projectID, instanceID string, allowArchivedProject bool) (*store.InstanceMessage, error) {
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:  workspaceID,
		ProjectID:  &projectID,
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get instance"))
	}
	if instance == nil || instance.Workspace != workspaceID {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", common.FormatProjectInstance(projectID, instanceID)))
	}
	if !allowArchivedProject {
		if err := ensureProjectInstanceIsActive(ctx, stores, instance); err != nil {
			return nil, err
		}
	}
	return instance, nil
}

func resolveProjectInstanceResourceForLifecycle(ctx context.Context, stores *store.Store, parts []string) (*common.Resource, error) {
	projectID, err := requiredResourceIdentifier(parts, 1, projectCollection)
	if err != nil {
		return nil, err
	}
	instanceID, err := requiredResourceIdentifier(parts, 3, instanceCollection)
	if err != nil {
		return nil, err
	}
	if _, err := getProjectScopedInstanceWithArchivedProject(ctx, stores, projectID, instanceID, true); err != nil {
		return nil, err
	}
	return &common.Resource{Type: common.ResourceTypeProject, ID: projectID}, nil
}

func allowsArchivedProjectResourceResolution(method string) bool {
	return method == v1connect.InstanceServiceDeleteInstanceProcedure ||
		method == v1connect.InstanceServiceUndeleteInstanceProcedure ||
		method == v1connect.InstanceServicePrepareSampleProjectInstanceProcedure
}

func getResourceFromRequest(ctx context.Context, request any, method string) ([]string, error) {
	pm, ok := request.(proto.Message)
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid request for method %q", method))
	}
	mr := pm.ProtoReflect()

	methodTokens := strings.Split(method, "/")
	if len(methodTokens) != 3 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid method %q", method))
	}
	shortMethod := methodTokens[2]

	var resources []string

	switch r := request.(type) {
	case *v1pb.CreateInstanceRequest:
		if r.Parent == nil {
			break
		}
		projectID, err := common.GetProjectID(*r.Parent)
		if err == nil && common.IsDefaultProject(common.GetWorkspaceIDFromContext(ctx), projectID) {
			// Default projects cannot own instances. Authorize at workspace scope so
			// InstanceService can return its canonical invalid-argument response.
			return []string{""}, nil
		}
	case *v1pb.PrepareSampleProjectInstanceRequest:
		projectID, err := common.GetProjectID(r.GetParent())
		if err == nil && common.IsDefaultProject(common.GetWorkspaceIDFromContext(ctx), projectID) {
			// Keep the default-project rejection in InstanceService, consistent
			// with ordinary project instance creation.
			return []string{""}, nil
		}
	default:
	}
	if r, ok := request.(*v1pb.UpdateInstanceRequest); ok && r.AllowMissing && r.Instance != nil {
		if projectID, _, err := common.GetProjectIDInstanceID(r.Instance.Name); err == nil {
			// The instance does not exist yet, so authorize the implicit creation
			// against its project collection instead of resolving the instance.
			return []string{common.FormatProject(projectID)}, nil
		}
		if _, err := common.GetInstanceID(r.Instance.Name); err == nil {
			return []string{""}, nil
		}
	}
	if r, ok := request.(*v1pb.BatchUpdateInstancesRequest); ok {
		// The batch parent authorizes every implicit creation. An empty parent
		// represents the workspace instance collection.
		resources = append(resources, r.GetParent())
		for _, updateRequest := range r.GetRequests() {
			if updateRequest.GetAllowMissing() {
				continue
			}
			resources = append(resources, getResourceFromSingleRequest(updateRequest.ProtoReflect(), "UpdateInstance"))
		}
		return resources, nil
	}

	if r, ok := request.(*v1pb.ListInstanceDatabaseRequest); ok && r.GetInstance() != nil {
		// During instance creation, the request carries an inline instance so
		// the handler can preview remote databases before the instance exists.
		// Use the nested parent project when present; a top-level candidate is
		// still authorized at workspace scope.
		if projectID, _, err := common.GetProjectIDInstanceID(r.GetName()); err == nil {
			resources = append(resources, common.FormatProject(projectID))
		} else {
			resources = append(resources, "")
		}
		return resources, nil
	}

	// Transferring database projects needs to check both projects.
	var updateDatabaseRequests []*v1pb.UpdateDatabaseRequest
	switch r := request.(type) {
	case *v1pb.UpdateDatabaseRequest:
		updateDatabaseRequests = append(updateDatabaseRequests, r)
	case *v1pb.BatchUpdateDatabasesRequest:
		updateDatabaseRequests = append(updateDatabaseRequests, r.Requests...)
	default:
	}
	for _, r := range updateDatabaseRequests {
		if hasPath(r.GetUpdateMask(), "project") {
			projectID, err := common.GetProjectID(r.GetDatabase().GetProject())
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get projectID from %q", r.GetDatabase().GetProject()))
			}
			// Allow to transfer databases to the default project.
			if common.IsDefaultProject(common.GetWorkspaceIDFromContext(ctx), projectID) {
				continue
			}
			resources = append(resources, r.GetDatabase().GetProject())
		}
	}

	// HACK(p0ny): unfortunately, BatchUpdateIssuesStatus doesn't comply to aip.
	if r, ok := request.(*v1pb.BatchUpdateIssuesStatusRequest); ok {
		resources = append(resources, r.Issues...)
		return resources, nil
	}

	if strings.HasPrefix(shortMethod, "Batch") {
		parentDesc := mr.Descriptor().Fields().ByName("parent")
		if parentDesc != nil && proto.HasExtension(parentDesc.Options(), annotationsproto.E_ResourceReference) && mr.Has(parentDesc) {
			resources = append(resources, mr.Get(parentDesc).String())
		}
		// Handle batch get requests.
		if strings.HasPrefix(shortMethod, "BatchGet") {
			namesDesc := mr.Descriptor().Fields().ByName("names")
			if namesDesc != nil {
				namesValue := mr.Get(namesDesc)
				namesValueList := namesValue.List()
				for i := 0; i < namesValueList.Len(); i++ {
					v := namesValueList.Get(i)
					resources = append(resources, v.String())
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
				resources = append(resources, getResourceFromSingleRequest(r, shortMethodWithoutBatch))
			}
			return resources, nil
		}
	}
	resources = append(resources, getResourceFromSingleRequest(mr, shortMethod))
	return resources, nil
}

func getResourceFromSingleRequest(mr protoreflect.Message, shortMethod string) string {
	parentDesc := mr.Descriptor().Fields().ByName("parent")
	if parentDesc != nil && proto.HasExtension(parentDesc.Options(), annotationsproto.E_ResourceReference) {
		return mr.Get(parentDesc).String()
	}
	nameDesc := mr.Descriptor().Fields().ByName("name")
	if nameDesc != nil && proto.HasExtension(nameDesc.Options(), annotationsproto.E_ResourceReference) {
		return mr.Get(nameDesc).String()
	}
	// This is primarily used by Get/SetIAMPolicy().
	resourceFieldDesc := mr.Descriptor().Fields().ByName("resource")
	if resourceFieldDesc != nil && proto.HasExtension(resourceFieldDesc.Options(), annotationsproto.E_ResourceReference) {
		return mr.Get(resourceFieldDesc).String()
	}
	// This is primarily used by AddWebhook().
	projectFieldDesc := mr.Descriptor().Fields().ByName("project")
	if projectFieldDesc != nil && proto.HasExtension(projectFieldDesc.Options(), annotationsproto.E_ResourceReference) {
		return mr.Get(projectFieldDesc).String()
	}

	// Listing top-level resources.
	if strings.HasPrefix(shortMethod, "List") {
		return ""
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
		return ""
	}
	if proto.HasExtension(resourceDesc.Message().Options(), annotationsproto.E_Resource) {
		// Parent-less resource. Return empty for Create() method (workspace resource).
		if isCreate {
			return ""
		}
		resourceValue := mr.Get(resourceDesc)
		resourceNameDesc := resourceDesc.Message().Fields().ByName("name")
		if resourceNameDesc != nil {
			return resourceValue.Message().Get(resourceNameDesc).String()
		}
	}
	return ""
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
