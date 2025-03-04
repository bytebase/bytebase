package v1

import (
	"context"
	"log/slog"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/secret"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// InstanceService implements the instance service.
type InstanceService struct {
	v1pb.UnimplementedInstanceServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
	metricReporter *metricreport.Reporter
	secret         string
	stateCfg       *state.State
	dbFactory      *dbfactory.DBFactory
	schemaSyncer   *schemasync.Syncer
	iamManager     *iam.Manager
}

// NewInstanceService creates a new InstanceService.
func NewInstanceService(store *store.Store, licenseService enterprise.LicenseService, metricReporter *metricreport.Reporter, secret string, stateCfg *state.State, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer, iamManager *iam.Manager) *InstanceService {
	return &InstanceService{
		store:          store,
		licenseService: licenseService,
		metricReporter: metricReporter,
		secret:         secret,
		stateCfg:       stateCfg,
		dbFactory:      dbFactory,
		schemaSyncer:   schemaSyncer,
		iamManager:     iamManager,
	}
}

// GetInstance gets an instance.
func (s *InstanceService) GetInstance(ctx context.Context, request *v1pb.GetInstanceRequest) (*v1pb.Instance, error) {
	instance, err := getInstanceMessage(ctx, s.store, request.Name)
	if err != nil {
		return nil, err
	}
	return convertToInstance(instance)
}

// ListInstances lists all instances.
func (s *InstanceService) ListInstances(ctx context.Context, request *v1pb.ListInstancesRequest) (*v1pb.ListInstancesResponse, error) {
	find := &store.FindInstanceMessage{
		ShowDeleted: request.ShowDeleted,
	}
	instances, err := s.store.ListInstancesV2(ctx, find)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	response := &v1pb.ListInstancesResponse{}
	for _, instance := range instances {
		ins, err := convertToInstance(instance)
		if err != nil {
			return nil, err
		}
		response.Instances = append(response.Instances, ins)
	}
	return response, nil
}

// ListInstanceDatabase list all databases in the instance.
func (s *InstanceService) ListInstanceDatabase(ctx context.Context, request *v1pb.ListInstanceDatabaseRequest) (*v1pb.ListInstanceDatabaseResponse, error) {
	var instanceMessage *store.InstanceMessage

	if request.Instance != nil {
		instanceID, err := common.GetInstanceID(request.Name)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		if instanceMessage, err = s.convertToInstanceMessage(instanceID, request.Instance); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	} else {
		instance, err := getInstanceMessage(ctx, s.store, request.Name)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		instanceMessage = instance
	}

	instanceMeta, err := s.schemaSyncer.GetInstanceMeta(ctx, instanceMessage)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &v1pb.ListInstanceDatabaseResponse{}
	for _, database := range instanceMeta.Databases {
		response.Databases = append(response.Databases, database.Name)
	}
	return response, nil
}

// CreateInstance creates an instance.
func (s *InstanceService) CreateInstance(ctx context.Context, request *v1pb.CreateInstanceRequest) (*v1pb.Instance, error) {
	if request.Instance == nil {
		return nil, status.Errorf(codes.InvalidArgument, "instance must be set")
	}
	if !isValidResourceID(request.InstanceId) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid instance ID %v", request.InstanceId)
	}

	if err := s.instanceCountGuard(ctx); err != nil {
		return nil, err
	}

	instanceMessage, err := s.convertToInstanceMessage(request.InstanceId, request.Instance)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Test connection.
	if request.ValidateOnly {
		for _, ds := range instanceMessage.DataSources {
			err := func() error {
				driver, err := s.dbFactory.GetDataSourceDriver(ctx, instanceMessage, ds, "", false /* datashare */, ds.Type == api.RO, db.ConnectionContext{})
				if err != nil {
					return status.Errorf(codes.Internal, "failed to get database driver with error: %v", err.Error())
				}
				defer driver.Close(ctx)
				if err := driver.Ping(ctx); err != nil {
					return status.Errorf(codes.InvalidArgument, "invalid datasource %s, error %s", ds.Type, err)
				}
				return nil
			}()
			if err != nil {
				return nil, err
			}
		}

		return convertToInstance(instanceMessage)
	}

	instanceCountLimit := s.licenseService.GetInstanceLicenseCount(ctx)
	if instanceMessage.Activation {
		if err := s.store.CheckActivationLimit(ctx, instanceCountLimit); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return nil, status.Error(codes.ResourceExhausted, err.Error())
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if err := s.checkInstanceDataSources(instanceMessage, instanceMessage.DataSources); err != nil {
		return nil, err
	}

	instance, err := s.store.CreateInstanceV2(ctx,
		instanceMessage,
		instanceCountLimit,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
			"engine": instance.Engine,
		},
	})

	return convertToInstance(instance)
}

func (s *InstanceService) checkInstanceDataSources(instance *store.InstanceMessage, dataSources []*store.DataSourceMessage) error {
	dsIDMap := map[string]bool{}
	for _, ds := range dataSources {
		if err := s.checkDataSource(instance, ds); err != nil {
			return err
		}
		if dsIDMap[ds.ID] {
			return status.Errorf(codes.InvalidArgument, `duplicate data source id "%s"`, ds.ID)
		}
		dsIDMap[ds.ID] = true
	}

	return nil
}

func (s *InstanceService) checkDataSource(instance *store.InstanceMessage, dataSource *store.DataSourceMessage) error {
	if dataSource.ID == "" {
		return status.Errorf(codes.InvalidArgument, "data source id is required")
	}
	password, err := common.Unobfuscate(dataSource.ObfuscatedPassword, s.secret)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureExternalSecretManager, instance); err != nil {
		missingFeatureError := status.Error(codes.PermissionDenied, err.Error())
		if dataSource.ExternalSecret != nil {
			return missingFeatureError
		}
		if ok, _ := secret.GetExternalSecretURL(password); !ok {
			return nil
		}
		return missingFeatureError
	}

	return nil
}

