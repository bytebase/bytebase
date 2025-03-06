package v1

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	tidbparser "github.com/pingcap/tidb/pkg/parser"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pkg/errors"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	filterKeyEnvironment = "environment"
	filterKeyDatabase    = "database"
	filterKeyStartTime   = "start_time"

	// Support order by count, latest_log_time, average_query_time, maximum_query_time,
	// average_rows_sent, maximum_rows_sent, average_rows_examined, maximum_rows_examined for now.
	orderByKeyCount               = "count"
	orderByKeyLatestLogTime       = "latest_log_time"
	orderByKeyAverageQueryTime    = "average_query_time"
	orderByKeyMaximumQueryTime    = "maximum_query_time"
	orderByKeyAverageRowsSent     = "average_rows_sent"
	orderByKeyMaximumRowsSent     = "maximum_rows_sent"
	orderByKeyAverageRowsExamined = "average_rows_examined"
	orderByKeyMaximumRowsExamined = "maximum_rows_examined"
)

// DatabaseService implements the database service.
type DatabaseService struct {
	v1pb.UnimplementedDatabaseServiceServer
	store          *store.Store
	schemaSyncer   *schemasync.Syncer
	licenseService enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(store *store.Store, schemaSyncer *schemasync.Syncer, licenseService enterprise.LicenseService, profile *config.Profile, iamManager *iam.Manager) *DatabaseService {
	return &DatabaseService{
		store:          store,
		schemaSyncer:   schemaSyncer,
		licenseService: licenseService,
		profile:        profile,
		iamManager:     iamManager,
	}
}

// GetDatabase gets a database.
func (s *DatabaseService) GetDatabase(ctx context.Context, request *v1pb.GetDatabaseRequest) (*v1pb.Database, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	find := &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	}
	databaseMessage, err := s.store.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if databaseMessage == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	database, err := s.convertToDatabase(ctx, databaseMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert database, error: %v", err)
	}
	return database, nil
}

// TODO(ed): test it.
func getFindDatabaseFilter(filter string) (*store.FindDatabaseFilter, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	getSubCondition := func(expr celast.Expr, join string) (string, error) {
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

	getVariavleAndValue := func(expr celast.Expr) (string, any) {
		var variable string
		var value any
		for _, arg := range expr.AsCall().Args() {
			switch arg.Kind() {
			case celast.IdentKind:
				variable = arg.AsIdent()
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
			}
		}
		return variable, value
	}

	parseToSQL := func(variable, value any) (string, error) {
		switch variable {
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return "", status.Errorf(codes.InvalidArgument, "invalid project filter %q", value)
			}
			positionalArgs = append(positionalArgs, projectID)
			return fmt.Sprintf("db.project = $%d", len(positionalArgs)), nil
		case "instance":
			instanceID, err := common.GetInstanceID(value.(string))
			if err != nil {
				return "", status.Errorf(codes.InvalidArgument, "invalid instance filter %q", value)
			}
			positionalArgs = append(positionalArgs, instanceID)
			return fmt.Sprintf("db.instance = $%d", len(positionalArgs)), nil
		case "environment":
			environmentID, err := common.GetEnvironmentID(value.(string))
			if err != nil {
				return "", status.Errorf(codes.InvalidArgument, "invalid environment filter %q", value)
			}
			positionalArgs = append(positionalArgs, environmentID)
			return fmt.Sprintf(`
			COALESCE(
				db.environment,
				instance.environment
			) = $%d`, len(positionalArgs)), nil
		case "engine":
			v1Engine, ok := v1pb.Engine_value[value.(string)]
			if !ok {
				return "", status.Errorf(codes.InvalidArgument, "invalid engine filter %q", value)
			}
			engine := convertEngine(v1pb.Engine(v1Engine))
			positionalArgs = append(positionalArgs, engine)
			return fmt.Sprintf("instance.engine = $%d", len(positionalArgs)), nil
		case "name":
			positionalArgs = append(positionalArgs, value)
			return fmt.Sprintf("db.name = $%d", len(positionalArgs)), nil
		case "label":
			keyVal := strings.Split(value.(string), ":")
			if len(keyVal) != 2 {
				return "", status.Errorf(codes.InvalidArgument, `invalid label filter %q, should in "{label key}:{label value} format"`, value)
			}
			labelKey := keyVal[0]
			labelValues := strings.Split(keyVal[1], ",")
			positionalArgs = append(positionalArgs, labelValues)
			return fmt.Sprintf("db.metadata->'labels'->>'%s' = ANY($%d)", labelKey, len(positionalArgs)), nil
		case "exclude_unassigned":
			if _, ok := value.(bool); ok {
				positionalArgs = append(positionalArgs, api.DefaultProjectID)
				return fmt.Sprintf("db.project != $%d", len(positionalArgs)), nil
			}
			return "TRUE", nil
		default:
			return "", status.Errorf(codes.InvalidArgument, "unsupport variable %q", variable)
		}
	}

	parseToEngineSQL := func(expr celast.Expr, relation string) (string, error) {
		variable, value := getVariavleAndValue(expr)
		if variable != "engine" {
			return "", status.Errorf(codes.InvalidArgument, `only "engine" support "engine in [xx]"/"!(engine in [xx])" operator`)
		}

		rawEngineList, ok := value.([]any)
		if !ok {
			return "", status.Errorf(codes.InvalidArgument, "invalid engine value %q", value)
		}
		engineList := []string{}
		for _, rawEngine := range rawEngineList {
			v1Engine, ok := v1pb.Engine_value[rawEngine.(string)]
			if !ok {
				return "", status.Errorf(codes.InvalidArgument, "invalid engine filter %q", rawEngine)
			}
			engine := convertEngine(v1pb.Engine(v1Engine))
			positionalArgs = append(positionalArgs, engine)
			engineList = append(engineList, fmt.Sprintf("$%d", len(positionalArgs)))
		}

		return fmt.Sprintf("instance.engine %s (%s)", relation, strings.Join(engineList, ",")), nil
	}

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				return getSubCondition(expr, "OR")
			case celoperators.LogicalAnd:
				return getSubCondition(expr, "AND")
			case celoperators.Equals:
				variable, value := getVariavleAndValue(expr)
				return parseToSQL(variable, value)
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", status.Errorf(codes.InvalidArgument, `invalid args for %q`, variable)
				}
				value := args[0].AsLiteral().Value()
				if variable != "name" {
					return "", status.Errorf(codes.InvalidArgument, `only "name" support %q operator, but found %q`, celoverloads.Matches, variable)
				}
				strValue, ok := value.(string)
				if !ok {
					return "", status.Errorf(codes.InvalidArgument, "expect string, got %T, hint: filter literals should be string", value)
				}
				return "LOWER(db.name) LIKE '%" + strings.ToLower(strValue) + "%'", nil
			case celoperators.In:
				return parseToEngineSQL(expr, "IN")
			case celoperators.LogicalNot:
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", status.Errorf(codes.InvalidArgument, `only support !(engine in ["{engine1}", "{engine2}"]) format`)
				}
				return parseToEngineSQL(args[0], "NOT IN")
			default:
				return "", status.Errorf(codes.InvalidArgument, "unexpected function %v", functionName)
			}
		default:
			return "", status.Errorf(codes.InvalidArgument, "unexpected expr kind %v", expr.Kind())
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}

	return &store.FindDatabaseFilter{
		Args:  positionalArgs,
		Where: "(" + where + ")",
	}, nil
}

