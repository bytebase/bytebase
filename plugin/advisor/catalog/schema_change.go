package catalog

import (
	"github.com/bytebase/bytebase/plugin/advisor/db"
)

func newInterDatabase(d *Database) *interDatabase {
	database := &interDatabase{
		name:         d.Name,
		characterSet: d.CharacterSet,
		collation:    d.Collation,
		dbType:       d.DbType,
		schemaSet:    make(interSchemaMap),
	}

	for _, schema := range d.SchemaList {
		database.schemaSet[schema.Name] = newInterSchema(schema)
	}

	return database
}

func newInterSchema(s *Schema) *interSchema {
	schema := &interSchema{
		name:         s.Name,
		tableSet:     make(interTableMap),
		viewSet:      make(interViewMap),
		extensionSet: make(interExtensionMap),
	}

	for _, table := range s.TableList {
		schema.tableSet[table.Name] = newInterTable(table)
	}

	for _, view := range s.ViewList {
		schema.viewSet[view.Name] = newInterView(view)
	}

	for _, extension := range s.ExtensionList {
		schema.extensionSet[extension.Name] = newInterExtension(extension)
	}

	return schema
}

func newInterView(v *View) *interView {
	return &interView{
		name:       v.Name,
		definition: v.Definition,
		comment:    v.Comment,
	}
}

func newInterExtension(e *Extension) *interExtension {
	return &interExtension{
		name:        e.Name,
		version:     e.Version,
		description: e.Description,
	}
}

func newInterTable(t *Table) *interTable {
	table := &interTable{
		name:          t.Name,
		tableType:     t.Type,
		engine:        t.Engine,
		collation:     t.Collation,
		rowCount:      t.RowCount,
		dataSize:      t.DataSize,
		indexSize:     t.IndexSize,
		dataFree:      t.DataFree,
		createOptions: t.CreateOptions,
		comment:       t.Comment,
		columnSet:     make(interColumnMap),
		indexSet:      make(interIndexMap),
	}

	for _, column := range t.ColumnList {
		table.columnSet[column.Name] = newInterColumn(column)
	}

	for _, index := range t.IndexList {
		table.indexSet[index.Name] = newInterIndex(index)
	}

	return table
}

func newInterColumn(c *Column) *interColumn {
	return &interColumn{
		name:         c.Name,
		position:     c.Position,
		defaultValue: c.Default,
		nullable:     c.Nullable,
		columnType:   c.Type,
		characterSet: c.CharacterSet,
		collation:    c.Collation,
		comment:      c.Comment,
	}
}

func newInterIndex(i *Index) *interIndex {
	index := &interIndex{
		name:      i.Name,
		indextype: i.Type,
		unique:    i.Unique,
		primary:   i.Primary,
		visible:   i.Visible,
		comment:   i.Comment,
	}
	index.expressionList = append(index.expressionList, i.ExpressionList...)
	return index
}

type interDatabase struct {
	name         string
	characterSet string
	collation    string
	dbType       db.Type
	schemaSet    interSchemaMap
}
type interSchema struct {
	name         string
	tableSet     interTableMap
	viewSet      interViewMap
	extensionSet interExtensionMap
}
type interSchemaMap map[string]*interSchema

type interTable struct {
	name      string
	tableType string
	// engine isn't supported for Postgres, Snowflake, SQLite.
	engine string
	// collation isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	collation string
	rowCount  int64
	// dataSize isn't supported for SQLite.
	dataSize int64
	// indexSize isn't supported for ClickHouse, Snowflake, SQLite.
	indexSize int64
	// dataFree isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	dataFree int64
	// createOptions isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	createOptions string
	// comment isn't supported for SQLite.
	comment   string
	columnSet interColumnMap
	// indexSet isn't supported for ClickHouse, Snowflake.
	indexSet interIndexMap
}
type interTableMap map[string]*interTable

func (table *interTable) convertToCatalog() *Table {
	return &Table{
		Name:          table.name,
		Type:          table.tableType,
		Engine:        table.engine,
		Collation:     table.collation,
		RowCount:      table.rowCount,
		DataSize:      table.dataSize,
		IndexSize:     table.indexSize,
		DataFree:      table.dataFree,
		CreateOptions: table.createOptions,
		Comment:       table.comment,
		ColumnList:    table.columnSet.convertToCatalog(),
		IndexList:     table.indexSet.convertToCatalog(),
	}
}

type interIndex struct {
	name string
	// This could refer to a column or an expression.
	expressionList []string
	// Type isn't supported for SQLite.
	indextype string
	unique    bool
	primary   bool
	// Visible isn't supported for Postgres, SQLite.
	visible bool
	// Comment isn't supported for SQLite.
	comment string
}
type interIndexMap map[string]*interIndex

func (m interIndexMap) convertToCatalog() []*Index {
	var res []*Index
	for _, index := range m {
		res = append(res, index.convertToCatalog())
	}
	return res
}

func (index *interIndex) convertToCatalog() *Index {
	return &Index{
		Name:           index.name,
		ExpressionList: index.expressionList,
		Type:           index.indextype,
		Unique:         index.unique,
		Primary:        index.primary,
		Visible:        index.visible,
		Comment:        index.comment,
	}
}

type interColumn struct {
	name         string
	position     int
	defaultValue *string
	// nullable isn't supported for ClickHouse.
	nullable   bool
	columnType string
	// characterSet isn't supported for Postgres, ClickHouse, SQLite.
	characterSet string
	// collation isn't supported for ClickHouse, SQLite.
	collation string
	// comment isn't supported for SQLite.
	comment string
}
type interColumnMap map[string]*interColumn

func (m interColumnMap) convertToCatalog() []*Column {
	var res []*Column
	for _, column := range m {
		res = append(res, column.convertToCatalog())
	}
	return res
}

func (column *interColumn) convertToCatalog() *Column {
	return &Column{
		Name:         column.name,
		Position:     column.position,
		Default:      column.defaultValue,
		Nullable:     column.nullable,
		Type:         column.columnType,
		CharacterSet: column.characterSet,
		Collation:    column.collation,
		Comment:      column.comment,
	}
}

type interView struct {
	name       string
	definition string
	comment    string
}
type interViewMap map[string]*interView

type interExtension struct {
	name        string
	version     string
	description string
}
type interExtensionMap map[string]*interExtension
