package v1

import (
	"context"
	"fmt"
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
	var project *store.ProjectMessage
	if request.Parent != "" {
		p, err := s.getProjectMessage(ctx, request.Parent)
		if err != nil {
			return nil, err
		}
		if p.Deleted {
			return nil, status.Errorf(codes.NotFound, "project %q has been deleted", request.Parent)
		}
		project = p
	}
	find := &store.FindInstanceMessage{
		ShowDeleted: request.ShowDeleted,
	}
	if project != nil {
		find.ProjectUID = &project.UID
	}
	instances, err := s.store.ListInstancesV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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

// SearchInstance searches for instances.
func (s *InstanceService) SearchInstances(ctx context.Context, request *v1pb.SearchInstancesRequest) (*v1pb.SearchInstancesResponse, error) {
	var project *store.ProjectMessage
	if request.Parent != "" {
		p, err := s.getProjectMessage(ctx, request.Parent)
		if err != nil {
			return nil, err
		}
		if p.Deleted {
			return nil, status.Errorf(codes.NotFound, "project %q has been deleted", request.Parent)
		}
		project = p
	}

	databaseFind := &store.FindDatabaseMessage{}
	if project != nil {
		databaseFind.ProjectID = &project.ResourceID
	}

	databases, err := searchDatabases(ctx, s.store, s.iamManager, databaseFind, iam.PermissionDatabasesGet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get databases, error: %v", err)
	}

	instanceResourceIDsSet := make(map[string]struct{})
	for _, db := range databases {
		instanceResourceIDsSet[db.InstanceID] = struct{}{}
	}
	var instanceResourceIDs []string
	for id := range instanceResourceIDsSet {
		instanceResourceIDs = append(instanceResourceIDs, id)
	}

	instances, err := s.store.ListInstancesV2(ctx, &store.FindInstanceMessage{
		ResourceIDs: &instanceResourceIDs,
		ShowDeleted: request.ShowDeleted,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.SearchInstancesResponse{}
	for _, instance := range instances {
		ins, err := convertToInstance(instance)
		if err != nil {
			return nil, err
		}
		response.Instances = append(response.Instances, ins)
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
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
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
				return nil, status.Errorf(codes.ResourceExhausted, err.Error())
			}
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	if err := s.checkInstanceDataSources(instanceMessage, instanceMessage.DataSources); err != nil {
		return nil, err
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	instance, err := s.store.CreateInstanceV2(ctx,
		instanceMessage,
		principalID,
		instanceCountLimit,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
	if err == nil {
		defer driver.Close(ctx)
		updatedInstance, err := s.schemaSyncer.SyncInstance(ctx, instance)
		if err != nil {
			slog.Warn("Failed to sync instance",
				slog.String("instance", instance.ResourceID),
				log.BBError(err))
		} else {
			instance = updatedInstance
		}
		// Sync all databases in the instance asynchronously.
		s.stateCfg.InstanceSyncs.Store(instance.UID, instance)
		select {
		case s.stateCfg.InstanceSyncTickleChan <- 0:
		default:
		}
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
		return status.Errorf(codes.Internal, err.Error())
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureExternalSecretManager, instance); err != nil {
		missingFeatureError := status.Errorf(codes.PermissionDenied, err.Error())
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

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	patch := &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
		UpdaterID:  principalID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &request.Instance.Title
		case "environment":
			patch.UpdateEnvironmentID = true
			if request.Instance.Environment != "" {
				environmentID, err := common.GetEnvironmentID(request.Instance.Environment)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, err.Error())
				}
				environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
					ResourceID:  &environmentID,
					ShowDeleted: true,
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, err.Error())
				}
				if environment == nil {
					return nil, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
				}
				if environment.Deleted {
					return nil, status.Errorf(codes.FailedPrecondition, "environment %q is deleted", environmentID)
				}
				patch.EnvironmentID = environment.ResourceID
			}
		case "external_link":
			patch.ExternalLink = &request.Instance.ExternalLink
		case "data_sources":
			datasources, err := s.convertToDataSourceMessages(request.Instance.DataSources)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			if err := s.checkInstanceDataSources(instance, datasources); err != nil {
				return nil, err
			}
			patch.DataSources = &datasources
		case "activation":
			patch.Activation = &request.Instance.Activation
		case "options.sync_interval":
			if patch.OptionsUpsert == nil {
				patch.OptionsUpsert = instance.Options
			}
			patch.OptionsUpsert.SyncInterval = request.Instance.Options.GetSyncInterval()
		case "options.maximum_connections":
			if patch.OptionsUpsert == nil {
				patch.OptionsUpsert = instance.Options
			}
			patch.OptionsUpsert.MaximumConnections = request.Instance.Options.GetMaximumConnections()
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupported update_mask "%s"`, path)
		}
	}

	instanceCountLimit := s.licenseService.GetInstanceLicenseCount(ctx)
	if v := patch.Activation; v != nil && *v {
		if err := s.store.CheckActivationLimit(ctx, instanceCountLimit); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return nil, status.Errorf(codes.ResourceExhausted, err.Error())
			}
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	ins, err := s.store.UpdateInstanceV2(ctx, patch, instanceCountLimit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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

	slowQueryPolicy, err := s.store.GetSlowQueryPolicy(ctx, api.PolicyResourceTypeInstance, instance.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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
		if project != nil {
			findDatabase.ProjectID = &project.ResourceID
		}
		databases, err := s.store.ListDatabases(ctx, findDatabase)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to list databases: %s", err.Error())
		}

		var enabledDatabases []*store.DatabaseMessage
		for _, database := range databases {
			if database.SyncState != api.OK {
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

			enabledDatabases = append(enabledDatabases, database)
		}

		if len(enabledDatabases) == 0 {
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
	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &project.ResourceID})
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

			slowQueryPolicy, err := s.store.GetSlowQueryPolicy(ctx, api.PolicyResourceTypeInstance, instance.UID)
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
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
			if _, err := s.store.BatchUpdateDatabaseProject(ctx, databases, api.DefaultProjectID, api.SystemBotID); err != nil {
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

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if _, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
		Delete:     &deletePatch,
		UpdaterID:  principalID,
	}, -1 /* don't need to pass the instance limition */); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	ins, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
		Delete:     &undeletePatch,
		UpdaterID:  principalID,
	}, -1 /* don't need to pass the instance limition */)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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

	updatedInstance, err := s.schemaSyncer.SyncInstance(ctx, instance)
	if err != nil {
		return nil, err
	}
	// Sync all databases in the instance asynchronously.
	s.stateCfg.InstanceSyncs.Store(instance.UID, updatedInstance)
	s.stateCfg.InstanceSyncTickleChan <- 0

	return &v1pb.SyncInstanceResponse{}, nil
}

// SyncInstance syncs the instance.
func (s *InstanceService) BatchSyncInstance(ctx context.Context, request *v1pb.BatchSyncInstanceRequest) (*v1pb.BatchSyncInstanceResponse, error) {
	for _, r := range request.Requests {
		instance, err := getInstanceMessage(ctx, s.store, r.Name)
		if err != nil {
			return nil, err
		}
		if instance.Deleted {
			return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", r.Name)
		}
		// Sync all databases in the instance asynchronously.
		select {
		case s.stateCfg.InstanceSyncTickleChan <- 0:
		default:
		}
	}
	select {
	case s.stateCfg.InstanceSyncTickleChan <- 0:
	default:
	}

	return &v1pb.BatchSyncInstanceResponse{}, nil
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

	instance, err := getInstanceMessage(ctx, s.store, request.Instance)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", request.Instance)
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
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if err := s.store.AddDataSourceToInstanceV2(ctx, instance.UID, principalID, instance.ResourceID, dataSource); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		UID: &instance.UID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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

	instance, err := getInstanceMessage(ctx, s.store, request.Instance)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", request.Instance)
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

	if dataSource.Type == api.RO {
		if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureReadReplicaConnection, instance); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	patch := &store.UpdateDataSourceMessage{
		UpdaterID:    principalID,
		InstanceUID:  instance.UID,
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
		case "direct_connection":
			dataSource.DirectConnection = request.DataSource.DirectConnection
		case "region":
			dataSource.Region = request.DataSource.Region
		case "account_id":
			dataSource.AccountID = request.DataSource.AccountId
			patch.AccountID = &request.DataSource.AccountId
		case "warehouse_id":
			dataSource.WarehouseID = request.DataSource.WarehouseId
			patch.WarehouseID = &request.DataSource.WarehouseId
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupport update_mask "%s"`, path)
		}
	}

	if err := s.checkDataSource(instance, &dataSource); err != nil {
		return nil, err
	}

	if patch.SSHHost != nil || patch.SSHPort != nil || patch.SSHUser != nil || patch.SSHObfuscatedPassword != nil || patch.SSHObfuscatedPrivateKey != nil {
		if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureInstanceSSHConnection, instance); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
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
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		UID: &instance.UID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToInstance(instance)
}

