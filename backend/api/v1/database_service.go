package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pkg/errors"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/runner/backuprun"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	filterKeyEnvironment = "environment"
	filterKeyProject     = "project"
	filterKeyStartTime   = "start_time"

	// Support order by count, latest_log_time, average_query_time, maximum_query_time,
	// average_rows_sent, maximum_rows_sent, average_rows_examined, maximum_rows_examined for now.
	orderByKeyCount               = "count"
	orderByKeyLatestLogTime       = "latest_log_time"
	orderByKeyAverageQueryTime    = "average_query_time"
	orderByKeyMaximumQueryTime    = "maximum_query_time"
	orderByKeyAverageRowsSent     = "average_rows_sent"
	orderByKeyMaximumRowsSent     = "maximum_rows_sent"
	orderByKeyAverageRowsExamined = "average_rows_examined"
	orderByKeyMaximumRowsExamined = "maximum_rows_examined"
)

// DatabaseService implements the database service.
type DatabaseService struct {
	v1pb.UnimplementedDatabaseServiceServer
	store          *store.Store
	backupRunner   *backuprun.Runner
	schemaSyncer   *schemasync.Syncer
	licenseService enterprise.LicenseService
	profile        *config.Profile
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(store *store.Store, br *backuprun.Runner, schemaSyncer *schemasync.Syncer, licenseService enterprise.LicenseService, profile *config.Profile) *DatabaseService {
	return &DatabaseService{
		store:          store,
		backupRunner:   br,
		schemaSyncer:   schemaSyncer,
		licenseService: licenseService,
		profile:        profile,
	}
}

// GetDatabase gets a database.
func (s *DatabaseService) GetDatabase(ctx context.Context, request *v1pb.GetDatabaseRequest) (*v1pb.Database, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	find := &store.FindDatabaseMessage{}
	databaseUID, isNumber := isNumber(databaseName)
	if isNumber {
		// Expected format: "instances/{ignored_value}/database/{uid}"
		find.UID = &databaseUID
	} else {
		// Expected format: "instances/{instance}/database/{database}"
		find.InstanceID = &instanceID
		find.DatabaseName = &databaseName
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
		}
		find.IgnoreCaseSensitive = store.IgnoreDatabaseAndTableCaseSensitive(instance)
	}
	database, err := s.store.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionManageGeneral); err != nil {
		return nil, err
	}
	return convertToDatabase(database), nil
}

// ListDatabases lists all databases.
func (s *DatabaseService) ListDatabases(ctx context.Context, request *v1pb.ListDatabasesRequest) (*v1pb.ListDatabasesResponse, error) {
	instanceID, err := common.GetInstanceID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	find := &store.FindDatabaseMessage{}
	if instanceID != "-" {
		find.InstanceID = &instanceID
	}
	if request.Filter != "" {
		projectFilter, err := getProjectFilter(request.Filter)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		projectID, err := common.GetProjectID(projectFilter)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid project %q in the filter", projectFilter)
		}
		find.ProjectID = &projectID
	}
	databases, err := s.store.ListDatabases(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListDatabasesResponse{}
	for _, database := range databases {
		if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionManageGeneral); err != nil {
			st := status.Convert(err)
			if st.Code() == codes.PermissionDenied {
				continue
			}
			return nil, err
		}
		response.Databases = append(response.Databases, convertToDatabase(database))
	}
	return response, nil
}

// SearchDatabases searches all databases.
func (s *DatabaseService) SearchDatabases(ctx context.Context, request *v1pb.SearchDatabasesRequest) (*v1pb.SearchDatabasesResponse, error) {
	instanceID, err := common.GetInstanceID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	find := &store.FindDatabaseMessage{}
	if instanceID != "-" {
		find.InstanceID = &instanceID
	}
	if request.Filter != "" {
		projectFilter, err := getProjectFilter(request.Filter)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		projectID, err := common.GetProjectID(projectFilter)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid project %q in the filter", projectFilter)
		}
		find.ProjectID = &projectID
	}
	databases, err := s.store.ListDatabases(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.SearchDatabasesResponse{}
	for _, database := range databases {
		if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionManageGeneral); err != nil {
			st := status.Convert(err)
			if st.Code() == codes.PermissionDenied {
				continue
			}
			return nil, err
		}
		response.Databases = append(response.Databases, convertToDatabase(database))
	}
	return response, nil
}

// UpdateDatabase updates a database.
func (s *DatabaseService) UpdateDatabase(ctx context.Context, request *v1pb.UpdateDatabaseRequest) (*v1pb.Database, error) {
	if request.Database == nil {
		return nil, status.Errorf(codes.InvalidArgument, "database must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Database.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionAdminDatabase); err != nil {
		return nil, err
	}

	var project *store.ProjectMessage
	patch := &store.UpdateDatabaseMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "project":
			projectID, err := common.GetProjectID(request.Database.Project)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID:  &projectID,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}
			if project == nil {
				return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
			}
			if project.Deleted {
				return nil, status.Errorf(codes.FailedPrecondition, "project %q is deleted", projectID)
			}
			patch.ProjectID = &project.ResourceID
		case "labels":
			labels := request.Database.Labels
			if labels == nil {
				labels = map[string]string{}
			}
			patch.MetadataUpsert = &storepb.DatabaseMetadata{
				Labels: labels,
			}
		case "environment":
			if request.Database.Environment == "" {
				unsetEnvironment := ""
				patch.EnvironmentID = &unsetEnvironment
			} else {
				environmentID, err := common.GetEnvironmentID(request.Database.Environment)
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
				patch.EnvironmentID = &environment.ResourceID
			}
		}
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	updatedDatabase, err := s.store.UpdateDatabase(ctx, patch, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project != nil {
		if err := s.createTransferProjectActivity(ctx, project, principalID, database); err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	return convertToDatabase(updatedDatabase), nil
}

// SyncDatabase syncs the schema of a database.
func (s *DatabaseService) SyncDatabase(ctx context.Context, request *v1pb.SyncDatabaseRequest) (*v1pb.SyncDatabaseResponse, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	find := &store.FindDatabaseMessage{}
	databaseUID, isNumber := isNumber(databaseName)
	if isNumber {
		// Expected format: "instances/{ignored_value}/database/{uid}"
		find.UID = &databaseUID
	} else {
		// Expected format: "instances/{instance}/database/{database}"
		find.InstanceID = &instanceID
		find.DatabaseName = &databaseName
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
		}
		find.IgnoreCaseSensitive = store.IgnoreDatabaseAndTableCaseSensitive(instance)
	}
	database, err := s.store.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionChangeDatabase); err != nil {
		return nil, err
	}
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		return nil, err
	}
	return &v1pb.SyncDatabaseResponse{}, nil
}

// BatchUpdateDatabases updates a database in batch.
func (s *DatabaseService) BatchUpdateDatabases(ctx context.Context, request *v1pb.BatchUpdateDatabasesRequest) (*v1pb.BatchUpdateDatabasesResponse, error) {
	var databases []*store.DatabaseMessage
	projectURI := ""
	for _, req := range request.Requests {
		if req.Database == nil {
			return nil, status.Errorf(codes.InvalidArgument, "database must be set")
		}
		if req.UpdateMask == nil {
			return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
		}
		instanceID, databaseName, err := common.GetInstanceDatabaseID(req.Database.Name)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
		}
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instanceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
		}
		if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionAdminDatabase); err != nil {
			return nil, err
		}
		if projectURI != "" && projectURI != req.Database.Project {
			return nil, status.Errorf(codes.InvalidArgument, "database should use the same project")
		}
		projectURI = req.Database.Project
		databases = append(databases, database)
	}
	// TODO(d): support batch update environment.
	projectID, err := common.GetProjectID(projectURI)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID:  &projectID,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.FailedPrecondition, "project %q is deleted", projectID)
	}

	response := &v1pb.BatchUpdateDatabasesResponse{}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if len(databases) > 0 {
		updatedDatabases, err := s.store.BatchUpdateDatabaseProject(ctx, databases, project.ResourceID, principalID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if err := s.createTransferProjectActivity(ctx, project, principalID, databases...); err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		for _, database := range updatedDatabases {
			response.Databases = append(response.Databases, convertToDatabase(database))
		}
	}
	return response, nil
}

// GetDatabaseMetadata gets the metadata of a database.
func (s *DatabaseService) GetDatabaseMetadata(ctx context.Context, request *v1pb.GetDatabaseMetadataRequest) (*v1pb.DatabaseMetadata, error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(request.Name, common.MetadataSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionManageGeneral); err != nil {
		return nil, err
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &database.ProjectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to sync database schema for database %q, error %v", databaseName, err)
		}
		newDBSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if newDBSchema == nil {
			return nil, status.Errorf(codes.NotFound, "database schema %q not found", databaseName)
		}
		dbSchema = newDBSchema
	}

	var filter *metadataFilter
	if request.Filter != "" {
		schema, table, err := common.GetSchemaTableName(request.Filter)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", filter)
		}
		filter = &metadataFilter{schema: schema, table: table}
	}
	v1pbMetadata := convertDatabaseMetadata(database, dbSchema.Metadata, dbSchema.Config, request.View, filter)

	// Set effective masking level only if filter is set for a table.
	if filter != nil && request.View == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
		dataClassificationSetting, err := s.store.GetDataClassificationSetting(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get data classification setting, error: %v", err)
		}
		maskingRulePolicy, err := s.store.GetMaskingRulePolicy(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get masking rule policy, error: %v", err)
		}
		// Convert the maskingPolicy to a map to reduce the time complexity of searching.
		maskingPolicy, err := s.store.GetMaskingPolicyByDatabaseUID(ctx, database.UID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find masking policy for database %q", databaseName)
		}
		maskingPolicyMap := make(map[maskingPolicyKey]*storepb.MaskData)
		if maskingPolicy != nil {
			for _, maskData := range maskingPolicy.MaskData {
				maskingPolicyMap[maskingPolicyKey{
					schema: maskData.Schema,
					table:  maskData.Table,
					column: maskData.Column,
				}] = maskData
			}
		}

		evaluator := newEmptyMaskingLevelEvaluator().withDataClassificationSetting(dataClassificationSetting).withMaskingRulePolicy(maskingRulePolicy)
		for _, schema := range v1pbMetadata.Schemas {
			if filter.schema != schema.Name {
				continue
			}
			for _, table := range schema.Tables {
				if filter.table != table.Name {
					continue
				}
				for _, column := range table.Columns {
					maskingLevel, err := evaluator.evaluateMaskingLevelOfColumn(database, schema.Name, table.Name, column.Name, column.Classification, project.DataClassificationConfigID, maskingPolicyMap, nil /* Exceptions*/)
					if err != nil {
						return nil, status.Errorf(codes.Internal, "failed to evaluate masking level of column %q, error: %v", column.Name, err)
					}
					v1pbMaskingLevel := convertToV1PBMaskingLevel(maskingLevel)
					column.EffectiveMaskingLevel = v1pbMaskingLevel
				}
			}
		}
	}

	return v1pbMetadata, nil
}

