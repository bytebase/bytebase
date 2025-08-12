package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/secret"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

// InstanceService implements the instance service.
type InstanceService struct {
	v1connect.UnimplementedInstanceServiceHandler
	store          *store.Store
	licenseService *enterprise.LicenseService
	metricReporter *metricreport.Reporter
	stateCfg       *state.State
	dbFactory      *dbfactory.DBFactory
	schemaSyncer   *schemasync.Syncer
	iamManager     *iam.Manager
}

// NewInstanceService creates a new InstanceService.
func NewInstanceService(store *store.Store, licenseService *enterprise.LicenseService, metricReporter *metricreport.Reporter, stateCfg *state.State, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer, iamManager *iam.Manager) *InstanceService {
	return &InstanceService{
		store:          store,
		licenseService: licenseService,
		metricReporter: metricReporter,
		stateCfg:       stateCfg,
		dbFactory:      dbFactory,
		schemaSyncer:   schemaSyncer,
		iamManager:     iamManager,
	}
}

// GetInstance gets an instance.
func (s *InstanceService) GetInstance(ctx context.Context, req *connect.Request[v1pb.GetInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

func parseListInstanceFilter(filter string) (*store.ListResourceFilter, error) {
	if filter == "" {
		return nil, nil
	}
	e, err := cel.NewEnv()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	parseToSQL := func(variable, value any) (string, error) {
		switch variable {
		case "name":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("instance.metadata->>'title' = $%d", len(positionalArgs)), nil
		case "resource_id":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("instance.resource_id = $%d", len(positionalArgs)), nil
		case "environment":
			environmentID, err := common.GetEnvironmentID(value.(string))
			if err != nil {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid environment filter %q", value))
			}
			positionalArgs = append(positionalArgs, environmentID)
			return fmt.Sprintf("instance.environment = $%d", len(positionalArgs)), nil
		case "state":
			v1State, ok := v1pb.State_value[value.(string)]
			if !ok {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid state filter %q", value))
			}
			positionalArgs = append(positionalArgs, v1pb.State(v1State) == v1pb.State_DELETED)
			return fmt.Sprintf("instance.deleted = $%d", len(positionalArgs)), nil
		case "engine":
			v1Engine, ok := v1pb.Engine_value[value.(string)]
			if !ok {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid engine filter %q", value))
			}
			engine := convertEngine(v1pb.Engine(v1Engine))
			positionalArgs = append(positionalArgs, engine)
			return fmt.Sprintf("instance.metadata->>'engine' = $%d", len(positionalArgs)), nil
		case "host":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("ds ->> 'host' = $%d", len(positionalArgs)), nil
		case "port":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("ds ->> 'port' = $%d", len(positionalArgs)), nil
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid project filter %q", value))
			}
			positionalArgs = append(positionalArgs, projectID)
			return fmt.Sprintf("db.project = $%d", len(positionalArgs)), nil
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
				if strValue == "" {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`empty value for %q`, variable))
				}

				switch variable {
				case "name":
					return "LOWER(instance.metadata->>'title') LIKE '%" + strings.ToLower(strValue) + "%'", nil
				case "resource_id":
					return "LOWER(instance.resource_id) LIKE '%" + strings.ToLower(strValue) + "%'", nil
				case "host", "port":
					return "ds ->> '" + variable + "' LIKE '%" + strValue + "%'", nil
				default:
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
				}
			case celoperators.In:
				return parseToEngineSQL(expr, "IN")
			case celoperators.LogicalNot:
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.New(`only support !(engine in ["{engine1}", "{engine2}"]) format`))
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

// ListInstances lists all instances.
func (s *InstanceService) ListInstances(ctx context.Context, req *connect.Request[v1pb.ListInstancesRequest]) (*connect.Response[v1pb.ListInstancesResponse], error) {
	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindInstanceMessage{
		ShowDeleted: req.Msg.ShowDeleted,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
	}
	filter, err := parseListInstanceFilter(req.Msg.Filter)
	if err != nil {
		return nil, err
	}
	find.Filter = filter
	instances, err := s.store.ListInstancesV2(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	nextPageToken := ""
	if len(instances) == limitPlusOne {
		instances = instances[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	response := &v1pb.ListInstancesResponse{
		NextPageToken: nextPageToken,
	}
	for _, instance := range instances {
		ins := convertInstanceMessage(instance)
		response.Instances = append(response.Instances, ins)
	}
	return connect.NewResponse(response), nil
}

// ListInstanceDatabase list all databases in the instance.
func (s *InstanceService) ListInstanceDatabase(ctx context.Context, req *connect.Request[v1pb.ListInstanceDatabaseRequest]) (*connect.Response[v1pb.ListInstanceDatabaseResponse], error) {
	var instanceMessage *store.InstanceMessage

	if req.Msg.Instance != nil {
		instanceID, err := common.GetInstanceID(req.Msg.Name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		if instanceMessage, err = convertInstanceToInstanceMessage(instanceID, req.Msg.Instance); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else {
		instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		instanceMessage = instance
	}

	instanceMeta, err := s.schemaSyncer.GetInstanceMeta(ctx, instanceMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1pb.ListInstanceDatabaseResponse{}
	for _, database := range instanceMeta.Databases {
		response.Databases = append(response.Databases, database.Name)
	}
	return connect.NewResponse(response), nil
}

// CreateInstance creates an instance.
func (s *InstanceService) CreateInstance(ctx context.Context, req *connect.Request[v1pb.CreateInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.Instance == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance must be set"))
	}
	if !isValidResourceID(req.Msg.InstanceId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid instance ID %v", req.Msg.InstanceId))
	}

	if err := s.instanceCountGuard(ctx); err != nil {
		return nil, err
	}

	instanceMessage, err := convertInstanceToInstanceMessage(req.Msg.InstanceId, req.Msg.Instance)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Test connection.
	if req.Msg.ValidateOnly {
		for _, ds := range instanceMessage.Metadata.GetDataSources() {
			err := func() error {
				driver, err := s.dbFactory.GetDataSourceDriver(
					ctx, instanceMessage, ds,
					db.ConnectionContext{
						ReadOnly: ds.GetType() == storepb.DataSourceType_READ_ONLY,
					},
				)
				if err != nil {
					return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database driver"))
				}
				defer driver.Close(ctx)
				if err := driver.Ping(ctx); err != nil {
					return connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid datasource %s", ds.GetType()))
				}
				return nil
			}()
			if err != nil {
				return nil, err
			}
		}

		result := convertInstanceMessage(instanceMessage)
		return connect.NewResponse(result), nil
	}

	activatedInstanceLimit := s.licenseService.GetActivatedInstanceLimit(ctx)
	if instanceMessage.Metadata.GetActivation() {
		count, err := s.store.GetActivatedInstanceCount(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if count >= activatedInstanceLimit {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.Errorf(instanceExceededError, activatedInstanceLimit))
		}
	}

	if err := s.checkInstanceDataSources(instanceMessage, instanceMessage.Metadata.GetDataSources()); err != nil {
		return nil, err
	}

	instance, err := s.store.CreateInstanceV2(ctx, instanceMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
	if err == nil {
		defer driver.Close(ctx)
		updatedInstance, _, _, err := s.schemaSyncer.SyncInstance(ctx, instance)
		if err != nil {
			slog.Warn("Failed to sync instance",
				slog.String("instance", instance.ResourceID),
				log.BBError(err))
		} else {
			instance = updatedInstance
		}
		// Sync all databases in the instance asynchronously.
		s.schemaSyncer.SyncAllDatabases(ctx, instance)
	}

	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricapi.InstanceCreateMetricName,
		Value: 1,
		Labels: map[string]any{
			"engine": instance.Metadata.GetEngine(),
		},
	})

	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

func (s *InstanceService) checkInstanceDataSources(instance *store.InstanceMessage, dataSources []*storepb.DataSource) error {
	dsIDMap := map[string]bool{}
	for _, ds := range dataSources {
		if err := s.checkDataSource(instance, ds); err != nil {
			return err
		}
		if dsIDMap[ds.GetId()] {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`duplicate data source id "%s"`, ds.GetId()))
		}
		dsIDMap[ds.GetId()] = true
	}

	return nil
}

const instanceExceededError = "activation instance count has reached the limit (%v)"

func (s *InstanceService) checkDataSource(instance *store.InstanceMessage, dataSource *storepb.DataSource) error {
	if dataSource.GetId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("data source id is required"))
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_EXTERNAL_SECRET_MANAGER, instance); err != nil {
		missingFeatureError := connect.NewError(connect.CodePermissionDenied, err)
		if dataSource.GetExternalSecret() != nil {
			return missingFeatureError
		}
		if ok, _ := secret.GetExternalSecretURL(dataSource.GetPassword()); !ok {
			return nil
		}
		return missingFeatureError
	}

	return nil
}

// UpdateInstance updates an instance.
func (s *InstanceService) UpdateInstance(ctx context.Context, req *connect.Request[v1pb.UpdateInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.Instance == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance must be set"))
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask must be set"))
	}

	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Instance.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Instance.Name))
	}

	metadata := proto.CloneOf(instance.Metadata)
	patch := &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
		Metadata:   metadata,
	}
	updateActivation := false
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Metadata.Title = req.Msg.Instance.Title
		case "environment":
			if req.Msg.Instance.Environment == nil || *req.Msg.Instance.Environment == "" {
				// Clear the environment if null or empty string is provided
				emptyStr := ""
				patch.EnvironmentID = &emptyStr
			} else {
				envID, err := common.GetEnvironmentID(*req.Msg.Instance.Environment)
				if err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, err)
				}
				environment, err := s.store.GetEnvironmentByID(ctx, envID)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, err)
				}
				if environment == nil {
					return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("environment %q not found", envID))
				}
				patch.EnvironmentID = &envID
			}
		case "external_link":
			patch.Metadata.ExternalLink = req.Msg.Instance.ExternalLink
		case "data_sources":
			dataSources, err := convertV1DataSources(req.Msg.Instance.DataSources)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			if err := s.checkInstanceDataSources(instance, dataSources); err != nil {
				return nil, err
			}
			patch.Metadata.DataSources = dataSources
		case "activation":
			if !instance.Metadata.GetActivation() && req.Msg.Instance.Activation {
				updateActivation = true
			}
			patch.Metadata.Activation = req.Msg.Instance.Activation
		case "sync_interval":
			if err := s.licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_CUSTOM_INSTANCE_SYNC_TIME, instance); err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}
			patch.Metadata.SyncInterval = req.Msg.Instance.SyncInterval
		case "maximum_connections":
			if err := s.licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_CUSTOM_INSTANCE_CONNECTION_LIMIT, instance); err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}
			patch.Metadata.MaximumConnections = req.Msg.Instance.MaximumConnections
		case "sync_databases":
			patch.Metadata.SyncDatabases = req.Msg.Instance.SyncDatabases
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`unsupported update_mask "%s"`, path))
		}
	}

	activatedInstanceLimit := s.licenseService.GetActivatedInstanceLimit(ctx)
	if updateActivation {
		count, err := s.store.GetActivatedInstanceCount(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if count >= activatedInstanceLimit {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.Errorf(instanceExceededError, activatedInstanceLimit))
		}
	}

	ins, err := s.store.UpdateInstanceV2(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result := convertInstanceMessage(ins)
	return connect.NewResponse(result), nil
}

