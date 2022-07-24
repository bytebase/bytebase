package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/plugin/db"
)

var (
	_ catalog.Catalog = (*Catalog)(nil)
)

// Catalog is the database catalog.
type Catalog struct {
	databaseID *int
	store      *Store
	engineType db.Type
}

// NewCatalog creates a new database catalog.
func NewCatalog(databaseID *int, store *Store, dbType db.Type) *Catalog {
	return &Catalog{
		databaseID: databaseID,
		store:      store,
		engineType: dbType,
	}
}

// GetDatabase implements the catalog.Catalog interface.
func (c *Catalog) GetDatabase(ctx context.Context) (*catalog.Database, error) {
	database, err := c.store.GetDatabase(ctx, &api.DatabaseFind{
		ID: c.databaseID,
	})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, nil
	}

	dbType, err := convertToCatalogDBType(c.engineType)
	if err != nil {
		return nil, err
	}

	databaseData := catalog.Database{
		Name:         database.Name,
		CharacterSet: database.CharacterSet,
		Collation:    database.Collation,
		DbType:       dbType,
	}

	if databaseData.SchemaList, err = c.getSchemaList(ctx); err != nil {
		return nil, err
	}

	return &databaseData, nil
}

type schemaMap map[string]*catalog.Schema

func (m schemaMap) getOrCreateSchema(name string) *catalog.Schema {
	if schema, found := m[name]; found {
		return schema
	}

	schema := &catalog.Schema{
		Name: name,
	}
	m[name] = schema
	return schema
}

func (c *Catalog) getSchemaList(ctx context.Context) ([]*catalog.Schema, error) {
	schemaSet := make(schemaMap)

	// find table list
	tableList, err := c.store.FindTable(ctx, &api.TableFind{
		DatabaseID: c.databaseID,
	})
	if err != nil {
		return nil, err
	}
	for _, table := range tableList {
		tableData := convertTable(table)
		schemaName := ""
		if c.engineType == db.Postgres {
			if schemaName, tableData.Name, err = splitPGSchema(tableData.Name); err != nil {
				return nil, err
			}
		}
		schema := schemaSet.getOrCreateSchema(schemaName)
		schema.TableList = append(schema.TableList, tableData)
	}

	// find view list
	viewList, err := c.store.FindView(ctx, &api.ViewFind{
		DatabaseID: c.databaseID,
	})
	if err != nil {
		return nil, err
	}
	for _, view := range viewList {
		viewData := convertView(view)
		schemaName := ""
		if c.engineType == db.Postgres {
			if schemaName, viewData.Name, err = splitPGSchema(viewData.Name); err != nil {
				return nil, err
			}
		}
		schema := schemaSet.getOrCreateSchema(schemaName)
		schema.ViewList = append(schema.ViewList, viewData)
	}

	// find extension list
	extensionList, err := c.store.FindDBExtension(ctx, &api.DBExtensionFind{
		DatabaseID: c.databaseID,
	})
	if err != nil {
		return nil, err
	}
	for _, extension := range extensionList {
		schema := schemaSet.getOrCreateSchema(extension.Schema)
		schema.ExtensionList = append(schema.ExtensionList, &catalog.Extension{
			Name:        extension.Name,
			Version:     extension.Version,
			Description: extension.Description,
		})
	}

	var schemaList []*catalog.Schema
	for _, schema := range schemaSet {
		schemaList = append(schemaList, schema)
	}
	return schemaList, nil
}

func splitPGSchema(name string) (string, string, error) {
	list := strings.Split(name, ".")
	if len(list) != 2 {
		return "", "", fmt.Errorf("split failed: the expected name is schemaName.name, but get %s", name)
	}
	return list[0], list[1], nil
}

func convertView(view *api.View) *catalog.View {
	return &catalog.View{
		Name:       view.Name,
		CreatedTs:  view.CreatedTs,
		UpdatedTs:  view.UpdatedTs,
		Definition: view.Definition,
		Comment:    view.Comment,
	}
}

func convertTable(table *api.Table) *catalog.Table {
	return &catalog.Table{
		Name:          table.Name,
		CreatedTs:     table.CreatedTs,
		UpdatedTs:     table.UpdatedTs,
		Type:          table.Type,
		Engine:        table.Engine,
		Collation:     table.Collation,
		RowCount:      table.RowCount,
		DataSize:      table.DataSize,
		IndexSize:     table.IndexSize,
		DataFree:      table.DataFree,
		CreateOptions: table.CreateOptions,
		Comment:       table.Comment,
		ColumnList:    convertColumnList(table.ColumnList),
		IndexList:     converIndexList(table.IndexList),
	}
}

func converIndexList(list []*api.Index) []*catalog.Index {
	var res []*catalog.Index
	for _, index := range list {
		res = append(res, &catalog.Index{
			Name:       index.Name,
			Expression: index.Expression,
			Position:   index.Position,
			Type:       index.Type,
			Unique:     index.Unique,
			Primary:    index.Primary,
			Visible:    index.Visible,
			Comment:    index.Comment,
		})
	}
	return res
}

func convertColumnList(list []*api.Column) []*catalog.Column {
	var res []*catalog.Column
	for _, column := range list {
		res = append(res, &catalog.Column{
			Name:         column.Name,
			Position:     column.Position,
			Default:      column.Default,
			Nullable:     column.Nullable,
			Type:         column.Type,
			CharacterSet: column.CharacterSet,
			Collation:    column.Collation,
			Comment:      column.Comment,
		})
	}
	return res
}

func convertToCatalogDBType(dbType db.Type) (catalog.DBType, error) {
	switch dbType {
	case db.MySQL:
		return catalog.MySQL, nil
	case db.Postgres:
		return catalog.Postgres, nil
	case db.TiDB:
		return catalog.TiDB, nil
	}

	return "", fmt.Errorf("unsupported db type %s for catalog", dbType)
}
