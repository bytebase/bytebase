package databricks

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

type tableUnion struct {
	externalTable *storepb.ExternalTableMetadata
	table         *storepb.TableMetadata
	view          *storepb.ViewMetadata
	materialView  *storepb.MaterializedViewMetadata
	typeName      catalog.TableType
	name          string
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
	catalogMap, err := d.listCatologTables(ctx, d.curCatalog)
	if err != nil {
		return nil, err
	}

	dbMetadataMeta := storepb.DatabaseSchemaMetadata{}
	schemaMap, ok := (catalogMap)[d.curCatalog]
	if !ok {
		return nil, errors.Errorf("cannot find metadata for catalog '%s'", d.curCatalog)
	}

	dbMetadataMeta.Name = d.curCatalog
	schemas := convertToStorepbSchemas(schemaMap)
	dbMetadataMeta.Schemas = schemas

	return &dbMetadataMeta, nil
}

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instanceMetadata := &db.InstanceMetadata{}

	// fetch version.
	versionData, _, err := d.execStatementSync(ctx, "SELECT VERSION()")
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
	catalogMap, err := d.listCatologTables(ctx, "")
	if err != nil {
		return nil, err
	}

	for catalogName, schemaMap := range catalogMap {
		dbMetadataMeta := storepb.DatabaseSchemaMetadata{}
		schemas := convertToStorepbSchemas(schemaMap)
		dbMetadataMeta.Name = catalogName
		dbMetadataMeta.Schemas = schemas
		instanceMetadata.Databases = append(instanceMetadata.Databases, &dbMetadataMeta)
	}

	// fetch workspace users.
	// TODO(tommy): complete this part when Permissions API for Golang is implemented.

	return instanceMetadata, nil
}

// list all tables in the workspace when catalogName is set to "".
func (d *Driver) listCatologTables(ctx context.Context, targetCatalogName string) (databricksCatalogMap, error) {
	catalogMap := make(databricksCatalogMap)

	catalogsInfo, err := d.Client.Catalogs.ListAll(ctx, catalog.ListCatalogsRequest{})
	if err != nil {
		return nil, err
	}

	for _, catalogInfo := range catalogsInfo {
		if targetCatalogName != "" && targetCatalogName != catalogInfo.Name {
			continue
		}
		schemasInfo, err := d.Client.Schemas.ListAll(ctx, catalog.ListSchemasRequest{
			CatalogName: catalogInfo.Name,
		})
		if err != nil {
			return nil, err
		}
		for _, schemaInfo := range schemasInfo {
			// init map.
			scMap, ok := catalogMap[catalogInfo.Name]
			if !ok {
				catalogMap[catalogInfo.Name] = databricksSchemaMap{
					schemaInfo.Name: []*tableUnion{},
				}
			} else {
				_, ok := scMap[schemaInfo.Name]
				if !ok {
					scMap[schemaInfo.Name] = []*tableUnion{}
				}
				catalogMap[catalogInfo.Name] = scMap
			}
			// list tables in the schema.
			tablesInfo, err := d.Client.Tables.ListAll(ctx, catalog.ListTablesRequest{
				CatalogName: catalogInfo.Name,
				SchemaName:  schemaInfo.Name,
			})
			if err != nil {
				return nil, err
			}
			if err := appendSchemaTables(catalogMap, tablesInfo); err != nil {
				return nil, err
			}
		}
	}

	return catalogMap, nil
}

// extract tables' metadata from 'tablesInfo' and store it in 'catalogMap'.
func appendSchemaTables(catalogMap databricksCatalogMap, tablesInfo []catalog.TableInfo) error {
	for _, tableInfo := range tablesInfo {
		table := tableUnion{
			typeName: tableInfo.TableType,
			name:     tableInfo.Name,
		}

		switch tableInfo.TableType {
		case catalog.TableTypeView:
			table.view = &storepb.ViewMetadata{
				Name:              table.name,
				Definition:        tableInfo.ViewDefinition,
				Comment:           tableInfo.Comment,
				DependencyColumns: convertToDependencyColumns(tableInfo.SchemaName, tableInfo.Name, tableInfo.Columns),
			}
		case catalog.TableTypeMaterializedView:
			table.materialView = &storepb.MaterializedViewMetadata{
				Name:       table.name,
				Definition: tableInfo.ViewDefinition,
				Comment:    tableInfo.Comment,
			}
		case catalog.TableTypeExternal:
			table.externalTable = &storepb.ExternalTableMetadata{
				Name: table.name,
			}
		case catalog.TableTypeManaged:
			table.table = &storepb.TableMetadata{
				Name:    table.name,
				Columns: convertToColumnMetadata(tableInfo.Columns),
				Comment: tableInfo.Comment,
			}
		default:
			// we do not sync streaming table.
			continue
		}

		// catalogMap[tableInfo.CatalogName] must not be nil.
		if tblList, ok := catalogMap[tableInfo.CatalogName][tableInfo.SchemaName]; ok {
			tblList = append(tblList, &table)
			catalogMap[tableInfo.CatalogName][tableInfo.SchemaName] = tblList
		} else {
			return errors.New("table list not initialized")
		}
	}
	return nil
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

func convertToDependencyColumns(schema, table string, columnInfo []catalog.ColumnInfo) []*storepb.DependencyColumn {
	columns := []*storepb.DependencyColumn{}
	for _, col := range columnInfo {
		columns = append(columns, &storepb.DependencyColumn{
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

func getQualifiedTblName(catalogName, schemaName, tblName string) (string, error) {
	if tblName == "" || schemaName == "" {
		return "", errors.New("table name and schema name must be specified")
	}
	qualifiedName := fmt.Sprintf("`%s`.`%s`", schemaName, tblName)
	if catalogName != "" {
		qualifiedName = fmt.Sprintf("`%s`.%s", catalogName, qualifiedName)
	}
	return qualifiedName, nil
}
