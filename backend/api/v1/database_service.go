package v1

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// DatabaseService implements the database service.
type DatabaseService struct {
	v1connect.UnimplementedDatabaseServiceHandler
	store        *store.Store
	schemaSyncer *schemasync.Syncer
	profile      *config.Profile
	iamManager   *iam.Manager
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(store *store.Store, schemaSyncer *schemasync.Syncer, profile *config.Profile, iamManager *iam.Manager) *DatabaseService {
	return &DatabaseService{
		store:        store,
		schemaSyncer: schemaSyncer,
		profile:      profile,
		iamManager:   iamManager,
	}
}

// GetDatabase gets a database.
func (s *DatabaseService) GetDatabase(ctx context.Context, req *connect.Request[v1pb.GetDatabaseRequest]) (*connect.Response[v1pb.Database], error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", req.Msg.Name))
	}
	databaseMessage, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if databaseMessage == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", req.Msg.Name))
	}
	database, err := s.convertToDatabase(ctx, databaseMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert database"))
	}
	return connect.NewResponse(database), nil
}

func (s *DatabaseService) BatchGetDatabases(ctx context.Context, req *connect.Request[v1pb.BatchGetDatabasesRequest]) (*connect.Response[v1pb.BatchGetDatabasesResponse], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	// Parse parent to extract project ID filter if specified.
	var projectIDFilter *string
	if strings.HasPrefix(req.Msg.Parent, common.ProjectNamePrefix) {
		projectID, err := common.GetProjectID(req.Msg.Parent)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid parent %q", req.Msg.Parent))
		}
		projectIDFilter = &projectID
	}
	// For instances/{instance} or "-" (wildcard), no project filter is applied.
	databases := make([]*v1pb.Database, 0, len(req.Msg.Names))
	for _, name := range req.Msg.Names {
		instanceID, databaseName, err := common.GetInstanceDatabaseID(name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", name))
		}
		databaseMessage, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
		}
		if databaseMessage == nil {
			// Ignore deleted database.
			continue
		}
		// If parent specifies a project, validate database belongs to that project.
		if projectIDFilter != nil && databaseMessage.ProjectID != *projectIDFilter {
			// Ignore database not in the specified project.
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
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert database"))
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

// ListDatabases lists all databases.
func (s *DatabaseService) ListDatabases(ctx context.Context, req *connect.Request[v1pb.ListDatabasesRequest]) (*connect.Response[v1pb.ListDatabasesResponse], error) {
	user, ok := GetUserFromContext(ctx)
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

	orderByKeys, err := store.GetDatabaseOrders(req.Msg.OrderBy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.OrderByKeys = orderByKeys

	filterQ, err := store.GetListDatabaseFilter(req.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.FilterQ = filterQ

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
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	response := &v1pb.ListDatabasesResponse{
		NextPageToken: nextPageToken,
	}
	for _, databaseMessage := range databaseMessages {
		database, err := s.convertToDatabase(ctx, databaseMessage)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert database"))
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
	instanceID, databaseName, err := common.GetInstanceDatabaseID(req.Msg.Database.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", req.Msg.Database.Name))
	}
	databaseMessage, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if databaseMessage == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", req.Msg.Database.Name))
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
			project, err = s.store.GetProject(ctx, &store.FindProjectMessage{
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
			instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &databaseMessage.InstanceID})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get instance for format version")
			}
			if instance == nil {
				return nil, errors.Errorf("instance %q not found", databaseMessage.InstanceID)
			}
			if _, err := s.store.CreateChangelog(ctx, &store.ChangelogMessage{
				InstanceID:     databaseMessage.InstanceID,
				DatabaseName:   databaseMessage.DatabaseName,
				Status:         store.ChangelogStatusDone,
				SyncHistoryUID: &syncHistory,
				Payload: &storepb.ChangelogPayload{
					Type:        storepb.ChangelogPayload_BASELINE,
					GitCommit:   s.profile.GitCommit,
					DumpVersion: schema.GetDumpFormatVersion(instance.Metadata.GetEngine()),
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
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert database"))
	}
	return connect.NewResponse(database), nil
}

// SyncDatabase syncs the schema of a database.
func (s *DatabaseService) SyncDatabase(ctx context.Context, req *connect.Request[v1pb.SyncDatabaseRequest]) (*connect.Response[v1pb.SyncDatabaseResponse], error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", req.Msg.Name))
	}
	databaseMessage, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if databaseMessage == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", req.Msg.Name))
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
		instanceID, databaseName, err := common.GetInstanceDatabaseID(name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", name))
		}
		databaseMessage, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
		}
		if databaseMessage == nil {
			continue
		}
		s.schemaSyncer.SyncDatabaseAsync(databaseMessage)
	}
	return connect.NewResponse(&v1pb.BatchSyncDatabasesResponse{}), nil
}

// BatchUpdateDatabases updates databases in batch.
func (s *DatabaseService) BatchUpdateDatabases(ctx context.Context, req *connect.Request[v1pb.BatchUpdateDatabasesRequest]) (*connect.Response[v1pb.BatchUpdateDatabasesResponse], error) {
	response := &v1pb.BatchUpdateDatabasesResponse{}
	for _, updateReq := range req.Msg.GetRequests() {
		updated, err := s.UpdateDatabase(ctx, connect.NewRequest(updateReq))
		if err != nil {
			return nil, err
		}
		response.Databases = append(response.Databases, updated.Msg)
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
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				if variable != "table" {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %v", variable))
				}
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`invalid args for %q`, variable))
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect string, got %T, hint: filter literals should be string", value))
				}
				metaFilter.table = &tableMetadataFilter{
					name:     strings.ToLower(strValue),
					wildcard: true,
				}
				return nil
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				strValue, ok := value.(string)
				if !ok {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected string but found %q", value))
				}
				switch variable {
				case "schema":
					lowerValue := strings.ToLower(strValue)
					metaFilter.schema = &lowerValue
				case "table":
					metaFilter.table = &tableMetadataFilter{
						name: strings.ToLower(strValue),
					}
				default:
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %v", variable))
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
	instanceID, databaseName, err := common.GetInstanceDatabaseID(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", name))
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", name))
	}
	dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	if dbMetadata == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to sync database schema for database %q, error %v", name, err))
		}
		newDBSchema, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: database.DatabaseName,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
		}
		if newDBSchema == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database schema %q not found", name))
		}
		dbMetadata = newDBSchema
	}

	filter, err := getDatabaseMetadataFilter(req.Msg.Filter)
	if err != nil {
		return nil, err
	}
	v1pbMetadata := convertStoreDatabaseMetadata(dbMetadata.GetProto(), filter, int(req.Msg.Limit))
	v1pbMetadata.Name = fmt.Sprintf("%s%s/%s%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName, common.MetadataSuffix)

	return connect.NewResponse(v1pbMetadata), nil
}