// DeleteInstance deletes an instance.
func (s *InstanceService) DeleteInstance(ctx context.Context, req *connect.Request[v1pb.DeleteInstanceRequest]) (*connect.Response[emptypb.Empty], error) {
	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
	if err != nil {
		return nil, err
	}
	if req.Msg.Force {
		if len(databases) > 0 {
			defaultProjectID := common.DefaultProjectID
			if _, err := s.store.BatchUpdateDatabases(ctx, databases, &store.BatchUpdateDatabases{ProjectID: &defaultProjectID}); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
	} else {
		var databaseNames []string
		for _, database := range databases {
			if database.ProjectID != common.DefaultProjectID {
				databaseNames = append(databaseNames, database.DatabaseName)
			}
		}
		if len(databaseNames) > 0 {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("all databases should be transferred to the unassigned project before deleting the instance"))
		}
	}

	metadata := proto.CloneOf(instance.Metadata)
	metadata.Activation = false
	if _, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
		Deleted:    &deletePatch,
		Metadata:   metadata,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// UndeleteInstance undeletes an instance.
func (s *InstanceService) UndeleteInstance(ctx context.Context, req *connect.Request[v1pb.UndeleteInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if !instance.Deleted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("instance %q is active", req.Msg.Name))
	}
	if err := s.instanceCountGuard(ctx); err != nil {
		return nil, err
	}

	ins, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
		Deleted:    &undeletePatch,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := convertInstanceMessage(ins)
	return connect.NewResponse(result), nil
}