// ListDatabases lists all databases.
func (s *DatabaseService) ListDatabases(ctx context.Context, request *v1pb.ListDatabasesRequest) (*v1pb.ListDatabasesResponse, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindDatabaseMessage{
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		ShowDeleted: request.ShowDeleted,
	}

	filter, err := getFindDatabaseFilter(request.Filter)
	if err != nil {
		return nil, err
	}
	find.Filter = filter

	switch {
	case strings.HasPrefix(request.Parent, common.ProjectNamePrefix):
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionProjectsGet, user)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "user does not have permission %q", iam.PermissionProjectsGet)
		}

		p, err := common.GetProjectID(request.Parent)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parent %q", request.Parent)
		}
		find.ProjectID = &p
	case strings.HasPrefix(request.Parent, common.WorkspacePrefix):
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionDatabasesList, user)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "user does not have permission %q", iam.PermissionDatabasesList)
		}
	case strings.HasPrefix(request.Parent, common.InstanceNamePrefix):
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionInstancesGet, user)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "user does not have permission %q", iam.PermissionInstancesGet)
		}

		instanceID, err := common.GetInstanceID(request.Parent)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parent %q", request.Parent)
		}
		find.InstanceID = &instanceID
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid parent %q", request.Parent)
	}

	databaseMessages, err := s.store.ListDatabases(ctx, find)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	nextPageToken := ""
	if len(databaseMessages) == limitPlusOne {
		databaseMessages = databaseMessages[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal next page token, error: %v", err)
		}
	}

	response := &v1pb.ListDatabasesResponse{
		NextPageToken: nextPageToken,
	}
	for _, databaseMessage := range databaseMessages {
		database, err := s.convertToDatabase(ctx, databaseMessage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert database, error: %v", err)
		}
		response.Databases = append(response.Databases, database)
	}
	return response, nil
}

// UpdateDatabase updates a database.
func (s *DatabaseService) UpdateDatabase(ctx context.Context, request *v1pb.UpdateDatabaseRequest) (*v1pb.Database, error) {
	if request.Database == nil {
		return nil, status.Errorf(codes.InvalidArgument, "database must be set")
	}
	if len(request.GetUpdateMask().GetPaths()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Database.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if databaseMessage == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	var project *store.ProjectMessage
	patch := &store.UpdateDatabaseMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		Metadata:     proto.Clone(databaseMessage.Metadata).(*storepb.DatabaseMetadata),
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "project":
			projectID, err := common.GetProjectID(request.Database.Project)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID:  &projectID,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			if project == nil {
				return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
			}
			if project.Deleted {
				return nil, status.Errorf(codes.FailedPrecondition, "project %q is deleted", projectID)
			}
			patch.ProjectID = &project.ResourceID
		case "labels":
			patch.Metadata.Labels = request.Database.Labels
		case "environment":
			if request.Database.Environment != "" {
				environmentID, err := common.GetEnvironmentID(request.Database.Environment)
				if err != nil {
					return nil, status.Error(codes.InvalidArgument, err.Error())
				}
				environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
					ResourceID:  &environmentID,
					ShowDeleted: true,
				})
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
				if environment == nil {
					return nil, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
				}
				if environment.Deleted {
					return nil, status.Errorf(codes.FailedPrecondition, "environment %q is deleted", environmentID)
				}
				patch.EnvironmentID = &environmentID
			} else {
				unsetEnvironment := ""
				patch.EnvironmentID = &unsetEnvironment
			}
		}
	}

	updatedDatabase, err := s.store.UpdateDatabase(ctx, patch)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	database, err := s.convertToDatabase(ctx, updatedDatabase)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert database, error: %v", err)
	}
	return database, nil
}

// SyncDatabase syncs the schema of a database.
func (s *DatabaseService) SyncDatabase(ctx context.Context, request *v1pb.SyncDatabaseRequest) (*v1pb.SyncDatabaseResponse, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	find := &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	}
	database, err := s.store.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
		return nil, err
	}
	return &v1pb.SyncDatabaseResponse{}, nil
}

