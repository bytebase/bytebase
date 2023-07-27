package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	"INSTANCE_CONNECTION":              api.AnomalyInstanceConnection,
	"MIGRATION_SCHEMA":                 api.AnomalyInstanceMigrationSchema,
	"DATABASE_BACKUP_POLICY_VIOLATION": api.AnomalyDatabaseBackupPolicyViolation,
	"DATABASE_BACKUP_MISSING":          api.AnomalyDatabaseBackupMissing,
	"DATABASE_CONNECTION":              api.AnomalyDatabaseConnection,
	"DATABASE_SCHEMA_DRIFT":            api.AnomalyDatabaseSchemaDrift,
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
	var find store.ListAnomalyMessage
	if request.Filter != "" {
		// We only support filter by type and resource now.
		types, err := getEBNFTokens(request.Filter, typeFilterKey)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
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
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if len(resources) > 1 {
			return nil, status.Errorf(codes.InvalidArgument, "only one resource can be specified")
		} else if len(resources) == 1 {
			sections := strings.Split(resources[0], "/")
			if len(sections) == 2 {
				// Treat as instances/{resource id}
				insID, err := common.GetInstanceID(resources[0])
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, `invalid resource filter "%s": %v`, resources[0], err.Error())
				}
				find.InstanceID = &insID
			} else if len(sections) == 4 {
				// Treat as instances/{resource id}/databases/{db name}
				insID, dbName, err := common.GetInstanceDatabaseID(resources[0])
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, `invalid resource filter "%s": %v`, resources[0], err.Error())
				}
				database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:   &insID,
					DatabaseName: &dbName,
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, err.Error())
				}
				if database == nil {
					return nil, status.Errorf(codes.NotFound, "cannot found the database %s", resources[0])
				}
				find.InstanceID = &insID
				find.DatabaseUID = &database.UID
			} else {
				return nil, status.Errorf(codes.InvalidArgument, `invalid resource filter "%s"`, resources[0])
			}
		}
	}

	anomalies, err := s.store.ListAnomalyV2(ctx, &find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	var response v1pb.SearchAnomaliesResponse
	for _, anomaly := range anomalies {
		pbAnomaly, err := s.convertToAnomaly(ctx, anomaly)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		response.Anomalies = append(response.Anomalies, pbAnomaly)
	}
	return &response, nil
}