// SyncInstance syncs the instance.
func (s *InstanceService) SyncInstance(ctx context.Context, req *connect.Request[v1pb.SyncInstanceRequest]) (*connect.Response[v1pb.SyncInstanceResponse], error) {
	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}

	updatedInstance, allDatabases, newDatabases, err := s.schemaSyncer.SyncInstance(ctx, instance)
	if err != nil {
		return nil, err
	}
	if req.Msg.EnableFullSync {
		// Sync all databases in the instance asynchronously.
		s.schemaSyncer.SyncAllDatabases(ctx, updatedInstance)
	} else {
		s.schemaSyncer.SyncDatabasesAsync(newDatabases)
	}

	response := &v1pb.SyncInstanceResponse{}
	for _, database := range allDatabases {
		response.Databases = append(response.Databases, database.Name)
	}
	return connect.NewResponse(response), nil
}

// BatchSyncInstances syncs multiple instances.
func (s *InstanceService) BatchSyncInstances(ctx context.Context, req *connect.Request[v1pb.BatchSyncInstancesRequest]) (*connect.Response[v1pb.BatchSyncInstancesResponse], error) {
	for _, r := range req.Msg.Requests {
		instance, err := getInstanceMessage(ctx, s.store, r.Name)
		if err != nil {
			return nil, err
		}
		if instance.Deleted {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", r.Name))
		}

		updatedInstance, _, newDatabases, err := s.schemaSyncer.SyncInstance(ctx, instance)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to sync instance"))
		}
		if r.EnableFullSync {
			// Sync all databases in the instance asynchronously.
			s.schemaSyncer.SyncAllDatabases(ctx, updatedInstance)
		} else {
			s.schemaSyncer.SyncDatabasesAsync(newDatabases)
		}
	}

	return connect.NewResponse(&v1pb.BatchSyncInstancesResponse{}), nil
}

// BatchUpdateInstances update multiple instances.
func (s *InstanceService) BatchUpdateInstances(ctx context.Context, req *connect.Request[v1pb.BatchUpdateInstancesRequest]) (*connect.Response[v1pb.BatchUpdateInstancesResponse], error) {
	response := &v1pb.BatchUpdateInstancesResponse{}
	for _, updateReq := range req.Msg.GetRequests() {
		updated, err := s.UpdateInstance(ctx, connect.NewRequest(updateReq))
		if err != nil {
			return nil, err
		}
		response.Instances = append(response.Instances, updated.Msg)
	}
	return connect.NewResponse(response), nil
}

// AddDataSource adds a data source to an instance.
func (s *InstanceService) AddDataSource(ctx context.Context, req *connect.Request[v1pb.AddDataSourceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.DataSource == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("data sources is required"))
	}
	// We only support add RO type datasouce to instance now, see more details in instance_service.proto.
	if req.Msg.DataSource.Type != v1pb.DataSourceType_READ_ONLY {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only support adding read-only data source"))
	}

	dataSource, err := convertV1DataSource(req.Msg.DataSource)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("failed to convert data source"))
	}

	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}
	for _, ds := range instance.Metadata.GetDataSources() {
		if ds.GetId() == req.Msg.DataSource.Id {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("data source already exists with the same name"))
		}
	}
	if err := s.checkDataSource(instance, dataSource); err != nil {
		return nil, err
	}

	// Test connection.
	if req.Msg.ValidateOnly {
		err := func() error {
			driver, err := s.dbFactory.GetDataSourceDriver(
				ctx, instance, dataSource,
				db.ConnectionContext{
					ReadOnly: dataSource.GetType() == storepb.DataSourceType_READ_ONLY,
				},
			)
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database driver"))
			}
			defer driver.Close(ctx)
			if err := driver.Ping(ctx); err != nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid datasource %s", dataSource.GetType()))
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
		result := convertInstanceMessage(instance)
		return connect.NewResponse(result), nil
	}

	if dataSource.GetType() != storepb.DataSourceType_READ_ONLY {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only read-only data source can be added"))
	}
	if err := s.licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_INSTANCE_READ_ONLY_CONNECTION, instance); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	metadata := proto.CloneOf(instance.Metadata)
	metadata.DataSources = append(metadata.DataSources, dataSource)
	instance, err = s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{ResourceID: instance.ResourceID, Metadata: metadata})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