// BatchUpdateDatabases updates a database in batch.
func (s *DatabaseService) BatchUpdateDatabases(ctx context.Context, request *v1pb.BatchUpdateDatabasesRequest) (*v1pb.BatchUpdateDatabasesResponse, error) {
	databases := []*store.DatabaseMessage{}
	batchUpdate := &store.BatchUpdateDatabases{}
	var updateMask *fieldmaskpb.FieldMask
	for _, req := range request.Requests {
		if req.Database == nil {
			return nil, status.Errorf(codes.InvalidArgument, "database must be set")
		}
		if req.UpdateMask == nil {
			return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
		}
		instanceID, databaseName, err := common.GetInstanceDatabaseID(req.Database.Name)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
		}
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:      &instanceID,
			DatabaseName:    &databaseName,
			IsCaseSensitive: store.IsObjectCaseSensitive(instance),
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
		}
		if updateMask == nil {
			updateMask = req.UpdateMask
		}
		if !slices.Equal(updateMask.Paths, req.UpdateMask.Paths) {
			return nil, status.Errorf(codes.InvalidArgument, "all databases should have the same update_mask")
		}
		for _, path := range req.UpdateMask.Paths {
			switch path {
			case "project":
				projectID, err := common.GetProjectID(req.Database.Project)
				if err != nil {
					return nil, status.Error(codes.InvalidArgument, err.Error())
				}
				if batchUpdate.ProjectID != nil && *batchUpdate.ProjectID != projectID {
					return nil, status.Errorf(codes.InvalidArgument, "all databases should use the same project")
				}
				batchUpdate.ProjectID = &projectID
			case "environment":
				if req.Database.Environment != "" {
					envID, err := common.GetEnvironmentID(req.Database.Environment)
					if err != nil {
						return nil, status.Error(codes.InvalidArgument, err.Error())
					}
					if batchUpdate.EnvironmentID != nil && *batchUpdate.EnvironmentID != envID {
						return nil, status.Errorf(codes.InvalidArgument, "all databases should use the same environment")
					}
					batchUpdate.EnvironmentID = &envID
				} else {
					if batchUpdate.EnvironmentID != nil && *batchUpdate.EnvironmentID != "" {
						return nil, status.Errorf(codes.InvalidArgument, "all databases should use the same environment")
					}
					unsetEnvironment := ""
					batchUpdate.EnvironmentID = &unsetEnvironment
				}
			default:
				return nil, status.Errorf(codes.InvalidArgument, "unsupported update_mask path %q", path)
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
			return nil, status.Error(codes.Internal, err.Error())
		}
		if project == nil {
			return nil, status.Errorf(codes.NotFound, "project %q not found", *batchUpdate.ProjectID)
		}
		if project.Deleted {
			return nil, status.Errorf(codes.FailedPrecondition, "project %q is deleted", *batchUpdate.ProjectID)
		}
	}
	if batchUpdate.EnvironmentID != nil && *batchUpdate.EnvironmentID != "" {
		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
			ResourceID:  batchUpdate.EnvironmentID,
			ShowDeleted: true,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if environment == nil {
			return nil, status.Errorf(codes.NotFound, "environment %q not found", *batchUpdate.EnvironmentID)
		}
		if environment.Deleted {
			return nil, status.Errorf(codes.FailedPrecondition, "environment %q is deleted", *batchUpdate.EnvironmentID)
		}
	}

	updatedDatabases, err := s.store.BatchUpdateDatabases(ctx, databases, batchUpdate)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	response := &v1pb.BatchUpdateDatabasesResponse{}
	for _, databaseMessage := range updatedDatabases {
		database, err := s.convertToDatabase(ctx, databaseMessage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert database, error: %v", err)
		}
		response.Databases = append(response.Databases, database)
	}
	return response, nil
}

// GetDatabaseMetadata gets the metadata of a database.
func (s *DatabaseService) GetDatabaseMetadata(ctx context.Context, request *v1pb.GetDatabaseMetadataRequest) (*v1pb.DatabaseMetadata, error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(request.Name, common.MetadataSuffix)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to sync database schema for database %q, error %v", databaseName, err)
		}
		newDBSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if newDBSchema == nil {
			return nil, status.Errorf(codes.NotFound, "database schema %q not found", databaseName)
		}
		dbSchema = newDBSchema
	}

	var filter *metadataFilter
	if request.Filter != "" {
		schema, table, err := common.GetSchemaTableName(request.Filter)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", filter)
		}
		filter = &metadataFilter{schema: schema, table: table}
	}
	v1pbMetadata, err := convertStoreDatabaseMetadata(dbSchema.GetMetadata(), filter)
	if err != nil {
		return nil, err
	}
	v1pbMetadata.Name = fmt.Sprintf("%s%s/%s%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName, common.MetadataSuffix)

	return v1pbMetadata, nil
}

// GetDatabaseSchema gets the schema of a database.
func (s *DatabaseService) GetDatabaseSchema(ctx context.Context, request *v1pb.GetDatabaseSchemaRequest) (*v1pb.DatabaseSchema, error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(request.Name, common.SchemaSuffix)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to sync database schema for database %q, error %v", databaseName, err)
		}
		newDBSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if newDBSchema == nil {
			return nil, status.Errorf(codes.NotFound, "database schema %q not found", databaseName)
		}
		dbSchema = newDBSchema
	}
	// We only support MySQL engine for now.
	schema := string(dbSchema.GetSchema())
	if request.SdlFormat {
		switch instance.Metadata.GetEngine() {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			sdlSchema, err := transform.SchemaTransform(storepb.Engine_MYSQL, schema)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert schema to sdl format, error %v", err.Error())
			}
			schema = sdlSchema
		}
	}
	return &v1pb.DatabaseSchema{Schema: schema}, nil
}

// DiffSchema diff the database schema.
func (s *DatabaseService) DiffSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (*v1pb.DiffSchemaResponse, error) {
	source, err := s.getSourceSchema(ctx, request)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get source schema, error: %v", err)
	}

	target, err := s.getTargetSchema(ctx, request)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get target schema, error: %v", err)
	}

	engine, err := s.getParserEngine(ctx, request)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get parser engine, error: %v", err)
	}

	strictMode := true
	if engine == storepb.Engine_ORACLE {
		strictMode = false
	}
	diff, err := base.SchemaDiff(engine, base.DiffContext{
		IgnoreCaseSensitive: false,
		StrictMode:          strictMode,
	}, source, target)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute diff between source and target schemas, error: %v", err)
	}

	return &v1pb.DiffSchemaResponse{
		Diff: diff,
	}, nil
}

func (s *DatabaseService) getSourceSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (string, error) {
	if strings.Contains(request.Name, common.ChangelogPrefix) {
		changeHistory, err := s.GetChangelog(ctx, &v1pb.GetChangelogRequest{
			Name:      request.Name,
			View:      v1pb.ChangelogView_CHANGELOG_VIEW_FULL,
			SdlFormat: true,
		})
		if err != nil {
			return "", err
		}
		return changeHistory.Schema, nil
	}

	databaseSchema, err := s.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{
		Name:      fmt.Sprintf("%s/schema", request.Name),
		SdlFormat: request.SdlFormat,
	})
	if err != nil {
		return "", err
	}
	return databaseSchema.Schema, nil
}