// UpdateDatabaseMetadata updates the metadata config of a database.
func (s *DatabaseService) UpdateDatabaseMetadata(ctx context.Context, request *v1pb.UpdateDatabaseMetadataRequest) (*v1pb.DatabaseMetadata, error) {
	if request.DatabaseMetadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty database config")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(request.DatabaseMetadata.Name, common.MetadataSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionAdminDatabase); err != nil {
		return nil, err
	}

	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if dbSchema == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "database schema metadata not found")
	}

	for _, path := range request.UpdateMask.Paths {
		if path == "schema_configs" {
			databaseConfig := convertV1DatabaseConfig(&v1pb.DatabaseConfig{
				Name:          databaseName,
				SchemaConfigs: request.DatabaseMetadata.SchemaConfigs,
			})
			if err := s.store.UpdateDBSchema(ctx, database.UID, &store.UpdateDBSchemaMessage{Config: databaseConfig}, principalID); err != nil {
				return nil, err
			}
		}
	}

	dbSchema, err = s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if dbSchema == nil {
		return nil, status.Errorf(codes.NotFound, "database schema %q not found", databaseName)
	}

	v1pbMetadata := convertDatabaseMetadata(database, dbSchema.Metadata, dbSchema.Config, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_BASIC, nil /* filter */)
	return v1pbMetadata, nil
}

// GetDatabaseSchema gets the schema of a database.
func (s *DatabaseService) GetDatabaseSchema(ctx context.Context, request *v1pb.GetDatabaseSchemaRequest) (*v1pb.DatabaseSchema, error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(request.Name, common.SchemaSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionManageGeneral); err != nil {
		return nil, err
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to sync database schema for database %q, error %v", databaseName, err)
		}
		newDBSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if newDBSchema == nil {
			return nil, status.Errorf(codes.NotFound, "database schema %q not found", databaseName)
		}
		dbSchema = newDBSchema
	}
	// We only support MySQL engine for now.
	schema := string(dbSchema.Schema)
	if request.SdlFormat {
		switch instance.Engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			sdlSchema, err := transform.SchemaTransform(storepb.Engine_MYSQL, schema)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert schema to sdl format, error %v", err.Error())
			}
			schema = sdlSchema
		}
	}
	return &v1pb.DatabaseSchema{Schema: schema}, nil
}

// GetBackupSetting gets the backup setting of a database.
func (s *DatabaseService) GetBackupSetting(ctx context.Context, request *v1pb.GetBackupSettingRequest) (*v1pb.BackupSetting, error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(request.Name, common.BackupSettingSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionManageGeneral); err != nil {
		return nil, err
	}
	backupSetting, err := s.store.GetBackupSettingV2(ctx, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if backupSetting == nil {
		// If the backup setting is not found, return the default backup setting.
		return getDefaultBackupSetting(instance.ResourceID, database.DatabaseName), nil
	}
	return convertToBackupSetting(backupSetting, instance.ResourceID, database.DatabaseName)
}

// UpdateBackupSetting updates the backup setting of a database.
func (s *DatabaseService) UpdateBackupSetting(ctx context.Context, request *v1pb.UpdateBackupSettingRequest) (*v1pb.BackupSetting, error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(request.Setting.Name, common.BackupSettingSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionAdminDatabase); err != nil {
		return nil, err
	}
	backupSetting, err := s.validateAndConvertToStoreBackupSetting(ctx, request.Setting, database)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	backupSetting, err = s.store.UpsertBackupSettingV2(ctx, principalID, backupSetting)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToBackupSetting(backupSetting, instance.ResourceID, database.DatabaseName)
}

// ListBackups lists the backups of a database.
func (s *DatabaseService) ListBackups(ctx context.Context, request *v1pb.ListBackupsRequest) (*v1pb.ListBackupsResponse, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionManageGeneral); err != nil {
		return nil, err
	}

	rowStatus := api.Normal
	existedBackupList, err := s.store.ListBackupV2(ctx, &store.FindBackupMessage{
		DatabaseUID: &database.UID,
		RowStatus:   &rowStatus,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	var backupList []*v1pb.Backup
	for _, existedBackup := range existedBackupList {
		backupList = append(backupList, convertToBackup(existedBackup, instance.ResourceID, database.DatabaseName))
	}
	return &v1pb.ListBackupsResponse{
		Backups: backupList,
	}, nil
}

// CreateBackup creates a backup of a database.
func (s *DatabaseService) CreateBackup(ctx context.Context, request *v1pb.CreateBackupRequest) (*v1pb.Backup, error) {
	instanceID, databaseName, backupName, err := common.GetInstanceDatabaseIDBackupName(request.Backup.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionAdminDatabase); err != nil {
		return nil, err
	}

	existedBackupList, err := s.store.ListBackupV2(ctx, &store.FindBackupMessage{
		DatabaseUID: &database.UID,
		Name:        &backupName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if len(existedBackupList) > 0 {
		return nil, status.Errorf(codes.AlreadyExists, "backup %q in database %q already exists", backupName, databaseName)
	}

	creatorID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	backup, err := s.backupRunner.ScheduleBackupTask(ctx, database, backupName, api.BackupTypeManual, creatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToBackup(backup, instanceID, databaseName), nil
}

// ListChangeHistories lists the change histories of a database.
func (s *DatabaseService) ListChangeHistories(ctx context.Context, request *v1pb.ListChangeHistoriesRequest) (*v1pb.ListChangeHistoriesResponse, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionManageGeneral); err != nil {
		return nil, err
	}

	var limit, offset int
	if request.PageToken != "" {
		var pageToken storepb.PageToken
		if err := unmarshalPageToken(request.PageToken, &pageToken); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page token: %v", err)
		}
		if pageToken.Limit < 0 {
			return nil, status.Errorf(codes.InvalidArgument, "page size cannot be negative")
		}
		limit = int(pageToken.Limit)
		offset = int(pageToken.Offset)
	} else {
		limit = int(request.PageSize)
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}
	limitPlusOne := limit + 1

	truncateSize := 512
	if s.profile.Mode == common.ReleaseModeDev {
		truncateSize = 4
	}
	find := &store.FindInstanceChangeHistoryMessage{
		InstanceID:   &instance.UID,
		DatabaseID:   &database.UID,
		Limit:        &limitPlusOne,
		Offset:       &offset,
		TruncateSize: truncateSize,
	}
	if request.View == v1pb.ChangeHistoryView_CHANGE_HISTORY_VIEW_FULL {
		find.ShowFull = true
	}
	if request.Filter != "" {
		find.ResourcesFilter = &request.Filter
	}
	changeHistories, err := s.store.ListInstanceChangeHistory(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list change history, error: %v", err)
	}

	if len(changeHistories) == limitPlusOne {
		nextPageToken, err := getPageToken(limit, offset+limit)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		converted, err := convertToChangeHistories(changeHistories[:limit])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert change histories, error: %v", err)
		}
		return &v1pb.ListChangeHistoriesResponse{
			ChangeHistories: converted,
			NextPageToken:   nextPageToken,
		}, nil
	}

	// no subsequent pages
	converted, err := convertToChangeHistories(changeHistories)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert change histories, error: %v", err)
	}
	return &v1pb.ListChangeHistoriesResponse{
		ChangeHistories: converted,
		NextPageToken:   "",
	}, nil
}

// GetChangeHistory gets a change history.
func (s *DatabaseService) GetChangeHistory(ctx context.Context, request *v1pb.GetChangeHistoryRequest) (*v1pb.ChangeHistory, error) {
	instanceID, databaseName, changeHistoryIDStr, err := common.GetInstanceDatabaseIDChangeHistory(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionManageGeneral); err != nil {
		return nil, err
	}

	truncateSize := 4 * 1024 * 1024
	if s.profile.Mode == common.ReleaseModeDev {
		truncateSize = 64
	}
	find := &store.FindInstanceChangeHistoryMessage{
		InstanceID:   &instance.UID,
		DatabaseID:   &database.UID,
		ID:           &changeHistoryIDStr,
		TruncateSize: truncateSize,
	}
	if request.View == v1pb.ChangeHistoryView_CHANGE_HISTORY_VIEW_FULL {
		find.ShowFull = true
	}
	changeHistory, err := s.store.ListInstanceChangeHistory(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list change history, error: %v", err)
	}
	if len(changeHistory) == 0 {
		return nil, status.Errorf(codes.NotFound, "change history %q not found", changeHistoryIDStr)
	}
	if len(changeHistory) > 1 {
		return nil, status.Errorf(codes.Internal, "expect to find one change history, got %d", len(changeHistory))
	}
	converted, err := convertToChangeHistory(changeHistory[0])
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert change history, error: %v", err)
	}
	if request.SdlFormat {
		switch instance.Engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			sdlSchema, err := transform.SchemaTransform(storepb.Engine_MYSQL, converted.Schema)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert schema to sdl format, error %v", err.Error())
			}
			converted.Schema = sdlSchema
			sdlSchema, err = transform.SchemaTransform(storepb.Engine_MYSQL, converted.PrevSchema)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert previous schema to sdl format, error %v", err.Error())
			}
			converted.PrevSchema = sdlSchema
		}
	}
	return converted, nil
}

