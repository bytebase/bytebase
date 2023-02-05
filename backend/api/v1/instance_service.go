package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// InstanceService implements the instance service.
type InstanceService struct {
	v1pb.UnimplementedInstanceServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
	secret         string
}

// NewInstanceService creates a new InstanceService.
func NewInstanceService(store *store.Store, licenseService enterpriseAPI.LicenseService, secret string) *InstanceService {
	return &InstanceService{
		store:          store,
		licenseService: licenseService,
		secret:         secret,
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
	environmentID, err := getEnvironmentID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	find := &store.FindInstanceMessage{
		ShowDeleted: request.ShowDeleted,
	}
	// Use "environments/-" to list all instances from all environments.
	if environmentID != "-" {
		find.EnvironmentID = &environmentID
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
	environmentID, err := getEnvironmentID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if !isValidResourceID(request.InstanceId) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid instance ID %v", request.InstanceId)
	}

	// Instance limit in the plan.
	if err := s.instanceCountGuard(ctx); err != nil {
		return nil, err
	}

	instanceMessage, err := s.convertToInstanceMessage(request.InstanceId, request.Instance)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	instance, err := s.store.CreateInstanceV2(ctx,
		environmentID,
		instanceMessage,
		principalID,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// TODO(d): sync instance databases.
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
		case "instance.title":
			patch.Title = &request.Instance.Title
		case "instance.external_link":
			patch.ExternalLink = &request.Instance.ExternalLink
		case "instance.data_sources":
			datasourceList, err := s.convertToDataSourceMessages(request.Instance.DataSources)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			patch.DataSources = &datasourceList
		}
	}

	ins, err := s.store.UpdateInstanceV2(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// TODO(d): sync instance databases.

	return convertToInstance(ins), nil
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

	if _, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		UpdaterID:     ctx.Value(common.PrincipalIDContextKey).(int),
		EnvironmentID: instance.EnvironmentID,
		ResourceID:    instance.ResourceID,
		Delete:        &deletePatch,
	}); err != nil {
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
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToInstance(ins), nil
}

// AddDataSource adds a data source to an instance.
func (s *InstanceService) AddDataSource(ctx context.Context, request *v1pb.AddDataSourceRequest) (*v1pb.Instance, error) {
	if request.DataSources == nil {
		return nil, status.Errorf(codes.InvalidArgument, "data sources is required")
	}
	// We only support add RO type datasouce to instance now, see more details in instance_service.proto.
	if request.DataSources.Type != v1pb.DataSourceType_READ_ONLY {
		return nil, status.Errorf(codes.InvalidArgument, "only support add read-only data source")
	}

	dataSource, err := s.convertToDataSourceMessage(request.DataSources)
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

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)

	if err := s.store.AddDataSourceToInstanceV2(ctx, instance.UID, principalID, instance.EnvironmentID, instance.ResourceID, dataSource); err != nil {
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
	if request.DataSources == nil {
		return nil, status.Errorf(codes.InvalidArgument, "data sources is required")
	}
	// We only support remove RO type datasource to instance now, see more details in instance_service.proto.
	if request.DataSources.Type != v1pb.DataSourceType_READ_ONLY {
		return nil, status.Errorf(codes.InvalidArgument, "only support remove read-only data source")
	}

	dataSource, err := s.convertToDataSourceMessage(request.DataSources)
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

	if err := s.store.RemoveDataSourceV2(ctx, instance.UID, instance.EnvironmentID, instance.ResourceID, dataSource.Type); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	instance, err = s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &instance.EnvironmentID,
		ResourceID:    &instance.ResourceID,
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

// UpdateDataSource updates a data source of an instance.
func (s *InstanceService) UpdateDataSource(ctx context.Context, request *v1pb.UpdateDataSourceRequest) (*v1pb.Instance, error) {
	if request.DataSources == nil {
		return nil, status.Errorf(codes.InvalidArgument, "datasource is required")
	}
	tp, err := convertDataSourceTp(request.DataSources.Type)
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

	patch := &store.UpdateDataSourceMessage{
		UpdaterID:     ctx.Value(common.PrincipalIDContextKey).(int),
		InstanceUID:   instance.UID,
		Type:          tp,
		EnvironmentID: instance.EnvironmentID,
		InstanceID:    instance.ResourceID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "username":
			patch.Username = &request.DataSources.Username
		case "password":
			obfuscated := common.Obfuscate(request.DataSources.Password, s.secret)
			patch.ObfuscatedPassword = &obfuscated
		case "ssl_ca":
			obfuscated := common.Obfuscate(request.DataSources.SslCa, s.secret)
			patch.ObfuscatedSslCa = &obfuscated
		case "ssl_cert":
			obfuscated := common.Obfuscate(request.DataSources.SslCert, s.secret)
			patch.ObfuscatedSslCert = &obfuscated
		case "ssl_key":
			obfuscated := common.Obfuscate(request.DataSources.SslKey, s.secret)
			patch.ObfuscatedSslKey = &obfuscated
		case "host":
			patch.Host = &request.DataSources.Host
		case "port":
			patch.Port = &request.DataSources.Port
		case "srv":
			patch.SRV = &request.DataSources.Srv
		case "authentication_database":
			patch.AuthenticationDatabase = &request.DataSources.AuthenticationDatabase
		}
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

func (s *InstanceService) instanceCountGuard(ctx context.Context) error {
	subscription := s.licenseService.LoadSubscription(ctx)
	if subscription.InstanceCount == -1 {
		return nil
	}

	count, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{})
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if count >= subscription.InstanceCount {
		return status.Errorf(codes.ResourceExhausted, "reached the maximum instance count %d", subscription.InstanceCount)
	}

	return nil
}

func (s *InstanceService) getInstanceMessage(ctx context.Context, name string) (*store.InstanceMessage, error) {
	environmentID, instanceID, err := getEnvironmentInstanceID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
	})
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
		})
	}

	return &v1pb.Instance{
		Name:         fmt.Sprintf("%s%s/%s%s", environmentNamePrefix, instance.EnvironmentID, instanceNamePrefix, instance.ResourceID),
		Uid:          fmt.Sprintf("%d", instance.UID),
		Title:        instance.Title,
		Engine:       engine,
		ExternalLink: instance.ExternalLink,
		DataSources:  dataSourceList,
		State:        convertDeletedToState(instance.Deleted),
	}
}

func (s *InstanceService) convertToInstanceMessage(instanceID string, instance *v1pb.Instance) (*store.InstanceMessage, error) {
	datasources, err := s.convertToDataSourceMessages(instance.DataSources)
	if err != nil {
		return nil, err
	}

	return &store.InstanceMessage{
		ResourceID:   instanceID,
		Title:        instance.Title,
		Engine:       convertEngine(instance.Engine),
		ExternalLink: instance.ExternalLink,
		DataSources:  datasources,
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
		Title:                  dataSource.Title,
		Type:                   dsType,
		Username:               dataSource.Username,
		ObfuscatedPassword:     common.Obfuscate(dataSource.Password, s.secret),
		ObfuscatedSslCa:        common.Obfuscate(dataSource.SslCa, s.secret),
		ObfuscatedSslCert:      common.Obfuscate(dataSource.SslCert, s.secret),
		ObfuscatedSslKey:       common.Obfuscate(dataSource.SslKey, s.secret),
		Host:                   dataSource.Host,
		Port:                   dataSource.Port,
		Database:               dataSource.Database,
		SRV:                    dataSource.Srv,
		AuthenticationDatabase: dataSource.AuthenticationDatabase,
	}, nil
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
