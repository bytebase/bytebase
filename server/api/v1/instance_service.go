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
	instanceID, err := getInstanceID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, instanceID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.InvalidArgument, "instance %q not found", instanceID)
	}
	return convertInstance(instance), nil
}

func getInstanceID(name string) (string, error) {
	if !strings.HasPrefix(name, instanceNamePrefix) {
		return "", errors.Errorf("invalid instance name %q", name)
	}
	return strings.TrimPrefix(name, instanceNamePrefix), nil
}

func convertInstance(instance *api.Instance) *v1pb.Instance {
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
	for _, ds := range instance.DataSourceList {
		dataSourceType := v1pb.DataSourceType_DATA_SOURCE_UNSPECIFIED
		switch ds.Type {
		case api.Admin:
			dataSourceType = v1pb.DataSourceType_DATA_SOURCE_ADMIN
		case api.RO:
			dataSourceType = v1pb.DataSourceType_DATA_SOURCE_RO
		}

		dataSourceList = append(dataSourceList, &v1pb.DataSource{
			Title:    ds.Name,
			Type:     dataSourceType,
			Username: ds.Username,
			SslCa:    ds.SslCa,
			SslCert:  ds.SslCert,
			SslKey:   ds.SslKey,
			Host:     ds.HostOverride,
			Port:     ds.PortOverride,
			Database: ds.DatabaseName,
		})
	}

	return &v1pb.Instance{
		Name:         fmt.Sprintf("%s%s", instanceNamePrefix, instance.ResourceID),
		Title:        instance.Name,
		Engine:       engine,
		ExternalLink: instance.ExternalLink,
		DataSources:  dataSourceList,
	}
}
