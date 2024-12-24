package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// DatabaseCatalogService implements the database catalog service.
type DatabaseCatalogService struct {
	v1pb.UnimplementedDatabaseCatalogServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
}

// NewDatabaseCatalogService creates a new DatabaseCatalogService.
func NewDatabaseCatalogService(store *store.Store, licenseService enterprise.LicenseService) *DatabaseCatalogService {
	return &DatabaseCatalogService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetDatabaseCatalog gets a database catalog.
func (s *DatabaseCatalogService) GetDatabaseCatalog(ctx context.Context, request *v1pb.GetDatabaseCatalogRequest) (*v1pb.DatabaseCatalog, error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(request.Name, common.CatalogSuffix)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get instance %s", instanceID)
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
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return convertDatabaseConfig(database, dbSchema.GetConfig()), nil
}

// UpdateDatabaseCatalog updates a database catalog.
func (s *DatabaseCatalogService) UpdateDatabaseCatalog(ctx context.Context, request *v1pb.UpdateDatabaseCatalogRequest) (*v1pb.DatabaseCatalog, error) {
	instanceID, databaseName, err := common.TrimSuffixAndGetInstanceDatabaseID(request.GetCatalog().GetName(), common.CatalogSuffix)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get instance %s", instanceID)
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
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if dbSchema == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "database schema metadata not found")
	}

	databaseConfig := convertDatabaseCatalog(request.GetCatalog())
	if err := s.store.UpdateDBSchema(ctx, database.UID, &store.UpdateDBSchemaMessage{Config: databaseConfig}, api.SystemBotID); err != nil {
		return nil, err
	}

	return request.GetCatalog(), nil
}

func convertDatabaseConfig(database *store.DatabaseMessage, config *storepb.DatabaseConfig) *v1pb.DatabaseCatalog {
	c := &v1pb.DatabaseCatalog{
		Name: fmt.Sprintf("%s%s/%s%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName, common.CatalogSuffix),
	}
	for _, sc := range config.SchemaConfigs {
		s := &v1pb.SchemaCatalog{Name: sc.Name}
		for _, tc := range sc.TableConfigs {
			t := &v1pb.TableCatalog{
				Name:             tc.Name,
				ClassificationId: tc.ClassificationId,
			}
			var columns []*v1pb.ColumnCatalog
			for _, cc := range tc.ColumnConfigs {
				columns = append(columns, &v1pb.ColumnCatalog{
					Name:                      cc.Name,
					SemanticTypeId:            cc.SemanticTypeId,
					Labels:                    cc.Labels,
					ClassificationId:          cc.ClassificationId,
					MaskingLevel:              convertToV1PBMaskingLevel(cc.MaskingLevel),
					FullMaskingAlgorithmId:    cc.FullMaskingAlgorithmId,
					PartialMaskingAlgorithmId: cc.PartialMaskingAlgorithmId,
					// ObjectSchema:              cc.ObjectSchema,
				})
			}
			t.Kind = &v1pb.TableCatalog_Columns_{Columns: &v1pb.TableCatalog_Columns{
				Columns: columns,
			}}
			s.Tables = append(s.Tables, t)
		}
		c.Schemas = append(c.Schemas, s)
	}
	return c
}

func convertDatabaseCatalog(catalog *v1pb.DatabaseCatalog) *storepb.DatabaseConfig {
	c := &storepb.DatabaseConfig{}
	for _, sc := range catalog.Schemas {
		s := &storepb.SchemaConfig{Name: sc.Name}
		for _, tc := range sc.Tables {
			t := &storepb.TableConfig{
				Name:             tc.Name,
				ClassificationId: tc.ClassificationId,
			}
			for _, cc := range tc.GetColumns().GetColumns() {
				t.ColumnConfigs = append(t.ColumnConfigs, &storepb.ColumnConfig{
					Name:                      cc.Name,
					SemanticTypeId:            cc.SemanticTypeId,
					Labels:                    cc.Labels,
					ClassificationId:          cc.ClassificationId,
					MaskingLevel:              convertToStorePBMaskingLevel(cc.MaskingLevel),
					FullMaskingAlgorithmId:    cc.FullMaskingAlgorithmId,
					PartialMaskingAlgorithmId: cc.PartialMaskingAlgorithmId,
					// ObjectSchema:              cc.ObjectSchema,
				})
			}
			s.TableConfigs = append(s.TableConfigs, t)
		}
		c.SchemaConfigs = append(c.SchemaConfigs, s)
	}
	return c
}