func (s *DatabaseService) getTargetSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (string, error) {
	schema := request.GetSchema()
	changeHistoryID := request.GetChangelog()
	// TODO: maybe we will support an empty schema as the target.
	if schema == "" && changeHistoryID == "" {
		return "", status.Errorf(codes.InvalidArgument, "must set the schema or change history id as the target")
	}

	// If the change history id is set, use the schema of the change history as the target.
	if changeHistoryID != "" {
		changeHistory, err := s.GetChangelog(ctx, &v1pb.GetChangelogRequest{
			Name:      changeHistoryID,
			View:      v1pb.ChangelogView_CHANGELOG_VIEW_FULL,
			SdlFormat: true,
		})
		if err != nil {
			return "", err
		}
		schema = changeHistory.Schema
	}

	return schema, nil
}

func (s *DatabaseService) getParserEngine(ctx context.Context, request *v1pb.DiffSchemaRequest) (storepb.Engine, error) {
	var instanceID string
	var engine storepb.Engine

	if strings.Contains(request.Name, common.ChangelogPrefix) {
		insID, _, _, err := common.GetInstanceDatabaseChangelogUID(request.Name)
		if err != nil {
			return engine, status.Error(codes.InvalidArgument, err.Error())
		}
		instanceID = insID
	} else {
		insID, _, err := common.GetInstanceDatabaseID(request.Name)
		if err != nil {
			return engine, status.Error(codes.InvalidArgument, err.Error())
		}
		instanceID = insID
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return engine, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return engine, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	switch instance.Metadata.GetEngine() {
	case storepb.Engine_POSTGRES:
		engine = storepb.Engine_POSTGRES
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		engine = storepb.Engine_MYSQL
	case storepb.Engine_TIDB:
		engine = storepb.Engine_TIDB
	case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE:
		engine = storepb.Engine_ORACLE
	case storepb.Engine_MSSQL:
		engine = storepb.Engine_MSSQL
	case storepb.Engine_COCKROACHDB:
		engine = storepb.Engine_COCKROACHDB
	default:
		return engine, status.Errorf(codes.InvalidArgument, "invalid engine type %v", instance.Metadata.GetEngine())
	}

	return engine, nil
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
			sort.Slice(v1Schema.Tables, func(i, j int) bool {
				return v1Schema.Tables[i].Name < v1Schema.Tables[j].Name
			})
			v1Database.Schemas = append(v1Database.Schemas, v1Schema)
		}
		sort.Slice(v1Database.Schemas, func(i, j int) bool {
			return v1Database.Schemas[i].Name < v1Database.Schemas[j].Name
		})
		result.Databases = append(result.Databases, v1Database)
	}
	sort.Slice(result.Databases, func(i, j int) bool {
		return result.Databases[i].Name < result.Databases[j].Name
	})
	return result
}

// ListSecrets lists the secrets of a database.
func (s *DatabaseService) ListSecrets(ctx context.Context, request *v1pb.ListSecretsRequest) (*v1pb.ListSecretsResponse, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	return &v1pb.ListSecretsResponse{
		Secrets: convertToV1Secrets(database.Metadata.GetSecrets(), database.InstanceID, database.DatabaseName),
	}, nil
}

// UpdateSecret updates a secret of a database.
func (s *DatabaseService) UpdateSecret(ctx context.Context, request *v1pb.UpdateSecretRequest) (*v1pb.Secret, error) {
	if request.Secret == nil {
		return nil, status.Errorf(codes.InvalidArgument, "secret is required")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	instanceID, databaseName, updateSecretName, err := common.GetInstanceDatabaseIDSecretName(request.Secret.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureEncryptedSecrets, instance); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	secretsMap := make(map[string]*storepb.Secret)
	for _, secret := range database.Metadata.GetSecrets() {
		secretsMap[secret.Name] = secret
	}

	newSecret := &storepb.Secret{}
	if _, ok := secretsMap[updateSecretName]; !ok {
		// If the secret is not existed and allow_missing is false, we will not create it.
		if !request.AllowMissing {
			return nil, status.Errorf(codes.NotFound, "secret %q not found", updateSecretName)
		}
		newSecret.Name = updateSecretName
		newSecret.Value = request.Secret.Value
		newSecret.Description = request.Secret.Description
	} else {
		oldSecret := secretsMap[updateSecretName]
		newSecret.Name = oldSecret.Name
		newSecret.Value = oldSecret.Value
		newSecret.Description = oldSecret.Description
		for _, path := range request.UpdateMask.Paths {
			switch path {
			case "value":
				newSecret.Value = request.Secret.Value
			case "name":
				// We don't allow users to update the name of a secret.
				return nil, status.Errorf(codes.InvalidArgument, "name of a secret is not allowed to be updated")
			case "description":
				newSecret.Description = request.Secret.Description
			}
		}
	}
	if err := isSecretValid(newSecret); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	secretsMap[updateSecretName] = newSecret

	var secrets []*storepb.Secret
	for _, secret := range secretsMap {
		secrets = append(secrets, secret)
	}

	metadata, ok := proto.Clone(database.Metadata).(*storepb.DatabaseMetadata)
	if !ok {
		return nil, errors.Errorf("failed to convert database metadata type")
	}
	metadata.Secrets = secrets
	if _, err := s.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		Metadata:     metadata,
	}); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return convertToV1Secret(newSecret, database.InstanceID, database.DatabaseName), nil
}

// DeleteSecret deletes a secret of a database.
func (s *DatabaseService) DeleteSecret(ctx context.Context, request *v1pb.DeleteSecretRequest) (*emptypb.Empty, error) {
	instanceID, databaseName, secretName, err := common.GetInstanceDatabaseIDSecretName(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureEncryptedSecrets, instance); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	secretsMap := make(map[string]*storepb.Secret)
	for _, secret := range database.Metadata.GetSecrets() {
		secretsMap[secret.Name] = secret
	}
	delete(secretsMap, secretName)
	var secrets []*storepb.Secret
	for _, secret := range secretsMap {
		secrets = append(secrets, secret)
	}

	metadata, ok := proto.Clone(database.Metadata).(*storepb.DatabaseMetadata)
	if !ok {
		return nil, errors.Errorf("failed to convert database metadata type")
	}
	metadata.Secrets = secrets
	if _, err := s.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		Metadata:     metadata,
	}); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

