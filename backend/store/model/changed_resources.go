package model

import (
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type ChangedResources struct {
	databases map[string]*ChangedDatabase

	// databaseOnly holds databases changed by a statement whose write target is a non-table
	// object (e.g. a qualified CREATE VIEW/PROCEDURE/... in another database) — there is no
	// table to scope, only the target database. It is auth-only: it is NOT included in Build()
	// (the changed-resource report stays table-shaped), and is consumed by the SQL-editor
	// write-authorization path to gate a qualified cross-database object DDL by its own database.
	databaseOnly map[string]bool

	dbMetadata *DatabaseMetadata
}

type ChangedDatabase struct {
	schemas map[string]*ChangedSchema
}

type ChangedSchema struct {
	tables map[string]*ChangedTable
}

type ChangedTable struct {
	table         *storepb.ChangedResourceTable
	affectedTable bool
}

func NewChangedResources(dbMetadata *DatabaseMetadata) *ChangedResources {
	return &ChangedResources{
		databases:    make(map[string]*ChangedDatabase),
		databaseOnly: make(map[string]bool),
		dbMetadata:   dbMetadata,
	}
}

func (r *ChangedResources) Build() *storepb.ChangedResources {
	changedResources := &storepb.ChangedResources{}
	for name, database := range r.databases {
		d := database.build()
		d.Name = name
		for _, schema := range d.Schemas {
			for _, table := range schema.Tables {
				if tableMetadata := r.getTableMetadata(name, schema.GetName(), table.GetName()); tableMetadata != nil {
					table.TableRows = tableMetadata.GetProto().GetRowCount()
				}
			}
		}
		changedResources.Databases = append(changedResources.Databases, d)
	}
	return changedResources
}

// getTableMetadata returns the synced metadata of a changed table. The metadata only
// describes the reviewed database, so a table qualified with another database has none,
// even when the reviewed database has a table of the same name.
func (r *ChangedResources) getTableMetadata(database, schema, table string) *TableMetadata {
	if r.dbMetadata == nil {
		return nil
	}
	if name := r.dbMetadata.DatabaseName(); name != "" {
		if r.dbMetadata.GetIsObjectCaseSensitive() {
			if database != name {
				return nil
			}
		} else if !strings.EqualFold(database, name) {
			return nil
		}
	}
	return r.dbMetadata.GetSchemaMetadata(schema).GetTable(table)
}

func (d *ChangedDatabase) build() *storepb.ChangedResourceDatabase {
	changedResourceDatabase := &storepb.ChangedResourceDatabase{}
	for name, schema := range d.schemas {
		s := schema.build()
		s.Name = name
		changedResourceDatabase.Schemas = append(changedResourceDatabase.Schemas, s)
	}
	slices.SortFunc(changedResourceDatabase.Schemas, func(a, b *storepb.ChangedResourceSchema) int {
		if a.GetName() < b.GetName() {
			return -1
		} else if a.GetName() > b.GetName() {
			return 1
		}
		return 0
	})
	return changedResourceDatabase
}

func (s *ChangedSchema) build() *storepb.ChangedResourceSchema {
	changedResourceSchema := &storepb.ChangedResourceSchema{}
	for _, table := range s.tables {
		changedResourceSchema.Tables = append(changedResourceSchema.Tables, table.table)
	}
	slices.SortFunc(changedResourceSchema.Tables, func(a, b *storepb.ChangedResourceTable) int {
		if a.GetName() < b.GetName() {
			return -1
		} else if a.GetName() > b.GetName() {
			return 1
		}
		return 0
	})

	return changedResourceSchema
}

func (r *ChangedResources) AddTable(database string, schema string, change *storepb.ChangedResourceTable, affectedTable bool) {
	if _, ok := r.databases[database]; !ok {
		r.databases[database] = &ChangedDatabase{
			schemas: make(map[string]*ChangedSchema),
		}
	}
	if _, ok := r.databases[database].schemas[schema]; !ok {
		r.databases[database].schemas[schema] = &ChangedSchema{
			tables: make(map[string]*ChangedTable),
		}
	}
	if r.databases[database].schemas[schema].tables == nil {
		r.databases[database].schemas[schema].tables = make(map[string]*ChangedTable)
	}
	v, ok := r.databases[database].schemas[schema].tables[change.GetName()]
	if !ok {
		r.databases[database].schemas[schema].tables[change.GetName()] = &ChangedTable{
			table:         change,
			affectedTable: affectedTable,
		}
		return
	}
	if affectedTable {
		v.affectedTable = true
	}
}

// AddDatabase records a database changed by a non-table-object write (Tier 2): a qualified
// object DDL (CREATE/ALTER/DROP VIEW/PROCEDURE/FUNCTION/TRIGGER/...) whose only auth-relevant
// identity is its target database. Auth-only; not surfaced in Build().
func (r *ChangedResources) AddDatabase(database string) {
	if r.databaseOnly == nil {
		r.databaseOnly = make(map[string]bool)
	}
	r.databaseOnly[database] = true
}

// GetDatabaseOnlyTargets returns the databases recorded via AddDatabase (non-table-object
// write targets). Used by the SQL-editor write-authorization path.
func (r *ChangedResources) GetDatabaseOnlyTargets() []string {
	result := make([]string, 0, len(r.databaseOnly))
	for database := range r.databaseOnly {
		result = append(result, database)
	}
	slices.Sort(result)
	return result
}

// CountAffectedTableRows sums the synced row counts of the tables changed by DDL. Partitions
// have no row count of their own and resolve to their table, which is counted once.
func (r *ChangedResources) CountAffectedTableRows() int64 {
	counted := make(map[*metadatapb.TableMetadata]bool)
	var totalAffectedRows int64
	for databaseName, d := range r.databases {
		for schemaName, schema := range d.schemas {
			for tableName, table := range schema.tables {
				if !table.affectedTable {
					continue
				}
				tableMeta := r.getTableMetadata(databaseName, schemaName, tableName)
				if tableMeta == nil || counted[tableMeta.GetProto()] {
					continue
				}
				counted[tableMeta.GetProto()] = true
				totalAffectedRows += tableMeta.GetProto().GetRowCount()
			}
		}
	}
	return totalAffectedRows
}
