package store

import (
	"context"
	"sort"
	"strings"

	"github.com/pkg/errors"

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
	Database *catalog.Database
	Finder   *catalog.Finder
}

// NewCatalog creates a new database catalog.
func (s *Store) NewCatalog(ctx context.Context, databaseID int, engineType db.Type) (catalog.Catalog, error) {
	database, err := s.GetDatabase(ctx, &api.DatabaseFind{
		ID: &databaseID,
	})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, nil
	}

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(engineType))
	if err != nil {
		return nil, err
	}

	databaseData := &catalog.Database{
		Name:         database.Name,
		CharacterSet: database.CharacterSet,
		Collation:    database.Collation,
		DbType:       dbType,
	}

	if databaseData.SchemaList, err = s.getSchemaList(ctx, databaseID, engineType); err != nil {
		return nil, err
	}
	c := &Catalog{Database: databaseData}
	c.Finder = catalog.NewFinder(c.Database, &catalog.FinderContext{CheckIntegrity: true})
	return c, nil
}

// GetDatabase implements the catalog.Catalog interface.
func (c *Catalog) GetDatabase() *catalog.Database {
	return c.Database
}

// GetFinder implements the catalog.Catalog interface.
func (c *Catalog) GetFinder() *catalog.Finder {
	return c.Finder
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

func (s *Store) getSchemaList(ctx context.Context, databaseID int, engineType db.Type) ([]*catalog.Schema, error) {
	schemaSet := make(schemaMap)

	// find table list
	tableList, err := s.FindTable(ctx, &api.TableFind{
		DatabaseID: &databaseID,
	})
	if err != nil {
		return nil, err
	}
	for _, table := range tableList {
		tableData := convertTable(table)
		schemaName := ""
		if engineType == db.Postgres {
			if schemaName, tableData.Name, err = splitPGSchema(tableData.Name); err != nil {
				return nil, err
			}
		}

		// find index list
		indexList, err := s.FindIndex(ctx, &api.IndexFind{
			DatabaseID: &databaseID,
			TableID:    &table.ID,
		})
		if err != nil {
			return nil, err
		}
		tableData.IndexList = convertIndexList(indexList)

		// find column list
		columnList, err := s.FindColumn(ctx, &api.ColumnFind{
			DatabaseID: &databaseID,
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
	viewList, err := s.FindView(ctx, &api.ViewFind{
		DatabaseID: &databaseID,
	})
	if err != nil {
		return nil, err
	}
	for _, view := range viewList {
		viewData := convertView(view)
		schemaName := ""
		if engineType == db.Postgres {
			if schemaName, viewData.Name, err = splitPGSchema(viewData.Name); err != nil {
				return nil, err
			}
		}
		schema := schemaSet.getOrCreateSchema(schemaName)
		schema.ViewList = append(schema.ViewList, viewData)
	}

	// find extension list
	extensionList, err := s.FindDBExtension(ctx, &api.DBExtensionFind{
		DatabaseID: &databaseID,
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
		return "", "", errors.Errorf("split failed: the expected name is schemaName.name, but get %s", name)
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