// DiffSchema diff the database schema.
func (s *DatabaseService) DiffSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (*v1pb.DiffSchemaResponse, error) {
	source, err := s.getSourceSchema(ctx, request)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get source schema, error: %v", err)
	}

	target, err := s.getTargetSchema(ctx, request)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get target schema, error: %v", err)
	}

	engine, err := s.getParserEngine(ctx, request)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get parser engine, error: %v", err)
	}

	diff, err := base.SchemaDiff(engine, source, target, false /* ignoreCaseSensitive */)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute diff between source and target schemas, error: %v", err)
	}

	return &v1pb.DiffSchemaResponse{
		Diff: diff,
	}, nil
}

func (s *DatabaseService) getSourceSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (string, error) {
	if strings.Contains(request.Name, common.ChangeHistoryPrefix) {
		changeHistory, err := s.GetChangeHistory(ctx, &v1pb.GetChangeHistoryRequest{
			Name:      request.Name,
			View:      v1pb.ChangeHistoryView_CHANGE_HISTORY_VIEW_FULL,
			SdlFormat: true,
		})
		if err != nil {
			return "", err
		}
		return changeHistory.Schema, nil
	}

	databaseSchema, err := s.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{
		Name:      fmt.Sprintf("%s/schema", request.Name),
		SdlFormat: request.SdlFormat,
	})
	if err != nil {
		return "", err
	}
	return databaseSchema.Schema, nil
}

func (s *DatabaseService) getTargetSchema(ctx context.Context, request *v1pb.DiffSchemaRequest) (string, error) {
	schema := request.GetSchema()
	changeHistoryID := request.GetChangeHistory()
	// TODO: maybe we will support an empty schema as the target.
	if schema == "" && changeHistoryID == "" {
		return "", status.Errorf(codes.InvalidArgument, "must set the schema or change history id as the target")
	}

	// If the change history id is set, use the schema of the change history as the target.
	if changeHistoryID != "" {
		changeHistory, err := s.GetChangeHistory(ctx, &v1pb.GetChangeHistoryRequest{
			Name:      request.Name,
			View:      v1pb.ChangeHistoryView_CHANGE_HISTORY_VIEW_FULL,
			SdlFormat: true,
		})
		if err != nil {
			return "", err
		}
		schema = changeHistory.Schema
	}

	return schema, nil
}

func (s *DatabaseService) getParserEngine(ctx context.Context, request *v1pb.DiffSchemaRequest) (storepb.Engine, error) {
	var instanceID string
	var engine storepb.Engine

	if strings.Contains(request.Name, common.ChangeHistoryPrefix) {
		insID, _, _, err := common.GetInstanceDatabaseIDChangeHistory(request.Name)
		if err != nil {
			return engine, status.Errorf(codes.InvalidArgument, err.Error())
		}
		instanceID = insID
	} else {
		insID, _, err := common.GetInstanceDatabaseID(request.Name)
		if err != nil {
			return engine, status.Errorf(codes.InvalidArgument, err.Error())
		}
		instanceID = insID
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return engine, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return engine, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	switch instance.Engine {
	case storepb.Engine_POSTGRES:
		engine = storepb.Engine_POSTGRES
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		engine = storepb.Engine_MYSQL
	case storepb.Engine_TIDB:
		engine = storepb.Engine_TIDB
	case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE:
		engine = storepb.Engine_ORACLE
	default:
		return engine, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid engine type %v", instance.Engine))
	}

	return engine, nil
}

func convertToChangeHistories(h []*store.InstanceChangeHistoryMessage) ([]*v1pb.ChangeHistory, error) {
	var changeHistories []*v1pb.ChangeHistory
	for _, history := range h {
		converted, err := convertToChangeHistory(history)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert change history")
		}
		changeHistories = append(changeHistories, converted)
	}
	return changeHistories, nil
}

func convertToChangeHistory(h *store.InstanceChangeHistoryMessage) (*v1pb.ChangeHistory, error) {
	v1pbHistory := &v1pb.ChangeHistory{
		Name:              fmt.Sprintf("%s%s/%s%s/%s%v", common.InstanceNamePrefix, h.InstanceID, common.DatabaseIDPrefix, h.DatabaseName, common.ChangeHistoryPrefix, h.UID),
		Uid:               h.UID,
		Creator:           fmt.Sprintf("users/%s", h.Creator.Email),
		Updater:           fmt.Sprintf("users/%s", h.Updater.Email),
		CreateTime:        timestamppb.New(time.Unix(h.CreatedTs, 0)),
		UpdateTime:        timestamppb.New(time.Unix(h.UpdatedTs, 0)),
		ReleaseVersion:    h.ReleaseVersion,
		Source:            convertToChangeHistorySource(h.Source),
		Type:              convertToChangeHistoryType(h.Type),
		Status:            convertToChangeHistoryStatus(h.Status),
		Version:           h.Version.Version,
		Description:       h.Description,
		Statement:         h.Statement,
		Schema:            h.Schema,
		PrevSchema:        h.SchemaPrev,
		ExecutionDuration: durationpb.New(time.Duration(h.ExecutionDurationNs)),
		Issue:             "",
	}
	if h.SheetID != nil {
		v1pbHistory.StatementSheet = fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, h.IssueProjectID, common.SheetIDPrefix, *h.SheetID)
	}
	if h.IssueUID != nil {
		v1pbHistory.Issue = fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, h.IssueProjectID, common.IssuePrefix, *h.IssueUID)
	}
	if h.Payload != nil && h.Payload.ChangedResources != nil {
		v1pbHistory.ChangedResources = convertToChangedResources(h.Payload.ChangedResources)
	}
	return v1pbHistory, nil
}

func convertToChangedResources(r *storepb.ChangedResources) *v1pb.ChangedResources {
	if r == nil {
		return nil
	}
	result := &v1pb.ChangedResources{}
	for _, database := range r.Databases {
		v1Database := &v1pb.ChangedResourceDatabase{
			Name:    database.Name,
			Schemas: []*v1pb.ChangedResourceSchema{},
		}
		for _, schema := range database.Schemas {
			v1Schema := &v1pb.ChangedResourceSchema{
				Name:   schema.Name,
				Tables: []*v1pb.ChangedResourceTable{},
			}
			for _, table := range schema.Tables {
				v1Schema.Tables = append(v1Schema.Tables, &v1pb.ChangedResourceTable{
					Name: table.Name,
				})
			}
			sort.Slice(v1Schema.Tables, func(i, j int) bool {
				return v1Schema.Tables[i].Name < v1Schema.Tables[j].Name
			})
			v1Database.Schemas = append(v1Database.Schemas, v1Schema)
		}
		sort.Slice(v1Database.Schemas, func(i, j int) bool {
			return v1Database.Schemas[i].Name < v1Database.Schemas[j].Name
		})
		result.Databases = append(result.Databases, v1Database)
	}
	sort.Slice(result.Databases, func(i, j int) bool {
		return result.Databases[i].Name < result.Databases[j].Name
	})
	return result
}

func convertToPushEvent(e *storepb.PushEvent) *v1pb.PushEvent {
	if e == nil {
		return nil
	}
	return &v1pb.PushEvent{
		VcsType:            convertToVcsType(e.VcsType),
		BaseDir:            e.BaseDir,
		Ref:                e.Ref,
		Before:             e.Before,
		After:              e.After,
		RepositoryId:       e.RepositoryId,
		RepositoryUrl:      e.RepositoryUrl,
		RepositoryFullPath: e.RepositoryFullPath,
		AuthorName:         e.AuthorName,
		Commits:            convertToCommits(e.Commits),
		FileCommit:         convertToFileCommit(e.FileCommit),
	}
}

func convertToVcsType(t storepb.VcsType) v1pb.VcsType {
	switch t {
	case storepb.VcsType_GITLAB:
		return v1pb.VcsType_GITLAB
	case storepb.VcsType_GITHUB:
		return v1pb.VcsType_GITHUB
	case storepb.VcsType_BITBUCKET:
		return v1pb.VcsType_BITBUCKET
	case storepb.VcsType_VCS_TYPE_UNSPECIFIED:
		return v1pb.VcsType_VCS_TYPE_UNSPECIFIED
	default:
		return v1pb.VcsType_VCS_TYPE_UNSPECIFIED
	}
}

func convertToCommits(commits []*storepb.Commit) []*v1pb.Commit {
	var converted []*v1pb.Commit
	for _, c := range commits {
		converted = append(converted, &v1pb.Commit{
			Id:           c.Id,
			Title:        c.Title,
			Message:      c.Message,
			CreatedTime:  timestamppb.New(time.Unix(c.CreatedTs, 0)),
			Url:          c.Url,
			AuthorName:   c.AuthorName,
			AuthorEmail:  c.AuthorEmail,
			AddedList:    c.AddedList,
			ModifiedList: c.ModifiedList,
		})
	}
	return converted
}

func convertToFileCommit(c *storepb.FileCommit) *v1pb.FileCommit {
	if c == nil {
		return nil
	}
	return &v1pb.FileCommit{
		Id:          c.Id,
		Title:       c.Title,
		Message:     c.Message,
		CreatedTime: timestamppb.New(time.Unix(c.CreatedTs, 0)),
		Url:         c.Url,
		AuthorName:  c.AuthorName,
		AuthorEmail: c.AuthorEmail,
		Added:       c.Added,
	}
}

func convertToChangeHistorySource(source db.MigrationSource) v1pb.ChangeHistory_Source {
	switch source {
	case db.UI:
		return v1pb.ChangeHistory_UI
	case db.VCS:
		return v1pb.ChangeHistory_VCS
	case db.LIBRARY:
		return v1pb.ChangeHistory_LIBRARY
	default:
		return v1pb.ChangeHistory_SOURCE_UNSPECIFIED
	}
}

func convertToChangeHistoryType(t db.MigrationType) v1pb.ChangeHistory_Type {
	switch t {
	case db.Baseline:
		return v1pb.ChangeHistory_BASELINE
	case db.Migrate:
		return v1pb.ChangeHistory_MIGRATE
	case db.MigrateSDL:
		return v1pb.ChangeHistory_MIGRATE_SDL
	case db.Branch:
		return v1pb.ChangeHistory_BRANCH
	case db.Data:
		return v1pb.ChangeHistory_DATA
	default:
		return v1pb.ChangeHistory_TYPE_UNSPECIFIED
	}
}