// UpdateInstance updates an instance.
func (s *InstanceService) UpdateInstance(ctx context.Context, request *v1pb.UpdateInstanceRequest) (*v1pb.Instance, error) {
	if request.Instance == nil {
		return nil, status.Errorf(codes.InvalidArgument, "instance must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	instance, err := getInstanceMessage(ctx, s.store, request.Instance.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", request.Instance.Name)
	}

	patch := &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &request.Instance.Title
		case "environment":
			environmentID, err := common.GetEnvironmentID(request.Instance.Environment)
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
			patch.EnvironmentID = &environment.ResourceID
		case "external_link":
			patch.ExternalLink = &request.Instance.ExternalLink
		case "data_sources":
			datasources, err := s.convertToDataSourceMessages(request.Instance.DataSources)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			if err := s.checkInstanceDataSources(instance, datasources); err != nil {
				return nil, err
			}
			patch.DataSources = &datasources
		case "activation":
			if request.Instance.Activation != instance.Activation {
				patch.Activation = &request.Instance.Activation
			}
		case "options.sync_interval":
			if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureCustomInstanceSynchronization, instance); err != nil {
				return nil, status.Error(codes.PermissionDenied, err.Error())
			}
			if patch.OptionsUpsert == nil {
				patch.OptionsUpsert = instance.Options
			}
			patch.OptionsUpsert.SyncInterval = request.Instance.Options.GetSyncInterval()
		case "options.maximum_connections":
			if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureCustomInstanceSynchronization, instance); err != nil {
				return nil, status.Error(codes.PermissionDenied, err.Error())
			}
			if patch.OptionsUpsert == nil {
				patch.OptionsUpsert = instance.Options
			}
			patch.OptionsUpsert.MaximumConnections = request.Instance.Options.GetMaximumConnections()
		case "options.sync_databases":
			if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureCustomInstanceSynchronization, instance); err != nil {
				return nil, status.Error(codes.PermissionDenied, err.Error())
			}
			if patch.OptionsUpsert == nil {
				patch.OptionsUpsert = instance.Options
			}
			patch.OptionsUpsert.SyncDatabases = request.Instance.Options.GetSyncDatabases()
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupported update_mask "%s"`, path)
		}
	}

	instanceCountLimit := s.licenseService.GetInstanceLicenseCount(ctx)
	if v := patch.Activation; v != nil && *v {
		if err := s.store.CheckActivationLimit(ctx, instanceCountLimit); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return nil, status.Error(codes.ResourceExhausted, err.Error())
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	ins, err := s.store.UpdateInstanceV2(ctx, patch, instanceCountLimit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return convertToInstance(ins)
}

func (s *InstanceService) syncSlowQueriesForInstance(ctx context.Context, instanceName string) (*emptypb.Empty, error) {
	instance, err := getInstanceMessage(ctx, s.store, instanceName)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", instanceName)
	}

	slowQueryPolicy, err := s.store.GetSlowQueryPolicy(ctx, instance.ResourceID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if slowQueryPolicy == nil || !slowQueryPolicy.Active {
		return nil, status.Errorf(codes.FailedPrecondition, "slow query policy is not active for instance %q", instanceName)
	}

	if err := s.syncSlowQueriesImpl(ctx, (*store.ProjectMessage)(nil), instance); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *InstanceService) syncSlowQueriesImpl(ctx context.Context, project *store.ProjectMessage, instance *store.InstanceMessage) error {
	switch instance.Engine {
	case storepb.Engine_MYSQL:
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
		if err != nil {
			return err
		}
		defer driver.Close(ctx)
		if err := driver.CheckSlowQueryLogEnabled(ctx); err != nil {
			slog.Warn("slow query log is not enabled", slog.String("instance", instance.ResourceID), log.BBError(err))
			return nil
		}

		// Sync slow queries for instance.
		message := &state.InstanceSlowQuerySyncMessage{
			InstanceID: instance.ResourceID,
		}
		if project != nil {
			message.ProjectID = project.ResourceID
		}
		s.stateCfg.InstanceSlowQuerySyncChan <- message
	case storepb.Engine_POSTGRES:
		findDatabase := &store.FindDatabaseMessage{
			InstanceID: &instance.ResourceID,
		}
		databases, err := s.store.ListDatabases(ctx, findDatabase)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to list databases: %s", err.Error())
		}

		enabled := false
		for _, database := range databases {
			if database.Deleted {
				continue
			}
			if pgparser.IsSystemDatabase(database.DatabaseName) {
				continue
			}
			if err := func() error {
				driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
				if err != nil {
					return err
				}
				defer driver.Close(ctx)
				return driver.CheckSlowQueryLogEnabled(ctx)
			}(); err != nil {
				slog.Warn("slow query log is not enabled", slog.String("database", database.DatabaseName), log.BBError(err))
				continue
			}

			enabled = true
			break
		}

		if !enabled {
			return nil
		}

		// Sync slow queries for instance.
		message := &state.InstanceSlowQuerySyncMessage{
			InstanceID: instance.ResourceID,
		}
		if project != nil {
			message.ProjectID = project.ResourceID
		}
		s.stateCfg.InstanceSlowQuerySyncChan <- message
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported engine %q", instance.Engine)
	}
	return nil
}

func (s *InstanceService) syncSlowQueriesForProject(ctx context.Context, projectName string) (*emptypb.Empty, error) {
	project, err := s.getProjectMessage(ctx, projectName)
	if err != nil {
		return nil, err
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectName)
	}
	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list databases: %s", err.Error())
	}

	instanceMap := make(map[string]bool)
	var errs error
	for _, database := range databases {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get instance %q: %s", database.InstanceID, err.Error())
		}

		switch instance.Engine {
		case storepb.Engine_MYSQL, storepb.Engine_POSTGRES:
			if instance.Deleted {
				continue
			}

			slowQueryPolicy, err := s.store.GetSlowQueryPolicy(ctx, instance.ResourceID)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			if slowQueryPolicy == nil || !slowQueryPolicy.Active {
				continue
			}

			if _, ok := instanceMap[instance.ResourceID]; ok {
				continue
			}

			if err := s.syncSlowQueriesImpl(ctx, project, instance); err != nil {
				errs = multierr.Append(errs, errors.Wrapf(err, "failed to sync slow queries for instance %q", instance.ResourceID))
			}
		default:
			continue
		}
	}

	if errs != nil {
		return nil, status.Errorf(codes.Internal, "failed to sync slow queries for following instances: %s", errs.Error())
	}

	return &emptypb.Empty{}, nil
}

// SyncSlowQueries syncs slow queries for an instance.
func (s *InstanceService) SyncSlowQueries(ctx context.Context, request *v1pb.SyncSlowQueriesRequest) (*emptypb.Empty, error) {
	switch {
	case strings.HasPrefix(request.Parent, common.InstanceNamePrefix):
		return s.syncSlowQueriesForInstance(ctx, request.Parent)
	case strings.HasPrefix(request.Parent, common.ProjectNamePrefix):
		return s.syncSlowQueriesForProject(ctx, request.Parent)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid parent %q", request.Parent)
	}
}

// DeleteInstance deletes an instance.
func (s *InstanceService) DeleteInstance(ctx context.Context, request *v1pb.DeleteInstanceRequest) (*emptypb.Empty, error) {
	instance, err := getInstanceMessage(ctx, s.store, request.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", request.Name)
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
	if err != nil {
		return nil, err
	}
	if request.Force {
		if len(databases) > 0 {
			if _, err := s.store.BatchUpdateDatabaseProject(ctx, databases, api.DefaultProjectID); err != nil {
				return nil, err
			}
		}
	} else {
		var databaseNames []string
		for _, database := range databases {
			if database.ProjectID != api.DefaultProjectID {
				databaseNames = append(databaseNames, database.DatabaseName)
			}
		}
		if len(databaseNames) > 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "all databases should be transferred to the unassigned project before deleting the instance")
		}
	}

	if _, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
		Deleted:    &deletePatch,
	}, -1 /* don't need to pass the instance limition */); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

// UndeleteInstance undeletes an instance.
func (s *InstanceService) UndeleteInstance(ctx context.Context, request *v1pb.UndeleteInstanceRequest) (*v1pb.Instance, error) {
	instance, err := getInstanceMessage(ctx, s.store, request.Name)
	if err != nil {
		return nil, err
	}
	if !instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q is active", request.Name)
	}

	ins, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
		Deleted:    &undeletePatch,
	}, -1 /* don't need to pass the instance limition */)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return convertToInstance(ins)
}

// SyncInstance syncs the instance.
func (s *InstanceService) SyncInstance(ctx context.Context, request *v1pb.SyncInstanceRequest) (*v1pb.SyncInstanceResponse, error) {
	instance, err := getInstanceMessage(ctx, s.store, request.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", request.Name)
	}

	updatedInstance, allDatabases, newDatabases, err := s.schemaSyncer.SyncInstance(ctx, instance)
	if err != nil {
		return nil, err
	}
	if request.EnableFullSync {
		// Sync all databases in the instance asynchronously.
		s.schemaSyncer.SyncAllDatabases(ctx, updatedInstance)
	} else {
		s.schemaSyncer.SyncDatabasesAsync(newDatabases)
	}

	response := &v1pb.SyncInstanceResponse{}
	for _, database := range allDatabases {
		response.Databases = append(response.Databases, database.Name)
	}
	return response, nil
}

// SyncInstance syncs the instance.
func (s *InstanceService) BatchSyncInstances(ctx context.Context, request *v1pb.BatchSyncInstancesRequest) (*v1pb.BatchSyncInstancesResponse, error) {
	for _, r := range request.Requests {
		instance, err := getInstanceMessage(ctx, s.store, r.Name)
		if err != nil {
			return nil, err
		}
		if instance.Deleted {
			return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", r.Name)
		}

		updatedInstance, _, newDatabases, err := s.schemaSyncer.SyncInstance(ctx, instance)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to sync instance, %v", err)
		}
		if r.EnableFullSync {
			// Sync all databases in the instance asynchronously.
			s.schemaSyncer.SyncAllDatabases(ctx, updatedInstance)
		} else {
			s.schemaSyncer.SyncDatabasesAsync(newDatabases)
		}
	}

	return &v1pb.BatchSyncInstancesResponse{}, nil
}

// AddDataSource adds a data source to an instance.
func (s *InstanceService) AddDataSource(ctx context.Context, request *v1pb.AddDataSourceRequest) (*v1pb.Instance, error) {
	if request.DataSource == nil {
		return nil, status.Errorf(codes.InvalidArgument, "data sources is required")
	}
	// We only support add RO type datasouce to instance now, see more details in instance_service.proto.
	if request.DataSource.Type != v1pb.DataSourceType_READ_ONLY {
		return nil, status.Errorf(codes.InvalidArgument, "only support adding read-only data source")
	}

	dataSource, err := s.convertToDataSourceMessage(request.DataSource)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert data source")
	}

	instance, err := getInstanceMessage(ctx, s.store, request.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", request.Name)
	}
	for _, ds := range instance.DataSources {
		if ds.ID == request.DataSource.Id {
			return nil, status.Errorf(codes.NotFound, "data source already exists with the same name")
		}
	}
	if err := s.checkDataSource(instance, dataSource); err != nil {
		return nil, err
	}

	// Test connection.
	if request.ValidateOnly {
		err := func() error {
			driver, err := s.dbFactory.GetDataSourceDriver(ctx, instance, dataSource, "", false /* datashare */, dataSource.Type == api.RO, db.ConnectionContext{})
			if err != nil {
				return status.Errorf(codes.Internal, "failed to get database driver with error: %v", err.Error())
			}
			defer driver.Close(ctx)
			if err := driver.Ping(ctx); err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid datasource %s, error %s", dataSource.Type, err)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
		return convertToInstance(instance)
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureReadReplicaConnection, instance); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if err := s.store.AddDataSourceToInstanceV2(ctx, instance.ResourceID, dataSource); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instance.ResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return convertToInstance(instance)
}

// UpdateDataSource updates a data source of an instance.
func (s *InstanceService) UpdateDataSource(ctx context.Context, request *v1pb.UpdateDataSourceRequest) (*v1pb.Instance, error) {
	if request.DataSource == nil {
		return nil, status.Errorf(codes.InvalidArgument, "datasource is required")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	instance, err := getInstanceMessage(ctx, s.store, request.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", request.Name)
	}
	// We create a new variable dataSource to not modify existing data source in the memory.
	var dataSource store.DataSourceMessage
	found := false
	for _, ds := range instance.DataSources {
		if ds.ID == request.DataSource.Id {
			dataSource = *ds
			found = true
			break
		}
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, `cannot found data source "%s"`, request.DataSource.Id)
	}

	if dataSource.Type == api.RO {
		if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureReadReplicaConnection, instance); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	patch := &store.UpdateDataSourceMessage{
		InstanceID:   instance.ResourceID,
		DataSourceID: request.DataSource.Id,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "username":
			patch.Username = &request.DataSource.Username
			dataSource.Username = *patch.Username
		case "password":
			obfuscated := common.Obfuscate(request.DataSource.Password, s.secret)
			patch.ObfuscatedPassword = &obfuscated
			dataSource.ObfuscatedPassword = obfuscated
		case "ssl_ca":
			obfuscated := common.Obfuscate(request.DataSource.SslCa, s.secret)
			patch.ObfuscatedSslCa = &obfuscated
			dataSource.ObfuscatedSslCa = obfuscated
		case "ssl_cert":
			obfuscated := common.Obfuscate(request.DataSource.SslCert, s.secret)
			patch.ObfuscatedSslCert = &obfuscated
			dataSource.ObfuscatedSslCert = obfuscated
		case "ssl_key":
			obfuscated := common.Obfuscate(request.DataSource.SslKey, s.secret)
			patch.ObfuscatedSslKey = &obfuscated
			dataSource.ObfuscatedSslKey = obfuscated
		case "host":
			patch.Host = &request.DataSource.Host
			dataSource.Host = request.DataSource.Host
		case "port":
			patch.Port = &request.DataSource.Port
			dataSource.Port = request.DataSource.Port
		case "database":
			patch.Database = &request.DataSource.Database
			dataSource.Database = request.DataSource.Database
		case "srv":
			patch.SRV = &request.DataSource.Srv
			dataSource.SRV = request.DataSource.Srv
		case "authentication_database":
			patch.AuthenticationDatabase = &request.DataSource.AuthenticationDatabase
			dataSource.AuthenticationDatabase = request.DataSource.AuthenticationDatabase
		case "sid":
			patch.SID = &request.DataSource.Sid
			dataSource.SID = request.DataSource.Sid
		case "service_name":
			patch.ServiceName = &request.DataSource.ServiceName
			dataSource.ServiceName = request.DataSource.ServiceName
		case "ssh_host":
			patch.SSHHost = &request.DataSource.SshHost
			dataSource.SSHHost = request.DataSource.SshHost
		case "ssh_port":
			patch.SSHPort = &request.DataSource.SshPort
			dataSource.SSHPort = request.DataSource.SshPort
		case "ssh_user":
			patch.SSHUser = &request.DataSource.SshUser
			dataSource.SSHUser = request.DataSource.SshUser
		case "ssh_password":
			obfuscated := common.Obfuscate(request.DataSource.SshPassword, s.secret)
			patch.SSHObfuscatedPassword = &obfuscated
			dataSource.SSHObfuscatedPassword = obfuscated
		case "ssh_private_key":
			obfuscated := common.Obfuscate(request.DataSource.SshPrivateKey, s.secret)
			patch.SSHObfuscatedPrivateKey = &obfuscated
			dataSource.SSHObfuscatedPrivateKey = obfuscated
		case "authentication_private_key":
			obfuscated := common.Obfuscate(request.DataSource.AuthenticationPrivateKey, s.secret)
			patch.AuthenticationPrivateKeyObfuscated = &obfuscated
			dataSource.AuthenticationPrivateKeyObfuscated = obfuscated
		case "external_secret":
			externalSecret, err := convertToStoreDataSourceExternalSecret(request.DataSource.ExternalSecret)
			if err != nil {
				return nil, err
			}
			dataSource.ExternalSecret = externalSecret
			patch.ExternalSecret = externalSecret
			patch.RemoveExternalSecret = externalSecret == nil
		case "sasl_config":
			dataSource.SASLConfig = convertToStoreDataSourceSaslConfig(request.DataSource.SaslConfig)
			patch.SASLConfig = dataSource.SASLConfig
			patch.RemoveSASLConfig = dataSource.SASLConfig == nil
		case "authentication_type":
			authType := convertToAuthenticationType(request.DataSource.AuthenticationType)
			dataSource.AuthenticationType = authType
			patch.AuthenticationType = &authType
		case "additional_addresses":
			additionalAddresses := convertToStoreAdditionalAddresses(request.DataSource.AdditionalAddresses)
			dataSource.AdditionalAddresses = additionalAddresses
			patch.AdditionalAddress = &additionalAddresses
		case "replica_set":
			dataSource.ReplicaSet = request.DataSource.ReplicaSet
			patch.ReplicaSet = &request.DataSource.ReplicaSet
		case "direct_connection":
			dataSource.DirectConnection = request.DataSource.DirectConnection
			patch.DirectConnection = &request.DataSource.DirectConnection
		case "region":
			dataSource.Region = request.DataSource.Region
			patch.Region = &request.DataSource.Region
		case "warehouse_id":
			dataSource.WarehouseID = request.DataSource.WarehouseId
			patch.WarehouseID = &request.DataSource.WarehouseId
		case "use_ssl":
			dataSource.UseSSL = request.DataSource.UseSsl
			patch.UseSSL = &request.DataSource.UseSsl
		case "redis_type":
			redisType := convertToStoreRedisType(request.DataSource.RedisType)
			dataSource.RedisType = redisType
			patch.RedisType = &redisType
		case "master_name":
			dataSource.MasterName = request.DataSource.MasterName
			patch.MasterName = &request.DataSource.MasterName
		case "master_username":
			dataSource.MasterUsername = request.DataSource.MasterUsername
			patch.MasterUsername = &request.DataSource.MasterUsername
		case "master_password":
			obfuscated := common.Obfuscate(request.DataSource.MasterPassword, s.secret)
			dataSource.MasterObfuscatedPassword = obfuscated
			patch.MasterObfuscatedPassword = &obfuscated
		case "iam_extension":
			if v := request.DataSource.IamExtension; v != nil {
				switch v := v.(type) {
				case *v1pb.DataSource_ClientSecretCredential_:
					dataSource.ClientSecretCredential = convertToStoreClientSecretCredential(v.ClientSecretCredential)
					patch.ClientSecretCredential = dataSource.ClientSecretCredential
				default:
				}
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupported update_mask "%s"`, path)
		}
	}

	if err := s.checkDataSource(instance, &dataSource); err != nil {
		return nil, err
	}

	if patch.SSHHost != nil || patch.SSHPort != nil || patch.SSHUser != nil || patch.SSHObfuscatedPassword != nil || patch.SSHObfuscatedPrivateKey != nil {
		if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureInstanceSSHConnection, instance); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	// Test connection.
	if request.ValidateOnly {
		err := func() error {
			driver, err := s.dbFactory.GetDataSourceDriver(ctx, instance, &dataSource, "", false /* datashare */, dataSource.Type == api.RO, db.ConnectionContext{})
			if err != nil {
				return status.Errorf(codes.Internal, "failed to get database driver with error: %v", err.Error())
			}
			defer driver.Close(ctx)
			if err := driver.Ping(ctx); err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid datasource %s, error %s", dataSource.Type, err)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
		return convertToInstance(instance)
	}

	if err := s.store.UpdateDataSourceV2(ctx, patch); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instance.ResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return convertToInstance(instance)
}