// UpdateDataSource updates a data source of an instance.
func (s *InstanceService) UpdateDataSource(ctx context.Context, req *connect.Request[v1pb.UpdateDataSourceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.DataSource == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("datasource is required"))
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask must be set"))
	}

	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}
	metadata := proto.CloneOf(instance.Metadata)
	var dataSource *storepb.DataSource
	for _, ds := range metadata.GetDataSources() {
		if ds.GetId() == req.Msg.DataSource.Id {
			dataSource = ds
			break
		}
	}
	if dataSource == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf(`cannot found data source "%s"`, req.Msg.DataSource.Id))
	}

	if dataSource.GetType() == storepb.DataSourceType_READ_ONLY {
		if err := s.licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_INSTANCE_READ_ONLY_CONNECTION, instance); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
	}

	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "username":
			dataSource.Username = req.Msg.DataSource.Username
		case "password":
			dataSource.Password = req.Msg.DataSource.Password
		case "ssl_ca":
			dataSource.SslCa = req.Msg.DataSource.SslCa
		case "ssl_cert":
			dataSource.SslCert = req.Msg.DataSource.SslCert
		case "ssl_key":
			dataSource.SslKey = req.Msg.DataSource.SslKey
		case "host":
			dataSource.Host = req.Msg.DataSource.Host
		case "port":
			dataSource.Port = req.Msg.DataSource.Port
		case "database":
			dataSource.Database = req.Msg.DataSource.Database
		case "srv":
			dataSource.Srv = req.Msg.DataSource.Srv
		case "authentication_database":
			dataSource.AuthenticationDatabase = req.Msg.DataSource.AuthenticationDatabase
		case "sid":
			dataSource.Sid = req.Msg.DataSource.Sid
		case "service_name":
			dataSource.ServiceName = req.Msg.DataSource.ServiceName
		case "ssh_host":
			dataSource.SshHost = req.Msg.DataSource.SshHost
		case "ssh_port":
			dataSource.SshPort = req.Msg.DataSource.SshPort
		case "ssh_user":
			dataSource.SshUser = req.Msg.DataSource.SshUser
		case "ssh_password":
			dataSource.SshPassword = req.Msg.DataSource.SshPassword
		case "ssh_private_key":
			dataSource.SshPrivateKey = req.Msg.DataSource.SshPrivateKey
		case "authentication_private_key":
			dataSource.AuthenticationPrivateKey = req.Msg.DataSource.AuthenticationPrivateKey
		case "external_secret":
			externalSecret, err := convertV1DataSourceExternalSecret(req.Msg.DataSource.ExternalSecret)
			if err != nil {
				return nil, err
			}
			dataSource.ExternalSecret = externalSecret
		case "sasl_config":
			dataSource.SaslConfig = convertV1DataSourceSaslConfig(req.Msg.DataSource.SaslConfig)
		case "authentication_type":
			dataSource.AuthenticationType = convertV1AuthenticationType(req.Msg.DataSource.AuthenticationType)
		case "additional_addresses":
			dataSource.AdditionalAddresses = convertAdditionalAddresses(req.Msg.DataSource.AdditionalAddresses)
		case "replica_set":
			dataSource.ReplicaSet = req.Msg.DataSource.ReplicaSet
		case "direct_connection":
			dataSource.DirectConnection = req.Msg.DataSource.DirectConnection
		case "region":
			dataSource.Region = req.Msg.DataSource.Region
		case "warehouse_id":
			dataSource.WarehouseId = req.Msg.DataSource.WarehouseId
		case "use_ssl":
			dataSource.UseSsl = req.Msg.DataSource.UseSsl
		case "redis_type":
			dataSource.RedisType = convertV1RedisType(req.Msg.DataSource.RedisType)
		case "master_name":
			dataSource.MasterName = req.Msg.DataSource.MasterName
		case "master_username":
			dataSource.MasterUsername = req.Msg.DataSource.MasterUsername
		case "master_password":
			dataSource.MasterPassword = req.Msg.DataSource.MasterPassword
		case "extra_connection_parameters":
			dataSource.ExtraConnectionParameters = req.Msg.DataSource.ExtraConnectionParameters
		case "azure_credential", "aws_credential", "gcp_credential":
			switch req.Msg.DataSource.AuthenticationType {
			case v1pb.DataSource_AZURE_IAM:
				if azureCredential := req.Msg.DataSource.GetAzureCredential(); azureCredential != nil {
					dataSource.IamExtension = &storepb.DataSource_AzureCredential_{
						AzureCredential: &storepb.DataSource_AzureCredential{
							TenantId:     azureCredential.TenantId,
							ClientId:     azureCredential.ClientId,
							ClientSecret: azureCredential.ClientSecret,
						},
					}
				} else {
					dataSource.IamExtension = nil
				}
			case v1pb.DataSource_AWS_RDS_IAM:
				if awsCredential := req.Msg.DataSource.GetAwsCredential(); awsCredential != nil {
					dataSource.IamExtension = &storepb.DataSource_AwsCredential{
						AwsCredential: &storepb.DataSource_AWSCredential{
							AccessKeyId:     awsCredential.AccessKeyId,
							SecretAccessKey: awsCredential.SecretAccessKey,
							SessionToken:    awsCredential.SessionToken,
						},
					}
				} else {
					dataSource.IamExtension = nil
				}
			case v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM:
				if gcpCredential := req.Msg.DataSource.GetGcpCredential(); gcpCredential != nil {
					dataSource.IamExtension = &storepb.DataSource_GcpCredential{
						GcpCredential: &storepb.DataSource_GCPCredential{
							Content: gcpCredential.Content,
						},
					}
				} else {
					dataSource.IamExtension = nil
				}
			default:
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`unsupported update_mask "%s"`, path))
		}
	}

	if err := s.checkDataSource(instance, dataSource); err != nil {
		return nil, err
	}

	// Test connection.
	if req.Msg.ValidateOnly {
		err := func() error {
			driver, err := s.dbFactory.GetDataSourceDriver(
				ctx, instance, dataSource,
				db.ConnectionContext{ReadOnly: dataSource.GetType() == storepb.DataSourceType_READ_ONLY},
			)
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database driver"))
			}
			defer driver.Close(ctx)
			if err := driver.Ping(ctx); err != nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid datasource %s", dataSource.GetType()))
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
		result := convertInstanceMessage(instance)
		return connect.NewResponse(result), nil
	}

	instance, err = s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{ResourceID: instance.ResourceID, Metadata: metadata})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

