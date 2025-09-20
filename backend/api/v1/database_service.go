package v1

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/transform"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// DatabaseService implements the database service.
type DatabaseService struct {
	v1connect.UnimplementedDatabaseServiceHandler
	store          *store.Store
	schemaSyncer   *schemasync.Syncer
	licenseService *enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(store *store.Store, schemaSyncer *schemasync.Syncer, licenseService *enterprise.LicenseService, profile *config.Profile, iamManager *iam.Manager) *DatabaseService {
	return &DatabaseService{
		store:          store,
		schemaSyncer:   schemaSyncer,
		licenseService: licenseService,
		profile:        profile,
		iamManager:     iamManager,
	}
}

// GetDatabase gets a database.
func (s *DatabaseService) GetDatabase(ctx context.Context, req *connect.Request[v1pb.GetDatabaseRequest]) (*connect.Response[v1pb.Database], error) {
	databaseMessage, err := getDatabaseMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if databaseMessage.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q was deleted", req.Msg.Name))
	}
	database, err := s.convertToDatabase(ctx, databaseMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert database, error: %v", err))
	}
	return connect.NewResponse(database), nil
}

func (s *DatabaseService) BatchGetDatabases(ctx context.Context, req *connect.Request[v1pb.BatchGetDatabasesRequest]) (*connect.Response[v1pb.BatchGetDatabasesResponse], error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	// TODO(steven): Filter out the databases based on `req.Msg.parent`.
	databases := make([]*v1pb.Database, 0, len(req.Msg.Names))
	for _, name := range req.Msg.Names {
		databaseMessage, err := getDatabaseMessage(ctx, s.store, name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database %q with error: %v", name, err.Error()))
		}
		if databaseMessage.Deleted {
			// Ignore no deleted database.
			continue
		}
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionDatabasesGet, user, databaseMessage.ProjectID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
		}
		if !ok {
			// Ignore no permission database.
			continue
		}
		database, err := s.convertToDatabase(ctx, databaseMessage)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert database, error: %v", err))
		}
		databases = append(databases, database)
	}
	return connect.NewResponse(&v1pb.BatchGetDatabasesResponse{Databases: databases}), nil
}

func getVariableAndValueFromExpr(expr celast.Expr) (string, any) {
	var variable string
	var value any
	for _, arg := range expr.AsCall().Args() {
		switch arg.Kind() {
		case celast.IdentKind:
			variable = arg.AsIdent()
		case celast.SelectKind:
			// Handle member selection like "labels.environment"
			sel := arg.AsSelect()
			if sel.Operand().Kind() == celast.IdentKind {
				variable = fmt.Sprintf("%s.%s", sel.Operand().AsIdent(), sel.FieldName())
			}
		case celast.LiteralKind:
			value = arg.AsLiteral().Value()
		case celast.ListKind:
			list := []any{}
			for _, e := range arg.AsList().Elements() {
				if e.Kind() == celast.LiteralKind {
					list = append(list, e.AsLiteral().Value())
				}
			}
			value = list
		default:
		}
	}
	return variable, value
}

func getSubConditionFromExpr(expr celast.Expr, getFilter func(expr celast.Expr) (string, error), join string) (string, error) {
	var args []string
	for _, arg := range expr.AsCall().Args() {
		s, err := getFilter(arg)
		if err != nil {
			return "", err
		}
		args = append(args, "("+s+")")
	}
	return strings.Join(args, fmt.Sprintf(" %s ", join)), nil
}