// RemoveDataSource removes a data source to an instance.
func (s *InstanceService) RemoveDataSource(ctx context.Context, request *v1pb.RemoveDataSourceRequest) (*v1pb.Instance, error) {
	if request.DataSource == nil {
		return nil, status.Errorf(codes.InvalidArgument, "data sources is required")
	}

	instance, err := getInstanceMessage(ctx, s.store, request.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", request.Name)
	}

	// We create a new variable dataSource to not modify existing data source in the memory.
	var dataSource store.DataSourceMessage
	found := false
	for _, ds := range instance.DataSources {
		if ds.ID == request.DataSource.Id {
			dataSource = *ds
			found = true
			break
		}
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, "data source not found")
	}

	// We only support remove RO type datasource to instance now, see more details in instance_service.proto.
	if dataSource.Type != api.RO {
		return nil, status.Errorf(codes.InvalidArgument, "only support remove read-only data source")
	}

	if err := s.store.RemoveDataSourceV2(ctx, instance.ResourceID, dataSource.ID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instance.ResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instance.ResourceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return convertToInstance(instance)
}

func (s *InstanceService) getProjectMessage(ctx context.Context, name string) (*store.ProjectMessage, error) {
	projectID, err := common.GetProjectID(name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID:  &projectID,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", name)
	}

	return project, nil
}

