package model

import (
	"slices"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type ChangedResources struct {
	databases map[string]*ChangedDatabase

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
		databases:  make(map[string]*ChangedDatabase),
		dbMetadata: dbMetadata,
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
