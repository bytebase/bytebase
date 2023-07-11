package v1

import (
	"context"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/metric"
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
	licenseService enterpriseAPI.LicenseService
	metricReporter *metricreport.Reporter
	secret         string
	stateCfg       *state.State
	dbFactory      *dbfactory.DBFactory
	schemaSyncer   *schemasync.Syncer
}

// NewInstanceService creates a new InstanceService.
func NewInstanceService(store *store.Store, licenseService enterpriseAPI.LicenseService, metricReporter *metricreport.Reporter, secret string, stateCfg *state.State, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer) *InstanceService {
	return &InstanceService{
		store:          store,
		licenseService: licenseService,
		metricReporter: metricReporter,
		secret:         secret,
		stateCfg:       stateCfg,
		dbFactory:      dbFactory,
		schemaSyncer:   schemaSyncer,
	}
}

// GetInstance gets an instance.
func (s *InstanceService) GetInstance(ctx context.Context, request *v1pb.GetInstanceRequest) (*v1pb.Instance, error) {
	instance, err := s.getInstanceMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	return convertToInstance(instance), nil
}

// ListInstances lists all instances.
func (s *InstanceService) ListInstances(ctx context.Context, request *v1pb.ListInstancesRequest) (*v1pb.ListInstancesResponse, error) {
	find := &store.FindInstanceMessage{
		ShowDeleted: request.ShowDeleted,
	}
	instances, err := s.store.ListInstancesV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListInstancesResponse{}
	for _, instance := range instances {
		response.Instances = append(response.Instances, convertToInstance(instance))
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
				driver, err := s.dbFactory.GetDataSourceDriver(ctx, instanceMessage.Engine, ds, "", "", 0, false /* datashare */, ds.Type == api.RO, false /* schemaTenantMode */)
				if err != nil {
					return err
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
		return convertToInstance(instanceMessage), nil
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

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	instance, err := s.store.CreateInstanceV2(ctx,
		instanceMessage,
		principalID,
		instanceCountLimit,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err == nil {
		defer driver.Close(ctx)
		if _, err := s.schemaSyncer.SyncInstance(ctx, instance); err != nil {
			log.Warn("Failed to sync instance",
				zap.String("instance", instance.ResourceID),
				zap.Error(err))
		}
		// Sync all databases in the instance asynchronously.
		s.stateCfg.InstanceDatabaseSyncChan <- instance
	}

	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricAPI.InstanceCreateMetricName,
		Value: 1,
		Labels: map[string]any{
			"engine": instance.Engine,
		},
	})

	return convertToInstance(instance), nil
}

// UpdateInstance updates an instance.
func (s *InstanceService) UpdateInstance(ctx context.Context, request *v1pb.UpdateInstanceRequest) (*v1pb.Instance, error) {
	if request.Instance == nil {
		return nil, status.Errorf(codes.InvalidArgument, "instance must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	instance, err := s.getInstanceMessage(ctx, request.Instance.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", request.Instance.Name)
	}

	patch := &store.UpdateInstanceMessage{
		UpdaterID:     ctx.Value(common.PrincipalIDContextKey).(int),
		EnvironmentID: instance.EnvironmentID,
		ResourceID:    instance.ResourceID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &request.Instance.Title
		case "external_link":
			patch.ExternalLink = &request.Instance.ExternalLink
		case "data_sources":
			datasourceList, err := s.convertToDataSourceMessages(request.Instance.DataSources)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			patch.DataSources = &datasourceList
		case "activation":
			patch.Activation = &request.Instance.Activation
		case "options.schema_tenant_mode":
			if patch.Options == nil {
				patch.Options = &storepb.InstanceOptions{
					SchemaTenantMode: request.Instance.Options.SchemaTenantMode,
				}
			} else {
				patch.Options.SchemaTenantMode = request.Instance.Options.SchemaTenantMode
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupport update_mask "%s"`, path)
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

	// TODO(d): sync instance databases.

	return convertToInstance(ins), nil
}

// SyncSlowQueries syncs slow queries for an instance.
func (s *InstanceService) SyncSlowQueries(ctx context.Context, request *v1pb.SyncSlowQueriesRequest) (*emptypb.Empty, error) {
	instance, err := s.getInstanceMessage(ctx, request.Instance)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", request.Instance)
	}

	slowQueryPolicy, err := s.store.GetSlowQueryPolicy(ctx, api.PolicyResourceTypeInstance, instance.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if slowQueryPolicy == nil || !slowQueryPolicy.Active {
		return nil, status.Errorf(codes.FailedPrecondition, "slow query policy is not active for instance %q", request.Instance)
	}

	switch instance.Engine {
	case db.MySQL:
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)
		if err := driver.CheckSlowQueryLogEnabled(ctx); err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "slow query log is not enabled: %s", err.Error())
		}

		// Sync slow queries for instance.
		s.stateCfg.InstanceSlowQuerySyncChan <- instance.ResourceID
	case db.Postgres:
		databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{
			InstanceID: &instance.ResourceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list databases: %s", err.Error())
		}

		var firstDatabase string
		var errs error
		for _, database := range databases {
			if database.SyncState != api.OK {
				continue
			}
			if _, exists := pg.ExcludedDatabaseList[database.DatabaseName]; exists {
				continue
			}
			if err := func() error {
				driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
				if err != nil {
					return err
				}
				defer driver.Close(ctx)
				return driver.CheckSlowQueryLogEnabled(ctx)
			}(); err != nil {
				errs = multierr.Append(errs, err)
			}

			if firstDatabase == "" {
				firstDatabase = database.DatabaseName
			}
		}

		if errs != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "slow query log is not enabled: %s", errs.Error())
		}

		if firstDatabase == "" {
			return nil, status.Errorf(codes.FailedPrecondition, "no database enabled pg_stat_statements")
		}

		// Sync slow queries for instance.
		s.stateCfg.InstanceSlowQuerySyncChan <- instance.ResourceID
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported engine %q", instance.Engine)
	}

	return &emptypb.Empty{}, nil
}

// DeleteInstance deletes an instance.
func (s *InstanceService) DeleteInstance(ctx context.Context, request *v1pb.DeleteInstanceRequest) (*emptypb.Empty, error) {
	instance, err := s.getInstanceMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", request.Name)
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
	if err != nil {
		return nil, err
	}
	if request.Force {
		if _, err := s.store.BatchUpdateDatabaseProject(ctx, databases, api.DefaultProjectID, api.SystemBotID); err != nil {
			return nil, err
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
		UpdaterID:     ctx.Value(common.PrincipalIDContextKey).(int),
		EnvironmentID: instance.EnvironmentID,
		ResourceID:    instance.ResourceID,
		Delete:        &deletePatch,
	}, -1 /* don't need to pass the instance limition */); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

// UndeleteInstance undeletes an instance.
func (s *InstanceService) UndeleteInstance(ctx context.Context, request *v1pb.UndeleteInstanceRequest) (*v1pb.Instance, error) {
	instance, err := s.getInstanceMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if !instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q is active", request.Name)
	}

	ins, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		UpdaterID:     ctx.Value(common.PrincipalIDContextKey).(int),
		EnvironmentID: instance.EnvironmentID,
		ResourceID:    instance.ResourceID,
		Delete:        &undeletePatch,
	}, -1 /* don't need to pass the instance limition */)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToInstance(ins), nil
}

// SyncInstance syncs the instance.
func (s *InstanceService) SyncInstance(ctx context.Context, request *v1pb.SyncInstanceRequest) (*v1pb.SyncInstanceResponse, error) {
	instance, err := s.getInstanceMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", request.Name)
	}

	if _, err := s.schemaSyncer.SyncInstance(ctx, instance); err != nil {
		return nil, err
	}
	// Sync all databases in the instance asynchronously.
	s.stateCfg.InstanceDatabaseSyncChan <- instance

	return &v1pb.SyncInstanceResponse{}, nil
}

// AddDataSource adds a data source to an instance.
func (s *InstanceService) AddDataSource(ctx context.Context, request *v1pb.AddDataSourceRequest) (*v1pb.Instance, error) {
	if request.DataSource == nil {
		return nil, status.Errorf(codes.InvalidArgument, "data sources is required")
	}
	// We only support add RO type datasouce to instance now, see more details in instance_service.proto.
	if request.DataSource.Type != v1pb.DataSourceType_READ_ONLY {
		return nil, status.Errorf(codes.InvalidArgument, "only support add read-only data source")
	}

	dataSource, err := s.convertToDataSourceMessage(request.DataSource)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert data source")
	}

	instance, err := s.getInstanceMessage(ctx, request.Instance)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", request.Instance)
	}

	// Test connection.
	if request.ValidateOnly {
		err := func() error {
			driver, err := s.dbFactory.GetDataSourceDriver(ctx, instance.Engine, dataSource, "", "", 0, false /* datashare */, dataSource.Type == api.RO, false /* schemaTenantMode */)
			if err != nil {
				return err
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
		return convertToInstance(instance), nil
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureReadReplicaConnection, instance); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if err := s.store.AddDataSourceToInstanceV2(ctx, instance.UID, principalID, instance.ResourceID, dataSource); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		UID: &instance.UID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToInstance(instance), nil
}

// UpdateDataSource updates a data source of an instance.
func (s *InstanceService) UpdateDataSource(ctx context.Context, request *v1pb.UpdateDataSourceRequest) (*v1pb.Instance, error) {
	if request.DataSource == nil {
		return nil, status.Errorf(codes.InvalidArgument, "datasource is required")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	tp, err := convertDataSourceTp(request.DataSource.Type)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.getInstanceMessage(ctx, request.Instance)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", request.Instance)
	}

	if tp == api.RO {
		if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureReadReplicaConnection, instance); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
	}

	// We create a new variable dataSource to not modify existing data source in the memory.
	var dataSource store.DataSourceMessage
	found := false
	for _, ds := range instance.DataSources {
		if ds.Type == tp {
			dataSource = *ds
			found = true
			break
		}
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, "data source not found")
	}

	patch := &store.UpdateDataSourceMessage{
		UpdaterID:   ctx.Value(common.PrincipalIDContextKey).(int),
		InstanceUID: instance.UID,
		Type:        tp,
		InstanceID:  instance.ResourceID,
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
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupport update_mask "%s"`, path)
		}
	}

	if patch.SSHHost != nil || patch.SSHPort != nil || patch.SSHUser != nil || patch.SSHObfuscatedPassword != nil || patch.SSHObfuscatedPrivateKey != nil {
		if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureInstanceSSHConnection, instance); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
	}

	// Test connection.
	if request.ValidateOnly {
		err := func() error {
			driver, err := s.dbFactory.GetDataSourceDriver(ctx, instance.Engine, &dataSource, "", "", 0, false /* datashare */, dataSource.Type == api.RO, false /* schemaTenantMode */)
			if err != nil {
				return err
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
		return convertToInstance(instance), nil
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

	return convertToInstance(instance), nil
}

// RemoveDataSource removes a data source to an instance.
func (s *InstanceService) RemoveDataSource(ctx context.Context, request *v1pb.RemoveDataSourceRequest) (*v1pb.Instance, error) {
	if request.DataSource == nil {
		return nil, status.Errorf(codes.InvalidArgument, "data sources is required")
	}
	// We only support remove RO type datasource to instance now, see more details in instance_service.proto.
	if request.DataSource.Type != v1pb.DataSourceType_READ_ONLY {
		return nil, status.Errorf(codes.InvalidArgument, "only support remove read-only data source")
	}

	dataSource, err := s.convertToDataSourceMessage(request.DataSource)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert data source")
	}

	instance, err := s.getInstanceMessage(ctx, request.Instance)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", request.Instance)
	}

	if err := s.store.RemoveDataSourceV2(ctx, instance.UID, instance.ResourceID, dataSource.Type); err != nil {
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

	return convertToInstance(instance), nil
}

func (s *InstanceService) getInstanceMessage(ctx context.Context, name string) (*store.InstanceMessage, error) {
	instanceID, err := getInstanceID(name)
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

	instance, err := s.store.GetInstanceV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", name)
	}

	return instance, nil
}

