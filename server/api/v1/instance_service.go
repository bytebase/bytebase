package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

const instanceNamePrefix = "instances/"

// InstanceService implements the instance service.
type InstanceService struct {
	v1pb.UnimplementedInstanceServiceServer
	store *store.Store
}

// NewInstanceService creates a new InstanceService.
func NewInstanceService(store *store.Store) *InstanceService {
	return &InstanceService{
		store: store,
	}
}

// GetInstance gets an instance.
func (s *InstanceService) GetInstance(ctx context.Context, request *v1pb.GetInstanceRequest) (*v1pb.Instance, error) {
	environmentID, instanceID, err := getEnvironmentAndInstanceID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, environmentID, instanceID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q not found", instanceID)
	}
	return convertInstance(environmentID, instance), nil
}

// ListInstances lists all instances.
func (s *EnvironmentService) ListInstances(ctx context.Context, request *v1pb.ListInstancesRequest) (*v1pb.ListInstancesResponse, error) {
	environmentID, err := getEnvironmentID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instances, err := s.store.ListInstanceV2(ctx, environmentID, request.ShowDeleted)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListInstancesResponse{}
	for _, instance := range instances {
		response.Instances = append(response.Instances, convertInstance(environmentID, instance))
	}
	return response, nil
}

func getEnvironmentAndInstanceID(name string) (string, string, error) {
	// the instance request should be environments/{environment-id}/instances/{instance-id}
	if !strings.HasPrefix(name, environmentNamePrefix) {
		return "", "", errors.Errorf("invalid request %q", name)
	}

	sections := strings.Split(name, "/")
	if len(sections) != 4 {
		return "", "", errors.Errorf("invalid request %q", name)
	}

	if fmt.Sprintf("%s/", sections[2]) != instanceNamePrefix {
		return "", "", errors.Errorf("invalid request %q", name)
	}

	if sections[1] == "" || sections[3] == "" {
		return "", "", errors.Errorf("invalid request %q", name)
	}

	return sections[1], sections[3], nil
}

func convertInstance(environmentID string, instance *store.InstanceMessage) *v1pb.Instance {
	engine := v1pb.Engine_ENGINE_UNSPECIFIED
	switch instance.Engine {
	case db.ClickHouse:
		engine = v1pb.Engine_ENGINE_CLICKHOUSE
	case db.MySQL:
		engine = v1pb.Engine_ENGINE_MYSQL
	case db.Postgres:
		engine = v1pb.Engine_ENGINE_POSTGRES
	case db.Snowflake:
		engine = v1pb.Engine_ENGINE_SNOWFLAKE
	case db.SQLite:
		engine = v1pb.Engine_ENGINE_SQLITE
	case db.TiDB:
		engine = v1pb.Engine_ENGINE_TIDB
	case db.MongoDB:
		engine = v1pb.Engine_ENGINE_MONGODB
	}

	dataSourceList := []*v1pb.DataSource{}
	for _, ds := range instance.DataSources {
		dataSourceType := v1pb.DataSourceType_DATA_SOURCE_UNSPECIFIED
		switch ds.Type {
		case api.Admin:
			dataSourceType = v1pb.DataSourceType_DATA_SOURCE_ADMIN
		case api.RO:
			dataSourceType = v1pb.DataSourceType_DATA_SOURCE_RO
		}

		dataSourceList = append(dataSourceList, &v1pb.DataSource{
			Title:    ds.Title,
			Type:     dataSourceType,
			Username: ds.Username,
			SslCa:    ds.SslCa,
			SslCert:  ds.SslCert,
			SslKey:   ds.SslKey,
			Host:     ds.Host,
			Port:     ds.Port,
			Database: ds.Database,
		})
	}

	state := v1pb.State_STATE_ACTIVE
	if instance.Deleted {
		state = v1pb.State_STATE_DELETED
	}

	return &v1pb.Instance{
		Name:         fmt.Sprintf("%s%s%s%s", environmentNamePrefix, environmentID, instanceNamePrefix, instance.InstanceID),
		Title:        instance.Title,
		Engine:       engine,
		ExternalLink: instance.ExternalLink,
		DataSources:  dataSourceList,
		State:        state,
	}
}
