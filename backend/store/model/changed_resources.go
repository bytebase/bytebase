package model

import (
	"slices"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type ChangedResources struct {
	databases map[string]*ChangedDatabase

	dbSchema *DatabaseSchema
}

type ChangedDatabase struct {
	schemas map[string]*ChangedSchema
}

type ChangedSchema struct {
	tables     map[string]*ChangedTable
	views      map[string]*storepb.ChangedResourceView
	functions  map[string]*storepb.ChangedResourceFunction
	procedures map[string]*storepb.ChangedResourceProcedure
}

type ChangedTable struct {
	table         *storepb.ChangedResourceTable
	affectedTable bool
}

func NewChangedResources(dbSchema *DatabaseSchema) *ChangedResources {
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

	for _, view := range s.views {
		changedResourceSchema.Views = append(changedResourceSchema.Views, view)
	}
	slices.SortFunc(changedResourceSchema.Views, func(a, b *storepb.ChangedResourceView) int {
		if a.GetName() < b.GetName() {
			return -1
		} else if a.GetName() > b.GetName() {
			return 1
		}
		return 0
	})

	for _, function := range s.functions {
		changedResourceSchema.Functions = append(changedResourceSchema.Functions, function)
	}
	slices.SortFunc(changedResourceSchema.Functions, func(a, b *storepb.ChangedResourceFunction) int {
		if a.GetName() < b.GetName() {
			return -1
		} else if a.GetName() > b.GetName() {
			return 1
		}
		return 0
	})

	for _, procedure := range s.procedures {
		changedResourceSchema.Procedures = append(changedResourceSchema.Procedures, procedure)
	}
	slices.SortFunc(changedResourceSchema.Procedures, func(a, b *storepb.ChangedResourceProcedure) int {
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
	v.table.Ranges = append(v.table.Ranges, change.GetRanges()...)
}

func (r *ChangedResources) AddView(database string, schema string, change *storepb.ChangedResourceView) {
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
	if r.databases[database].schemas[schema].views == nil {
		r.databases[database].schemas[schema].views = make(map[string]*storepb.ChangedResourceView)
	}
	v, ok := r.databases[database].schemas[schema].views[change.GetName()]
	if !ok {
		r.databases[database].schemas[schema].views[change.GetName()] = change
		return
	}
	v.Ranges = append(v.Ranges, change.GetRanges()...)
}

func (r *ChangedResources) AddFunction(database string, schema string, change *storepb.ChangedResourceFunction) {
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
	if r.databases[database].schemas[schema].functions == nil {
		r.databases[database].schemas[schema].functions = make(map[string]*storepb.ChangedResourceFunction)
	}
	v, ok := r.databases[database].schemas[schema].functions[change.GetName()]
	if !ok {
		r.databases[database].schemas[schema].functions[change.GetName()] = change
		return
	}
	v.Ranges = append(v.Ranges, change.GetRanges()...)
}

func (r *ChangedResources) AddProcedure(database string, schema string, change *storepb.ChangedResourceProcedure) {
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
	if r.databases[database].schemas[schema].procedures == nil {
		r.databases[database].schemas[schema].procedures = make(map[string]*storepb.ChangedResourceProcedure)
	}
	v, ok := r.databases[database].schemas[schema].procedures[change.GetName()]
	if !ok {
		r.databases[database].schemas[schema].procedures[change.GetName()] = change
		return
	}
	v.Ranges = append(v.Ranges, change.GetRanges()...)
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
