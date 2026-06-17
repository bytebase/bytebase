package model

import (
	"slices"

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
				if r.dbMetadata == nil {
					continue
				}
				schemaMetadata := r.dbMetadata.GetSchemaMetadata(schema.GetName())
				if schemaMetadata == nil {
					continue
				}
				tableMetadata := schemaMetadata.GetTable(table.GetName())
				if tableMetadata != nil {
					table.TableRows = tableMetadata.GetProto().GetRowCount()
				}
			}
		}
		changedResources.Databases = append(changedResources.Databases, d)
	}
	return changedResources
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

func (r *ChangedResources) CountAffectedTableRows() int64 {
	if r.dbMetadata == nil {
		return 0
	}

	var totalAffectedRows int64
	for _, d := range r.databases {
		for schemaName, schema := range d.schemas {
			for tableName, table := range schema.tables {
				if !table.affectedTable {
					continue
				}
				if r.dbMetadata == nil {
					continue
				}
				schemaMeta := r.dbMetadata.GetSchemaMetadata(schemaName)
				if schemaMeta == nil {
					continue
				}
				tableMeta := schemaMeta.GetTable(tableName)
				if tableMeta == nil {
					continue
				}
				totalAffectedRows += tableMeta.GetProto().GetRowCount()
			}
		}
	}
	return totalAffectedRows
}
