package export

import (
	"context"

	"connectrpc.com/connect"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

// GetResources extracts the resource list from the statement for exporting results as SQL.
// It analyzes the SQL query to determine which database tables/views are being queried.
func GetResources(
	ctx context.Context,
	storeInstance *store.Store,
	engine storepb.Engine,
	databaseName string,
	statement string,
	instance *store.InstanceMessage,
	getDatabaseMetadataFunc base.GetDatabaseMetadataFunc,
	listDatabaseNamesFunc base.ListDatabaseNamesFunc,
	getLinkedDatabaseMetadataFunc base.GetLinkedDatabaseMetadataFunc,
) ([]base.SchemaResource, error) {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return getResourcesForMySQL(ctx, storeInstance, engine, databaseName, statement, instance, getDatabaseMetadataFunc, listDatabaseNamesFunc, getLinkedDatabaseMetadataFunc)
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT:
		return getResourcesForPostgres(ctx, storeInstance, engine, databaseName, statement, instance, "public", getDatabaseMetadataFunc, listDatabaseNamesFunc, getLinkedDatabaseMetadataFunc)
	case storepb.Engine_ORACLE:
		return getResourcesForOracle(ctx, storeInstance, engine, databaseName, statement, instance, getDatabaseMetadataFunc, listDatabaseNamesFunc, getLinkedDatabaseMetadataFunc)
	case storepb.Engine_SNOWFLAKE:
		return getResourcesForSnowflake(ctx, storeInstance, engine, databaseName, statement, instance, getDatabaseMetadataFunc, listDatabaseNamesFunc, getLinkedDatabaseMetadataFunc)
	case storepb.Engine_MSSQL:
		return getResourcesForMSSQL(ctx, storeInstance, engine, databaseName, statement, instance, getDatabaseMetadataFunc, listDatabaseNamesFunc, getLinkedDatabaseMetadataFunc)
	default:
		if databaseName == "" {
			return nil, errors.Errorf("database must be specified")
		}
		return []base.SchemaResource{{Database: databaseName}}, nil
	}
}

func getResourcesForMySQL(
	ctx context.Context,
	storeInstance *store.Store,
	engine storepb.Engine,
	databaseName string,
	statement string,
	instance *store.InstanceMessage,
	getDatabaseMetadataFunc base.GetDatabaseMetadataFunc,
	listDatabaseNamesFunc base.ListDatabaseNamesFunc,
	getLinkedDatabaseMetadataFunc base.GetLinkedDatabaseMetadataFunc,
) ([]base.SchemaResource, error) {
	spans, err := base.GetQuerySpan(ctx, base.GetQuerySpanContext{
		InstanceID:                    instance.ResourceID,
		GetDatabaseMetadataFunc:       getDatabaseMetadataFunc,
		ListDatabaseNamesFunc:         listDatabaseNamesFunc,
		GetLinkedDatabaseMetadataFunc: getLinkedDatabaseMetadataFunc,
	}, engine, statement, databaseName, "", !store.IsObjectCaseSensitive(instance))
	if err != nil {
		return nil, err
	} else if databaseName == "" {
		var list []base.SchemaResource
		for _, span := range spans {
			for sourceColumn := range span.SourceColumns {
				list = append(list, base.SchemaResource{
					Database:     sourceColumn.Database,
					Schema:       sourceColumn.Schema,
					Table:        sourceColumn.Table,
					LinkedServer: sourceColumn.Server,
				})
			}
		}
		return list, nil
	}

	database, err := storeInstance.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instance.ResourceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
			return nil, nil
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to fetch database"))
	}
	if database == nil {
		return nil, nil
	}

	dbMetadata, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to fetch database schema"))
	}

	var result []base.SchemaResource
	for _, span := range spans {
		for sourceColumn := range span.SourceColumns {
			sr := base.SchemaResource{
				Database:     sourceColumn.Database,
				Schema:       sourceColumn.Schema,
				Table:        sourceColumn.Table,
				LinkedServer: sourceColumn.Server,
			}
			if sourceColumn.Database != dbMetadata.GetProto().Name {
				resourceDB, err := storeInstance.GetDatabase(ctx, &store.FindDatabaseMessage{
					InstanceID:   &instance.ResourceID,
					DatabaseName: &sourceColumn.Database,
				})
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database %v in instance %v", sourceColumn.Database, instance.ResourceID))
				}
				if resourceDB == nil {
					continue
				}
				resourceDBSchema, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
					InstanceID:   resourceDB.InstanceID,
					DatabaseName: resourceDB.DatabaseName,
				})
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database schema %v in instance %v", sourceColumn.Database, instance.ResourceID))
				}
				if !resourceExists(resourceDBSchema, sr) {
					continue
				}
				result = append(result, sr)
				continue
			}
			if !resourceExists(dbMetadata, sr) {
				continue
			}
			result = append(result, sr)
		}
	}
	return result, nil
}