func (s *AnomalyService) convertToAnomaly(ctx context.Context, anomaly *store.AnomalyMessage) (*v1pb.Anomaly, error) {
	pbAnomaly := &v1pb.Anomaly{
		CreateTime: timestamppb.New(time.Unix(anomaly.CreatedTs, 0)),
		UpdateTime: timestamppb.New(time.Unix(anomaly.UpdatedTs, 0)),
	}

	if v := anomaly.DatabaseUID; v != nil {
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			UID:         v,
			ShowDeleted: true,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find database with id %d", v)
		}
		if database == nil {
			return nil, errors.Errorf("cannot found database with id %d", v)
		}
		pbAnomaly.Resource = fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName)
	} else {
		pbAnomaly.Resource = fmt.Sprintf("%s%s", common.InstanceNamePrefix, anomaly.InstanceID)
	}

	switch anomaly.Type {
	case api.AnomalyInstanceConnection:
		var detail api.AnomalyInstanceConnectionPayload
		if err := json.Unmarshal([]byte(anomaly.Payload), &detail); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal instance connection anomaly payload")
		}
		pbAnomaly.Type = v1pb.Anomaly_INSTANCE_CONNECTION
		pbAnomaly.Detail = &v1pb.Anomaly_InstanceConnectionDetail_{
			InstanceConnectionDetail: &v1pb.Anomaly_InstanceConnectionDetail{
				Detail: detail.Detail,
			},
		}
	case api.AnomalyDatabaseBackupPolicyViolation:
		var detail api.AnomalyDatabaseBackupPolicyViolationPayload
		if err := json.Unmarshal([]byte(anomaly.Payload), &detail); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal database backup policy violation anomaly payload")
		}
		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
			UID: &detail.EnvironmentID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find environment with id %d", detail.EnvironmentID)
		}
		pbAnomaly.Type = v1pb.Anomaly_DATABASE_BACKUP_POLICY_VIOLATION
		pbAnomaly.Detail = &v1pb.Anomaly_DatabaseBackupPolicyViolationDetail_{
			DatabaseBackupPolicyViolationDetail: &v1pb.Anomaly_DatabaseBackupPolicyViolationDetail{
				// The instance are bind to a specify environment, and cannot be moved to another environment in Bytebase.
				// So it's safe to use environmentID here.
				Parent:           fmt.Sprintf("%s%s", common.EnvironmentNamePrefix, environment.ResourceID),
				ExpectedSchedule: convertBackupPlanSchedule(detail.ExpectedBackupSchedule),
				ActualSchedule:   convertBackupPlanSchedule(detail.ActualBackupSchedule),
			},
		}
	case api.AnomalyDatabaseBackupMissing:
		var detail api.AnomalyDatabaseBackupMissingPayload
		if err := json.Unmarshal([]byte(anomaly.Payload), &detail); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal database backup missing anomaly payload")
		}
		pbAnomaly.Type = v1pb.Anomaly_DATABASE_BACKUP_MISSING
		pbAnomaly.Detail = &v1pb.Anomaly_DatabaseBackupMissingDetail_{
			DatabaseBackupMissingDetail: &v1pb.Anomaly_DatabaseBackupMissingDetail{
				ExpectedSchedule: convertBackupPlanSchedule(detail.ExpectedBackupSchedule),
				LatestBackupTime: timestamppb.New(time.Unix(detail.LastBackupTs, 0)),
			},
		}
	case api.AnomalyDatabaseConnection:
		var detail api.AnomalyDatabaseConnectionPayload
		if err := json.Unmarshal([]byte(anomaly.Payload), &detail); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal database connection anomaly payload")
		}
		pbAnomaly.Type = v1pb.Anomaly_DATABASE_CONNECTION
		pbAnomaly.Detail = &v1pb.Anomaly_DatabaseConnectionDetail_{
			DatabaseConnectionDetail: &v1pb.Anomaly_DatabaseConnectionDetail{
				Detail: detail.Detail,
			},
		}
	case api.AnomalyDatabaseSchemaDrift:
		var detail api.AnomalyDatabaseSchemaDriftPayload
		if err := json.Unmarshal([]byte(anomaly.Payload), &detail); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal database schema drift anomaly payload")
		}
		pbAnomaly.Type = v1pb.Anomaly_DATABASE_SCHEMA_DRIFT
		pbAnomaly.Detail = &v1pb.Anomaly_DatabaseSchemaDriftDetail_{
			DatabaseSchemaDriftDetail: &v1pb.Anomaly_DatabaseSchemaDriftDetail{
				RecordVersion:  detail.Version,
				ExpectedSchema: detail.Expect,
				ActualSchema:   detail.Actual,
			},
		}
	}
	pbAnomaly.Severity = getSeverityFromAnomalyType(pbAnomaly.Type)
	return pbAnomaly, nil
}

func getSeverityFromAnomalyType(tp v1pb.Anomaly_AnomalyType) v1pb.Anomaly_AnomalySeverity {
	switch tp {
	case v1pb.Anomaly_DATABASE_BACKUP_POLICY_VIOLATION:
		return v1pb.Anomaly_MEDIUM
	case v1pb.Anomaly_DATABASE_BACKUP_MISSING:
		return v1pb.Anomaly_HIGH
	case v1pb.Anomaly_INSTANCE_CONNECTION, v1pb.Anomaly_MIGRATION_SCHEMA, v1pb.Anomaly_DATABASE_CONNECTION, v1pb.Anomaly_DATABASE_SCHEMA_DRIFT:
		return v1pb.Anomaly_CRITICAL
	}
	return v1pb.Anomaly_ANOMALY_SEVERITY_UNSPECIFIED
}

func convertBackupPlanSchedule(s api.BackupPlanPolicySchedule) v1pb.BackupPlanSchedule {
	switch s {
	case api.BackupPlanPolicyScheduleUnset:
		return v1pb.BackupPlanSchedule_UNSET
	case api.BackupPlanPolicyScheduleDaily:
		return v1pb.BackupPlanSchedule_DAILY
	case api.BackupPlanPolicyScheduleWeekly:
		return v1pb.BackupPlanSchedule_WEEKLY
	}
	return v1pb.BackupPlanSchedule_SCHEDULE_UNSPECIFIED
}
