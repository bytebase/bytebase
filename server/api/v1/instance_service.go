package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/db"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

// InstanceService implements the instance service.
type InstanceService struct {
	v1pb.UnimplementedInstanceServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// NewInstanceService creates a new InstanceService.
func NewInstanceService(store *store.Store, licenseService enterpriseAPI.LicenseService) *InstanceService {
	return &InstanceService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetInstance gets an instance.
func (s *InstanceService) GetInstance(ctx context.Context, request *v1pb.GetInstanceRequest) (*v1pb.Instance, error) {
	environmentID, instanceID, err := getEnvironmentInstanceID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
		ShowDeleted:   true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
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
	count, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	subscription := s.licenseService.LoadSubscription(ctx)
	if count >= subscription.InstanceCount {
		return nil, status.Errorf(codes.ResourceExhausted, "reached the maximum instance count %d", subscription.InstanceCount)
	}

	instanceMessage, err := convertToInstanceMessage(request.InstanceId, request.Instance)
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

	environmentID, instanceID, err := getEnvironmentInstanceID(request.Instance.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
		ShowDeleted:   true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", request.Instance.Name)
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", request.Instance.Name)
	}

	patch := &store.UpdateInstanceMessage{
		UpdaterID:     ctx.Value(common.PrincipalIDContextKey).(int),
		EnvironmentID: environmentID,
		ResourceID:    instanceID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "instance.title":
			patch.Title = &request.Instance.Title
		case "instance.external_link":
			patch.ExternalLink = &request.Instance.ExternalLink
		case "instance.data_sources":
			datasourceList, err := convertToDataSourceMessageList(request.Instance.DataSources)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			patch.DataSources = datasourceList
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
	environmentID, instanceID, err := getEnvironmentInstanceID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
		ShowDeleted:   true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", request.Name)
	}
	if instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", request.Name)
	}

	rowStatus := api.Archived
	if _, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		UpdaterID:     ctx.Value(common.PrincipalIDContextKey).(int),
		EnvironmentID: environmentID,
		ResourceID:    instanceID,
		RowStatus:     &rowStatus,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

// UndeleteInstance undeletes an instance.
func (s *InstanceService) UndeleteInstance(ctx context.Context, request *v1pb.UndeleteInstanceRequest) (*v1pb.Instance, error) {
	environmentID, instanceID, err := getEnvironmentInstanceID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
		ShowDeleted:   true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", request.Name)
	}
	if !instance.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q is active", request.Name)
	}

	rowStatus := api.Normal
	ins, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
		UpdaterID:     ctx.Value(common.PrincipalIDContextKey).(int),
		EnvironmentID: environmentID,
		ResourceID:    instanceID,
		RowStatus:     &rowStatus,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToInstance(ins), nil
}

// AddDataSource adds a data source to an instance.
func (*InstanceService) AddDataSource(_ context.Context, _ *v1pb.AddDataSourceRequest) (*v1pb.Instance, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddDataSource not implemented")
}

// RemoveDataSource removes a data source to an instance.
func (*InstanceService) RemoveDataSource(_ context.Context, _ *v1pb.RemoveDataSourceRequest) (*v1pb.Instance, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveDataSource not implemented")
}

// UpdateDataSource updates a data source of an instance.
func (*InstanceService) UpdateDataSource(_ context.Context, _ *v1pb.UpdateDataSourceRequest) (*v1pb.Instance, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateDataSource not implemented")
}

func convertToInstance(instance *store.InstanceMessage) *v1pb.Instance {
	engine := v1pb.Engine_ENGINE_UNSPECIFIED
	switch instance.Engine {
	case db.ClickHouse:
		engine = v1pb.Engine_CLICKHOUSE
	case db.MySQL:
		engine = v1pb.Engine_MYSQL
	case db.Postgres:
		engine = v1pb.Engine_POSTGRES
	case db.Snowflake:
		engine = v1pb.Engine_SNOWFLAKE
	case db.SQLite:
		engine = v1pb.Engine_SQLITE
	case db.TiDB:
		engine = v1pb.Engine_TIDB
	case db.MongoDB:
		engine = v1pb.Engine_MONGODB
	}

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
			// We don't return the password on reads.
			SslCa:                  ds.SslCa,
			SslCert:                ds.SslCert,
			SslKey:                 ds.SslKey,
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

func convertToInstanceMessage(instanceID string, instance *v1pb.Instance) (*store.InstanceMessage, error) {
	var engine db.Type
	switch instance.Engine {
	case v1pb.Engine_CLICKHOUSE:
		engine = db.ClickHouse
	case v1pb.Engine_MYSQL:
		engine = db.MySQL
	case v1pb.Engine_POSTGRES:
		engine = db.Postgres
	case v1pb.Engine_SNOWFLAKE:
		engine = db.Snowflake
	case v1pb.Engine_SQLITE:
		engine = db.SQLite
	case v1pb.Engine_TIDB:
		engine = db.TiDB
	case v1pb.Engine_MONGODB:
		engine = db.MongoDB
	default:
		return nil, errors.Errorf("invalid instance engine %v", instance.Engine)
	}

	datasourceList, err := convertToDataSourceMessageList(instance.DataSources)
	if err != nil {
		return nil, err
	}

	return &store.InstanceMessage{
		ResourceID:   instanceID,
		Title:        instance.Title,
		Engine:       engine,
		ExternalLink: instance.ExternalLink,
		DataSources:  datasourceList,
	}, nil
}

func convertToDataSourceMessageList(dataSources []*v1pb.DataSource) ([]*store.DataSourceMessage, error) {
	datasourceList := []*store.DataSourceMessage{}
	for _, ds := range dataSources {
		var dsType api.DataSourceType
		switch ds.Type {
		case v1pb.DataSourceType_READ_ONLY:
			dsType = api.RO
		case v1pb.DataSourceType_ADMIN:
			dsType = api.Admin
		default:
			return nil, errors.Errorf("invalid data source type %v", ds.Type)
		}
		datasourceList = append(datasourceList, &store.DataSourceMessage{
			Title:    ds.Title,
			Type:     dsType,
			Username: ds.Username,
			Password: ds.Password,
			SslCa:    ds.SslCa,
			SslCert:  ds.SslCert,
			SslKey:   ds.SslKey,
			Host:     ds.Host,
			Port:     ds.Port,
			Database: ds.Database,
		})
	}

	return datasourceList, nil
}
