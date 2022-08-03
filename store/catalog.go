package store

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
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

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(c.engineType))
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

		// find index list
		indexList, err := c.store.FindIndex(ctx, &api.IndexFind{
			DatabaseID: c.databaseID,
			TableID:    &table.ID,
		})
		if err != nil {
			return nil, err
		}
		tableData.IndexList = convertIndexList(indexList)

		// find column list
		columnList, err := c.store.FindColumn(ctx, &api.ColumnFind{
			DatabaseID: c.databaseID,
			TableID:    &table.ID,
		})
		if err != nil {
			return nil, err
		}
		tableData.ColumnList = convertColumnList(columnList)

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
	}
}

func convertIndexList(list []*api.Index) []*catalog.Index {
	if len(list) == 0 {
		return nil
	}
	var res []*catalog.Index
	indexMap := make(map[string][]*api.Index)
	for _, expression := range list {
		indexMap[expression.Name] = append(indexMap[expression.Name], expression)
	}

	for _, expressionList := range indexMap {
		res = append(res, convertIndexExceptExpression(expressionList))
	}
	// Order it cause the random iteration order in Go, see https://go.dev/blog/maps
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res
}

func convertIndexExceptExpression(list []*api.Index) *catalog.Index {
	if len(list) == 0 {
		return nil
	}
	res := &catalog.Index{
		Name:    list[0].Name,
		Type:    list[0].Type,
		Unique:  list[0].Unique,
		Primary: list[0].Primary,
		Visible: list[0].Visible,
		Comment: list[0].Comment,
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Position < list[j].Position
	})
	for _, expression := range list {
		res.ExpressionList = append(res.ExpressionList, expression.Expression)
	}
	return res
}

func convertColumnList(list []*api.Column) []*catalog.Column {
	if len(list) == 0 {
		return nil
	}
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