func getInstanceMessage(ctx context.Context, stores *store.Store, name string) (*store.InstanceMessage, error) {
	instanceID, err := common.GetInstanceID(name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	find := &store.FindInstanceMessage{
		ResourceID: &instanceID,
	}
	instance, err := stores.GetInstanceV2(ctx, find)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", name)
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
func buildEnvironmentName(environmentID string) string {
	var b strings.Builder
	b.Grow(len("environments/") + len(environmentID))
	_, _ = b.WriteString("environments/")
	_, _ = b.WriteString(environmentID)
	return b.String()
}

func convertToInstance(instance *store.InstanceMessage) (*v1pb.Instance, error) {
	engine := convertToEngine(instance.Engine)
	dataSourceList, err := convertToV1DataSources(instance.DataSources)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert data source with error: %v", err.Error())
	}

	return &v1pb.Instance{
		Name:          buildInstanceName(instance.ResourceID),
		Title:         instance.Title,
		Engine:        engine,
		EngineVersion: instance.EngineVersion,
		ExternalLink:  instance.ExternalLink,
		DataSources:   dataSourceList,
		State:         convertDeletedToState(instance.Deleted),
		Environment:   buildEnvironmentName(instance.EnvironmentID),
		Activation:    instance.Activation,
		Options:       convertToInstanceOptions(instance.Options),
		Roles:         convertToInstanceRoles(instance, instance.Metadata.GetRoles()),
	}, nil
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

func convertToInstanceRoles(instance *store.InstanceMessage, roles []*storepb.InstanceRole) []*v1pb.InstanceRole {
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

func (s *InstanceService) convertToInstanceMessage(instanceID string, instance *v1pb.Instance) (*store.InstanceMessage, error) {
	datasources, err := s.convertToDataSourceMessages(instance.DataSources)
	if err != nil {
		return nil, err
	}
	environmentID, err := common.GetEnvironmentID(instance.Environment)
	if err != nil {
		return nil, err
	}

	return &store.InstanceMessage{
		ResourceID:    instanceID,
		Title:         instance.Title,
		Engine:        convertEngine(instance.Engine),
		ExternalLink:  instance.ExternalLink,
		DataSources:   datasources,
		EnvironmentID: environmentID,
		Activation:    instance.Activation,
		Options:       convertInstanceOptions(instance.Options),
	}, nil
}

func convertToInstanceResource(instanceMessage *store.InstanceMessage) (*v1pb.InstanceResource, error) {
	instance, err := convertToInstance(instanceMessage)
	if err != nil {
		return nil, err
	}
	return &v1pb.InstanceResource{
		Name:          instance.Name,
		Title:         instance.Title,
		Engine:        instance.Engine,
		EngineVersion: instance.EngineVersion,
		DataSources:   instance.DataSources,
		Activation:    instance.Activation,
		Environment:   instance.Environment,
	}, nil
}

func (s *InstanceService) convertToDataSourceMessages(dataSources []*v1pb.DataSource) ([]*store.DataSourceMessage, error) {
	var datasources []*store.DataSourceMessage
	for _, ds := range dataSources {
		dataSource, err := s.convertToDataSourceMessage(ds)
		if err != nil {
			return nil, err
		}
		datasources = append(datasources, dataSource)
	}

	return datasources, nil
}

func convertToV1DataSourceExternalSecret(externalSecret *storepb.DataSourceExternalSecret) (*v1pb.DataSourceExternalSecret, error) {
	if externalSecret == nil {
		return nil, nil
	}
	secret := new(v1pb.DataSourceExternalSecret)
	if err := convertProtoToProto(externalSecret, secret); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert external secret with error: %v", err.Error())
	}

	resp := &v1pb.DataSourceExternalSecret{
		SecretType:      secret.SecretType,
		Url:             secret.Url,
		AuthType:        secret.AuthType,
		EngineName:      secret.EngineName,
		SecretName:      secret.SecretName,
		PasswordKeyName: secret.PasswordKeyName,
	}

	// clear sensitive data.
	switch resp.AuthType {
	case v1pb.DataSourceExternalSecret_VAULT_APP_ROLE:
		appRole := secret.GetAppRole()
		resp.AuthOption = &v1pb.DataSourceExternalSecret_AppRole{
			AppRole: &v1pb.DataSourceExternalSecret_AppRoleAuthOption{
				Type:      appRole.Type,
				MountPath: appRole.MountPath,
			},
		}
	case v1pb.DataSourceExternalSecret_TOKEN:
		resp.AuthOption = &v1pb.DataSourceExternalSecret_Token{
			Token: "",
		}
	}

	return resp, nil
}

func convertToV1DataSources(dataSources []*store.DataSourceMessage) ([]*v1pb.DataSource, error) {
	dataSourceList := []*v1pb.DataSource{}
	for _, ds := range dataSources {
		externalSecret, err := convertToV1DataSourceExternalSecret(ds.ExternalSecret)
		if err != nil {
			return nil, err
		}

		dataSourceType := v1pb.DataSourceType_DATA_SOURCE_UNSPECIFIED
		switch ds.Type {
		case api.Admin:
			dataSourceType = v1pb.DataSourceType_ADMIN
		case api.RO:
			dataSourceType = v1pb.DataSourceType_READ_ONLY
		}

		authenticationType := v1pb.DataSource_AUTHENTICATION_UNSPECIFIED
		switch ds.AuthenticationType {
		case storepb.DataSourceOptions_AUTHENTICATION_UNSPECIFIED, storepb.DataSourceOptions_PASSWORD:
			authenticationType = v1pb.DataSource_PASSWORD
		case storepb.DataSourceOptions_GOOGLE_CLOUD_SQL_IAM:
			authenticationType = v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM
		case storepb.DataSourceOptions_AWS_RDS_IAM:
			authenticationType = v1pb.DataSource_AWS_RDS_IAM
		case storepb.DataSourceOptions_AZURE_IAM:
			authenticationType = v1pb.DataSource_AZURE_IAM
		}

		dataSource := &v1pb.DataSource{
			Id:       ds.ID,
			Type:     dataSourceType,
			Username: ds.Username,
			// We don't return the password and SSLs on reads.
			Host:                   ds.Host,
			Port:                   ds.Port,
			Database:               ds.Database,
			Srv:                    ds.SRV,
			AuthenticationDatabase: ds.AuthenticationDatabase,
			Sid:                    ds.SID,
			ServiceName:            ds.ServiceName,
			ExternalSecret:         externalSecret,
			AuthenticationType:     authenticationType,
			SaslConfig:             convertToV1DataSourceSaslConfig(ds.SASLConfig),
			AdditionalAddresses:    convertToV1DataSourceAddresses(ds.AdditionalAddresses),
			ReplicaSet:             ds.ReplicaSet,
			DirectConnection:       ds.DirectConnection,
			Region:                 ds.Region,
			WarehouseId:            ds.WarehouseID,
			UseSsl:                 ds.UseSSL,
			RedisType:              convertToV1RedisType(ds.RedisType),
			MasterName:             ds.MasterName,
			MasterUsername:         ds.MasterUsername,
		}
		if clientSecretCredential := convertToV1ClientSecretCredential(ds.ClientSecretCredential); clientSecretCredential != nil {
			dataSource.IamExtension = &v1pb.DataSource_ClientSecretCredential_{
				ClientSecretCredential: clientSecretCredential,
			}
		}

		dataSourceList = append(dataSourceList, dataSource)
	}

	return dataSourceList, nil
}

func convertToV1ClientSecretCredential(clientSecretCredential *storepb.DataSourceOptions_ClientSecretCredential) *v1pb.DataSource_ClientSecretCredential {
	if clientSecretCredential == nil {
		return nil
	}
	return &v1pb.DataSource_ClientSecretCredential{
		TenantId:     clientSecretCredential.TenantId,
		ClientId:     clientSecretCredential.ClientId,
		ClientSecret: clientSecretCredential.ClientSecret,
	}
}

func convertToStoreDataSourceExternalSecret(externalSecret *v1pb.DataSourceExternalSecret) (*storepb.DataSourceExternalSecret, error) {
	if externalSecret == nil {
		return nil, nil
	}
	secret := new(storepb.DataSourceExternalSecret)
	if err := convertProtoToProto(externalSecret, secret); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert external secret with error: %v", err.Error())
	}
	switch secret.SecretType {
	case storepb.DataSourceExternalSecret_VAULT_KV_V2:
		if secret.Url == "" {
			return nil, status.Errorf(codes.InvalidArgument, "missing Vault URL")
		}
		if secret.EngineName == "" {
			return nil, status.Errorf(codes.InvalidArgument, "missing Vault engine name")
		}
		if secret.SecretName == "" || secret.PasswordKeyName == "" {
			return nil, status.Errorf(codes.InvalidArgument, "missing secret name or key name")
		}
	case storepb.DataSourceExternalSecret_AWS_SECRETS_MANAGER:
		if secret.SecretName == "" || secret.PasswordKeyName == "" {
			return nil, status.Errorf(codes.InvalidArgument, "missing secret name or key name")
		}
	case storepb.DataSourceExternalSecret_GCP_SECRET_MANAGER:
		if secret.SecretName == "" {
			return nil, status.Errorf(codes.InvalidArgument, "missing GCP secret name")
		}
	}

	switch secret.AuthType {
	case storepb.DataSourceExternalSecret_TOKEN:
		if secret.GetToken() == "" {
			return nil, status.Errorf(codes.InvalidArgument, "missing token")
		}
	case storepb.DataSourceExternalSecret_VAULT_APP_ROLE:
		if secret.GetAppRole() == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing Vault approle")
		}
	}

	return secret, nil
}