func convertToChangeHistoryStatus(s db.MigrationStatus) v1pb.ChangeHistory_Status {
	switch s {
	case db.Pending:
		return v1pb.ChangeHistory_PENDING
	case db.Done:
		return v1pb.ChangeHistory_DONE
	case db.Failed:
		return v1pb.ChangeHistory_FAILED
	default:
		return v1pb.ChangeHistory_STATUS_UNSPECIFIED
	}
}

// ListSecrets lists the secrets of a database.
func (s *DatabaseService) ListSecrets(ctx context.Context, request *v1pb.ListSecretsRequest) (*v1pb.ListSecretsResponse, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionAdminDatabase); err != nil {
		return nil, err
	}

	return &v1pb.ListSecretsResponse{
		Secrets: stripeAndConvertToServiceSecrets(database.Secrets, database.InstanceID, database.DatabaseName),
	}, nil
}

// UpdateSecret updates a secret of a database.
func (s *DatabaseService) UpdateSecret(ctx context.Context, request *v1pb.UpdateSecretRequest) (*v1pb.Secret, error) {
	if request.Secret == nil {
		return nil, status.Errorf(codes.InvalidArgument, "secret is required")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	instanceID, databaseName, updateSecretName, err := common.GetInstanceDatabaseIDSecretName(request.Secret.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureEncryptedSecrets, instance); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionAdminDatabase); err != nil {
		return nil, err
	}

	// We retrieve the secret from the database, convert secrets to map, upsert the new secret and store it back.
	// But if two processes are doing this at the same time, the second one will override the first one.
	// It is not a big deal for now because it's not a common case and users can find that secret he set is not existed or not correct.
	secretsMap := make(map[string]*storepb.SecretItem)
	if database.Secrets != nil {
		for _, secret := range database.Secrets.Items {
			secretsMap[secret.Name] = secret
		}
	}

	var newSecret storepb.SecretItem
	if _, ok := secretsMap[updateSecretName]; !ok {
		// If the secret is not existed and allow_missing is false, we will not create it.
		if !request.AllowMissing {
			return nil, status.Errorf(codes.NotFound, "secret %q not found", updateSecretName)
		}
		newSecret.Name = updateSecretName
		newSecret.Value = request.Secret.Value
		newSecret.Description = request.Secret.Description
	} else {
		oldSecret := secretsMap[updateSecretName]
		newSecret.Name = oldSecret.Name
		newSecret.Value = oldSecret.Value
		newSecret.Description = oldSecret.Description
		for _, path := range request.UpdateMask.Paths {
			switch path {
			case "value":
				newSecret.Value = request.Secret.Value
			case "name":
				// We don't allow users to update the name of a secret.
				return nil, status.Errorf(codes.InvalidArgument, "name of a secret is not allowed to be updated")
			case "description":
				newSecret.Description = request.Secret.Description
			}
		}
	}
	if err := isSecretValid(&newSecret); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	secretsMap[updateSecretName] = &newSecret
	// Flatten the map to a slice.
	var secretItems []*storepb.SecretItem
	for _, secret := range secretsMap {
		secretItems = append(secretItems, secret)
	}
	var updateDatabaseMessage store.UpdateDatabaseMessage
	updateDatabaseMessage.Secrets = &storepb.Secrets{
		Items: secretItems,
	}
	updateDatabaseMessage.InstanceID = database.InstanceID
	updateDatabaseMessage.DatabaseName = database.DatabaseName
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	updatedDatabase, err := s.store.UpdateDatabase(ctx, &updateDatabaseMessage, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// Get the secret from the updated database.
	for _, secret := range updatedDatabase.Secrets.Items {
		if secret.Name == updateSecretName {
			return stripeAndConvertToServiceSecret(secret, updatedDatabase.InstanceID, updatedDatabase.DatabaseName), nil
		}
	}
	return &v1pb.Secret{}, nil
}

// DeleteSecret deletes a secret of a database.
func (s *DatabaseService) DeleteSecret(ctx context.Context, request *v1pb.DeleteSecretRequest) (*emptypb.Empty, error) {
	instanceID, databaseName, secretName, err := common.GetInstanceDatabaseIDSecretName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(api.FeatureEncryptedSecrets, instance); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	if err := s.checkDatabasePermission(ctx, database.ProjectID, api.ProjectPermissionAdminDatabase); err != nil {
		return nil, err
	}

	// We retrieve the secret from the database, convert secrets to map, upsert the new secret and store it back.
	// But if two processes are doing this at the same time, the second one will override the first one.
	// It is not a big deal for now because it's not a common case and users can find that secret he set is not existed or not correct.
	secretsMap := make(map[string]*storepb.SecretItem)
	if database.Secrets != nil {
		for _, secret := range database.Secrets.Items {
			secretsMap[secret.Name] = secret
		}
	}
	delete(secretsMap, secretName)

	// Flatten the map to a slice.
	var secretItems []*storepb.SecretItem
	for _, secret := range secretsMap {
		secretItems = append(secretItems, secret)
	}
	var updateDatabaseMessage store.UpdateDatabaseMessage
	updateDatabaseMessage.Secrets = &storepb.Secrets{
		Items: secretItems,
	}
	updateDatabaseMessage.InstanceID = database.InstanceID
	updateDatabaseMessage.DatabaseName = database.DatabaseName
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if _, err := s.store.UpdateDatabase(ctx, &updateDatabaseMessage, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *DatabaseService) checkDatabasePermission(ctx context.Context, projectID string, permission api.ProjectPermissionType) error {
	role, ok := ctx.Value(common.RoleContextKey).(api.Role)
	if !ok {
		return status.Errorf(codes.Internal, "role not found")
	}
	if isOwnerOrDBA(role) {
		return nil
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return status.Errorf(codes.Internal, "principal ID not found")
	}
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &projectID})
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}

	if permission == api.ProjectPermissionManageGeneral {
		if !isProjectMember(principalID, policy) {
			return status.Errorf(codes.PermissionDenied, "permission denied")
		}
		return nil
	}

	projectRoles := make(map[common.ProjectRole]bool)
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member.ID == principalID {
				projectRoles[common.ProjectRole(binding.Role)] = true
				break
			}
		}
	}

	if !api.ProjectPermission(permission, s.licenseService.GetEffectivePlan(), projectRoles) {
		return status.Errorf(codes.PermissionDenied, "permission denied")
	}

	return nil
}

type totalValue struct {
	totalQueryTime time.Duration
	totalCount     int64
}

// ListSlowQueries lists the slow queries.
func (s *DatabaseService) ListSlowQueries(ctx context.Context, request *v1pb.ListSlowQueriesRequest) (*v1pb.ListSlowQueriesResponse, error) {
	findDatabase := &store.FindDatabaseMessage{}
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if instanceID != "-" {
		findDatabase.InstanceID = &instanceID
	}
	if databaseName != "-" {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
		}
		findDatabase.DatabaseName = &databaseName
		findDatabase.IgnoreCaseSensitive = store.IgnoreDatabaseAndTableCaseSensitive(instance)
	}

	filters, err := parseFilter(request.Filter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var startLogDate, endLogDate *time.Time
	for _, expr := range filters {
		switch expr.key {
		case filterKeyEnvironment:
			reg := regexp.MustCompile(`^environments/(.+)`)
			match := reg.FindStringSubmatch(expr.value)
			if len(match) != 2 {
				return nil, status.Errorf(codes.InvalidArgument, "invalid environment filter %q", expr.value)
			}
			findDatabase.EffectiveEnvironmentID = &match[1]
		case filterKeyProject:
			reg := regexp.MustCompile(`^projects/(.+)`)
			match := reg.FindStringSubmatch(expr.value)
			if len(match) != 2 {
				return nil, status.Errorf(codes.InvalidArgument, "invalid project filter %q", expr.value)
			}
			findDatabase.ProjectID = &match[1]
		case filterKeyStartTime:
			switch expr.operator {
			case comparatorTypeGreater:
				if startLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.value)
				}
				t = t.AddDate(0, 0, 1).UTC()
				startLogDate = &t
			case comparatorTypeGreaterEqual:
				if startLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.value)
				}
				t = t.UTC()
				startLogDate = &t
			case comparatorTypeLess:
				if endLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.value)
				}
				t = t.UTC()
				endLogDate = &t
			case comparatorTypeLessEqual:
				if endLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.value)
				}
				t = t.AddDate(0, 0, 1).UTC()
				endLogDate = &t
			default:
				return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q %q %q", expr.key, expr.operator, expr.value)
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter key %q", expr.key)
		}
	}

	orderByKeys, err := parseOrderBy(request.OrderBy)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := validSlowQueryOrderByKey(orderByKeys); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	databases, err := s.store.ListDatabases(ctx, findDatabase)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find database list %q", err.Error())
	}

	var canAccessDBs []*store.DatabaseMessage

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user %q", err.Error())
	}
	switch user.Role {
	case api.Owner, api.DBA:
		canAccessDBs = databases
	case api.Developer:
		for _, database := range databases {
			policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &database.ProjectID})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find project policy %q", err.Error())
			}
			if isProjectOwnerOrDeveloper(principalID, policy) {
				canAccessDBs = append(canAccessDBs, database)
			}
		}
	default:
		return nil, status.Errorf(codes.PermissionDenied, "unknown role %q", user.Role)
	}

	result := &v1pb.ListSlowQueriesResponse{}
	instanceMap := make(map[string]*totalValue)

	for _, database := range canAccessDBs {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &database.InstanceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find instance %q", err.Error())
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, "instance %q not found", database.InstanceID)
		}
		listSlowQuery := &store.ListSlowQueryMessage{
			InstanceUID:  &instance.UID,
			DatabaseUID:  &database.UID,
			StartLogDate: startLogDate,
			EndLogDate:   endLogDate,
		}
		logs, err := s.store.ListSlowQuery(ctx, listSlowQuery)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find slow query %q", err.Error())
		}

		for _, log := range logs {
			result.SlowQueryLogs = append(result.SlowQueryLogs, convertToSlowQueryLog(database.InstanceID, database.DatabaseName, database.ProjectID, log))
			if value, exists := instanceMap[database.InstanceID]; exists {
				value.totalQueryTime += log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)
				value.totalCount += log.Statistics.Count
			} else {
				instanceMap[database.InstanceID] = &totalValue{
					totalQueryTime: log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count),
					totalCount:     log.Statistics.Count,
				}
			}
		}
	}

	for _, log := range result.SlowQueryLogs {
		instanceID, _, err := common.GetInstanceDatabaseID(log.Resource)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get instance id %q", err.Error())
		}
		totalQueryTime := log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)
		log.Statistics.QueryTimePercent = float64(totalQueryTime) / float64(instanceMap[instanceID].totalQueryTime)
		log.Statistics.CountPercent = float64(log.Statistics.Count) / float64(instanceMap[instanceID].totalCount)
	}

	result, err = sortSlowQueryLogResponse(result, orderByKeys)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sort slow query logs %q", err.Error())
	}

	return result, nil
}