// GetDatabaseSchema gets the schema of a database.
func (s *DatabaseService) GetDatabaseSchema(ctx context.Context, req *connect.Request[v1pb.GetDatabaseSchemaRequest]) (*connect.Response[v1pb.DatabaseSchema], error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(req.Msg.Name, common.SchemaSuffix)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", databaseName))
	}
	dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	if dbMetadata == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to sync database schema for database %q, error %v", databaseName, err))
		}
		newDBSchema, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: database.DatabaseName,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
		}
		if newDBSchema == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database schema %q not found", databaseName))
		}
		dbMetadata = newDBSchema
	}
	schemaString := string(dbMetadata.GetRawDump())
	return connect.NewResponse(&v1pb.DatabaseSchema{Schema: schemaString}), nil
}

// GetDatabaseSDLSchema gets the SDL schema of a database.
func (s *DatabaseService) GetDatabaseSDLSchema(ctx context.Context, req *connect.Request[v1pb.GetDatabaseSDLSchemaRequest]) (*connect.Response[v1pb.DatabaseSDLSchema], error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(req.Msg.Name, common.SDLSchemaSuffix)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
	}

	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", databaseName))
	}
	dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
	}
	if dbMetadata == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to sync database schema for database %q, error %v", databaseName, err))
		}
		newDBSchema, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: database.DatabaseName,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("%v", err.Error()))
		}
		if newDBSchema == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database schema %q not found", databaseName))
		}
		dbMetadata = newDBSchema
	}

	metadata := dbMetadata.GetProto()
	if metadata == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("database metadata not found for database %q", databaseName))
	}

	format := req.Msg.Format
	if format == v1pb.GetDatabaseSDLSchemaRequest_SDL_FORMAT_UNSPECIFIED {
		format = v1pb.GetDatabaseSDLSchemaRequest_SINGLE_FILE
	}

	switch format {
	case v1pb.GetDatabaseSDLSchemaRequest_SINGLE_FILE:
		return s.getSingleFileSDL(database.Engine, metadata)
	case v1pb.GetDatabaseSDLSchemaRequest_MULTI_FILE:
		return s.getMultiFileSDL(database.Engine, metadata)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported SDL format: %v", format))
	}
}