// RemoveDataSource removes a data source to an instance.
func (s *InstanceService) RemoveDataSource(ctx context.Context, req *connect.Request[v1pb.RemoveDataSourceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.DataSource == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("data sources is required"))
	}

	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}

	metadata := proto.CloneOf(instance.Metadata)
	var updatedDataSources []*storepb.DataSource
	var dataSource *storepb.DataSource
	for _, ds := range instance.Metadata.GetDataSources() {
		if ds.GetId() == req.Msg.DataSource.Id {
			dataSource = ds
		} else {
			updatedDataSources = append(updatedDataSources, ds)
		}
	}
	if dataSource == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("data source not found"))
	}

	// We only support remove RO type datasource to instance now, see more details in instance_service.proto.
	if dataSource.GetType() != storepb.DataSourceType_READ_ONLY {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only support remove read-only data source"))
	}

	metadata.DataSources = updatedDataSources
	instance, err = s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{ResourceID: instance.ResourceID, Metadata: metadata})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instance.ResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

func getInstanceMessage(ctx context.Context, stores *store.Store, name string) (*store.InstanceMessage, error) {
	instanceID, err := common.GetInstanceID(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	find := &store.FindInstanceMessage{
		ResourceID: &instanceID,
	}
	instance, err := stores.GetInstanceV2(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", name))
	}

	return instance, nil
}

// buildInstanceName builds the instance name with the given instance ID.
func buildInstanceName(instanceID string) string {
	var b strings.Builder
	b.Grow(len(common.InstanceNamePrefix) + len(instanceID))
	_, _ = b.WriteString(common.InstanceNamePrefix)
	_, _ = b.WriteString(instanceID)
	return b.String()
}

// buildEnvironmentName builds the environment name with the given environment ID.
func buildEnvironmentName(environmentID *string) *string {
	if environmentID == nil || *environmentID == "" {
		return nil
	}
	var b strings.Builder
	b.Grow(len("environments/") + len(*environmentID))
	_, _ = b.WriteString("environments/")
	_, _ = b.WriteString(*environmentID)
	result := b.String()
	return &result
}

func convertInstanceMessage(instance *store.InstanceMessage) *v1pb.Instance {
	engine := convertToEngine(instance.Metadata.GetEngine())
	dataSources := convertDataSources(instance.Metadata.GetDataSources())

	return &v1pb.Instance{
		Name:               buildInstanceName(instance.ResourceID),
		Title:              instance.Metadata.GetTitle(),
		Engine:             engine,
		EngineVersion:      instance.Metadata.GetVersion(),
		ExternalLink:       instance.Metadata.GetExternalLink(),
		DataSources:        dataSources,
		State:              convertDeletedToState(instance.Deleted),
		Environment:        buildEnvironmentName(instance.EnvironmentID),
		Activation:         instance.Metadata.GetActivation(),
		SyncInterval:       instance.Metadata.GetSyncInterval(),
		MaximumConnections: instance.Metadata.GetMaximumConnections(),
		SyncDatabases:      instance.Metadata.GetSyncDatabases(),
		Roles:              convertInstanceRoles(instance, instance.Metadata.GetRoles()),
		LastSyncTime:       instance.Metadata.GetLastSyncTime(),
	}
}

// buildRoleName builds the role name with the given instance ID and role name.
func buildRoleName(b *strings.Builder, instanceID, roleName string) string {
	b.Reset()
	_, _ = b.WriteString(common.InstanceNamePrefix)
	_, _ = b.WriteString(instanceID)
	_, _ = b.WriteString("/")
	_, _ = b.WriteString(common.RolePrefix)
	_, _ = b.WriteString(roleName)
	return b.String()
}

func convertInstanceRoles(instance *store.InstanceMessage, roles []*storepb.InstanceRole) []*v1pb.InstanceRole {
	var v1Roles []*v1pb.InstanceRole
	var b strings.Builder

	// preallocate memory for the builder
	b.Grow(len(common.InstanceNamePrefix) + len(instance.ResourceID) + 1 + len(common.RolePrefix) + 20)

	for _, role := range roles {
		v1Roles = append(v1Roles, &v1pb.InstanceRole{
			Name:      buildRoleName(&b, instance.ResourceID, role.Name),
			RoleName:  role.Name,
			Attribute: role.Attribute,
		})
	}
	return v1Roles
}

func convertInstanceToInstanceMessage(instanceID string, instance *v1pb.Instance) (*store.InstanceMessage, error) {
	datasources, err := convertV1DataSources(instance.DataSources)
	if err != nil {
		return nil, err
	}

	var environmentID *string
	if instance.Environment != nil && *instance.Environment != "" {
		envID, err := common.GetEnvironmentID(*instance.Environment)
		if err != nil {
			return nil, err
		}
		environmentID = &envID
	}

	return &store.InstanceMessage{
		ResourceID:    instanceID,
		EnvironmentID: environmentID,
		Metadata: &storepb.Instance{
			Title:              instance.GetTitle(),
			Engine:             convertEngine(instance.Engine),
			ExternalLink:       instance.GetExternalLink(),
			Activation:         instance.GetActivation(),
			DataSources:        datasources,
			SyncInterval:       instance.GetSyncInterval(),
			MaximumConnections: instance.GetMaximumConnections(),
			SyncDatabases:      instance.GetSyncDatabases(),
		},
	}, nil
}

func convertInstanceMessageToInstanceResource(instanceMessage *store.InstanceMessage) *v1pb.InstanceResource {
	instance := convertInstanceMessage(instanceMessage)
	return &v1pb.InstanceResource{
		Name:          instance.Name,
		Title:         instance.Title,
		Engine:        instance.Engine,
		EngineVersion: instance.EngineVersion,
		DataSources:   instance.DataSources,
		Activation:    instance.Activation,
		Environment:   instance.Environment,
	}
}

func convertV1DataSources(dataSources []*v1pb.DataSource) ([]*storepb.DataSource, error) {
	var values []*storepb.DataSource
	for _, ds := range dataSources {
		dataSource, err := convertV1DataSource(ds)
		if err != nil {
			return nil, err
		}
		values = append(values, dataSource)
	}

	return values, nil
}

func convertDataSourceExternalSecret(externalSecret *storepb.DataSourceExternalSecret) *v1pb.DataSourceExternalSecret {
	if externalSecret == nil {
		return nil
	}

	resp := &v1pb.DataSourceExternalSecret{
		SecretType:      v1pb.DataSourceExternalSecret_SecretType(externalSecret.SecretType),
		Url:             externalSecret.Url,
		AuthType:        v1pb.DataSourceExternalSecret_AuthType(externalSecret.AuthType),
		EngineName:      externalSecret.EngineName,
		SecretName:      externalSecret.SecretName,
		PasswordKeyName: externalSecret.PasswordKeyName,
	}

	// clear sensitive data.
	switch resp.AuthType {
	case v1pb.DataSourceExternalSecret_VAULT_APP_ROLE:
		appRole := externalSecret.GetAppRole()
		if appRole != nil {
			resp.AuthOption = &v1pb.DataSourceExternalSecret_AppRole{
				AppRole: &v1pb.DataSourceExternalSecret_AppRoleAuthOption{
					Type:      v1pb.DataSourceExternalSecret_AppRoleAuthOption_SecretType(appRole.Type),
					MountPath: appRole.MountPath,
				},
			}
		}
	case v1pb.DataSourceExternalSecret_TOKEN:
		resp.AuthOption = &v1pb.DataSourceExternalSecret_Token{
			Token: "",
		}
	default:
	}

	return resp
}

func convertDataSources(dataSources []*storepb.DataSource) []*v1pb.DataSource {
	var v1DataSources []*v1pb.DataSource
	for _, ds := range dataSources {
		externalSecret := convertDataSourceExternalSecret(ds.GetExternalSecret())

		dataSourceType := v1pb.DataSourceType_DATA_SOURCE_UNSPECIFIED
		switch ds.GetType() {
		case storepb.DataSourceType_ADMIN:
			dataSourceType = v1pb.DataSourceType_ADMIN
		case storepb.DataSourceType_READ_ONLY:
			dataSourceType = v1pb.DataSourceType_READ_ONLY
		default:
		}

		authenticationType := v1pb.DataSource_AUTHENTICATION_UNSPECIFIED
		switch ds.GetAuthenticationType() {
		case storepb.DataSource_AUTHENTICATION_UNSPECIFIED, storepb.DataSource_PASSWORD:
			authenticationType = v1pb.DataSource_PASSWORD
		case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
			authenticationType = v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM
		case storepb.DataSource_AWS_RDS_IAM:
			authenticationType = v1pb.DataSource_AWS_RDS_IAM
		case storepb.DataSource_AZURE_IAM:
			authenticationType = v1pb.DataSource_AZURE_IAM
		default:
		}

		dataSource := &v1pb.DataSource{
			Id:       ds.GetId(),
			Type:     dataSourceType,
			Username: ds.GetUsername(),
			// We don't return the password and SSLs on reads.
			Host:                      ds.GetHost(),
			Port:                      ds.GetPort(),
			Database:                  ds.GetDatabase(),
			Srv:                       ds.GetSrv(),
			AuthenticationDatabase:    ds.GetAuthenticationDatabase(),
			Sid:                       ds.GetSid(),
			ServiceName:               ds.GetServiceName(),
			SshHost:                   ds.GetSshHost(),
			SshPort:                   ds.GetSshPort(),
			SshUser:                   ds.GetSshUser(),
			ExternalSecret:            externalSecret,
			AuthenticationType:        authenticationType,
			SaslConfig:                convertDataSourceSaslConfig(ds.GetSaslConfig()),
			AdditionalAddresses:       convertDataSourceAddresses(ds.GetAdditionalAddresses()),
			ReplicaSet:                ds.GetReplicaSet(),
			DirectConnection:          ds.GetDirectConnection(),
			Region:                    ds.GetRegion(),
			WarehouseId:               ds.GetWarehouseId(),
			UseSsl:                    ds.GetUseSsl(),
			RedisType:                 convertRedisType(ds.GetRedisType()),
			MasterName:                ds.GetMasterName(),
			MasterUsername:            ds.GetMasterUsername(),
			ExtraConnectionParameters: ds.GetExtraConnectionParameters(),
		}

		switch dataSource.AuthenticationType {
		case v1pb.DataSource_AZURE_IAM:
			if azureCredential := ds.GetAzureCredential(); azureCredential != nil {
				dataSource.IamExtension = &v1pb.DataSource_AzureCredential_{
					AzureCredential: &v1pb.DataSource_AzureCredential{
						TenantId: azureCredential.TenantId,
						ClientId: azureCredential.ClientId,
					},
				}
			}
		case v1pb.DataSource_AWS_RDS_IAM:
			if awsCredential := ds.GetAwsCredential(); awsCredential != nil {
				dataSource.IamExtension = &v1pb.DataSource_AwsCredential{
					AwsCredential: &v1pb.DataSource_AWSCredential{},
				}
			}
		case v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM:
			if gcpCredential := ds.GetGcpCredential(); gcpCredential != nil {
				dataSource.IamExtension = &v1pb.DataSource_GcpCredential{
					GcpCredential: &v1pb.DataSource_GCPCredential{},
				}
			}
		default:
		}

		v1DataSources = append(v1DataSources, dataSource)
	}

	return v1DataSources
}

func convertV1DataSourceExternalSecret(externalSecret *v1pb.DataSourceExternalSecret) (*storepb.DataSourceExternalSecret, error) {
	if externalSecret == nil {
		return nil, nil
	}

	secret := &storepb.DataSourceExternalSecret{
		SecretType:      storepb.DataSourceExternalSecret_SecretType(externalSecret.SecretType),
		Url:             externalSecret.Url,
		AuthType:        storepb.DataSourceExternalSecret_AuthType(externalSecret.AuthType),
		EngineName:      externalSecret.EngineName,
		SecretName:      externalSecret.SecretName,
		PasswordKeyName: externalSecret.PasswordKeyName,
	}

	// Convert auth options
	switch externalSecret.AuthOption.(type) {
	case *v1pb.DataSourceExternalSecret_Token:
		secret.AuthOption = &storepb.DataSourceExternalSecret_Token{
			Token: externalSecret.GetToken(),
		}
	case *v1pb.DataSourceExternalSecret_AppRole:
		appRole := externalSecret.GetAppRole()
		if appRole != nil {
			secret.AuthOption = &storepb.DataSourceExternalSecret_AppRole{
				AppRole: &storepb.DataSourceExternalSecret_AppRoleAuthOption{
					Type:      storepb.DataSourceExternalSecret_AppRoleAuthOption_SecretType(appRole.Type),
					MountPath: appRole.MountPath,
				},
			}
		}
	}

	switch secret.SecretType {
	case storepb.DataSourceExternalSecret_VAULT_KV_V2:
		if secret.Url == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing Vault URL"))
		}
		if secret.EngineName == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing Vault engine name"))
		}
		if secret.SecretName == "" || secret.PasswordKeyName == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing secret name or key name"))
		}
	case storepb.DataSourceExternalSecret_AWS_SECRETS_MANAGER:
		if secret.SecretName == "" || secret.PasswordKeyName == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing secret name or key name"))
		}
	case storepb.DataSourceExternalSecret_GCP_SECRET_MANAGER:
		if secret.SecretName == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing GCP secret name"))
		}
	default:
	}

	switch secret.AuthType {
	case storepb.DataSourceExternalSecret_TOKEN:
		if secret.GetToken() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing token"))
		}
	case storepb.DataSourceExternalSecret_VAULT_APP_ROLE:
		if secret.GetAppRole() == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing Vault approle"))
		}
	default:
	}

	return secret, nil
}