// RemoveDataSource removes a data source to an instance.
func (s *InstanceService) RemoveDataSource(ctx context.Context, request *v1pb.RemoveDataSourceRequest) (*v1pb.Instance, error) {
	if request.DataSource == nil {
		return nil, status.Errorf(codes.InvalidArgument, "data sources is required")
	}

	instance, err := getInstanceMessage(ctx, s.store, request.Instance)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.NotFound, "instance %q has been deleted", request.Instance)
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

	if err := s.store.RemoveDataSourceV2(ctx, instance.UID, instance.ResourceID, dataSource.ID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instance.ResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		UID: &instance.UID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToInstance(instance)
}

func (s *InstanceService) getProjectMessage(ctx context.Context, name string) (*store.ProjectMessage, error) {
	projectID, err := common.GetProjectID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	var project *store.ProjectMessage
	projectUID, isNumber := isNumber(projectID)
	if isNumber {
		project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			UID:         &projectUID,
			ShowDeleted: true,
		})
	} else {
		project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID:  &projectID,
			ShowDeleted: true,
		})
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", name)
	}

	return project, nil
}

func getInstanceMessage(ctx context.Context, stores *store.Store, name string) (*store.InstanceMessage, error) {
	instanceID, err := common.GetInstanceID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	find := &store.FindInstanceMessage{}
	instanceUID, isNumber := isNumber(instanceID)
	if isNumber {
		find.UID = &instanceUID
	} else {
		find.ResourceID = &instanceID
	}

	instance, err := stores.GetInstanceV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", name)
	}

	return instance, nil
}