func convertToInstance(instance *store.InstanceMessage) *v1pb.Instance {
	engine := convertToEngine(instance.Engine)
	dataSourceList := []*v1pb.DataSource{}
	for _, ds := range instance.DataSources {
		dataSourceType := v1pb.DataSourceType_DATA_SOURCE_UNSPECIFIED
		switch ds.Type {
		case api.Admin:
			dataSourceType = v1pb.DataSourceType_ADMIN
		case api.RO:
			dataSourceType = v1pb.DataSourceType_READ_ONLY
		}

		dataSourceList = append(dataSourceList, &v1pb.DataSource{
			Title:    ds.Title,
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
		})
	}
	var options *v1pb.InstanceOptions
	if instance.Options != nil {
		options = &v1pb.InstanceOptions{
			SchemaTenantMode: instance.Options.SchemaTenantMode,
		}
	}

	return &v1pb.Instance{
		Name:          fmt.Sprintf("%s%s", instanceNamePrefix, instance.ResourceID),
		Uid:           fmt.Sprintf("%d", instance.UID),
		Title:         instance.Title,
		Engine:        engine,
		EngineVersion: instance.EngineVersion,
		ExternalLink:  instance.ExternalLink,
		DataSources:   dataSourceList,
		State:         convertDeletedToState(instance.Deleted),
		Environment:   fmt.Sprintf("environments/%s", instance.EnvironmentID),
		Activation:    instance.Activation,
		Options:       options,
	}
}