func parseToEngineSQL(expr celast.Expr, relation string) (string, error) {
	variable, value := getVariableAndValueFromExpr(expr)
	if variable != "engine" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only "engine" support "engine in [xx]"/"!(engine in [xx])" operator`))
	}

	rawEngineList, ok := value.([]any)
	if !ok {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid engine value %q", value))
	}
	if len(rawEngineList) == 0 {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("empty engine filter"))
	}
	engineList := []string{}
	for _, rawEngine := range rawEngineList {
		v1Engine, ok := v1pb.Engine_value[rawEngine.(string)]
		if !ok {
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid engine filter %q", rawEngine))
		}
		engine := convertEngine(v1pb.Engine(v1Engine))
		engineList = append(engineList, fmt.Sprintf(`'%s'`, engine.String()))
	}

	return fmt.Sprintf("instance.metadata->>'engine' %s (%s)", relation, strings.Join(engineList, ",")), nil
}

func getListDatabaseFilter(filter string) (*store.ListResourceFilter, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	parseToSQL := func(variable, value any) (string, error) {
		switch variable {
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid project filter %q", value))
			}
			positionalArgs = append(positionalArgs, projectID)
			return fmt.Sprintf("db.project = $%d", len(positionalArgs)), nil
		case "instance":
			instanceID, err := common.GetInstanceID(value.(string))
			if err != nil {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid instance filter %q", value))
			}
			positionalArgs = append(positionalArgs, instanceID)
			return fmt.Sprintf("db.instance = $%d", len(positionalArgs)), nil
		case "environment":
			environment, ok := value.(string)
			if !ok {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse value %v to string", value))
			}
			if environment != "" {
				environmentID, err := common.GetEnvironmentID(environment)
				if err != nil {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid environment filter %q", value))
				}
				positionalArgs = append(positionalArgs, environmentID)
				return fmt.Sprintf(`COALESCE(db.environment, instance.environment) = $%d`, len(positionalArgs)), nil
			}
			return "db.environment IS NULL AND instance.environment IS NULL", nil
		case "engine":
			v1Engine, ok := v1pb.Engine_value[value.(string)]
			if !ok {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid engine filter %q", value))
			}
			engine := convertEngine(v1pb.Engine(v1Engine))
			positionalArgs = append(positionalArgs, engine)
			return fmt.Sprintf("instance.metadata->>'engine' = $%d", len(positionalArgs)), nil
		case "name":
			positionalArgs = append(positionalArgs, value)
			return fmt.Sprintf("db.name = $%d", len(positionalArgs)), nil
		case "label":
			keyVal := strings.Split(value.(string), ":")
			if len(keyVal) != 2 {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`invalid label filter %q, should be in "{label key}:{label value} format"`, value))
			}
			labelKey := keyVal[0]
			labelValues := strings.Split(keyVal[1], ",")
			positionalArgs = append(positionalArgs, labelValues)
			return fmt.Sprintf("db.metadata->'labels'->>'%s' = ANY($%d)", labelKey, len(positionalArgs)), nil
		case "drifted":
			drifted, ok := value.(bool)
			if !ok {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid drifted filter %q", value))
			}
			condition := "IS"
			if !drifted {
				condition = "IS NOT"
			}
			return fmt.Sprintf("(db.metadata->>'drifted')::boolean %s TRUE", condition), nil
		case "exclude_unassigned":
			if excludeUnassigned, ok := value.(bool); excludeUnassigned && ok {
				positionalArgs = append(positionalArgs, common.DefaultProjectID)
				return fmt.Sprintf("db.project != $%d", len(positionalArgs)), nil
			}
			return "TRUE", nil
		case "table":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf(`
				EXISTS (
					SELECT 1
					FROM json_array_elements(ds.metadata->'schemas') AS s,
						 json_array_elements(s->'tables') AS t
					WHERE t->>'name' = $%d
				)
			`, len(positionalArgs)), nil
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
		}
	}

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				return getSubConditionFromExpr(expr, getFilter, "OR")
			case celoperators.LogicalAnd:
				return getSubConditionFromExpr(expr, getFilter, "AND")
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`invalid args for %q`, variable))
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect string, got %T, hint: filter literals should be string", value))
				}
				strValue = strings.ToLower(strValue)

				switch variable {
				case "name":
					return "LOWER(db.name) LIKE '%" + strValue + "%'", nil
				case "table":
					return `EXISTS (
						SELECT 1
						FROM json_array_elements(ds.metadata->'schemas') AS s,
						 	 json_array_elements(s->'tables') AS t
						WHERE t->>'name' LIKE '%` + strValue + `%')`, nil
				default:
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only "name" or "table" support %q operator, but found %q`, celoverloads.Matches, variable))
				}
			case celoperators.In:
				return parseToEngineSQL(expr, "IN")
			case celoperators.LogicalNot:
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only support !(engine in ["{engine1}", "{engine2}"]) format`))
				}
				return parseToEngineSQL(args[0], "NOT IN")
			default:
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function %v", functionName))
			}
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected expr kind %v", expr.Kind()))
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}

	return &store.ListResourceFilter{
		Args:  positionalArgs,
		Where: "(" + where + ")",
	}, nil
}

// ListDatabases lists all databases.
func (s *DatabaseService) ListDatabases(ctx context.Context, req *connect.Request[v1pb.ListDatabasesRequest]) (*connect.Response[v1pb.ListDatabasesResponse], error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindDatabaseMessage{
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		ShowDeleted: req.Msg.ShowDeleted,
	}

	filter, err := getListDatabaseFilter(req.Msg.Filter)
	if err != nil {
		return nil, err
	}
	find.Filter = filter

	switch {
	case strings.HasPrefix(req.Msg.Parent, common.ProjectNamePrefix):
		p, err := common.GetProjectID(req.Msg.Parent)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid parent %q", req.Msg.Parent))
		}
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionProjectsGet, user, p)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
		}
		if !ok {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q in %q", iam.PermissionProjectsGet, req.Msg.Parent))
		}
		find.ProjectID = &p
	case strings.HasPrefix(req.Msg.Parent, common.WorkspacePrefix):
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionDatabasesList, user)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
		}
		if !ok {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionDatabasesList))
		}
	case strings.HasPrefix(req.Msg.Parent, common.InstanceNamePrefix):
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionInstancesGet, user)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
		}
		if !ok {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionInstancesGet))
		}

		instanceID, err := common.GetInstanceID(req.Msg.Parent)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid parent %q", req.Msg.Parent))
		}
		find.InstanceID = &instanceID
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid parent %q", req.Msg.Parent))
	}

	databaseMessages, err := s.store.ListDatabases(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}

	nextPageToken := ""
	if len(databaseMessages) == limitPlusOne {
		databaseMessages = databaseMessages[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal next page token, error: %v", err))
		}
	}

	response := &v1pb.ListDatabasesResponse{
		NextPageToken: nextPageToken,
	}
	for _, databaseMessage := range databaseMessages {
		database, err := s.convertToDatabase(ctx, databaseMessage)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert database, error: %v", err))
		}
		response.Databases = append(response.Databases, database)
	}
	return connect.NewResponse(response), nil
}

// UpdateDatabase updates a database.
func (s *DatabaseService) UpdateDatabase(ctx context.Context, req *connect.Request[v1pb.UpdateDatabaseRequest]) (*connect.Response[v1pb.Database], error) {
	if req.Msg.Database == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database must be set"))
	}
	if len(req.Msg.GetUpdateMask().GetPaths()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
	}

	// Use the helper function to get the database
	databaseMessage, err := getDatabaseMessage(ctx, s.store, req.Msg.Database.Name)
	if err != nil {
		// Check if it's a not found error and allow_missing is true
		if strings.Contains(err.Error(), "not found") && req.Msg.AllowMissing {
			// Database creation is not supported through UpdateDatabase API.
			// Databases must be created through the plan and rollout system.
			return nil, connect.NewError(connect.CodeUnimplemented, errors.Errorf("database creation is not supported through UpdateDatabase, use plan and rollout instead"))
		}
		// For other errors or when allow_missing is false, return the error
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if databaseMessage.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q was deleted", req.Msg.Database.Name))
	}

	var project *store.ProjectMessage
	patch := &store.UpdateDatabaseMessage{
		InstanceID:   databaseMessage.InstanceID,
		DatabaseName: databaseMessage.DatabaseName,
	}
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "project":
			projectID, err := common.GetProjectID(req.Msg.Database.Project)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
			}
			project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID:  &projectID,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
			}
			if project == nil {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
			}
			if project.Deleted {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("project %q is deleted", projectID))
			}
			patch.ProjectID = &project.ResourceID
		case "labels":
			patch.MetadataUpdates = append(patch.MetadataUpdates, func(dm *storepb.DatabaseMetadata) {
				dm.Labels = req.Msg.Database.Labels
			})
		case "environment":
			if req.Msg.Database.Environment != nil && *req.Msg.Database.Environment != "" {
				environmentID, err := common.GetEnvironmentID(*req.Msg.Database.Environment)
				if err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
				}
				environment, err := s.store.GetEnvironmentByID(ctx, environmentID)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
				}
				if environment == nil {
					return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("environment %q not found", environmentID))
				}
				patch.EnvironmentID = &environmentID
			} else {
				unsetEnvironment := ""
				patch.EnvironmentID = &unsetEnvironment
			}
		case "drifted":
			// Create a new base schema.
			syncHistory, err := s.schemaSyncer.SyncDatabaseSchemaToHistory(ctx, databaseMessage)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to sync database metadata and schema")
			}
			if _, err := s.store.CreateChangelog(ctx, &store.ChangelogMessage{
				InstanceID:   databaseMessage.InstanceID,
				DatabaseName: databaseMessage.DatabaseName,
				Status:       store.ChangelogStatusDone,
				// TODO(d): Revisit the previous sync history UID.
				PrevSyncHistoryUID: &syncHistory,
				SyncHistoryUID:     &syncHistory,
				Payload: &storepb.ChangelogPayload{
					Type:      storepb.ChangelogPayload_BASELINE,
					GitCommit: s.profile.GitCommit,
				}}); err != nil {
				return nil, errors.Wrapf(err, "failed to create changelog")
			}
			patch.MetadataUpdates = append(patch.MetadataUpdates, func(dm *storepb.DatabaseMetadata) {
				dm.Drifted = false
			})
		default:
		}
	}

	updatedDatabase, err := s.store.UpdateDatabase(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}

	database, err := s.convertToDatabase(ctx, updatedDatabase)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert database, error: %v", err))
	}
	return connect.NewResponse(database), nil
}

// SyncDatabase syncs the schema of a database.
func (s *DatabaseService) SyncDatabase(ctx context.Context, req *connect.Request[v1pb.SyncDatabaseRequest]) (*connect.Response[v1pb.SyncDatabaseResponse], error) {
	databaseMessage, err := getDatabaseMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if databaseMessage.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q was deleted", req.Msg.Name))
	}
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, databaseMessage); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1pb.SyncDatabaseResponse{}), nil
}

// BatchSyncDatabases sync multiply database asynchronously.
func (s *DatabaseService) BatchSyncDatabases(ctx context.Context, req *connect.Request[v1pb.BatchSyncDatabasesRequest]) (*connect.Response[v1pb.BatchSyncDatabasesResponse], error) {
	for _, name := range req.Msg.Names {
		databaseMessage, err := getDatabaseMessage(ctx, s.store, name)
		if err != nil {
			return nil, err
		}
		if databaseMessage.Deleted {
			continue
		}
		s.schemaSyncer.SyncDatabaseAsync(databaseMessage)
	}
	return connect.NewResponse(&v1pb.BatchSyncDatabasesResponse{}), nil
}

// BatchUpdateDatabases updates a database in batch.
func (s *DatabaseService) BatchUpdateDatabases(ctx context.Context, req *connect.Request[v1pb.BatchUpdateDatabasesRequest]) (*connect.Response[v1pb.BatchUpdateDatabasesResponse], error) {
	databases := []*store.DatabaseMessage{}
	batchUpdate := &store.BatchUpdateDatabases{}
	var updateMask *fieldmaskpb.FieldMask
	for _, req := range req.Msg.Requests {
		if req.Database == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database must be set"))
		}
		if req.UpdateMask == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
		}
		database, err := getDatabaseMessage(ctx, s.store, req.Database.Name)
		if err != nil {
			return nil, err
		}
		if database.Deleted {
			continue
		}
		if updateMask == nil {
			updateMask = req.UpdateMask
		}
		if !slices.Equal(updateMask.Paths, req.UpdateMask.Paths) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("all databases should have the same update_mask"))
		}
		for _, path := range req.UpdateMask.Paths {
			switch path {
			case "project":
				projectID, err := common.GetProjectID(req.Database.Project)
				if err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
				}
				if batchUpdate.ProjectID != nil && *batchUpdate.ProjectID != projectID {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("all databases should use the same project"))
				}
				batchUpdate.ProjectID = &projectID
			case "environment":
				if req.Database.Environment != nil && *req.Database.Environment != "" {
					envID, err := common.GetEnvironmentID(*req.Database.Environment)
					if err != nil {
						return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
					}
					if batchUpdate.EnvironmentID != nil && *batchUpdate.EnvironmentID != envID {
						return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("all databases should use the same environment"))
					}
					batchUpdate.EnvironmentID = &envID
				} else {
					if batchUpdate.EnvironmentID != nil && *batchUpdate.EnvironmentID != "" {
						return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("all databases should use the same environment"))
					}
					unsetEnvironment := ""
					batchUpdate.EnvironmentID = &unsetEnvironment
				}
			default:
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported update_mask path %q", path))
			}
		}
		databases = append(databases, database)
	}
	if batchUpdate.ProjectID != nil {
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID:  batchUpdate.ProjectID,
			ShowDeleted: true,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
		}
		if project == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", *batchUpdate.ProjectID))
		}
		if project.Deleted {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("project %q is deleted", *batchUpdate.ProjectID))
		}
	}
	if batchUpdate.EnvironmentID != nil && *batchUpdate.EnvironmentID != "" {
		environment, err := s.store.GetEnvironmentByID(ctx, *batchUpdate.EnvironmentID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
		}
		if environment == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("environment %q not found", *batchUpdate.EnvironmentID))
		}
	}

	response := &v1pb.BatchUpdateDatabasesResponse{}
	if len(databases) == 0 {
		return connect.NewResponse(response), nil
	}

	updatedDatabases, err := s.store.BatchUpdateDatabases(ctx, databases, batchUpdate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	for _, databaseMessage := range updatedDatabases {
		database, err := s.convertToDatabase(ctx, databaseMessage)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert database, error: %v", err))
		}
		response.Databases = append(response.Databases, database)
	}
	return connect.NewResponse(response), nil
}

func getDatabaseMetadataFilter(filter string) (*metadataFilter, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var getFilter func(expr celast.Expr) error
	metaFilter := &metadataFilter{}

	getFilter = func(expr celast.Expr) error {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalAnd:
				for _, arg := range expr.AsCall().Args() {
					if err := getFilter(arg); err != nil {
						return err
					}
				}
				return nil
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				stringVal, ok := value.(string)
				if !ok {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected string but found %q", value))
				}
				switch variable {
				case "schema":
					metaFilter.schema = &stringVal
				case "table":
					metaFilter.table = &stringVal
				default:
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected variable %v", variable))
				}
				return nil
			default:
				return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function %v", functionName))
			}
		default:
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected expr kind %v", expr.Kind()))
		}
	}

	if err := getFilter(ast.NativeRep().Expr()); err != nil {
		return nil, err
	}

	return metaFilter, nil
}

// GetDatabaseMetadata gets the metadata of a database.
func (s *DatabaseService) GetDatabaseMetadata(ctx context.Context, req *connect.Request[v1pb.GetDatabaseMetadataRequest]) (*connect.Response[v1pb.DatabaseMetadata], error) {
	name, err := common.TrimSuffix(req.Msg.Name, common.MetadataSuffix)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
	}
	database, err := getDatabaseMessage(ctx, s.store, name)
	if err != nil {
		return nil, err
	}
	if database.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q was deleted", name))
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to sync database schema for database %q, error %v", name, err))
		}
		newDBSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
		}
		if newDBSchema == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database schema %q not found", name))
		}
		dbSchema = newDBSchema
	}

	filter, err := getDatabaseMetadataFilter(req.Msg.Filter)
	if err != nil {
		return nil, err
	}
	v1pbMetadata := convertStoreDatabaseMetadata(dbSchema.GetMetadata(), filter)
	v1pbMetadata.Name = fmt.Sprintf("%s%s/%s%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName, common.MetadataSuffix)

	return connect.NewResponse(v1pbMetadata), nil
}

// GetDatabaseSchema gets the schema of a database.
func (s *DatabaseService) GetDatabaseSchema(ctx context.Context, req *connect.Request[v1pb.GetDatabaseSchemaRequest]) (*connect.Response[v1pb.DatabaseSchema], error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(req.Msg.Name, common.SchemaSuffix)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", instanceID))
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", databaseName))
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to sync database schema for database %q, error %v", databaseName, err))
		}
		newDBSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
		}
		if newDBSchema == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database schema %q not found", databaseName))
		}
		dbSchema = newDBSchema
	}
	// We only support MySQL engine for now.
	schema := string(dbSchema.GetSchema())
	if req.Msg.SdlFormat {
		switch instance.Metadata.GetEngine() {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			sdlSchema, err := transform.SchemaTransform(storepb.Engine_MYSQL, schema)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert schema to sdl format, error %v", err.Error()))
			}
			schema = sdlSchema
		default:
		}
	}
	return connect.NewResponse(&v1pb.DatabaseSchema{Schema: schema}), nil
}

// DiffSchema diff the database schema.
func (s *DatabaseService) DiffSchema(ctx context.Context, req *connect.Request[v1pb.DiffSchemaRequest]) (*connect.Response[v1pb.DiffSchemaResponse], error) {
	// Check if target is changelog - use new metadata-based approach
	changeHistoryID := req.Msg.GetChangelog()
	if changeHistoryID != "" {
		sourceDBSchema, err := s.getSourceDBSchema(ctx, req.Msg)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get source schema, error: %v", err))
		}

		targetDBSchema, err := s.getTargetDBSchema(ctx, req.Msg)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get target schema, error: %v", err))
		}

		engine, err := s.getParserEngine(ctx, req.Msg)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get parser engine, error: %v", err))
		}

		schemaDiff, err := schema.GetDatabaseSchemaDiff(engine, sourceDBSchema, targetDBSchema)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to compute schema diff, error: %v", err))
		}

		// Filter out bbdataarchive schema changes for Postgres
		if engine == storepb.Engine_POSTGRES {
			schemaDiff = schema.FilterPostgresArchiveSchema(schemaDiff)
		}

		migrationSQL, err := schema.GenerateMigration(engine, schemaDiff)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate migration SQL, error: %v", err))
		}

		return connect.NewResponse(&v1pb.DiffSchemaResponse{
			Diff: migrationSQL,
		}), nil
	}

	// Fallback to old string-based approach for schema string targets
	source, err := s.getSourceSchema(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get source schema, error: %v", err))
	}

	target, err := s.getTargetSchema(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get target schema, error: %v", err))
	}

	engine, err := s.getParserEngine(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get parser engine, error: %v", err))
	}

	strictMode := engine != storepb.Engine_ORACLE
	diff, err := parserbase.SchemaDiff(engine, parserbase.DiffContext{
		IgnoreCaseSensitive: false,
		StrictMode:          strictMode,
	}, source, target)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to compute diff between source and target schemas, error: %v", err))
	}

	return connect.NewResponse(&v1pb.DiffSchemaResponse{
		Diff: diff,
	}), nil
}

func (s *DatabaseService) getSourceDBSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (*model.DatabaseSchema, error) {
	if strings.Contains(request.Name, common.ChangelogPrefix) {
		instanceID, databaseName, changelogUID, err := common.GetInstanceDatabaseChangelogUID(request.Name)
		if err != nil {
			return nil, err
		}

		changelog, err := s.store.GetChangelog(ctx, &store.FindChangelogMessage{
			UID: &changelogUID,
		})
		if err != nil {
			return nil, err
		}
		if changelog == nil {
			return nil, errors.Errorf("changelog %d not found", changelogUID)
		}

		// Use SyncHistoryUID to get historical metadata
		if changelog.SyncHistoryUID != nil {
			syncHistory, err := s.store.GetSyncHistoryByUID(ctx, *changelog.SyncHistoryUID)
			if err != nil {
				return nil, err
			}
			if syncHistory == nil {
				return nil, errors.Errorf("sync history %d not found", *changelog.SyncHistoryUID)
			}

			// Get instance to determine engine and case sensitivity
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
			if err != nil {
				return nil, err
			}
			if instance == nil {
				return nil, errors.Errorf("instance %s not found", instanceID)
			}

			return model.NewDatabaseSchema(
				syncHistory.Metadata,
				[]byte(syncHistory.Schema),
				&storepb.DatabaseConfig{},
				instance.Metadata.GetEngine(),
				store.IsObjectCaseSensitive(instance),
			), nil
		}

		// Fallback to current database schema if no sync history
		dbSchema, err := s.store.GetDBSchema(ctx, instanceID, databaseName)
		if err != nil {
			return nil, err
		}
		if dbSchema == nil {
			return nil, errors.Errorf("database schema not found for %s/%s", instanceID, databaseName)
		}
		return dbSchema, nil
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Name)
	if err != nil {
		return nil, err
	}

	dbSchema, err := s.store.GetDBSchema(ctx, instanceID, databaseName)
	if err != nil {
		return nil, err
	}
	if dbSchema == nil {
		return nil, errors.Errorf("database schema not found for %s/%s", instanceID, databaseName)
	}
	return dbSchema, nil
}

func (s *DatabaseService) getTargetDBSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (*model.DatabaseSchema, error) {
	changeHistoryID := request.GetChangelog()

	// If the change history id is set, use the schema of the change history as the target.
	if changeHistoryID != "" {
		instanceID, databaseName, changelogUID, err := common.GetInstanceDatabaseChangelogUID(changeHistoryID)
		if err != nil {
			return nil, err
		}

		changelog, err := s.store.GetChangelog(ctx, &store.FindChangelogMessage{
			UID: &changelogUID,
		})
		if err != nil {
			return nil, err
		}
		if changelog == nil {
			return nil, errors.Errorf("changelog %d not found", changelogUID)
		}

		// Use SyncHistoryUID to get historical metadata
		if changelog.SyncHistoryUID != nil {
			syncHistory, err := s.store.GetSyncHistoryByUID(ctx, *changelog.SyncHistoryUID)
			if err != nil {
				return nil, err
			}
			if syncHistory == nil {
				return nil, errors.Errorf("sync history %d not found", *changelog.SyncHistoryUID)
			}

			// Get instance to determine engine and case sensitivity
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
			if err != nil {
				return nil, err
			}
			if instance == nil {
				return nil, errors.Errorf("instance %s not found", instanceID)
			}

			return model.NewDatabaseSchema(
				syncHistory.Metadata,
				[]byte(syncHistory.Schema),
				&storepb.DatabaseConfig{},
				instance.Metadata.GetEngine(),
				store.IsObjectCaseSensitive(instance),
			), nil
		}

		// Fallback to current database schema if no sync history
		dbSchema, err := s.store.GetDBSchema(ctx, instanceID, databaseName)
		if err != nil {
			return nil, err
		}
		if dbSchema == nil {
			return nil, errors.Errorf("database schema not found for %s/%s", instanceID, databaseName)
		}
		return dbSchema, nil
	}

	// If schema is provided, we need to parse it using GetDatabaseMetadata
	schemaStr := request.GetSchema()
	if schemaStr != "" {
		// Get the engine from the source database
		engine, err := s.getParserEngine(ctx, request)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get parser engine")
		}

		// Parse the schema string into metadata
		metadata, err := schema.GetDatabaseMetadata(engine, schemaStr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse target schema")
		}

		// Get instance to determine case sensitivity
		instanceID, _, err := common.GetInstanceDatabaseID(request.Name)
		if err != nil {
			return nil, err
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, err
		}
		if instance == nil {
			return nil, errors.Errorf("instance %s not found", instanceID)
		}

		// Create DatabaseSchema from the parsed metadata
		return model.NewDatabaseSchema(
			metadata,
			[]byte(schemaStr),
			&storepb.DatabaseConfig{},
			engine,
			store.IsObjectCaseSensitive(instance),
		), nil
	}

	return nil, errors.Errorf("must set the schema or change history id as the target")
}

func (s *DatabaseService) getSourceSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (string, error) {
	if strings.Contains(request.Name, common.ChangelogPrefix) {
		changeHistory, err := s.GetChangelog(ctx, &connect.Request[v1pb.GetChangelogRequest]{
			Msg: &v1pb.GetChangelogRequest{
				Name:      request.Name,
				View:      v1pb.ChangelogView_CHANGELOG_VIEW_FULL,
				SdlFormat: true,
			},
		})
		if err != nil {
			return "", err
		}
		return changeHistory.Msg.Schema, nil
	}

	databaseSchema, err := s.GetDatabaseSchema(ctx, &connect.Request[v1pb.GetDatabaseSchemaRequest]{
		Msg: &v1pb.GetDatabaseSchemaRequest{
			Name:      fmt.Sprintf("%s/schema", request.Name),
			SdlFormat: request.SdlFormat,
		},
	})
	if err != nil {
		return "", err
	}
	return databaseSchema.Msg.Schema, nil
}

func (s *DatabaseService) getTargetSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (string, error) {
	schema := request.GetSchema()
	changeHistoryID := request.GetChangelog()
	// TODO: maybe we will support an empty schema as the target.
	if schema == "" && changeHistoryID == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("must set the schema or change history id as the target"))
	}

	// If the change history id is set, use the schema of the change history as the target.
	if changeHistoryID != "" {
		changeHistory, err := s.GetChangelog(ctx, &connect.Request[v1pb.GetChangelogRequest]{
			Msg: &v1pb.GetChangelogRequest{
				Name:      changeHistoryID,
				View:      v1pb.ChangelogView_CHANGELOG_VIEW_FULL,
				SdlFormat: true,
			},
		})
		if err != nil {
			return "", err
		}
		schema = changeHistory.Msg.Schema
	}

	return schema, nil
}

func (s *DatabaseService) getParserEngine(ctx context.Context, request *v1pb.DiffSchemaRequest) (storepb.Engine, error) {
	var instanceID string

	if strings.Contains(request.Name, common.ChangelogPrefix) {
		insID, _, _, err := common.GetInstanceDatabaseChangelogUID(request.Name)
		if err != nil {
			return storepb.Engine_ENGINE_UNSPECIFIED, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
		}
		instanceID = insID
	} else {
		insID, _, err := common.GetInstanceDatabaseID(request.Name)
		if err != nil {
			return storepb.Engine_ENGINE_UNSPECIFIED, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
		}
		instanceID = insID
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return storepb.Engine_ENGINE_UNSPECIFIED, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return storepb.Engine_ENGINE_UNSPECIFIED, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", instanceID))
	}

	return common.ConvertToParserEngine(instance.Metadata.GetEngine())
}

func convertToChangedResources(r *storepb.ChangedResources) *v1pb.ChangedResources {
	if r == nil {
		return nil
	}
	result := &v1pb.ChangedResources{}
	for _, database := range r.Databases {
		v1Database := &v1pb.ChangedResourceDatabase{
			Name: database.Name,
		}
		for _, schema := range database.Schemas {
			v1Schema := &v1pb.ChangedResourceSchema{
				Name: schema.Name,
			}
			for _, table := range schema.Tables {
				var ranges []*v1pb.Range
				for _, r := range table.Ranges {
					ranges = append(ranges, &v1pb.Range{
						Start: r.Start,
						End:   r.End,
					})
				}
				v1Schema.Tables = append(v1Schema.Tables, &v1pb.ChangedResourceTable{
					Name:   table.Name,
					Ranges: ranges,
				})
			}
			slices.SortFunc(v1Schema.Tables, func(a, b *v1pb.ChangedResourceTable) int {
				if a.Name < b.Name {
					return -1
				} else if a.Name > b.Name {
					return 1
				}
				return 0
			})
			v1Database.Schemas = append(v1Database.Schemas, v1Schema)
		}
		slices.SortFunc(v1Database.Schemas, func(a, b *v1pb.ChangedResourceSchema) int {
			if a.Name < b.Name {
				return -1
			} else if a.Name > b.Name {
				return 1
			}
			return 0
		})
		result.Databases = append(result.Databases, v1Database)
	}
	slices.SortFunc(result.Databases, func(a, b *v1pb.ChangedResourceDatabase) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return result
}

func (s *DatabaseService) convertToDatabase(ctx context.Context, database *store.DatabaseMessage) (*v1pb.Database, error) {
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &database.InstanceID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to find instance")
	}

	var environment, effectiveEnvironment *string
	if database.EnvironmentID != nil && *database.EnvironmentID != "" {
		env := common.FormatEnvironment(*database.EnvironmentID)
		environment = &env
	}
	if database.EffectiveEnvironmentID != nil && *database.EffectiveEnvironmentID != "" {
		effEnv := common.FormatEnvironment(*database.EffectiveEnvironmentID)
		effectiveEnvironment = &effEnv
	}
	instanceResource := convertInstanceMessageToInstanceResource(instance)
	return &v1pb.Database{
		Name:                 common.FormatDatabase(database.InstanceID, database.DatabaseName),
		State:                convertDeletedToState(database.Deleted),
		SuccessfulSyncTime:   database.Metadata.GetLastSyncTime(),
		Project:              common.FormatProject(database.ProjectID),
		Environment:          environment,
		EffectiveEnvironment: effectiveEnvironment,
		SchemaVersion:        database.Metadata.GetVersion(),
		Labels:               database.Metadata.Labels,
		InstanceResource:     instanceResource,
		BackupAvailable:      database.Metadata.GetBackupAvailable(),
		Drifted:              database.Metadata.GetDrifted(),
	}, nil
}

type metadataFilter struct {
	schema *string
	table  *string
}

func (s *DatabaseService) GetSchemaString(ctx context.Context, req *connect.Request[v1pb.GetSchemaStringRequest]) (*connect.Response[v1pb.GetSchemaStringResponse], error) {
	database, err := getDatabaseMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	if instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", database.InstanceID))
	}

	dbSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get database schema: %v", err))
	}

	switch req.Msg.Type {
	case v1pb.GetSchemaStringRequest_OBJECT_TYPE_UNSPECIFIED:
		if req.Msg.Metadata == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("metadata is required"))
		}
		storeSchema := convertV1DatabaseMetadata(req.Msg.Metadata)
		s, err := schema.GetDatabaseDefinition(instance.Metadata.Engine, schema.GetDefinitionContext{
			SkipBackupSchema: false,
			PrintHeader:      false,
		}, storeSchema)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get database schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_DATABASE:
		metadata := dbSchema.GetMetadata()
		s, err := schema.GetDatabaseDefinition(instance.Metadata.Engine, schema.GetDefinitionContext{
			SkipBackupSchema: false,
			PrintHeader:      false,
		}, metadata)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get database schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_SCHEMA:
		schemaMetadata := dbSchema.GetDatabaseMetadata().GetSchema(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}

		s, err := schema.GetSchemaDefinition(instance.Metadata.Engine, schemaMetadata.GetProto())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get schema schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_TABLE:
		schemaMetadata := dbSchema.GetDatabaseMetadata().GetSchema(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		tableMetadata := schemaMetadata.GetTable(req.Msg.Object)
		if tableMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("table %q not found", req.Msg.Object))
		}
		sequences := schemaMetadata.GetSequencesByOwnerTable(req.Msg.Object)
		var sequencesProto []*storepb.SequenceMetadata
		for _, sequence := range sequences {
			sequencesProto = append(sequencesProto, sequence.GetProto())
		}
		s, err := schema.GetTableDefinition(instance.Metadata.Engine, req.Msg.Schema, tableMetadata.GetProto(), sequencesProto)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get table schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_VIEW:
		schemaMetadata := dbSchema.GetDatabaseMetadata().GetSchema(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		viewMetadata := schemaMetadata.GetView(req.Msg.Object)
		if viewMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("view %q not found", req.Msg.Object))
		}
		s, err := schema.GetViewDefinition(instance.Metadata.Engine, req.Msg.Schema, viewMetadata.GetProto())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get view schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_MATERIALIZED_VIEW:
		schemaMetadata := dbSchema.GetDatabaseMetadata().GetSchema(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		materializedViewMetadata := schemaMetadata.GetMaterializedView(req.Msg.Object)
		if materializedViewMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("materialized view %q not found", req.Msg.Object))
		}
		s, err := schema.GetMaterializedViewDefinition(instance.Metadata.Engine, req.Msg.Schema, materializedViewMetadata.GetProto())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get materialized view schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_FUNCTION:
		schemaMetadata := dbSchema.GetDatabaseMetadata().GetSchema(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		functionMetadata := schemaMetadata.GetFunction(req.Msg.Object)
		if functionMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("function %q not found", req.Msg.Object))
		}
		s, err := schema.GetFunctionDefinition(instance.Metadata.Engine, req.Msg.Schema, functionMetadata.GetProto())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get function schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_PROCEDURE:
		schemaMetadata := dbSchema.GetDatabaseMetadata().GetSchema(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		procedureMetadata := schemaMetadata.GetProcedure(req.Msg.Object)
		if procedureMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("procedure %q not found", req.Msg.Object))
		}
		s, err := schema.GetProcedureDefinition(instance.Metadata.Engine, req.Msg.Schema, procedureMetadata.GetProto())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get procedure schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_SEQUENCE:
		schemaMetadata := dbSchema.GetDatabaseMetadata().GetSchema(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		sequenceMetadata := schemaMetadata.GetSequence(req.Msg.Object)
		if sequenceMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("sequence %q not found", req.Msg.Object))
		}
		s, err := schema.GetSequenceDefinition(instance.Metadata.Engine, req.Msg.Schema, sequenceMetadata.GetProto())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get sequence schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported schema type %v", req.Msg.Type))
	}
}