func sortSlowQueryLogResponse(response *v1pb.ListSlowQueriesResponse, orderByKeys []orderByKey) (*v1pb.ListSlowQueriesResponse, error) {
	if len(orderByKeys) == 0 {
		orderByKeys = []orderByKey{
			{
				key:      orderByKeyAverageQueryTime,
				isAscend: false,
			},
		}
	}

	if err := validSlowQueryOrderByKey(orderByKeys); err != nil {
		return nil, err
	}

	sort.Slice(response.SlowQueryLogs, func(i, j int) bool {
		for _, key := range orderByKeys {
			switch key.key {
			case orderByKeyCount:
				lCount := response.SlowQueryLogs[i].Statistics.Count
				rCount := response.SlowQueryLogs[j].Statistics.Count
				if lCount != rCount {
					if key.isAscend {
						return lCount < rCount
					}
					return lCount > rCount
				}
			case orderByKeyLatestLogTime:
				lTime := response.SlowQueryLogs[i].Statistics.LatestLogTime.AsTime()
				rTime := response.SlowQueryLogs[j].Statistics.LatestLogTime.AsTime()
				if !lTime.Equal(rTime) {
					if key.isAscend {
						return lTime.Before(rTime)
					}
					return lTime.After(rTime)
				}
			case orderByKeyAverageQueryTime:
				lTime := response.SlowQueryLogs[i].Statistics.AverageQueryTime.AsDuration()
				rTime := response.SlowQueryLogs[j].Statistics.AverageQueryTime.AsDuration()
				if lTime != rTime {
					if key.isAscend {
						return lTime < rTime
					}
					return lTime > rTime
				}
			case orderByKeyMaximumQueryTime:
				lDuration := response.SlowQueryLogs[i].Statistics.MaximumQueryTime.AsDuration()
				rDuration := response.SlowQueryLogs[j].Statistics.MaximumQueryTime.AsDuration()
				if lDuration != rDuration {
					if key.isAscend {
						return lDuration < rDuration
					}
					return lDuration > rDuration
				}
			case orderByKeyAverageRowsSent:
				lSent := response.SlowQueryLogs[i].Statistics.AverageRowsSent
				rSent := response.SlowQueryLogs[j].Statistics.AverageRowsSent
				if lSent != rSent {
					if key.isAscend {
						return lSent < rSent
					}
					return lSent > rSent
				}
			case orderByKeyMaximumRowsSent:
				lSent := response.SlowQueryLogs[i].Statistics.MaximumRowsSent
				rSent := response.SlowQueryLogs[j].Statistics.MaximumRowsSent
				if lSent != rSent {
					if key.isAscend {
						return lSent < rSent
					}
					return lSent > rSent
				}
			case orderByKeyAverageRowsExamined:
				lExamined := response.SlowQueryLogs[i].Statistics.AverageRowsExamined
				rExamined := response.SlowQueryLogs[j].Statistics.AverageRowsExamined
				if lExamined != rExamined {
					if key.isAscend {
						return lExamined < rExamined
					}
					return lExamined > rExamined
				}
			case orderByKeyMaximumRowsExamined:
				lExamined := response.SlowQueryLogs[i].Statistics.MaximumRowsExamined
				rExamined := response.SlowQueryLogs[j].Statistics.MaximumRowsExamined
				if lExamined != rExamined {
					if key.isAscend {
						return lExamined < rExamined
					}
					return lExamined > rExamined
				}
			}
		}
		return false
	})

	return response, nil
}

func validSlowQueryOrderByKey(keys []orderByKey) error {
	for _, key := range keys {
		switch key.key {
		// Support order by count, latest_log_time, average_query_time, maximum_query_time,
		// average_rows_sent, maximum_rows_sent, average_rows_examined, maximum_rows_examined for now.
		case orderByKeyCount, orderByKeyLatestLogTime, orderByKeyAverageQueryTime, orderByKeyMaximumQueryTime,
			orderByKeyAverageRowsSent, orderByKeyMaximumRowsSent, orderByKeyAverageRowsExamined, orderByKeyMaximumRowsExamined:
		default:
			return errors.Errorf("invalid order_by key %q", key.key)
		}
	}
	return nil
}

func convertToSlowQueryLog(instanceID string, databaseName string, projectID string, log *v1pb.SlowQueryLog) *v1pb.SlowQueryLog {
	return &v1pb.SlowQueryLog{
		Resource:   fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, instanceID, common.DatabaseIDPrefix, databaseName),
		Project:    fmt.Sprintf("%s%s", common.ProjectNamePrefix, projectID),
		Statistics: log.Statistics,
	}
}

func convertToBackup(backup *store.BackupMessage, instanceID string, databaseName string) *v1pb.Backup {
	createTime := timestamppb.New(time.Unix(backup.CreatedTs, 0))
	updateTime := timestamppb.New(time.Unix(backup.UpdatedTs, 0))
	backupState := v1pb.Backup_BACKUP_STATE_UNSPECIFIED
	switch backup.Status {
	case api.BackupStatusPendingCreate:
		backupState = v1pb.Backup_PENDING_CREATE
	case api.BackupStatusDone:
		backupState = v1pb.Backup_DONE
	case api.BackupStatusFailed:
		backupState = v1pb.Backup_FAILED
	}
	backupType := v1pb.Backup_BACKUP_TYPE_UNSPECIFIED
	switch backup.BackupType {
	case api.BackupTypeManual:
		backupType = v1pb.Backup_MANUAL
	case api.BackupTypeAutomatic:
		backupType = v1pb.Backup_AUTOMATIC
	case api.BackupTypePITR:
		backupType = v1pb.Backup_PITR
	}
	return &v1pb.Backup{
		Name:       fmt.Sprintf("%s%s/%s%s/%s%s", common.InstanceNamePrefix, instanceID, common.DatabaseIDPrefix, databaseName, common.BackupPrefix, backup.Name),
		CreateTime: createTime,
		UpdateTime: updateTime,
		State:      backupState,
		BackupType: backupType,
		Comment:    backup.Comment,
		Uid:        fmt.Sprintf("%d", backup.UID),
	}
}

func convertToDatabase(database *store.DatabaseMessage) *v1pb.Database {
	syncState := v1pb.State_STATE_UNSPECIFIED
	switch database.SyncState {
	case api.OK:
		syncState = v1pb.State_ACTIVE
	case api.NotFound:
		syncState = v1pb.State_DELETED
	}
	environment, effectiveEnvironment := "", ""
	if database.EnvironmentID != "" {
		environment = fmt.Sprintf("%s%s", common.EnvironmentNamePrefix, database.EnvironmentID)
	}
	if database.EffectiveEnvironmentID != "" {
		effectiveEnvironment = fmt.Sprintf("%s%s", common.EnvironmentNamePrefix, database.EffectiveEnvironmentID)
	}
	return &v1pb.Database{
		Name:                 fmt.Sprintf("instances/%s/databases/%s", database.InstanceID, database.DatabaseName),
		Uid:                  fmt.Sprintf("%d", database.UID),
		SyncState:            syncState,
		SuccessfulSyncTime:   timestamppb.New(time.Unix(database.SuccessfulSyncTimeTs, 0)),
		Project:              fmt.Sprintf("%s%s", common.ProjectNamePrefix, database.ProjectID),
		Environment:          environment,
		EffectiveEnvironment: effectiveEnvironment,
		SchemaVersion:        database.SchemaVersion.Version,
		Labels:               database.Metadata.Labels,
	}
}

type metadataFilter struct {
	schema string
	table  string
}

func convertDatabaseMetadata(database *store.DatabaseMessage, metadata *storepb.DatabaseSchemaMetadata, config *storepb.DatabaseConfig, requestView v1pb.DatabaseMetadataView, filter *metadataFilter) *v1pb.DatabaseMetadata {
	m := &v1pb.DatabaseMetadata{
		Name:         fmt.Sprintf("%s%s/%s%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName, common.MetadataSuffix),
		CharacterSet: metadata.CharacterSet,
		Collation:    metadata.Collation,
	}
	for _, schema := range metadata.Schemas {
		if filter != nil && filter.schema != schema.Name {
			continue
		}
		s := &v1pb.SchemaMetadata{
			Name: schema.Name,
		}
		for _, table := range schema.Tables {
			if filter != nil && filter.table != table.Name {
				continue
			}
			s.Tables = append(s.Tables, convertTableMetadata(table, requestView))
		}
		// Only return table for request with a filter.
		if filter != nil {
			continue
		}
		for _, view := range schema.Views {
			v1View := &v1pb.ViewMetadata{
				Name: view.Name,
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				var dependentColumnList []*v1pb.DependentColumn
				for _, dependentColumn := range view.DependentColumns {
					dependentColumnList = append(dependentColumnList, &v1pb.DependentColumn{
						Schema: dependentColumn.Schema,
						Table:  dependentColumn.Table,
						Column: dependentColumn.Column,
					})
				}
				v1View.Definition = view.Definition
				v1View.Comment = view.Comment
				v1View.DependentColumns = dependentColumnList
			}

			s.Views = append(s.Views, v1View)
		}
		for _, function := range schema.Functions {
			v1Func := &v1pb.FunctionMetadata{
				Name: function.Name,
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				v1Func.Definition = function.Definition
			}
			s.Functions = append(s.Functions, v1Func)
		}
		for _, task := range schema.Tasks {
			v1Task := &v1pb.TaskMetadata{
				Name: task.Name,
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				v1Task.Id = task.Id
				v1Task.Owner = task.Owner
				v1Task.Comment = task.Comment
				v1Task.Warehouse = task.Warehouse
				v1Task.Schedule = task.Schedule
				v1Task.Predecessors = task.Predecessors
				v1Task.State = v1pb.TaskMetadata_State(task.State)
				v1Task.Condition = task.Condition
				v1Task.Definition = task.Definition
			}
			s.Tasks = append(s.Tasks, v1Task)
		}
		for _, stream := range schema.Streams {
			v1Stream := &v1pb.StreamMetadata{
				Name: stream.Name,
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				v1Stream.TableName = stream.TableName
				v1Stream.Owner = stream.Owner
				v1Stream.Comment = stream.Comment
				v1Stream.Type = v1pb.StreamMetadata_Type(stream.Type)
				v1Stream.Stale = stream.Stale
				v1Stream.Mode = v1pb.StreamMetadata_Mode(stream.Mode)
				v1Stream.Definition = stream.Definition
			}
			s.Streams = append(s.Streams, v1Stream)
		}
		m.Schemas = append(m.Schemas, s)
	}
	for _, extension := range metadata.Extensions {
		m.Extensions = append(m.Extensions, &v1pb.ExtensionMetadata{
			Name:        extension.Name,
			Schema:      extension.Schema,
			Version:     extension.Version,
			Description: extension.Description,
		})
	}

	if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
		databaseConfig := convertDatabaseConfig(config, filter)
		if databaseConfig != nil {
			m.SchemaConfigs = databaseConfig.SchemaConfigs
		}
	}
	return m
}