type totalValue struct {
	totalQueryTime time.Duration
	totalCount     int64
}

// ListSlowQueries lists the slow queries.
func (s *DatabaseService) ListSlowQueries(ctx context.Context, request *v1pb.ListSlowQueriesRequest) (*v1pb.ListSlowQueriesResponse, error) {
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	findDatabase := &store.FindDatabaseMessage{
		ProjectID: &projectID,
	}

	filters, err := ParseFilter(request.Filter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var startLogDate, endLogDate *time.Time
	for _, expr := range filters {
		switch expr.Key {
		case filterKeyEnvironment:
			reg := regexp.MustCompile(`^environments/(.+)`)
			match := reg.FindStringSubmatch(expr.Value)
			if len(match) != 2 {
				return nil, status.Errorf(codes.InvalidArgument, "invalid environment filter %q", expr.Value)
			}
			findDatabase.EffectiveEnvironmentID = &match[1]
		case filterKeyDatabase:
			instanceID, databaseName, err := common.GetInstanceDatabaseID(expr.Value)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			if instance == nil {
				return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
			}
			findDatabase.InstanceID = &instanceID
			findDatabase.DatabaseName = &databaseName
			findDatabase.IsCaseSensitive = store.IsObjectCaseSensitive(instance)
		case filterKeyStartTime:
			switch expr.Operator {
			case ComparatorTypeGreater:
				if startLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.Value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.Value)
				}
				t = t.AddDate(0, 0, 1).UTC()
				startLogDate = &t
			case ComparatorTypeGreaterEqual:
				if startLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.Value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.Value)
				}
				t = t.UTC()
				startLogDate = &t
			case ComparatorTypeLess:
				if endLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.Value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.Value)
				}
				t = t.UTC()
				endLogDate = &t
			case ComparatorTypeLessEqual:
				if endLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.Value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.Value)
				}
				t = t.AddDate(0, 0, 1).UTC()
				endLogDate = &t
			default:
				return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q %q %q", expr.Key, expr.Operator, expr.Value)
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter key %q", expr.Key)
		}
	}

	orderByKeys, err := parseOrderBy(request.OrderBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := validSlowQueryOrderByKey(orderByKeys); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	databases, err := s.store.ListDatabases(ctx, findDatabase)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find database list %q", err.Error())
	}

	result := &v1pb.ListSlowQueriesResponse{}
	instanceMap := make(map[string]*totalValue)

	for _, database := range databases {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &database.InstanceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find instance %q", err.Error())
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, "instance %q not found", database.InstanceID)
		}
		listSlowQuery := &store.ListSlowQueryMessage{
			InstanceID:   &database.InstanceID,
			DatabaseName: &database.DatabaseName,
			StartLogDate: startLogDate,
			EndLogDate:   endLogDate,
		}
		logs, err := s.store.ListSlowQuery(ctx, listSlowQuery)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find slow query %q", err.Error())
		}

		for _, log := range logs {
			result.SlowQueryLogs = append(result.SlowQueryLogs, convertToSlowQueryLog(database.InstanceID, database.DatabaseName, database.ProjectID, log))
			if value, exists := instanceMap[database.InstanceID]; exists {
				value.totalQueryTime += log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)
				value.totalCount += log.Statistics.Count
			} else {
				instanceMap[database.InstanceID] = &totalValue{
					totalQueryTime: log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count),
					totalCount:     log.Statistics.Count,
				}
			}
		}
	}

	for _, log := range result.SlowQueryLogs {
		instanceID, _, err := common.GetInstanceDatabaseID(log.Resource)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get instance id %q", err.Error())
		}
		totalQueryTime := log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)
		log.Statistics.QueryTimePercent = float64(totalQueryTime) / float64(instanceMap[instanceID].totalQueryTime)
		log.Statistics.CountPercent = float64(log.Statistics.Count) / float64(instanceMap[instanceID].totalCount)
	}

	result, err = sortSlowQueryLogResponse(result, orderByKeys)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sort slow query logs %q", err.Error())
	}

	return result, nil
}

func sortSlowQueryLogResponse(response *v1pb.ListSlowQueriesResponse, orderByKeys []orderByKey) (*v1pb.ListSlowQueriesResponse, error) {
	if len(orderByKeys) == 0 {
		orderByKeys = []orderByKey{
			{
				key:      orderByKeyAverageQueryTime,
				isAscend: false,
			},
		}
	}

	if err := validSlowQueryOrderByKey(orderByKeys); err != nil {
		return nil, err
	}

	sort.Slice(response.SlowQueryLogs, func(i, j int) bool {
		for _, key := range orderByKeys {
			switch key.key {
			case orderByKeyCount:
				lCount := response.SlowQueryLogs[i].Statistics.Count
				rCount := response.SlowQueryLogs[j].Statistics.Count
				if lCount != rCount {
					if key.isAscend {
						return lCount < rCount
					}
					return lCount > rCount
				}
			case orderByKeyLatestLogTime:
				lTime := response.SlowQueryLogs[i].Statistics.LatestLogTime.AsTime()
				rTime := response.SlowQueryLogs[j].Statistics.LatestLogTime.AsTime()
				if !lTime.Equal(rTime) {
					if key.isAscend {
						return lTime.Before(rTime)
					}
					return lTime.After(rTime)
				}
			case orderByKeyAverageQueryTime:
				lTime := response.SlowQueryLogs[i].Statistics.AverageQueryTime.AsDuration()
				rTime := response.SlowQueryLogs[j].Statistics.AverageQueryTime.AsDuration()
				if lTime != rTime {
					if key.isAscend {
						return lTime < rTime
					}
					return lTime > rTime
				}
			case orderByKeyMaximumQueryTime:
				lDuration := response.SlowQueryLogs[i].Statistics.MaximumQueryTime.AsDuration()
				rDuration := response.SlowQueryLogs[j].Statistics.MaximumQueryTime.AsDuration()
				if lDuration != rDuration {
					if key.isAscend {
						return lDuration < rDuration
					}
					return lDuration > rDuration
				}
			case orderByKeyAverageRowsSent:
				lSent := response.SlowQueryLogs[i].Statistics.AverageRowsSent
				rSent := response.SlowQueryLogs[j].Statistics.AverageRowsSent
				if lSent != rSent {
					if key.isAscend {
						return lSent < rSent
					}
					return lSent > rSent
				}
			case orderByKeyMaximumRowsSent:
				lSent := response.SlowQueryLogs[i].Statistics.MaximumRowsSent
				rSent := response.SlowQueryLogs[j].Statistics.MaximumRowsSent
				if lSent != rSent {
					if key.isAscend {
						return lSent < rSent
					}
					return lSent > rSent
				}
			case orderByKeyAverageRowsExamined:
				lExamined := response.SlowQueryLogs[i].Statistics.AverageRowsExamined
				rExamined := response.SlowQueryLogs[j].Statistics.AverageRowsExamined
				if lExamined != rExamined {
					if key.isAscend {
						return lExamined < rExamined
					}
					return lExamined > rExamined
				}
			case orderByKeyMaximumRowsExamined:
				lExamined := response.SlowQueryLogs[i].Statistics.MaximumRowsExamined
				rExamined := response.SlowQueryLogs[j].Statistics.MaximumRowsExamined
				if lExamined != rExamined {
					if key.isAscend {
						return lExamined < rExamined
					}
					return lExamined > rExamined
				}
			}
		}
		return false
	})

	return response, nil
}

