package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	typeFilterKey     = "type"
	resourceFilterKey = "resource"
)

var (
	typesMap = map[string]api.AnomalyType{
		"INSTANCE_CONNECTION":              api.AnomalyInstanceConnection,
		"MIGRATION_SCHEMA":                 api.AnomalyInstanceMigrationSchema,
		"DATABASE_BACKUP_POLICY_VIOLATION": api.AnomalyDatabaseBackupPolicyViolation,
		"DATABASE_BACKUP_MISSING":          api.AnomalyDatabaseBackupMissing,
		"DATABASE_CONNECTION":              api.AnomalyDatabaseConnection,
		"DATABASE_SCHEMA_DRIFT":            api.AnomalyDatabaseSchemaDrift,
	}
)

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
	SearchAnomaliesResponse, error) {
	var find store.ListAnomalyMessage
	var environmentID, instanceID, databaseName string
	if request.Filter != "" {
		// We only support filter by type and resource now.
		types, err := getEBNFTokens(request.Filter, typeFilterKey)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		for _, tp := range types {
			if v, ok := typesMap[tp]; ok {
				find.Types = append(find.Types, &v)
			} else {
				return nil, status.Errorf(codes.InvalidArgument, "Invalid type filter %q", tp)
			}
		}
		resources, err := getEBNFTokens(request.Filter, resourceFilterKey)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if len(resources) > 1 {
			return nil, status.Errorf(codes.InvalidArgument, "Only one resource can be specified")
		}

		// For resources filter, we only support filter by instance and database.
		envID, insID, err := getEnvironmentInstanceID(resources[0])
		if err != nil {
			// Try to treat as database resource.
			envID, insID, dbName, err := getEnvironmentInstanceDatabaseID(resources[0])
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Only support filter by instance and database in resource filter")
			}
			environmentID, instanceID, databaseName = envID, insID, dbName
		} else {
			// Treat as instance resource.
			environmentID, instanceID = envID, insID
		}
		if environmentID != "" {
			env, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
				ResourceID: &environmentID,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}
			if env == nil {
				return nil, status.Errorf(codes.NotFound, "Environment %q not found", environmentID)
			}
		}
		if environmentID != "" && instanceID != "" {
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
				EnvironmentID: &environmentID,
				ResourceID:    &instanceID,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}
			if instance == nil {
				return nil, status.Errorf(codes.NotFound, "Instance %q not found", instanceID)
			}
			find.InstanceUID = &instance.UID
		}
		if environmentID != "" && instanceID != "" && databaseName != "" {
			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				EnvironmentID: &environmentID,
				InstanceID:    &instanceID,
				DatabaseName:  &databaseName,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}
			if database == nil {
				return nil, status.Errorf(codes.NotFound, "Database %q not found in environment %q instance %q", databaseName, environmentID, instanceID)
			}
			find.DatabaseUID = &database.UID
		}
	}

	anomalies, err := s.store.ListAnomalyV2(ctx, &find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	var response v1pb.SearchAnomaliesResponse
	for _, anomaly := range anomalies {
		pbAnomaly, err := convertToAnomaly(anomaly, environmentID, instanceID, databaseName)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		response.Anomalies = append(response.Anomalies, pbAnomaly)
	}
	return &response, nil
}

func convertToAnomaly(anomaly *store.AnomalyMessage, environmentID, instanceID, databaseName string) (*v1pb.Anomaly, error) {
	var pbAnomaly v1pb.Anomaly
	if environmentID != "" && instanceID != "" && databaseName != "" {
		pbAnomaly.Resource = fmt.Sprintf("%s%s/%s%s/%s%s", environmentNamePrefix, environmentID, instanceNamePrefix, instanceID, databaseIDPrefix, databaseName)
	} else if environmentID != "" && instanceID != "" {
		pbAnomaly.Resource = fmt.Sprintf("%s%s/%s%s", environmentNamePrefix, environmentID, instanceNamePrefix, instanceID)
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
		pbAnomaly.Type = v1pb.Anomaly_DATABASE_BACKUP_POLICY_VIOLATION
		pbAnomaly.Detail = &v1pb.Anomaly_DatabaseBackupPolicyViolationDetail_{
			DatabaseBackupPolicyViolationDetail: &v1pb.Anomaly_DatabaseBackupPolicyViolationDetail{
				// The instance are bind to a specify environment, and cannot be moved to another environment in Bytebase.
				// So it's safe to use environmentID here.
				Parent:           fmt.Sprintf("%s%s/%s%s", environmentNamePrefix, environmentID, instanceNamePrefix, instanceID),
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
	return &pbAnomaly, nil
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

func convertBackupPlanSchedule(s api.BackupPlanPolicySchedule) v1pb.Anomaly_BackupPlanSchedule {
	switch s {
	case api.BackupPlanPolicyScheduleUnset:
		return v1pb.Anomaly_UNSET
	case api.BackupPlanPolicyScheduleDaily:
		return v1pb.Anomaly_DAILY
	case api.BackupPlanPolicyScheduleWeekly:
		return v1pb.Anomaly_WEEKLY
	}
	return v1pb.Anomaly_BACKUP_PLAN_SCHEDULE_UNSPECIFIED
}