func convertDatabaseConfig(config *storepb.DatabaseConfig, filter *metadataFilter) *v1pb.DatabaseConfig {
	if config == nil {
		return nil
	}
	databaseConfig := &v1pb.DatabaseConfig{
		Name: config.Name,
	}
	for _, schema := range config.SchemaConfigs {
		if filter != nil && filter.schema != schema.Name {
			continue
		}
		s := &v1pb.SchemaConfig{
			Name: schema.Name,
		}
		for _, table := range schema.TableConfigs {
			if filter != nil && filter.table != table.Name {
				continue
			}
			s.TableConfigs = append(s.TableConfigs, convertTableConfig(table))
		}
		databaseConfig.SchemaConfigs = append(databaseConfig.SchemaConfigs, s)
	}
	return databaseConfig
}

func convertTableConfig(table *storepb.TableConfig) *v1pb.TableConfig {
	if table == nil {
		return nil
	}
	t := &v1pb.TableConfig{
		Name: table.Name,
	}
	for _, column := range table.ColumnConfigs {
		t.ColumnConfigs = append(t.ColumnConfigs, convertColumnConfig(column))
	}
	return t
}

func convertColumnConfig(column *storepb.ColumnConfig) *v1pb.ColumnConfig {
	if column == nil {
		return nil
	}
	return &v1pb.ColumnConfig{
		Name:           column.Name,
		SemanticTypeId: column.SemanticTypeId,
		Labels:         column.Labels,
	}
}

func convertV1DatabaseConfig(databaseConfig *v1pb.DatabaseConfig) *storepb.DatabaseConfig {
	if databaseConfig == nil {
		return nil
	}

	config := &storepb.DatabaseConfig{
		Name: databaseConfig.Name,
	}
	for _, schema := range databaseConfig.SchemaConfigs {
		s := &storepb.SchemaConfig{
			Name: schema.Name,
		}
		for _, table := range schema.TableConfigs {
			t := &storepb.TableConfig{
				Name: table.Name,
			}
			for _, column := range table.ColumnConfigs {
				t.ColumnConfigs = append(t.ColumnConfigs, &storepb.ColumnConfig{
					Name:           column.Name,
					SemanticTypeId: column.SemanticTypeId,
					Labels:         column.Labels,
				})
			}
			s.TableConfigs = append(s.TableConfigs, t)
		}
		config.SchemaConfigs = append(config.SchemaConfigs, s)
	}
	return config
}

func convertV1TableConfig(table *v1pb.TableConfig) *storepb.TableConfig {
	if table == nil {
		return nil
	}

	t := &storepb.TableConfig{
		Name: table.Name,
	}
	for _, column := range table.ColumnConfigs {
		t.ColumnConfigs = append(t.ColumnConfigs, convertV1ColumnConfig(column))
	}
	return t
}

func convertV1ColumnConfig(column *v1pb.ColumnConfig) *storepb.ColumnConfig {
	if column == nil {
		return nil
	}

	return &storepb.ColumnConfig{
		Name:           column.Name,
		SemanticTypeId: column.SemanticTypeId,
		Labels:         column.Labels,
	}
}

func (s *DatabaseService) createTransferProjectActivity(ctx context.Context, newProject *store.ProjectMessage, updaterID int, databases ...*store.DatabaseMessage) error {
	var creates []*store.ActivityMessage
	for _, database := range databases {
		oldProject, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
		if err != nil {
			return err
		}
		bytes, err := json.Marshal(api.ActivityProjectDatabaseTransferPayload{
			DatabaseID:   database.UID,
			DatabaseName: database.DatabaseName,
		})
		if err != nil {
			return err
		}
		creates = append(creates,
			&store.ActivityMessage{
				CreatorUID:   updaterID,
				ContainerUID: oldProject.UID,
				Type:         api.ActivityProjectDatabaseTransfer,
				Level:        api.ActivityInfo,
				Comment:      fmt.Sprintf("Transferred out database %q to project %q.", database.DatabaseName, newProject.Title),
				Payload:      string(bytes),
			},
			&store.ActivityMessage{
				CreatorUID:   updaterID,
				ContainerUID: newProject.UID,
				Type:         api.ActivityProjectDatabaseTransfer,
				Level:        api.ActivityInfo,
				Comment:      fmt.Sprintf("Transferred in database %q from project %q.", database.DatabaseName, oldProject.Title),
				Payload:      string(bytes),
			},
		)
	}
	if _, err := s.store.BatchCreateActivityV2(ctx, creates); err != nil {
		slog.Warn("failed to create activities for database project updates", log.BBError(err))
	}
	return nil
}

func getDefaultBackupSetting(instanceID, databaseName string) *v1pb.BackupSetting {
	sevenDays, err := convertPeriodTsToDuration(int(time.Duration(7 * 24 * time.Hour).Seconds()))
	if err != nil {
		slog.Warn("failed to convert period ts to duration", log.BBError(err))
	}
	return &v1pb.BackupSetting{
		Name:                 fmt.Sprintf("%s%s/%s%s/%s", common.InstanceNamePrefix, instanceID, common.DatabaseIDPrefix, databaseName, common.BackupSettingSuffix),
		BackupRetainDuration: sevenDays,
		CronSchedule:         "", /* Disable automatic backup */
		HookUrl:              "",
	}
}

func convertToBackupSetting(backupSetting *store.BackupSettingMessage, instanceID, databaseName string) (*v1pb.BackupSetting, error) {
	period, err := convertPeriodTsToDuration(backupSetting.RetentionPeriodTs)
	if err != nil {
		return nil, err
	}
	cronSchedule := ""
	if backupSetting.Enabled {
		cronSchedule = buildSimpleCron(backupSetting.HourOfDay, backupSetting.DayOfWeek)
	}
	return &v1pb.BackupSetting{
		Name:                 fmt.Sprintf("%s%s/%s%s/%s", common.InstanceNamePrefix, instanceID, common.DatabaseIDPrefix, databaseName, common.BackupSettingSuffix),
		BackupRetainDuration: period,
		CronSchedule:         cronSchedule,
		HookUrl:              backupSetting.HookURL,
	}, nil
}

func (s *DatabaseService) validateAndConvertToStoreBackupSetting(ctx context.Context, backupSetting *v1pb.BackupSetting, database *store.DatabaseMessage) (*store.BackupSettingMessage, error) {
	enable := backupSetting.CronSchedule != ""
	hourOfDay := 0
	dayOfWeek := -1
	var err error
	if enable {
		hourOfDay, dayOfWeek, err = parseSimpleCron(backupSetting.CronSchedule)
		if err != nil {
			return nil, err
		}
	}
	periodTs, err := convertDurationToPeriodTs(backupSetting.BackupRetainDuration)
	if err != nil {
		return nil, err
	}
	setting := &store.BackupSettingMessage{
		DatabaseUID:       database.UID,
		Enabled:           enable,
		HourOfDay:         hourOfDay,
		DayOfWeek:         dayOfWeek,
		RetentionPeriodTs: periodTs,
		HookURL:           backupSetting.HookUrl,
	}

	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID:  &database.EffectiveEnvironmentID,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", database.EffectiveEnvironmentID)
	}
	backupPlanPolicy, err := s.store.GetBackupPlanPolicyByEnvID(ctx, environment.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if backupPlanPolicy.Schedule != api.BackupPlanPolicyScheduleUnset {
		if !setting.Enabled {
			return nil, &common.Error{Code: common.Invalid, Err: errors.Errorf("backup setting should not be disabled for backup plan policy schedule %q", backupPlanPolicy.Schedule)}
		}
		switch backupPlanPolicy.Schedule {
		case api.BackupPlanPolicyScheduleDaily:
			if setting.DayOfWeek != -1 {
				return nil, &common.Error{Code: common.Invalid, Err: errors.Errorf("backup setting DayOfWeek should be unset for backup plan policy schedule %q", backupPlanPolicy.Schedule)}
			}
		case api.BackupPlanPolicyScheduleWeekly:
			if setting.DayOfWeek == -1 {
				return nil, &common.Error{Code: common.Invalid, Err: errors.Errorf("backup setting DayOfWeek should be set for backup plan policy schedule %q", backupPlanPolicy.Schedule)}
			}
		}
	}

	return setting, nil
}

// parseSimpleCron parses a simple cron expression(only support hour of day and day of week), and returns them as int.
func parseSimpleCron(cron string) (int, int, error) {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return 0, 0, errors.New("invalid cron expression")
	}
	hourOfDay, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, errors.New("invalid cron hour field")
	}
	if hourOfDay < 0 || hourOfDay > 23 {
		return 0, 0, errors.New("invalid cron hour range")
	}
	weekDay := fields[4]
	// "*" means any day of week.
	if weekDay == "*" {
		return hourOfDay, -1, nil
	}
	dayOfWeek, err := strconv.Atoi(weekDay)
	if err != nil {
		return 0, 0, err
	}
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return 0, 0, errors.New("invalid cron day of week range")
	}
	return hourOfDay, dayOfWeek, nil
}