func validSlowQueryOrderByKey(keys []orderByKey) error {
	for _, key := range keys {
		switch key.key {
		// Support order by count, latest_log_time, average_query_time, maximum_query_time,
		// average_rows_sent, maximum_rows_sent, average_rows_examined, maximum_rows_examined for now.
		case orderByKeyCount, orderByKeyLatestLogTime, orderByKeyAverageQueryTime, orderByKeyMaximumQueryTime,
			orderByKeyAverageRowsSent, orderByKeyMaximumRowsSent, orderByKeyAverageRowsExamined, orderByKeyMaximumRowsExamined:
		default:
			return errors.Errorf("invalid order_by key %q", key.key)
		}
	}
	return nil
}

func convertToSlowQueryLog(instanceID string, databaseName string, projectID string, log *v1pb.SlowQueryLog) *v1pb.SlowQueryLog {
	return &v1pb.SlowQueryLog{
		Resource:   fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, instanceID, common.DatabaseIDPrefix, databaseName),
		Project:    common.FormatProject(projectID),
		Statistics: log.Statistics,
	}
}

func (s *DatabaseService) convertToDatabase(ctx context.Context, database *store.DatabaseMessage) (*v1pb.Database, error) {
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &database.InstanceID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to find instance")
	}

	environment, effectiveEnvironment := "", ""
	if database.EnvironmentID != "" {
		environment = common.FormatEnvironment(database.EnvironmentID)
	}
	if database.EffectiveEnvironmentID != "" {
		effectiveEnvironment = common.FormatEnvironment(database.EffectiveEnvironmentID)
	}
	instanceResource, err := convertInstanceMessageToInstanceResource(instance)
	if err != nil {
		return nil, err
	}
	return &v1pb.Database{
		Name:                 common.FormatDatabase(database.InstanceID, database.DatabaseName),
		State:                convertDeletedToState(database.Deleted),
		SuccessfulSyncTime:   database.Metadata.GetLastSyncTime(),
		Project:              common.FormatProject(database.ProjectID),
		Environment:          environment,
		EffectiveEnvironment: effectiveEnvironment,
		SchemaVersion:        database.SchemaVersion,
		Labels:               database.Metadata.Labels,
		InstanceResource:     instanceResource,
		BackupAvailable:      database.Metadata.GetBackupAvailable(),
	}, nil
}

type metadataFilter struct {
	schema string
	table  string
}

func convertToV1Secrets(secrets []*storepb.Secret, instanceID, databaseName string) []*v1pb.Secret {
	var v1Secrets []*v1pb.Secret
	for _, secret := range secrets {
		v1Secrets = append(v1Secrets, convertToV1Secret(secret, instanceID, databaseName))
	}
	return v1Secrets
}

func convertToV1Secret(secret *storepb.Secret, instanceID, databaseName string) *v1pb.Secret {
	return &v1pb.Secret{
		Name:        fmt.Sprintf("%s%s/%s%s/%s%s", common.InstanceNamePrefix, instanceID, common.DatabaseIDPrefix, databaseName, common.SecretNamePrefix, secret.Name),
		Value:       "", /* stripped */
		Description: secret.Description,
	}
}
func isSecretValid(secret *storepb.Secret) error {
	// Names can not be empty.
	if secret.Name == "" {
		return errors.Errorf("invalid secret name: %s, name can not be empty", secret.Name)
	}
	// Values can not be empty.
	if secret.Value == "" {
		return errors.Errorf("the value of secret: %s can not be empty", secret.Name)
	}

	// Names must not start with the 'BYTEBASE_' prefix.
	bytebaseCaseInsensitivePrefixRegexp := regexp.MustCompile(`(?i)^BYTEBASE_`)
	if bytebaseCaseInsensitivePrefixRegexp.MatchString(secret.Name) {
		return errors.Errorf("invalid secret name: %s, name must not start with the 'BYTEBASE_' prefix", secret.Name)
	}
	// Names must not start with a number.
	if unicode.IsDigit(rune(secret.Name[0])) {
		return errors.Errorf("invalid secret name: %s, name must not start with a number", secret.Name)
	}

	// Names can only contain alphanumeric characters ([A-Z], [0-9]) or underscores (_). Spaces are not allowed.
	for _, c := range secret.Name {
		if !isUpperCaseLetter(c) && !unicode.IsDigit(c) && c != '_' {
			return errors.Errorf("invalid secret name: %s, expect [A-Z], [0-9], '_', but meet: %v", secret.Name, c)
		}
	}
	return nil
}

func isUpperCaseLetter(c rune) bool {
	return 'A' <= c && c <= 'Z'
}