func convertToStoreDataSourceSaslConfig(saslConfig *v1pb.SASLConfig) *storepb.SASLConfig {
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

func convertToV1DataSourceSaslConfig(saslConfig *storepb.SASLConfig) *v1pb.SASLConfig {
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

func convertToV1DataSourceAddresses(addresses []*storepb.DataSourceOptions_Address) []*v1pb.DataSource_Address {
	res := make([]*v1pb.DataSource_Address, 0, len(addresses))
	for _, address := range addresses {
		res = append(res, &v1pb.DataSource_Address{
			Host: address.Host,
			Port: address.Port,
		})
	}
	return res
}

func convertToStoreAdditionalAddresses(addresses []*v1pb.DataSource_Address) []*storepb.DataSourceOptions_Address {
	res := make([]*storepb.DataSourceOptions_Address, 0, len(addresses))
	for _, address := range addresses {
		res = append(res, &storepb.DataSourceOptions_Address{
			Host: address.Host,
			Port: address.Port,
		})
	}
	return res
}

func convertToAuthenticationType(authType v1pb.DataSource_AuthenticationType) storepb.DataSourceOptions_AuthenticationType {
	authenticationType := storepb.DataSourceOptions_AUTHENTICATION_UNSPECIFIED
	switch authType {
	case v1pb.DataSource_AUTHENTICATION_UNSPECIFIED, v1pb.DataSource_PASSWORD:
		authenticationType = storepb.DataSourceOptions_PASSWORD
	case v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		authenticationType = storepb.DataSourceOptions_GOOGLE_CLOUD_SQL_IAM
	case v1pb.DataSource_AWS_RDS_IAM:
		authenticationType = storepb.DataSourceOptions_AWS_RDS_IAM
	case v1pb.DataSource_AZURE_IAM:
		authenticationType = storepb.DataSourceOptions_AZURE_IAM
	}
	return authenticationType
}

func convertToStoreRedisType(redisType v1pb.DataSource_RedisType) storepb.DataSourceOptions_RedisType {
	authenticationType := storepb.DataSourceOptions_REDIS_TYPE_UNSPECIFIED
	switch redisType {
	case v1pb.DataSource_STANDALONE:
		authenticationType = storepb.DataSourceOptions_STANDALONE
	case v1pb.DataSource_SENTINEL:
		authenticationType = storepb.DataSourceOptions_SENTINEL
	case v1pb.DataSource_CLUSTER:
		authenticationType = storepb.DataSourceOptions_CLUSTER
	}
	return authenticationType
}

func convertToV1RedisType(redisType storepb.DataSourceOptions_RedisType) v1pb.DataSource_RedisType {
	authenticationType := v1pb.DataSource_STANDALONE
	switch redisType {
	case storepb.DataSourceOptions_STANDALONE:
		authenticationType = v1pb.DataSource_STANDALONE
	case storepb.DataSourceOptions_SENTINEL:
		authenticationType = v1pb.DataSource_SENTINEL
	case storepb.DataSourceOptions_CLUSTER:
		authenticationType = v1pb.DataSource_CLUSTER
	}
	return authenticationType
}

func (s *InstanceService) convertToDataSourceMessage(dataSource *v1pb.DataSource) (*store.DataSourceMessage, error) {
	dsType, err := convertDataSourceTp(dataSource.Type)
	if err != nil {
		return nil, err
	}
	externalSecret, err := convertToStoreDataSourceExternalSecret(dataSource.ExternalSecret)
	if err != nil {
		return nil, err
	}
	saslConfig := convertToStoreDataSourceSaslConfig(dataSource.SaslConfig)
	clientSecretCredential := convertToStoreClientSecretCredential(dataSource.GetClientSecretCredential())

	return &store.DataSourceMessage{
		ID:                                 dataSource.Id,
		Type:                               dsType,
		Username:                           dataSource.Username,
		ObfuscatedPassword:                 common.Obfuscate(dataSource.Password, s.secret),
		ObfuscatedSslCa:                    common.Obfuscate(dataSource.SslCa, s.secret),
		ObfuscatedSslCert:                  common.Obfuscate(dataSource.SslCert, s.secret),
		ObfuscatedSslKey:                   common.Obfuscate(dataSource.SslKey, s.secret),
		Host:                               dataSource.Host,
		Port:                               dataSource.Port,
		Database:                           dataSource.Database,
		SRV:                                dataSource.Srv,
		AuthenticationDatabase:             dataSource.AuthenticationDatabase,
		SID:                                dataSource.Sid,
		ServiceName:                        dataSource.ServiceName,
		SSHHost:                            dataSource.SshHost,
		SSHPort:                            dataSource.SshPort,
		SSHUser:                            dataSource.SshUser,
		SSHObfuscatedPassword:              common.Obfuscate(dataSource.SshPassword, s.secret),
		SSHObfuscatedPrivateKey:            common.Obfuscate(dataSource.SshPrivateKey, s.secret),
		AuthenticationPrivateKeyObfuscated: common.Obfuscate(dataSource.AuthenticationPrivateKey, s.secret),
		ExternalSecret:                     externalSecret,
		SASLConfig:                         saslConfig,
		AuthenticationType:                 convertToAuthenticationType(dataSource.AuthenticationType),
		AdditionalAddresses:                convertToStoreAdditionalAddresses(dataSource.AdditionalAddresses),
		ReplicaSet:                         dataSource.ReplicaSet,
		DirectConnection:                   dataSource.DirectConnection,
		Region:                             dataSource.Region,
		WarehouseID:                        dataSource.WarehouseId,
		UseSSL:                             dataSource.UseSsl,
		RedisType:                          convertToStoreRedisType(dataSource.RedisType),
		MasterName:                         dataSource.MasterName,
		MasterUsername:                     dataSource.MasterUsername,
		MasterObfuscatedPassword:           common.Obfuscate(dataSource.MasterPassword, s.secret),
		ClientSecretCredential:             clientSecretCredential,
	}, nil
}

func convertToStoreClientSecretCredential(credential *v1pb.DataSource_ClientSecretCredential) *storepb.DataSourceOptions_ClientSecretCredential {
	if credential == nil {
		return nil
	}
	return &storepb.DataSourceOptions_ClientSecretCredential{
		TenantId:     credential.TenantId,
		ClientId:     credential.ClientId,
		ClientSecret: credential.ClientSecret,
	}
}

func (s *InstanceService) instanceCountGuard(ctx context.Context) error {
	instanceLimit := s.licenseService.GetPlanLimitValue(ctx, enterprise.PlanLimitMaximumInstance)

	count, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if count >= instanceLimit {
		return status.Errorf(codes.ResourceExhausted, "reached the maximum instance count %d", instanceLimit)
	}

	return nil
}

func convertDataSourceTp(tp v1pb.DataSourceType) (api.DataSourceType, error) {
	var dsType api.DataSourceType
	switch tp {
	case v1pb.DataSourceType_READ_ONLY:
		dsType = api.RO
	case v1pb.DataSourceType_ADMIN:
		dsType = api.Admin
	default:
		return "", errors.Errorf("invalid data source type %v", tp)
	}
	return dsType, nil
}

func convertToInstanceOptions(options *storepb.InstanceOptions) *v1pb.InstanceOptions {
	if options == nil {
		return &v1pb.InstanceOptions{}
	}

	return &v1pb.InstanceOptions{
		SyncInterval:       options.SyncInterval,
		MaximumConnections: options.MaximumConnections,
		SyncDatabases:      options.GetSyncDatabases(),
	}
}

func convertInstanceOptions(options *v1pb.InstanceOptions) *storepb.InstanceOptions {
	if options == nil {
		return &storepb.InstanceOptions{}
	}

	return &storepb.InstanceOptions{
		SyncInterval:       options.SyncInterval,
		MaximumConnections: options.MaximumConnections,
		SyncDatabases:      options.GetSyncDatabases(),
	}
}