// DiffSchema diff the database schema.
func (s *DatabaseService) DiffSchema(ctx context.Context, req *connect.Request[v1pb.DiffSchemaRequest]) (*connect.Response[v1pb.DiffSchemaResponse], error) {
	// Use unified SDL-based approach for all scenarios
	sourceDBSchema, err := s.getSourceDBMetadata(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get source schema"))
	}

	targetDBSchema, err := s.getTargetDBMetadata(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get target schema"))
	}

	engine, err := s.getParserEngine(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get parser engine"))
	}

	schemaDiff, err := schema.GetDatabaseSchemaDiff(engine, sourceDBSchema, targetDBSchema)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to compute schema diff"))
	}

	// Filter out bbdataarchive schema changes for Postgres
	if engine == storepb.Engine_POSTGRES {
		schemaDiff = schema.FilterPostgresArchiveSchema(schemaDiff)
	}

	migrationSQL, err := schema.GenerateMigration(engine, schemaDiff)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate migration SQL"))
	}

	return connect.NewResponse(&v1pb.DiffSchemaResponse{
		Diff: migrationSQL,
	}), nil
}

func (s *DatabaseService) getSourceDBMetadata(ctx context.Context, request *v1pb.DiffSchemaRequest) (*model.DatabaseMetadata, error) {
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
			instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
			if err != nil {
				return nil, err
			}
			if instance == nil {
				return nil, errors.Errorf("instance %s not found", instanceID)
			}

			return model.NewDatabaseMetadata(
				syncHistory.Metadata,
				[]byte(syncHistory.Schema),
				&storepb.DatabaseConfig{},
				instance.Metadata.GetEngine(),
				store.IsObjectCaseSensitive(instance),
			), nil
		}

		// Fallback to current database schema if no sync history
		dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   instanceID,
			DatabaseName: databaseName,
		})
		if err != nil {
			return nil, err
		}
		if dbMetadata == nil {
			return nil, errors.Errorf("database schema not found for %s/%s", instanceID, databaseName)
		}
		return dbMetadata, nil
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Name)
	if err != nil {
		return nil, err
	}

	dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
	})
	if err != nil {
		return nil, err
	}
	if dbMetadata == nil {
		return nil, errors.Errorf("database schema not found for %s/%s", instanceID, databaseName)
	}
	return dbMetadata, nil
}

func (s *DatabaseService) getTargetDBMetadata(ctx context.Context, request *v1pb.DiffSchemaRequest) (*model.DatabaseMetadata, error) {
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
			instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
			if err != nil {
				return nil, err
			}
			if instance == nil {
				return nil, errors.Errorf("instance %s not found", instanceID)
			}

			return model.NewDatabaseMetadata(
				syncHistory.Metadata,
				[]byte(syncHistory.Schema),
				&storepb.DatabaseConfig{},
				instance.Metadata.GetEngine(),
				store.IsObjectCaseSensitive(instance),
			), nil
		}

		// Fallback to current database schema if no sync history
		dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   instanceID,
			DatabaseName: databaseName,
		})
		if err != nil {
			return nil, err
		}
		if dbMetadata == nil {
			return nil, errors.Errorf("database schema not found for %s/%s", instanceID, databaseName)
		}
		return dbMetadata, nil
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
		instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, err
		}
		if instance == nil {
			return nil, errors.Errorf("instance %s not found", instanceID)
		}

		// Create DatabaseSchema from the parsed metadata
		return model.NewDatabaseMetadata(
			metadata,
			[]byte(schemaStr),
			&storepb.DatabaseConfig{},
			engine,
			store.IsObjectCaseSensitive(instance),
		), nil
	}

	return nil, errors.Errorf("must set the schema or change history id as the target")
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

	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return storepb.Engine_ENGINE_UNSPECIFIED, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return storepb.Engine_ENGINE_UNSPECIFIED, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", instanceID))
	}

	return common.ConvertToParserEngine(instance.Metadata.GetEngine())
}