func buildSimpleCron(hourOfDay int, dayOfWeek int) string {
	if dayOfWeek == -1 {
		return fmt.Sprintf("0 %d * * *", hourOfDay)
	}
	return fmt.Sprintf("0 %d * * %d", hourOfDay, dayOfWeek)
}

func convertDurationToPeriodTs(duration *durationpb.Duration) (int, error) {
	if err := duration.CheckValid(); err != nil {
		return 0, errors.Wrap(err, "invalid duration")
	}
	// Round up to days
	return int(duration.AsDuration().Round(time.Hour * 24).Seconds()), nil
}

func convertPeriodTsToDuration(periodTs int) (*durationpb.Duration, error) {
	if periodTs < 0 {
		return nil, errors.New("invalid period")
	}
	return durationpb.New(time.Duration(periodTs) * time.Second), nil
}

// isProjectOwnerOrDeveloper returns whether a principal is a project owner or developer in the project.
func isProjectOwnerOrDeveloper(principalID int, projectPolicy *store.IAMPolicyMessage) bool {
	for _, binding := range projectPolicy.Bindings {
		if binding.Role != api.Owner && binding.Role != api.Developer {
			continue
		}
		for _, member := range binding.Members {
			if member.ID == principalID {
				return true
			}
		}
	}
	return false
}

func stripeAndConvertToServiceSecrets(secrets *storepb.Secrets, instanceID, databaseName string) []*v1pb.Secret {
	var serviceSecrets []*v1pb.Secret
	if secrets == nil || len(secrets.Items) == 0 {
		return serviceSecrets
	}
	for _, secret := range secrets.Items {
		serviceSecrets = append(serviceSecrets, stripeAndConvertToServiceSecret(secret, instanceID, databaseName))
	}
	return serviceSecrets
}

func stripeAndConvertToServiceSecret(secretEntry *storepb.SecretItem, instanceID, databaseName string) *v1pb.Secret {
	return &v1pb.Secret{
		Name:        fmt.Sprintf("%s%s/%s%s/%s%s", common.InstanceNamePrefix, instanceID, common.DatabaseIDPrefix, databaseName, common.SecretNamePrefix, secretEntry.Name),
		Value:       "", /* stripped */
		Description: secretEntry.Description,
	}
}

func isSecretValid(secret *storepb.SecretItem) error {
	// Names can not be empty.
	if secret.Name == "" {
		return errors.Errorf("invalid secret name: %s, name can not be empty", secret.Name)
	}
	// Values can not be empty.
	if secret.Value == "" {
		return errors.Errorf("the value of secret: %s can not be empty", secret.Name)
	}

	// Names must not start with the 'BYTEBASE_' prefix.
	bytebaseCaseInsensitivePrefixRegexp := regexp.MustCompile(`(?i)^BYTEBASE_`)
	if bytebaseCaseInsensitivePrefixRegexp.MatchString(secret.Name) {
		return errors.Errorf("invalid secret name: %s, name must not start with the 'BYTEBASE_' prefix", secret.Name)
	}
	// Names must not start with a number.
	if unicode.IsDigit(rune(secret.Name[0])) {
		return errors.Errorf("invalid secret name: %s, name must not start with a number", secret.Name)
	}

	// Names can only contain alphanumeric characters ([A-Z], [0-9]) or underscores (_). Spaces are not allowed.
	for _, c := range secret.Name {
		if !isUpperCaseLetter(c) && !unicode.IsDigit(c) && c != '_' {
			return errors.Errorf("invalid secret name: %s, expect [A-Z], [0-9], '_', but meet: %v", secret.Name, c)
		}
	}
	return nil
}

func isUpperCaseLetter(c rune) bool {
	return 'A' <= c && c <= 'Z'
}

// AdviseIndex advises the index of a table.
func (s *DatabaseService) AdviseIndex(ctx context.Context, request *v1pb.AdviseIndexRequest) (*v1pb.AdviseIndexResponse, error) {
	if err := s.licenseService.IsFeatureEnabled(api.FeaturePluginOpenAI); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	findDatabase := &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	}
	database, err := s.store.GetDatabaseV2(ctx, findDatabase)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get database: %v", err)
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	switch instance.Engine {
	case storepb.Engine_POSTGRES:
		return s.pgAdviseIndex(ctx, request, database)
	case storepb.Engine_MYSQL:
		return s.mysqlAdviseIndex(ctx, request, instance, database)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "AdviseIndex is not implemented for engine: %v", instance.Engine)
	}
}

func (s *DatabaseService) mysqlAdviseIndex(ctx context.Context, request *v1pb.AdviseIndexRequest, instance *store.InstanceMessage, database *store.DatabaseMessage) (*v1pb.AdviseIndexResponse, error) {
	openaiKeyName := api.SettingPluginOpenAIKey
	key, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &openaiKeyName})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get setting: %v", err)
	}
	if key.Value == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "OpenAI key is not set")
	}

	var schemas []*store.DBSchema

	// Deal with the cross database query.
	resources, err := base.ExtractResourceList(instance.Engine, database.DatabaseName, "", request.Statement)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to extract resource list: %v", err)
	}
	databaseMap := make(map[string]bool)
	for _, resource := range resources {
		databaseMap[resource.Database] = true
	}
	var databases []string
	for database := range databaseMap {
		databases = append(databases, database)
	}
	if len(databases) == 0 {
		databases = append(databases, database.DatabaseName)
	}

	for _, db := range databases {
		findDatabase := &store.FindDatabaseMessage{
			InstanceID:          &instance.ResourceID,
			DatabaseName:        &db,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		}
		database, err := s.store.GetDatabaseV2(ctx, findDatabase)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get database: %v", err)
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, "database %q not found", db)
		}
		schema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get database schema: %v", err)
		}
		schemas = append(schemas, schema)
	}

	var compactBuf bytes.Buffer
	for _, schema := range schemas {
		compactSchema, err := schema.CompactText()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to compact database schema: %v", err)
		}
		if _, err := compactBuf.WriteString(compactSchema); err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to write compact database schema: %v", err)
		}
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: `You are a MySQL index advisor. You answer the question about the index of tables and SQLs. DO NOT EXPLAIN THE ANSWER.`,
		},
		{
			Role: openai.ChatMessageRoleUser,
			Content: `You are an assistant who works as a Magic: The strict MySQL index advisor. Analyze the SQL with schema and existing indexes, then give the advice in the JSON format.
			If the SQL will use the existing index, the current_index field is the index name with database name and table name. Otherwise, the current_index field is "N/A".
			If it is possible to create a new index to speed up the query, the create_index_statement field is the SQL statement to create the index. Otherwise, the create_index_statement field is empty string.
			YOUR ADVICE MUST FOLLOW JSON FORMAT. DO NOT EXPLAIN THE ADVICE.
			Here two examples:
			{"current_index": "index_schema_table_age ON db1.schema_table", "create_index_statement":""}
			{"current_index": "N/A", "create_index_statement":"CREATE INDEX ON db1.schema_table(collected_at, schema_index_id)"}
			` + fmt.Sprintf(`### MySQL schema:\n### %s\n###The SQL is:\n### %s###`, compactBuf.String(), request.Statement),
		},
	}

	generateFunc := func(resp *v1pb.AdviseIndexResponse) error {
		// Generate current index.
		if resp.CurrentIndex != "N/A" {
			// Use regex to extract the index name, database name and table name from "index_schema_table_age ON public.schema_table".
			reg := regexp.MustCompile(`(?i)(.*) ON (.*)\.(.*)`)
			matches := reg.FindStringSubmatch(resp.CurrentIndex)
			if len(matches) != 4 {
				return errors.Errorf("failed to extract index name, database name and table name from %s", resp.CurrentIndex)
			}
			var dbSchema *store.DBSchema
			for _, schema := range schemas {
				if schema.Metadata.Name == matches[2] {
					dbSchema = schema
					break
				}
			}
			if dbSchema == nil {
				return errors.Errorf("database %s doesn't exist", matches[2])
			}
			indexMetadata := dbSchema.FindIndex("", matches[3], matches[1])
			if indexMetadata == nil {
				return errors.Errorf("index %s doesn't exist", resp.CurrentIndex)
			}
			resp.CurrentIndex = fmt.Sprintf("USING %s (%s)", indexMetadata.Type, strings.Join(indexMetadata.Expressions, ", "))
		} else {
			resp.CurrentIndex = "No usable index"
		}

		// Generate suggestion and create index statement.
		if resp.CreateIndexStatement != "" {
			p := tidbparser.New()
			node, err := p.ParseOneStmt(resp.CreateIndexStatement, "", "")
			if err != nil {
				return errors.Errorf("failed to parse create index statement: %v", err)
			}
			switch createIndex := node.(type) {
			case *tidbast.CreateIndexStmt:
				defineString, err := mysqlIndexExpressionList(createIndex)
				if err != nil {
					return errors.Errorf("failed to generate create index statement: %v", err)
				}
				indexType := createIndex.IndexOption.Tp.String()
				if indexType == "" {
					indexType = "BTREE"
				}
				resp.Suggestion = fmt.Sprintf("USING %s (%s)", indexType, defineString)
			default:
				return errors.Errorf("expect create index statement, but got %T", node)
			}
		} else {
			resp.Suggestion = "N/A"
		}

		return nil
	}

	result, err := getOpenAIResponse(ctx, messages, key.Value, generateFunc)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func mysqlIndexExpressionList(node *tidbast.CreateIndexStmt) (string, error) {
	var buf bytes.Buffer
	for i, item := range node.IndexPartSpecifications {
		text, err := restoreNode(item)
		if err != nil {
			return "", err
		}
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return "", err
			}
		}
		if _, err := buf.WriteString(text); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func restoreNode(node tidbast.Node) (string, error) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (s *DatabaseService) pgAdviseIndex(ctx context.Context, request *v1pb.AdviseIndexRequest, database *store.DatabaseMessage) (*v1pb.AdviseIndexResponse, error) {
	openaiKeyName := api.SettingPluginOpenAIKey
	key, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &openaiKeyName})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get setting: %v", err)
	}
	if key.Value == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "OpenAI key is not set")
	}

	schema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get database schema: %v", err)
	}
	compactSchema, err := schema.CompactText()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to compact database schema: %v", err)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: `You are a PostgreSQL index advisor. You answer the question about the index of tables and SQLs. DO NOT EXPLAIN THE ANSWER.`,
		},
		{
			Role: openai.ChatMessageRoleUser,
			Content: `You are an assistant who works as a Magic: The strict PostgreSQL index advisor. Analyze the SQL with schema and existing indexes, then give the advice in the JSON format.
			If the SQL will use the existing index, the current_index field is the index name with schema name and table name. Otherwise, the current_index field is "N/A".
			If it is possible to create a new index to speed up the query, the create_index_statement field is the SQL statement to create the index. Otherwise, the create_index_statement field is empty string.
			YOUR ADVICE MUST FOLLOW JSON FORMAT. DO NOT EXPLAIN THE ADVICE.
			Here two examples:
			{"current_index": "index_schema_table_age ON public.schema_table", "create_index_statement":""}
			{"current_index": "N/A", "create_index_statement":"CREATE INDEX ON public.schema_table(collected_at, schema_index_id)"}
			` + fmt.Sprintf(`### Postgres schema:\n### %s\n###The SQL is:\n### %s###`, compactSchema, request.Statement),
		},
	}

	generateFunc := func(resp *v1pb.AdviseIndexResponse) error {
		// Generate current index.
		if resp.CurrentIndex != "N/A" {
			// Use regex to extract the index name, schema name and table name from "index_schema_table_age ON public.schema_table".
			reg := regexp.MustCompile(`(?i)(.*) ON (.*)\.(.*)`)
			matches := reg.FindStringSubmatch(resp.CurrentIndex)
			if len(matches) != 4 {
				return errors.Errorf("failed to extract index name, schema name and table name from %s", resp.CurrentIndex)
			}
			indexMetadata := schema.FindIndex(matches[2], matches[3], matches[1])
			if indexMetadata == nil {
				return errors.Errorf("index %s doesn't exist", resp.CurrentIndex)
			}
			resp.CurrentIndex = fmt.Sprintf("USING %s (%s)", indexMetadata.Type, strings.Join(indexMetadata.Expressions, ", "))
		} else {
			resp.CurrentIndex = "No usable index"
		}

		// Generate suggestion and create index statement.
		if resp.CreateIndexStatement != "" {
			nodes, err := pgrawparser.Parse(pgrawparser.ParseContext{}, resp.CreateIndexStatement)
			if err != nil {
				return errors.Errorf("failed to parse create index statement: %v", err)
			}
			if len(nodes) != 1 {
				return errors.Errorf("expect 1 statement, but got %d", len(nodes))
			}
			switch node := nodes[0].(type) {
			case *ast.CreateIndexStmt:
				resp.Suggestion = fmt.Sprintf("USING %s (%s)", node.Index.Method, strings.Join(node.Index.GetKeyNameList(), ", "))
			default:
				return errors.Errorf("expect CreateIndexStmt, but got %T", node)
			}
		} else {
			resp.Suggestion = "N/A"
		}

		return nil
	}

	result, err := getOpenAIResponse(ctx, messages, key.Value, generateFunc)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func getOpenAIResponse(ctx context.Context, messages []openai.ChatCompletionMessage, key string, generateResponse func(*v1pb.AdviseIndexResponse) error) (*v1pb.AdviseIndexResponse, error) {
	var result v1pb.AdviseIndexResponse
	successful := false
	var retErr error
	// Retry 5 times if failed.
	for i := 0; i < 5; i++ {
		client := openai.NewClient(key)
		resp, err := client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model:            openai.GPT3Dot5Turbo,
				Messages:         messages,
				Temperature:      0,
				Stop:             []string{"#", ";"},
				TopP:             1.0,
				FrequencyPenalty: 0.0,
				PresencePenalty:  0.0,
			},
		)
		if err != nil {
			retErr = err
			continue
		}
		if err := protojson.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
			retErr = err
			continue
		}
		if err = generateResponse(&result); err != nil {
			retErr = err
			continue
		}
		successful = true
		break
	}

	if !successful {
		return nil, status.Errorf(codes.Internal, "Failed to get index advice, error %v", retErr)
	}
	return &result, nil
}