func convertToInstance(instance *store.InstanceMessage) (*v1pb.Instance, error) {
	engine := convertToEngine(instance.Engine)
	dataSourceList, err := convertToV1DataSources(instance.DataSources)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert data source with error: %v", err.Error())
	}

	return &v1pb.Instance{
		Name:          fmt.Sprintf("%s%s", common.InstanceNamePrefix, instance.ResourceID),
		Uid:           fmt.Sprintf("%d", instance.UID),
		Title:         instance.Title,
		Engine:        engine,
		EngineVersion: instance.EngineVersion,
		ExternalLink:  instance.ExternalLink,
		DataSources:   dataSourceList,
		State:         convertDeletedToState(instance.Deleted),
		Environment:   fmt.Sprintf("environments/%s", instance.EnvironmentID),
		Activation:    instance.Activation,
		Options:       convertToInstanceOptions(instance.Options),
	}, nil
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
		Title:         instance.Title,
		Engine:        instance.Engine,
		EngineVersion: instance.EngineVersion,
		DataSources:   instance.DataSources,
		Activation:    instance.Activation,
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
	if err := convertV1PbToStorePb(externalSecret, secret); err != nil {
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
		}

		dataSourceList = append(dataSourceList, &v1pb.DataSource{
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
		})
	}

	return dataSourceList, nil
}

func convertToStoreDataSourceExternalSecret(externalSecret *v1pb.DataSourceExternalSecret) (*storepb.DataSourceExternalSecret, error) {
	if externalSecret == nil {
		return nil, nil
	}
	secret := new(storepb.DataSourceExternalSecret)
	if err := convertV1PbToStorePb(externalSecret, secret); err != nil {
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
		AccountID:                          dataSource.AccountId,
		WarehouseID:                        dataSource.WarehouseId,
	}, nil
}

func (s *InstanceService) instanceCountGuard(ctx context.Context) error {
	instanceLimit := s.licenseService.GetPlanLimitValue(ctx, enterprise.PlanLimitMaximumInstance)

	count, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{})
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if int64(count) >= instanceLimit {
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
		return nil
	}

	return &v1pb.InstanceOptions{
		SyncInterval:       options.SyncInterval,
		MaximumConnections: options.MaximumConnections,
	}
}

func convertInstanceOptions(options *v1pb.InstanceOptions) *storepb.InstanceOptions {
	if options == nil {
		return nil
	}

	return &storepb.InstanceOptions{
		SyncInterval:       options.SyncInterval,
		MaximumConnections: options.MaximumConnections,
	}
}
