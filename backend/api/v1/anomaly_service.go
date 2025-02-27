package v1

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	typeFilterKey     = "type"
	resourceFilterKey = "resource"
)

var typesMap = map[string]api.AnomalyType{
	"DATABASE_CONNECTION":   api.AnomalyDatabaseConnection,
	"DATABASE_SCHEMA_DRIFT": api.AnomalyDatabaseSchemaDrift,
}

// AnomalyService implements the anomaly service.
type AnomalyService struct {
	v1pb.UnimplementedAnomalyServiceServer
	store *store.Store
}

// NewAnomalyService creates a new anomaly service.
func NewAnomalyService(store *store.Store) *AnomalyService {
	return &AnomalyService{store: store}
}

// SearchAnomalies implements the SearchAnomalies RPC.
func (s *AnomalyService) SearchAnomalies(ctx context.Context, request *v1pb.SearchAnomaliesRequest) (*v1pb.
	SearchAnomaliesResponse, error,
) {
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	find := &store.ListAnomalyMessage{
		ProjectID: projectID,
	}
	if request.Filter != "" {
		// We only support filter by type and resource now.
		types, err := getEBNFTokens(request.Filter, typeFilterKey)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		for _, tp := range types {
			if v, ok := typesMap[tp]; ok {
				find.Types = append(find.Types, v)
			} else {
				return nil, status.Errorf(codes.InvalidArgument, "invalid type filter %q", tp)
			}
		}
		resources, err := getEBNFTokens(request.Filter, resourceFilterKey)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if len(resources) > 1 {
			return nil, status.Errorf(codes.InvalidArgument, "only one resource can be specified")
		} else if len(resources) == 1 {
			insID, dbName, err := common.GetInstanceDatabaseID(resources[0])
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, `invalid resource filter "%s": %v`, resources[0], err.Error())
			}
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &insID})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get instance %s", insID)
			}
			if instance == nil {
				return nil, status.Errorf(codes.NotFound, "instance %q not found", insID)
			}
			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:      &insID,
				DatabaseName:    &dbName,
				IsCaseSensitive: store.IsObjectCaseSensitive(instance),
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			if database == nil {
				return nil, status.Errorf(codes.NotFound, "cannot found the database %s", resources[0])
			}
			find.InstanceID = &insID
			find.DatabaseName = &database.DatabaseName
		}
	}

	anomalies, err := s.store.ListAnomalyV2(ctx, find)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var response v1pb.SearchAnomaliesResponse
	for _, anomaly := range anomalies {
		pbAnomaly, err := s.convertToAnomaly(ctx, anomaly)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		response.Anomalies = append(response.Anomalies, pbAnomaly)
	}
	return &response, nil
}

func (s *AnomalyService) convertToAnomaly(ctx context.Context, anomaly *store.AnomalyMessage) (*v1pb.Anomaly, error) {
	pbAnomaly := &v1pb.Anomaly{
		CreateTime: timestamppb.New(anomaly.UpdatedAt),
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &anomaly.InstanceID,
		DatabaseName: &anomaly.DatabaseName,
		ShowDeleted:  true,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find database %s", anomaly.DatabaseName)
	}
	if database == nil {
		return nil, errors.Errorf("cannot found database %s", anomaly.DatabaseName)
	}
	pbAnomaly.Resource = common.FormatDatabase(database.InstanceID, database.DatabaseName)
	pbAnomaly.Type = convertAnomalyType(anomaly.Type)
	pbAnomaly.Severity = getSeverityFromAnomalyType(pbAnomaly.Type)
	return pbAnomaly, nil
}

func convertAnomalyType(tp api.AnomalyType) v1pb.Anomaly_AnomalyType {
	switch tp {
	case api.AnomalyDatabaseConnection:
		return v1pb.Anomaly_DATABASE_CONNECTION
	case api.AnomalyDatabaseSchemaDrift:
		return v1pb.Anomaly_DATABASE_SCHEMA_DRIFT
	default:
		return v1pb.Anomaly_ANOMALY_TYPE_UNSPECIFIED
	}
}

func getSeverityFromAnomalyType(tp v1pb.Anomaly_AnomalyType) v1pb.Anomaly_AnomalySeverity {
	switch tp {
	case v1pb.Anomaly_DATABASE_CONNECTION, v1pb.Anomaly_DATABASE_SCHEMA_DRIFT:
		return v1pb.Anomaly_CRITICAL
	}
	return v1pb.Anomaly_ANOMALY_SEVERITY_UNSPECIFIED
}
