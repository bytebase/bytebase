package model

import (
	"sort"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type ChangedResources struct {
	databases map[string]*ChangedDatabase

	dbSchema *DBSchema
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

func NewChangedResources(dbSchema *DBSchema) *ChangedResources {
	return &ChangedResources{
		databases: make(map[string]*ChangedDatabase),
		dbSchema:  dbSchema,
	}
}

func (r *ChangedResources) Build() *storepb.ChangedResources {
	changedResources := &storepb.ChangedResources{}
	for name, database := range r.databases {
		d := database.build()
		d.Name = name
		for _, schema := range d.Schemas {
			for _, table := range schema.Tables {
				if r.dbSchema == nil {
					continue
				}
				if r.dbSchema.GetDatabaseMetadata() == nil {
					continue
				}
				schemaMetadata := r.dbSchema.GetDatabaseMetadata().GetSchema(schema.GetName())
				if schemaMetadata == nil {
					continue
				}
				tableMetadata := schemaMetadata.GetTable(table.GetName())
				if tableMetadata != nil {
					table.TableRows = tableMetadata.GetRowCount()
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
	sort.Slice(changedResourceDatabase.Schemas, func(i, j int) bool {
		return changedResourceDatabase.Schemas[i].GetName() < changedResourceDatabase.Schemas[j].GetName()
	})
	return changedResourceDatabase
}

func (s *ChangedSchema) build() *storepb.ChangedResourceSchema {
	changedResourceSchema := &storepb.ChangedResourceSchema{}
	for _, table := range s.tables {
		changedResourceSchema.Tables = append(changedResourceSchema.Tables, table.table)
	}
	sort.Slice(changedResourceSchema.Tables, func(i, j int) bool {
		return changedResourceSchema.Tables[i].GetName() < changedResourceSchema.Tables[j].GetName()
	})
	return changedResourceSchema
}

func (r *ChangedResources) AddTable(database string, schema string, tableChange *storepb.ChangedResourceTable, affectedTable bool) {
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
	v, ok := r.databases[database].schemas[schema].tables[tableChange.GetName()]
	if !ok {
		r.databases[database].schemas[schema].tables[tableChange.GetName()] = &ChangedTable{
			table:         tableChange,
			affectedTable: affectedTable,
		}
		return
	}
	if affectedTable {
		v.affectedTable = true
	}
	v.table.Ranges = append(v.table.Ranges, tableChange.GetRanges()...)
}

func (r *ChangedResources) CountAffectedTableRows() int64 {
	if r.dbSchema == nil {
		return 0
	}

	var totalAffectedRows int64
	for _, d := range r.databases {
		for schemaName, schema := range d.schemas {
			for tableName, table := range schema.tables {
				if !table.affectedTable {
					continue
				}
				dbMeta := r.dbSchema.GetDatabaseMetadata()
				if dbMeta == nil {
					continue
				}
				schemaMeta := dbMeta.GetSchema(schemaName)
				if schemaMeta == nil {
					continue
				}
				tableMeta := schemaMeta.GetTable(tableName)
				if tableMeta == nil {
					continue
				}
				totalAffectedRows += tableMeta.GetRowCount()
			}
		}
	}
	return totalAffectedRows
}