func getResourcesForPostgres(
	ctx context.Context,
	storeInstance *store.Store,
	engine storepb.Engine,
	databaseName string,
	statement string,
	instance *store.InstanceMessage,
	defaultSchema string,
	getDatabaseMetadataFunc base.GetDatabaseMetadataFunc,
	listDatabaseNamesFunc base.ListDatabaseNamesFunc,
	getLinkedDatabaseMetadataFunc base.GetLinkedDatabaseMetadataFunc,
) ([]base.SchemaResource, error) {
	spans, err := base.GetQuerySpan(ctx, base.GetQuerySpanContext{
		InstanceID:                    instance.ResourceID,
		GetDatabaseMetadataFunc:       getDatabaseMetadataFunc,
		ListDatabaseNamesFunc:         listDatabaseNamesFunc,
		GetLinkedDatabaseMetadataFunc: getLinkedDatabaseMetadataFunc,
	}, engine, statement, databaseName, defaultSchema, !store.IsObjectCaseSensitive(instance))
	if err != nil {
		return nil, err
	}

	database, err := storeInstance.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instance.ResourceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to fetch database"))
	}
	if database == nil {
		return nil, nil
	}

	dbMetadata, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to fetch database schema"))
	}

	var result []base.SchemaResource
	for _, span := range spans {
		for sourceColumn := range span.SourceColumns {
			sr := base.SchemaResource{
				Database:     sourceColumn.Database,
				Schema:       sourceColumn.Schema,
				Table:        sourceColumn.Table,
				LinkedServer: sourceColumn.Server,
			}

			if sourceColumn.Database != dbMetadata.GetProto().Name {
				continue
			}

			if !resourceExists(dbMetadata, sr) {
				continue
			}

			result = append(result, sr)
		}
	}

	return result, nil
}

func getResourcesForOracle(
	ctx context.Context,
	storeInstance *store.Store,
	engine storepb.Engine,
	databaseName string,
	statement string,
	instance *store.InstanceMessage,
	getDatabaseMetadataFunc base.GetDatabaseMetadataFunc,
	listDatabaseNamesFunc base.ListDatabaseNamesFunc,
	getLinkedDatabaseMetadataFunc base.GetLinkedDatabaseMetadataFunc,
) ([]base.SchemaResource, error) {
	return getResourcesForPostgres(ctx, storeInstance, engine, databaseName, statement, instance, databaseName, getDatabaseMetadataFunc, listDatabaseNamesFunc, getLinkedDatabaseMetadataFunc)
}

func getResourcesForSnowflake(
	ctx context.Context,
	storeInstance *store.Store,
	engine storepb.Engine,
	databaseName string,
	statement string,
	instance *store.InstanceMessage,
	getDatabaseMetadataFunc base.GetDatabaseMetadataFunc,
	listDatabaseNamesFunc base.ListDatabaseNamesFunc,
	getLinkedDatabaseMetadataFunc base.GetLinkedDatabaseMetadataFunc,
) ([]base.SchemaResource, error) {
	dataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_READ_ONLY)
	adminDataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
	if dataSource == nil {
		dataSource = adminDataSource
	}
	if dataSource == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find data source for instance: %s", instance.ResourceID))
	}
	return getResourcesForPostgres(ctx, storeInstance, engine, databaseName, statement, instance, "PUBLIC", getDatabaseMetadataFunc, listDatabaseNamesFunc, getLinkedDatabaseMetadataFunc)
}

func getResourcesForMSSQL(
	ctx context.Context,
	storeInstance *store.Store,
	engine storepb.Engine,
	databaseName string,
	statement string,
	instance *store.InstanceMessage,
	getDatabaseMetadataFunc base.GetDatabaseMetadataFunc,
	listDatabaseNamesFunc base.ListDatabaseNamesFunc,
	getLinkedDatabaseMetadataFunc base.GetLinkedDatabaseMetadataFunc,
) ([]base.SchemaResource, error) {
	dataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_READ_ONLY)
	adminDataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
	if dataSource == nil {
		dataSource = adminDataSource
	}
	if dataSource == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find data source for instance: %s", instance.ResourceID))
	}
	return getResourcesForPostgres(ctx, storeInstance, engine, databaseName, statement, instance, "dbo", getDatabaseMetadataFunc, listDatabaseNamesFunc, getLinkedDatabaseMetadataFunc)
}

func resourceExists(dbMetadata *model.DatabaseMetadata, resource base.SchemaResource) bool {
	schema := dbMetadata.GetSchemaMetadata(resource.Schema)
	if schema == nil {
		return false
	}
	if schema.GetTable(resource.Table) != nil {
		return true
	}
	if schema.GetView(resource.Table) != nil {
		return true
	}
	if schema.GetMaterializedView(resource.Table) != nil {
		return true
	}
	return false
}
