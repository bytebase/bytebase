package databricks

import (
	"context"
	"time"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

type Table struct {
	externalTable *store.ExternalTableMetadata
	table         *store.TableMetadata
	view          *store.ViewMetadata
	materialView  *store.MaterializedViewMetadata
	typeName      catalog.TableType
}

type SchemaMap = map[string][]*Table
type CatalogMap = map[string]SchemaMap

// sync catalog.
func (d *Driver) SyncDBSchema(ctx context.Context) (*store.DatabaseSchemaMetadata, error) {
	// return nothing if no catalogs are specified.
	if d.curCatalog == "" {
		return nil, nil
	}

	// fetch table data from databricks.
	catalogMap, err := d.listTables(ctx)
	if err != nil {
		return nil, err
	}

	dbSchemaMeta := store.DatabaseSchemaMetadata{}
	schemaMap, ok := (*catalogMap)[d.curCatalog]
	if !ok {
		return nil, errors.Errorf("cannot find metadata for catalog '%s'", d.curCatalog)
	}

	dbSchemaMeta.Name = d.curCatalog
	schemas := convertToStorepbSchemas(&schemaMap)
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

	for catalogName, schemaMap := range *catalogMap {
		dbSchemaMeta := store.DatabaseSchemaMetadata{}
		schemas := convertToStorepbSchemas(&schemaMap)
		dbSchemaMeta.Name = catalogName
		dbSchemaMeta.Schemas = schemas
		instanceMetadata.Databases = append(instanceMetadata.Databases, &dbSchemaMeta)
	}

	return instanceMetadata, nil
}

func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*store.SlowQueryStatistics, error) {
	return nil, nil
}

func (d *Driver) listTables(ctx context.Context) (*CatalogMap, error) {
	tablesInfo, err := d.client.Tables.ListAll(ctx, catalog.ListTablesRequest{})
	if err != nil {
		return nil, err
	}

	catalogMap := CatalogMap{}

	for _, tableInfo := range tablesInfo {
		table := Table{
			typeName: tableInfo.TableType,
		}

		switch tableInfo.TableType {
		case catalog.TableTypeView:
			table.view = &store.ViewMetadata{
				Name:             tableInfo.Name,
				Definition:       tableInfo.ViewDefinition,
				Comment:          tableInfo.Comment,
				DependentColumns: convertToDependentColumns(tableInfo.SchemaName, tableInfo.Name, tableInfo.Columns),
			}
		case catalog.TableTypeMaterializedView:
			table.materialView = &store.MaterializedViewMetadata{
				Name:       tableInfo.Name,
				Definition: tableInfo.ViewDefinition,
				Comment:    tableInfo.Comment,
			}
		case catalog.TableTypeExternal:
			table.externalTable = &store.ExternalTableMetadata{
				Name: tableInfo.Name,
			}
			// TODO(tommy): find the responding string for the normal table type.
		default:
			table.table = &store.TableMetadata{
				Name:    tableInfo.Name,
				Columns: convertToColumnMetadata(tableInfo.Columns),
				Comment: tableInfo.Comment,
			}
		}

		if schemaMap, ok := catalogMap[tableInfo.CatalogName]; !ok {
			catalogMap[tableInfo.CatalogName] = SchemaMap{
				tableInfo.SchemaName: []*Table{&table},
			}
		} else {
			if tableList, ok := schemaMap[tableInfo.SchemaName]; ok {
				tableList = append(tableList, &table)
				schemaMap[tableInfo.SchemaName] = tableList
			} else {
				schemaMap[tableInfo.SchemaName] = []*Table{&table}
			}
		}
	}

	return &catalogMap, nil
}

func convertToColumnMetadata(columnInfo []catalog.ColumnInfo) []*store.ColumnMetadata {
	columns := []*store.ColumnMetadata{}
	for _, col := range columnInfo {
		columns = append(columns, &store.ColumnMetadata{
			Name:     col.Name,
			Position: int32(col.Position),
			Nullable: col.Nullable,
			Type:     string(col.TypeName),
			Comment:  col.Comment,
		})
	}
	return columns
}

func convertToDependentColumns(schema, table string, columnInfo []catalog.ColumnInfo) []*store.DependentColumn {
	columns := []*store.DependentColumn{}
	for _, col := range columnInfo {
		columns = append(columns, &store.DependentColumn{
			Schema: schema,
			Table:  table,
			Column: col.Name,
		})
	}
	return columns
}

func convertToStorepbSchemas(schemaMap *SchemaMap) []*store.SchemaMetadata {
	schemas := []*store.SchemaMetadata{}
	for schemaName, tableList := range *schemaMap {
		schemaMetadata := &store.SchemaMetadata{
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