func (s *DatabaseService) convertToDatabase(ctx context.Context, database *store.DatabaseMessage) (*v1pb.Database, error) {
	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{
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

type tableMetadataFilter struct {
	name     string
	wildcard bool
}

type metadataFilter struct {
	schema *string // exact match
	table  *tableMetadataFilter
}

func (s *DatabaseService) GetSchemaString(ctx context.Context, req *connect.Request[v1pb.GetSchemaStringRequest]) (*connect.Response[v1pb.GetSchemaStringResponse], error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", req.Msg.Name))
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database %q", req.Msg.Name))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", req.Msg.Name))
	}
	dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get database schema: %v", err))
	}

	switch req.Msg.Type {
	case v1pb.GetSchemaStringRequest_OBJECT_TYPE_UNSPECIFIED:
		if req.Msg.Metadata == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("metadata is required"))
		}
		storeSchema := convertV1DatabaseMetadata(req.Msg.Metadata)
		s, err := schema.GetDatabaseDefinition(database.Engine, schema.GetDefinitionContext{
			SkipBackupSchema: false,
			PrintHeader:      false,
		}, storeSchema)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get database schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_DATABASE:
		metadata := dbMetadata.GetProto()
		s, err := schema.GetDatabaseDefinition(database.Engine, schema.GetDefinitionContext{
			SkipBackupSchema: false,
			PrintHeader:      false,
		}, metadata)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get database schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_SCHEMA:
		schemaMetadata := dbMetadata.GetSchemaMetadata(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}

		s, err := schema.GetSchemaDefinition(database.Engine, schemaMetadata.GetProto())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get schema schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_TABLE:
		schemaMetadata := dbMetadata.GetSchemaMetadata(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		tableMetadata := schemaMetadata.GetTable(req.Msg.Object)
		if tableMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("table %q not found", req.Msg.Object))
		}
		sequences := schemaMetadata.GetSequencesByOwnerTable(req.Msg.Object)
		s, err := schema.GetTableDefinition(database.Engine, req.Msg.Schema, tableMetadata.GetProto(), sequences)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get table schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_VIEW:
		schemaMetadata := dbMetadata.GetSchemaMetadata(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		viewMetadata := schemaMetadata.GetView(req.Msg.Object)
		if viewMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("view %q not found", req.Msg.Object))
		}
		s, err := schema.GetViewDefinition(database.Engine, req.Msg.Schema, viewMetadata)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get view schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_MATERIALIZED_VIEW:
		schemaMetadata := dbMetadata.GetSchemaMetadata(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		materializedViewMetadata := schemaMetadata.GetMaterializedView(req.Msg.Object)
		if materializedViewMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("materialized view %q not found", req.Msg.Object))
		}
		s, err := schema.GetMaterializedViewDefinition(database.Engine, req.Msg.Schema, materializedViewMetadata)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get materialized view schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_FUNCTION:
		schemaMetadata := dbMetadata.GetSchemaMetadata(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		var functionMetadata *storepb.FunctionMetadata
		for _, fn := range schemaMetadata.GetProto().GetFunctions() {
			if fn.Name == req.Msg.Object {
				functionMetadata = fn
				break
			}
		}
		if functionMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("function %q not found", req.Msg.Object))
		}
		s, err := schema.GetFunctionDefinition(database.Engine, req.Msg.Schema, functionMetadata)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get function schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_PROCEDURE:
		schemaMetadata := dbMetadata.GetSchemaMetadata(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		procedureMetadata := schemaMetadata.GetProcedure(req.Msg.Object)
		if procedureMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("procedure %q not found", req.Msg.Object))
		}
		s, err := schema.GetProcedureDefinition(database.Engine, req.Msg.Schema, procedureMetadata)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get procedure schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	case v1pb.GetSchemaStringRequest_SEQUENCE:
		schemaMetadata := dbMetadata.GetSchemaMetadata(req.Msg.Schema)
		if schemaMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("schema %q not found", req.Msg.Schema))
		}
		sequenceMetadata := schemaMetadata.GetSequence(req.Msg.Object)
		if sequenceMetadata == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("sequence %q not found", req.Msg.Object))
		}
		s, err := schema.GetSequenceDefinition(database.Engine, req.Msg.Schema, sequenceMetadata)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("Failed to get sequence schema: %v", err))
		}
		return connect.NewResponse(&v1pb.GetSchemaStringResponse{SchemaString: s}), nil
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported schema type %v", req.Msg.Type))
	}
}

func (*DatabaseService) getSingleFileSDL(engine storepb.Engine, metadata *storepb.DatabaseSchemaMetadata) (*connect.Response[v1pb.DatabaseSDLSchema], error) {
	sdlText, err := schema.GetDatabaseDefinition(engine, schema.GetDefinitionContext{
		SkipBackupSchema: true,
		SDLFormat:        true,
	}, metadata)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate SDL schema: %v", err))
	}

	return connect.NewResponse(&v1pb.DatabaseSDLSchema{
		Schema:      []byte(sdlText),
		ContentType: "text/plain; charset=utf-8",
	}), nil
}

func (*DatabaseService) getMultiFileSDL(engine storepb.Engine, metadata *storepb.DatabaseSchemaMetadata) (*connect.Response[v1pb.DatabaseSDLSchema], error) {
	// Get multi-file schema from schema package
	result, err := schema.GetMultiFileDatabaseDefinition(engine, schema.GetDefinitionContext{
		SkipBackupSchema: true,
		SDLFormat:        true,
		MultiFileFormat:  true,
	}, metadata)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate multi-file SDL schema: %v", err))
	}

	// Create ZIP archive
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for _, file := range result.Files {
		writer, err := zipWriter.Create(file.Name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create zip entry %s: %v", file.Name, err))
		}
		if _, err := writer.Write([]byte(file.Content)); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to write content to %s: %v", file.Name, err))
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to close zip writer: %v", err))
	}

	return connect.NewResponse(&v1pb.DatabaseSDLSchema{
		Schema:      buf.Bytes(),
		ContentType: "application/zip",
	}), nil
}