func convertV1DataSourceSaslConfig(saslConfig *v1pb.SASLConfig) *storepb.SASLConfig {
	if saslConfig == nil {
		return nil
	}
	storeSaslConfig := &storepb.SASLConfig{}
	switch m := saslConfig.Mechanism.(type) {
	case *v1pb.SASLConfig_KrbConfig:
		storeSaslConfig.Mechanism = &storepb.SASLConfig_KrbConfig{
			KrbConfig: &storepb.KerberosConfig{
				Primary:              m.KrbConfig.Primary,
				Instance:             m.KrbConfig.Instance,
				Realm:                m.KrbConfig.Realm,
				Keytab:               m.KrbConfig.Keytab,
				KdcHost:              m.KrbConfig.KdcHost,
				KdcPort:              m.KrbConfig.KdcPort,
				KdcTransportProtocol: m.KrbConfig.KdcTransportProtocol,
			},
		}
	default:
		return nil
	}
	return storeSaslConfig
}

func convertDataSourceSaslConfig(saslConfig *storepb.SASLConfig) *v1pb.SASLConfig {
	if saslConfig == nil {
		return nil
	}
	storeSaslConfig := &v1pb.SASLConfig{}
	switch m := saslConfig.Mechanism.(type) {
	case *storepb.SASLConfig_KrbConfig:
		storeSaslConfig.Mechanism = &v1pb.SASLConfig_KrbConfig{
			KrbConfig: &v1pb.KerberosConfig{
				Primary:              m.KrbConfig.Primary,
				Instance:             m.KrbConfig.Instance,
				Realm:                m.KrbConfig.Realm,
				Keytab:               m.KrbConfig.Keytab,
				KdcHost:              m.KrbConfig.KdcHost,
				KdcPort:              m.KrbConfig.KdcPort,
				KdcTransportProtocol: m.KrbConfig.KdcTransportProtocol,
			},
		}
	default:
		return nil
	}
	return storeSaslConfig
}