func convertTableMetadata(table *storepb.TableMetadata, view v1pb.DatabaseMetadataView) *v1pb.TableMetadata {
	if table == nil {
		return nil
	}
	t := &v1pb.TableMetadata{
		Name:           table.Name,
		Engine:         table.Engine,
		Collation:      table.Collation,
		RowCount:       table.RowCount,
		DataSize:       table.DataSize,
		IndexSize:      table.IndexSize,
		DataFree:       table.DataFree,
		CreateOptions:  table.CreateOptions,
		Comment:        table.Comment,
		Classification: table.Classification,
		UserComment:    table.UserComment,
	}
	// We only return the table info for basic view.
	if view != v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
		return t
	}
	for _, column := range table.Columns {
		t.Columns = append(t.Columns, convertColumnMetadata(column))
	}
	for _, index := range table.Indexes {
		t.Indexes = append(t.Indexes, &v1pb.IndexMetadata{
			Name:        index.Name,
			Expressions: index.Expressions,
			Type:        index.Type,
			Unique:      index.Unique,
			Primary:     index.Primary,
			Visible:     index.Visible,
			Comment:     index.Comment,
		})
	}
	for _, foreignKey := range table.ForeignKeys {
		t.ForeignKeys = append(t.ForeignKeys, &v1pb.ForeignKeyMetadata{
			Name:              foreignKey.Name,
			Columns:           foreignKey.Columns,
			ReferencedSchema:  foreignKey.ReferencedSchema,
			ReferencedTable:   foreignKey.ReferencedTable,
			ReferencedColumns: foreignKey.ReferencedColumns,
			OnDelete:          foreignKey.OnDelete,
			OnUpdate:          foreignKey.OnUpdate,
			MatchType:         foreignKey.MatchType,
		})
	}
	return t
}

func convertColumnMetadata(column *storepb.ColumnMetadata) *v1pb.ColumnMetadata {
	if column == nil {
		return nil
	}
	metadata := &v1pb.ColumnMetadata{
		Name:           column.Name,
		Position:       column.Position,
		HasDefault:     column.DefaultValue != nil,
		Nullable:       column.Nullable,
		Type:           column.Type,
		CharacterSet:   column.CharacterSet,
		Collation:      column.Collation,
		Comment:        column.Comment,
		Classification: column.Classification,
		UserComment:    column.UserComment,
	}
	if metadata.HasDefault {
		switch value := column.DefaultValue.(type) {
		case *storepb.ColumnMetadata_Default:
			if value.Default == nil {
				metadata.Default = &v1pb.ColumnMetadata_DefaultNull{DefaultNull: true}
			} else {
				metadata.Default = &v1pb.ColumnMetadata_DefaultString{DefaultString: value.Default.Value}
			}
		case *storepb.ColumnMetadata_DefaultNull:
			metadata.Default = &v1pb.ColumnMetadata_DefaultNull{DefaultNull: true}
		case *storepb.ColumnMetadata_DefaultExpression:
			metadata.Default = &v1pb.ColumnMetadata_DefaultExpression{DefaultExpression: value.DefaultExpression}
		}
	}
	return metadata
}

func convertV1TableMetadata(table *v1pb.TableMetadata) *storepb.TableMetadata {
	t := &storepb.TableMetadata{
		Name:           table.Name,
		Engine:         table.Engine,
		Collation:      table.Collation,
		RowCount:       table.RowCount,
		DataSize:       table.DataSize,
		IndexSize:      table.IndexSize,
		DataFree:       table.DataFree,
		CreateOptions:  table.CreateOptions,
		Comment:        table.Comment,
		Classification: table.Classification,
		UserComment:    table.UserComment,
	}
	for _, column := range table.Columns {
		t.Columns = append(t.Columns, convertV1ColumnMetadata(column))
	}
	for _, index := range table.Indexes {
		t.Indexes = append(t.Indexes, &storepb.IndexMetadata{
			Name:        index.Name,
			Expressions: index.Expressions,
			Type:        index.Type,
			Unique:      index.Unique,
			Primary:     index.Primary,
			Visible:     index.Visible,
			Comment:     index.Comment,
		})
	}
	for _, foreignKey := range table.ForeignKeys {
		t.ForeignKeys = append(t.ForeignKeys, &storepb.ForeignKeyMetadata{
			Name:              foreignKey.Name,
			Columns:           foreignKey.Columns,
			ReferencedSchema:  foreignKey.ReferencedSchema,
			ReferencedTable:   foreignKey.ReferencedTable,
			ReferencedColumns: foreignKey.ReferencedColumns,
			OnDelete:          foreignKey.OnDelete,
			OnUpdate:          foreignKey.OnUpdate,
			MatchType:         foreignKey.MatchType,
		})
	}
	return t
}

func convertV1ColumnMetadata(column *v1pb.ColumnMetadata) *storepb.ColumnMetadata {
	metadata := &storepb.ColumnMetadata{
		Name:           column.Name,
		Position:       column.Position,
		Nullable:       column.Nullable,
		Type:           column.Type,
		CharacterSet:   column.CharacterSet,
		Collation:      column.Collation,
		Comment:        column.Comment,
		Classification: column.Classification,
		UserComment:    column.UserComment,
	}

	if column.HasDefault {
		switch value := column.Default.(type) {
		case *v1pb.ColumnMetadata_DefaultString:
			metadata.DefaultValue = &storepb.ColumnMetadata_Default{Default: wrapperspb.String(value.DefaultString)}
		case *v1pb.ColumnMetadata_DefaultNull:
			metadata.DefaultValue = &storepb.ColumnMetadata_DefaultNull{DefaultNull: true}
		case *v1pb.ColumnMetadata_DefaultExpression:
			metadata.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: value.DefaultExpression}
		}
	}
	return metadata
}
