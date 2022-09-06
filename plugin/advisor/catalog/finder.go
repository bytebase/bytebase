package catalog

import "github.com/bytebase/bytebase/plugin/advisor/db"

// Finder is the service for finding schema information in database.
type Finder struct {
	database *databaseState
}

// NewFinder creates a new finder.
func NewFinder(database *Database) *Finder {
	return &Finder{database: newDatabaseState(database)}
}

// HasNoTable returns true if the current database has no table.
func (f *Finder) HasNoTable() bool {
	for _, schema := range f.database.schemaSet {
		if len(schema.tableSet) != 0 {
			return false
		}
	}

	return true
}

// DatabaseName returns the database name.
func (f *Finder) DatabaseName() string {
	return f.database.name
}

// IndexFind is for find index.
type IndexFind struct {
	SchemaName string
	TableName  string
	IndexName  string
}

// FindIndex finds the index.
func (f *Finder) FindIndex(find *IndexFind) (string, *Index) {
	// needMatchTable is used for PostgreSQL. In PostgreSQL, the index name is unique in a schema, not a table.
	needMatchTable := (f.database.dbType != db.Postgres || find.SchemaName == "" || find.TableName != "")
	if needMatchTable {
		schema, exists := f.database.schemaSet[find.SchemaName]
		if !exists {
			return "", nil
		}
		table, exists := schema.tableSet[find.TableName]
		if !exists {
			return "", nil
		}
		index, exists := table.indexSet[find.IndexName]
		if !exists {
			return "", nil
		}
		return table.name, index.convertToCatalog()
	}
	for _, schema := range f.database.schemaSet {
		if schema.name != find.SchemaName {
			continue
		}
		for _, table := range schema.tableSet {
			// no need to match table name.
			index, exists := table.indexSet[find.IndexName]
			if !exists {
				return "", nil
			}
			return table.name, index.convertToCatalog()
		}
	}
	return "", nil
}

// PrimaryKeyFind is for find primary key.
type PrimaryKeyFind struct {
	SchemaName string
	TableName  string
}

// FindPrimaryKey finds the primary key.
func (f *Finder) FindPrimaryKey(find *PrimaryKeyFind) *Index {
	for _, schema := range f.database.schemaSet {
		if schema.name != find.SchemaName {
			continue
		}
		for _, table := range schema.tableSet {
			if table.name != find.TableName {
				continue
			}
			for _, index := range table.indexSet {
				if index.primary {
					return index.convertToCatalog()
				}
			}
		}
	}
	return nil
}

// ColumnFind is for find column.
type ColumnFind struct {
	SchemaName string
	TableName  string
	ColumnName string
}

// FindColumn finds the column.
func (f *Finder) FindColumn(find *ColumnFind) *Column {
	schema, exists := f.database.schemaSet[find.SchemaName]
	if !exists {
		return nil
	}
	table, exists := schema.tableSet[find.TableName]
	if !exists {
		return nil
	}
	column, exists := table.columnSet[find.ColumnName]
	if !exists {
		return nil
	}
	return column.convertToCatalog()
}

// TableFind is for find table.
type TableFind struct {
	SchemaName string
	TableName  string
}

// FindTable finds the table.
func (f *Finder) FindTable(find *TableFind) *Table {
	schema, exists := f.database.schemaSet[find.SchemaName]
	if !exists {
		return nil
	}
	table, exists := schema.tableSet[find.TableName]
	if !exists {
		return nil
	}
	return table.convertToCatalog()
}