func convertDataSourceAddresses(addresses []*storepb.DataSource_Address) []*v1pb.DataSource_Address {
	res := make([]*v1pb.DataSource_Address, 0, len(addresses))
	for _, address := range addresses {
		res = append(res, &v1pb.DataSource_Address{
			Host: address.Host,
			Port: address.Port,
		})
	}
	return res
}

func convertAdditionalAddresses(addresses []*v1pb.DataSource_Address) []*storepb.DataSource_Address {
	res := make([]*storepb.DataSource_Address, 0, len(addresses))
	for _, address := range addresses {
		res = append(res, &storepb.DataSource_Address{
			Host: address.Host,
			Port: address.Port,
		})
	}
	return res
}

func convertV1AuthenticationType(authType v1pb.DataSource_AuthenticationType) storepb.DataSource_AuthenticationType {
	authenticationType := storepb.DataSource_AUTHENTICATION_UNSPECIFIED
	switch authType {
	case v1pb.DataSource_AUTHENTICATION_UNSPECIFIED, v1pb.DataSource_PASSWORD:
		authenticationType = storepb.DataSource_PASSWORD
	case v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		authenticationType = storepb.DataSource_GOOGLE_CLOUD_SQL_IAM
	case v1pb.DataSource_AWS_RDS_IAM:
		authenticationType = storepb.DataSource_AWS_RDS_IAM
	case v1pb.DataSource_AZURE_IAM:
		authenticationType = storepb.DataSource_AZURE_IAM
	default:
	}
	return authenticationType
}