// AdviseIndex advises the index of a table.
func (s *DatabaseService) AdviseIndex(ctx context.Context, request *v1pb.AdviseIndexRequest) (*v1pb.AdviseIndexResponse, error) {
	if err := s.licenseService.IsFeatureEnabled(api.FeatureAIAssistant); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	findDatabase := &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	}
	database, err := s.store.GetDatabaseV2(ctx, findDatabase)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get database: %v", err)
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	switch instance.Metadata.GetEngine() {
	case storepb.Engine_POSTGRES:
		return s.pgAdviseIndex(ctx, request, database)
	case storepb.Engine_MYSQL:
		return s.mysqlAdviseIndex(ctx, request, instance, database)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "AdviseIndex is not implemented for engine: %v", instance.Metadata.GetEngine())
	}
}

func (s *DatabaseService) mysqlAdviseIndex(ctx context.Context, request *v1pb.AdviseIndexRequest, instance *store.InstanceMessage, database *store.DatabaseMessage) (*v1pb.AdviseIndexResponse, error) {
	key, endpoint, modelName, err := s.getOpenAISetting((ctx))
	if err != nil {
		return nil, err
	}

	var schemas []*model.DatabaseSchema
	// Deal with the cross database query.
	resources, err := base.ExtractResourceList(instance.Metadata.GetEngine(), database.DatabaseName, "", request.Statement)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to extract resource list: %v", err)
	}
	databaseMap := make(map[string]bool)
	for _, resource := range resources {
		databaseMap[resource.Database] = true
	}
	var databases []string
	for database := range databaseMap {
		databases = append(databases, database)
	}
	if len(databases) == 0 {
		databases = append(databases, database.DatabaseName)
	}

	for _, db := range databases {
		findDatabase := &store.FindDatabaseMessage{
			InstanceID:      &instance.ResourceID,
			DatabaseName:    &db,
			IsCaseSensitive: store.IsObjectCaseSensitive(instance),
		}
		database, err := s.store.GetDatabaseV2(ctx, findDatabase)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get database: %v", err)
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, "database %q not found", db)
		}
		schema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get database schema: %v", err)
		}
		schemas = append(schemas, schema)
	}

	var compactBuf bytes.Buffer
	for _, schema := range schemas {
		compactSchema, err := schema.CompactText()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to compact database schema: %v", err)
		}
		if _, err := compactBuf.WriteString(compactSchema); err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to write compact database schema: %v", err)
		}
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: `You are a MySQL index advisor. You answer the question about the index of tables and SQLs. DO NOT EXPLAIN THE ANSWER.`,
		},
		{
			Role: openai.ChatMessageRoleUser,
			Content: `You are an assistant who works as a Magic: The strict MySQL index advisor. Analyze the SQL with schema and existing indexes, then give the advice in the JSON format.
			If the SQL will use the existing index, the current_index field is the index name with database name and table name. Otherwise, the current_index field is "N/A".
			If it is possible to create a new index to speed up the query, the create_index_statement field is the SQL statement to create the index. Otherwise, the create_index_statement field is empty string.
			YOUR ADVICE MUST FOLLOW JSON FORMAT. DO NOT EXPLAIN THE ADVICE.
			Here two examples:
			{"current_index": "index_schema_table_age ON db1.schema_table", "create_index_statement":""}
			{"current_index": "N/A", "create_index_statement":"CREATE INDEX ON db1.schema_table(collected_at, schema_index_id)"}
			` + fmt.Sprintf(`### MySQL schema:\n### %s\n###The SQL is:\n### %s###`, compactBuf.String(), request.Statement),
		},
	}

	generateFunc := func(resp *v1pb.AdviseIndexResponse) error {
		// Generate current index.
		if resp.CurrentIndex != "N/A" {
			// Use regex to extract the index name, database name and table name from "index_schema_table_age ON public.schema_table".
			reg := regexp.MustCompile(`(?i)(.*) ON (.*)\.(.*)`)
			matches := reg.FindStringSubmatch(resp.CurrentIndex)
			if len(matches) != 4 {
				return errors.Errorf("failed to extract index name, database name and table name from %s", resp.CurrentIndex)
			}
			var dbSchema *model.DatabaseSchema
			for _, schema := range schemas {
				if schema.GetMetadata().Name == matches[2] {
					dbSchema = schema
					break
				}
			}
			if dbSchema == nil {
				return errors.Errorf("database %s doesn't exist", matches[2])
			}
			tableMetadata := dbSchema.GetDatabaseMetadata().GetSchema("").GetTable(matches[3])
			if tableMetadata == nil {
				return errors.Errorf("table %s doesn't exist", matches[3])
			}
			indexMetadata := tableMetadata.GetIndex(matches[1])
			if indexMetadata == nil {
				return errors.Errorf("index %s doesn't exist", resp.CurrentIndex)
			}
			indexProto := indexMetadata.GetProto()
			resp.CurrentIndex = fmt.Sprintf("USING %s (%s)", indexProto.Type, strings.Join(indexProto.Expressions, ", "))
		} else {
			resp.CurrentIndex = "No usable index"
		}

		// Generate suggestion and create index statement.
		if resp.CreateIndexStatement != "" {
			p := tidbparser.New()
			node, err := p.ParseOneStmt(resp.CreateIndexStatement, "", "")
			if err != nil {
				return errors.Errorf("failed to parse create index statement: %v", err)
			}
			switch createIndex := node.(type) {
			case *tidbast.CreateIndexStmt:
				defineString, err := mysqlIndexExpressionList(createIndex)
				if err != nil {
					return errors.Errorf("failed to generate create index statement: %v", err)
				}
				indexType := createIndex.IndexOption.Tp.String()
				if indexType == "" {
					indexType = "BTREE"
				}
				resp.Suggestion = fmt.Sprintf("USING %s (%s)", indexType, defineString)
			default:
				return errors.Errorf("expect create index statement, but got %T", node)
			}
		} else {
			resp.Suggestion = "N/A"
		}

		return nil
	}

	result, err := getOpenAIResponse(ctx, messages, key, endpoint, modelName, generateFunc)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func mysqlIndexExpressionList(node *tidbast.CreateIndexStmt) (string, error) {
	var buf bytes.Buffer
	for i, item := range node.IndexPartSpecifications {
		text, err := restoreNode(item)
		if err != nil {
			return "", err
		}
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return "", err
			}
		}
		if _, err := buf.WriteString(text); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func restoreNode(node tidbast.Node) (string, error) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (s *DatabaseService) pgAdviseIndex(ctx context.Context, request *v1pb.AdviseIndexRequest, database *store.DatabaseMessage) (*v1pb.AdviseIndexResponse, error) {
	key, endpoint, modelName, err := s.getOpenAISetting((ctx))
	if err != nil {
		return nil, err
	}
	schema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get database schema: %v", err)
	}
	compactSchema, err := schema.CompactText()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to compact database schema: %v", err)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: `You are a PostgreSQL index advisor. You answer the question about the index of tables and SQLs. DO NOT EXPLAIN THE ANSWER.`,
		},
		{
			Role: openai.ChatMessageRoleUser,
			Content: `You are an assistant who works as a Magic: The strict PostgreSQL index advisor. Analyze the SQL with schema and existing indexes, then give the advice in the JSON format.
			If the SQL will use the existing index, the current_index field is the index name with schema name and table name. Otherwise, the current_index field is "N/A".
			If it is possible to create a new index to speed up the query, the create_index_statement field is the SQL statement to create the index. Otherwise, the create_index_statement field is empty string.
			YOUR ADVICE MUST FOLLOW JSON FORMAT. DO NOT EXPLAIN THE ADVICE.
			Here two examples:
			{"current_index": "index_schema_table_age ON public.schema_table", "create_index_statement":""}
			{"current_index": "N/A", "create_index_statement":"CREATE INDEX ON public.schema_table(collected_at, schema_index_id)"}
			` + fmt.Sprintf(`### Postgres schema:\n### %s\n###The SQL is:\n### %s###`, compactSchema, request.Statement),
		},
	}

	generateFunc := func(resp *v1pb.AdviseIndexResponse) error {
		// Generate current index.
		if resp.CurrentIndex != "N/A" {
			// Use regex to extract the index name, schema name and table name from "index_schema_table_age ON public.schema_table".
			reg := regexp.MustCompile(`(?i)(.*) ON (.*)\.(.*)`)
			matches := reg.FindStringSubmatch(resp.CurrentIndex)
			if len(matches) != 4 {
				return errors.Errorf("failed to extract index name, schema name and table name from %s", resp.CurrentIndex)
			}
			schemaMetadata := schema.GetDatabaseMetadata().GetSchema(matches[2])
			if schemaMetadata == nil {
				return errors.Errorf("schema %s doesn't exist", matches[2])
			}
			tableMetadata := schemaMetadata.GetTable(matches[3])
			if tableMetadata == nil {
				return errors.Errorf("table %s doesn't exist", matches[3])
			}
			indexMetadata := tableMetadata.GetIndex(matches[1])
			if indexMetadata == nil {
				return errors.Errorf("index %s doesn't exist", resp.CurrentIndex)
			}
			indexProto := indexMetadata.GetProto()
			resp.CurrentIndex = fmt.Sprintf("USING %s (%s)", indexProto.Type, strings.Join(indexProto.Expressions, ", "))
		} else {
			resp.CurrentIndex = "No usable index"
		}

		// Generate suggestion and create index statement.
		if resp.CreateIndexStatement != "" {
			nodes, err := pgrawparser.Parse(pgrawparser.ParseContext{}, resp.CreateIndexStatement)
			if err != nil {
				return errors.Errorf("failed to parse create index statement: %v", err)
			}
			if len(nodes) != 1 {
				return errors.Errorf("expect 1 statement, but got %d", len(nodes))
			}
			switch node := nodes[0].(type) {
			case *ast.CreateIndexStmt:
				resp.Suggestion = fmt.Sprintf("USING %s (%s)", node.Index.Method, strings.Join(node.Index.GetKeyNameList(), ", "))
			default:
				return errors.Errorf("expect CreateIndexStmt, but got %T", node)
			}
		} else {
			resp.Suggestion = "N/A"
		}

		return nil
	}

	result, err := getOpenAIResponse(ctx, messages, key, endpoint, modelName, generateFunc)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *DatabaseService) getOpenAISetting(ctx context.Context) (string, string, string, error) {
	key, err := s.store.GetSettingV2(ctx, api.SettingPluginOpenAIKey)
	if err != nil {
		return "", "", "", status.Errorf(codes.Internal, "Failed to get setting: %v", err)
	}
	if key.Value == "" {
		return "", "", "", status.Errorf(codes.FailedPrecondition, "OpenAI key is not set")
	}
	endpointSetting, err := s.store.GetSettingV2(ctx, api.SettingPluginOpenAIEndpoint)
	if err != nil {
		return "", "", "", status.Errorf(codes.Internal, "Failed to get setting: %v", err)
	}
	var endpoint string
	if endpointSetting != nil {
		endpoint = endpointSetting.Value
	}
	model, err := s.store.GetSettingV2(ctx, api.SettingPluginOpenAIModel)
	if err != nil {
		return "", "", "", status.Errorf(codes.Internal, "Failed to get setting: %v", err)
	}
	return key.Value, endpoint, model.Value, nil
}

