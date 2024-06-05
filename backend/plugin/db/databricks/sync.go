package databricks

import (
	"context"
	"time"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type tableUnion struct {
	externalTable *storepb.ExternalTableMetadata
	table         *storepb.TableMetadata
	view          *storepb.ViewMetadata
	materialView  *storepb.MaterializedViewMetadata
	typeName      catalog.TableType
}

type databricksSchemaMap = map[string][]*tableUnion
type databricksCatalogMap = map[string]databricksSchemaMap

// sync catalog.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	// return nothing if no catalogs are specified.
	if d.curCatalog == "" {
		return nil, nil
	}

	// fetch table data from databricks.
	catalogMap, err := d.listTables(ctx)
	if err != nil {
		return nil, err
	}

	dbSchemaMeta := storepb.DatabaseSchemaMetadata{}
	schemaMap, ok := (catalogMap)[d.curCatalog]
	if !ok {
		return nil, errors.Errorf("cannot find metadata for catalog '%s'", d.curCatalog)
	}

	dbSchemaMeta.Name = d.curCatalog
	schemas := convertToStorepbSchemas(schemaMap)
	dbSchemaMeta.Schemas = schemas

	return &dbSchemaMeta, nil
}

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instanceMetadata := &db.InstanceMetadata{}

	// fetch table data from databricks.
	catalogMap, err := d.listTables(ctx)
	if err != nil {
		return nil, err
	}

	for catalogName, schemaMap := range catalogMap {
		dbSchemaMeta := storepb.DatabaseSchemaMetadata{}
		schemas := convertToStorepbSchemas(schemaMap)
		dbSchemaMeta.Name = catalogName
		dbSchemaMeta.Schemas = schemas
		instanceMetadata.Databases = append(instanceMetadata.Databases, &dbSchemaMeta)
	}

	return instanceMetadata, nil
}

func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, nil
}

func (d *Driver) listTables(ctx context.Context) (databricksCatalogMap, error) {
	tablesInfo, err := d.client.Tables.ListAll(ctx, catalog.ListTablesRequest{})
	if err != nil {
		return nil, err
	}

	catalogMap := make(databricksCatalogMap)

	for _, tableInfo := range tablesInfo {
		table := tableUnion{
			typeName: tableInfo.TableType,
		}

		switch tableInfo.TableType {
		case catalog.TableTypeView:
			table.view = &storepb.ViewMetadata{
				Name:             tableInfo.Name,
				Definition:       tableInfo.ViewDefinition,
				Comment:          tableInfo.Comment,
				DependentColumns: convertToDependentColumns(tableInfo.SchemaName, tableInfo.Name, tableInfo.Columns),
			}
		case catalog.TableTypeMaterializedView:
			table.materialView = &storepb.MaterializedViewMetadata{
				Name:       tableInfo.Name,
				Definition: tableInfo.ViewDefinition,
				Comment:    tableInfo.Comment,
			}
		case catalog.TableTypeExternal:
			table.externalTable = &storepb.ExternalTableMetadata{
				Name: tableInfo.Name,
			}
			// TODO(tommy): find the responding string for the normal table type.
		default:
			table.table = &storepb.TableMetadata{
				Name:    tableInfo.Name,
				Columns: convertToColumnMetadata(tableInfo.Columns),
				Comment: tableInfo.Comment,
			}
		}

		if schemaMap, ok := catalogMap[tableInfo.CatalogName]; !ok {
			catalogMap[tableInfo.CatalogName] = databricksSchemaMap{
				tableInfo.SchemaName: []*tableUnion{&table},
			}
		} else {
			if tableList, ok := schemaMap[tableInfo.SchemaName]; ok {
				tableList = append(tableList, &table)
				schemaMap[tableInfo.SchemaName] = tableList
			} else {
				schemaMap[tableInfo.SchemaName] = []*tableUnion{&table}
			}
		}
	}

	return catalogMap, nil
}

func convertToColumnMetadata(columnInfo []catalog.ColumnInfo) []*storepb.ColumnMetadata {
	columns := []*storepb.ColumnMetadata{}
	for _, col := range columnInfo {
		columns = append(columns, &storepb.ColumnMetadata{
			Name:     col.Name,
			Position: int32(col.Position),
			Nullable: col.Nullable,
			Type:     string(col.TypeName),
			Comment:  col.Comment,
		})
	}
	return columns
}

func convertToDependentColumns(schema, table string, columnInfo []catalog.ColumnInfo) []*storepb.DependentColumn {
	columns := []*storepb.DependentColumn{}
	for _, col := range columnInfo {
		columns = append(columns, &storepb.DependentColumn{
			Schema: schema,
			Table:  table,
			Column: col.Name,
		})
	}
	return columns
}

func convertToStorepbSchemas(schemaMap databricksSchemaMap) []*storepb.SchemaMetadata {
	schemas := []*storepb.SchemaMetadata{}
	for schemaName, tableList := range schemaMap {
		schemaMetadata := &storepb.SchemaMetadata{
			Name: schemaName,
		}

		for _, table := range tableList {
			switch table.typeName {
			case catalog.TableTypeView:
				schemaMetadata.Views = append(schemaMetadata.Views, table.view)
			case catalog.TableTypeMaterializedView:
				schemaMetadata.MaterializedViews = append(schemaMetadata.MaterializedViews, table.materialView)
			case catalog.TableTypeExternal:
				schemaMetadata.ExternalTables = append(schemaMetadata.ExternalTables, table.externalTable)
			default:
				// TODO(tommy): find the responding string for the normal table type.
				schemaMetadata.Tables = append(schemaMetadata.Tables, table.table)
			}
		}
		schemas = append(schemas, schemaMetadata)
	}
	return schemas
}