func convertV1RedisType(redisType v1pb.DataSource_RedisType) storepb.DataSource_RedisType {
	authenticationType := storepb.DataSource_REDIS_TYPE_UNSPECIFIED
	switch redisType {
	case v1pb.DataSource_STANDALONE:
		authenticationType = storepb.DataSource_STANDALONE
	case v1pb.DataSource_SENTINEL:
		authenticationType = storepb.DataSource_SENTINEL
	case v1pb.DataSource_CLUSTER:
		authenticationType = storepb.DataSource_CLUSTER
	default:
	}
	return authenticationType
}

func convertRedisType(redisType storepb.DataSource_RedisType) v1pb.DataSource_RedisType {
	authenticationType := v1pb.DataSource_STANDALONE
	switch redisType {
	case storepb.DataSource_STANDALONE:
		authenticationType = v1pb.DataSource_STANDALONE
	case storepb.DataSource_SENTINEL:
		authenticationType = v1pb.DataSource_SENTINEL
	case storepb.DataSource_CLUSTER:
		authenticationType = v1pb.DataSource_CLUSTER
	default:
	}
	return authenticationType
}

func convertV1DataSource(dataSource *v1pb.DataSource) (*storepb.DataSource, error) {
	dsType, err := convertV1DataSourceType(dataSource.Type)
	if err != nil {
		return nil, err
	}
	externalSecret, err := convertV1DataSourceExternalSecret(dataSource.ExternalSecret)
	if err != nil {
		return nil, err
	}
	saslConfig := convertV1DataSourceSaslConfig(dataSource.SaslConfig)

	storeDataSource := &storepb.DataSource{
		Id:                        dataSource.Id,
		Type:                      dsType,
		Username:                  dataSource.Username,
		Password:                  dataSource.Password,
		SslCa:                     dataSource.SslCa,
		SslCert:                   dataSource.SslCert,
		SslKey:                    dataSource.SslKey,
		Host:                      dataSource.Host,
		Port:                      dataSource.Port,
		Database:                  dataSource.Database,
		Srv:                       dataSource.Srv,
		AuthenticationDatabase:    dataSource.AuthenticationDatabase,
		Sid:                       dataSource.Sid,
		ServiceName:               dataSource.ServiceName,
		SshHost:                   dataSource.SshHost,
		SshPort:                   dataSource.SshPort,
		SshUser:                   dataSource.SshUser,
		SshPassword:               dataSource.SshPassword,
		SshPrivateKey:             dataSource.SshPrivateKey,
		AuthenticationPrivateKey:  dataSource.AuthenticationPrivateKey,
		ExternalSecret:            externalSecret,
		SaslConfig:                saslConfig,
		AuthenticationType:        convertV1AuthenticationType(dataSource.AuthenticationType),
		AdditionalAddresses:       convertAdditionalAddresses(dataSource.AdditionalAddresses),
		ReplicaSet:                dataSource.ReplicaSet,
		DirectConnection:          dataSource.DirectConnection,
		Region:                    dataSource.Region,
		WarehouseId:               dataSource.WarehouseId,
		UseSsl:                    dataSource.UseSsl,
		RedisType:                 convertV1RedisType(dataSource.RedisType),
		MasterName:                dataSource.MasterName,
		MasterUsername:            dataSource.MasterUsername,
		MasterPassword:            dataSource.MasterPassword,
		ExtraConnectionParameters: dataSource.ExtraConnectionParameters,
	}

	switch dataSource.AuthenticationType {
	case v1pb.DataSource_AZURE_IAM:
		if azureCredential := dataSource.GetAzureCredential(); azureCredential != nil {
			storeDataSource.IamExtension = &storepb.DataSource_AzureCredential_{
				AzureCredential: &storepb.DataSource_AzureCredential{
					TenantId:     azureCredential.TenantId,
					ClientId:     azureCredential.ClientId,
					ClientSecret: azureCredential.ClientSecret,
				},
			}
		}
	case v1pb.DataSource_AWS_RDS_IAM:
		if awsCredential := dataSource.GetAwsCredential(); awsCredential != nil {
			storeDataSource.IamExtension = &storepb.DataSource_AwsCredential{
				AwsCredential: &storepb.DataSource_AWSCredential{
					AccessKeyId:     awsCredential.AccessKeyId,
					SecretAccessKey: awsCredential.SecretAccessKey,
					SessionToken:    awsCredential.SessionToken,
				},
			}
		}
	case v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		if gcpCredential := dataSource.GetGcpCredential(); gcpCredential != nil {
			storeDataSource.IamExtension = &storepb.DataSource_GcpCredential{
				GcpCredential: &storepb.DataSource_GCPCredential{
					Content: gcpCredential.Content,
				},
			}
		}
	default:
	}

	return storeDataSource, nil
}

func convertV1DataSourceType(tp v1pb.DataSourceType) (storepb.DataSourceType, error) {
	switch tp {
	case v1pb.DataSourceType_READ_ONLY:
		return storepb.DataSourceType_READ_ONLY, nil
	case v1pb.DataSourceType_ADMIN:
		return storepb.DataSourceType_ADMIN, nil
	default:
		return storepb.DataSourceType_DATA_SOURCE_UNSPECIFIED, errors.Errorf("invalid data source type %v", tp)
	}
}

func (s *InstanceService) instanceCountGuard(ctx context.Context) error {
	instanceLimit := s.licenseService.GetInstanceLimit(ctx)

	count, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{})
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if count >= instanceLimit {
		return connect.NewError(connect.CodeResourceExhausted, errors.Errorf("reached the maximum instance count %d", instanceLimit))
	}

	return nil
}