func getOpenAIResponse(ctx context.Context, messages []openai.ChatCompletionMessage, key, endpoint, model string, generateResponse func(*v1pb.AdviseIndexResponse) error) (*v1pb.AdviseIndexResponse, error) {
	var result v1pb.AdviseIndexResponse
	successful := false
	var retErr error
	if model == "" {
		model = openai.GPT3Dot5Turbo
	}
	// Retry 5 times if failed.
	for i := 0; i < 5; i++ {
		cfg := openai.DefaultConfig(key)
		if endpoint != "" {
			cfg.BaseURL = endpoint
		}
		client := openai.NewClientWithConfig(cfg)
		resp, err := client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model:            model,
				Messages:         messages,
				Temperature:      0,
				Stop:             []string{"#", ";"},
				TopP:             1.0,
				FrequencyPenalty: 0.0,
				PresencePenalty:  0.0,
			},
		)
		if err != nil {
			retErr = errors.Wrap(err, "failed to create chat completion")
			continue
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
			retErr = errors.Wrapf(err, "failed to unmarshal chat completion response content: %s", resp.Choices[0].Message.Content)
			continue
		}
		if err = generateResponse(&result); err != nil {
			retErr = errors.Wrap(err, "failed to generate response")
			continue
		}
		successful = true
		break
	}

	if !successful {
		return nil, status.Errorf(codes.Internal, "Failed to get index advice, error %v", retErr)
	}
	return &result, nil
}