func (s *InstanceService) convertToInstanceMessage(instanceID string, instance *v1pb.Instance) (*store.InstanceMessage, error) {
	datasources, err := s.convertToDataSourceMessages(instance.DataSources)
	if err != nil {
		return nil, err
	}
	environmentID, err := getEnvironmentID(instance.Environment)
	if err != nil {
		return nil, err
	}
	var options *storepb.InstanceOptions
	if instance.Options != nil {
		options = &storepb.InstanceOptions{
			SchemaTenantMode: instance.Options.SchemaTenantMode,
		}
	}

	return &store.InstanceMessage{
		ResourceID:    instanceID,
		Title:         instance.Title,
		Engine:        convertEngine(instance.Engine),
		ExternalLink:  instance.ExternalLink,
		DataSources:   datasources,
		EnvironmentID: environmentID,
		Activation:    instance.Activation,
		Options:       options,
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

func (s *InstanceService) convertToDataSourceMessage(dataSource *v1pb.DataSource) (*store.DataSourceMessage, error) {
	dsType, err := convertDataSourceTp(dataSource.Type)
	if err != nil {
		return nil, err
	}

	return &store.DataSourceMessage{
		Title:                   dataSource.Title,
		Type:                    dsType,
		Username:                dataSource.Username,
		ObfuscatedPassword:      common.Obfuscate(dataSource.Password, s.secret),
		ObfuscatedSslCa:         common.Obfuscate(dataSource.SslCa, s.secret),
		ObfuscatedSslCert:       common.Obfuscate(dataSource.SslCert, s.secret),
		ObfuscatedSslKey:        common.Obfuscate(dataSource.SslKey, s.secret),
		Host:                    dataSource.Host,
		Port:                    dataSource.Port,
		Database:                dataSource.Database,
		SRV:                     dataSource.Srv,
		AuthenticationDatabase:  dataSource.AuthenticationDatabase,
		SID:                     dataSource.Sid,
		ServiceName:             dataSource.ServiceName,
		SSHHost:                 dataSource.SshHost,
		SSHPort:                 dataSource.SshPort,
		SSHUser:                 dataSource.SshUser,
		SSHObfuscatedPassword:   common.Obfuscate(dataSource.SshPassword, s.secret),
		SSHObfuscatedPrivateKey: common.Obfuscate(dataSource.SshPrivateKey, s.secret),
	}, nil
}

func (s *InstanceService) instanceCountGuard(ctx context.Context) error {
	instanceLimit := s.licenseService.GetPlanLimitValue(enterpriseAPI.PlanLimitMaximumInstance)

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
