package databricks

import (
	"context"
	"strings"
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
	catalogMap, err := d.listAllTables(ctx)
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

	// fetch version.
	versionData, _, err := d.execSingleSQLSync(ctx, "SELECT VERSION()")
	if err != nil {
		return nil, err
	}
	if len(versionData) != 1 || len(versionData[0]) != 1 {
		return nil, errors.New("invalid version format")
	}
	splitVersion := strings.Split(versionData[0][0], " ")
	if len(splitVersion) != 2 {
		return nil, errors.New("invalid version format")
	}
	instanceMetadata.Version = splitVersion[0]

	// fetch table data from databricks.
	catalogMap, err := d.listAllTables(ctx)
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

	// fetch workspace users.
	// TODO(tommy): complete this part when Permissions API for Golang is implemented.

	return instanceMetadata, nil
}

func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, nil
}

func (d *Driver) listAllTables(ctx context.Context) (databricksCatalogMap, error) {
	catalogMap := make(databricksCatalogMap)

	catalogsInfo, err := d.Client.Catalogs.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, catalogInfo := range catalogsInfo {
		schemasInfo, err := d.Client.Schemas.ListAll(ctx, catalog.ListSchemasRequest{
			CatalogName: catalogInfo.Name,
		})
		if err != nil {
			return nil, err
		}
		for _, schemaInfo := range schemasInfo {
			tablesInfo, err := d.Client.Tables.ListAll(ctx, catalog.ListTablesRequest{
				CatalogName: catalogInfo.Name,
				SchemaName:  schemaInfo.Name,
			})
			if err != nil {
				return nil, err
			}
			appendSchemaTables(catalogMap, tablesInfo)
		}
	}

	return catalogMap, nil
}

// extract tables' metadata from 'tablesInfo' and store it in 'catalogMap'.
func appendSchemaTables(catalogMap databricksCatalogMap, tablesInfo []catalog.TableInfo) {
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
		case catalog.TableTypeManaged:
			table.table = &storepb.TableMetadata{
				Name:    tableInfo.Name,
				Columns: convertToColumnMetadata(tableInfo.Columns),
				Comment: tableInfo.Comment,
			}
		default:
			// we do not sync streaming table.
			continue
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
			case catalog.TableTypeManaged:
				schemaMetadata.Tables = append(schemaMetadata.Tables, table.table)
			default:
				// we do not sync streaming table.
			}
		}
		schemas = append(schemas, schemaMetadata)
	}
	return schemas
}
