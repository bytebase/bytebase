package model

import "strings"

// DatabaseSearcher provides a fluent interface for searching database objects with a specific search path.
// This helper avoids repetitive searchPath parameter passing when performing multiple searches.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
type DatabaseSearcher struct {
	db         *DatabaseMetadata
	searchPath []string
}

// NewSearcher creates a searcher for database objects.
// If schemaName is provided (non-empty), it searches only in that schema; otherwise uses the database's search path.
// If the database's search path is empty, defaults to ["public"] for PostgreSQL.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) NewSearcher(schemaName string) *DatabaseSearcher {
	searchPath := d.GetSearchPath()
	if schemaName != "" {
		searchPath = []string{schemaName}
	} else if len(searchPath) == 0 {
		// Default to "public" schema if no search path is set (PostgreSQL default)
		searchPath = []string{"public"}
	}
	return &DatabaseSearcher{
		db:         d,
		searchPath: searchPath,
	}
}

// SearchTable searches for a table in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (s *DatabaseSearcher) SearchTable(name string) (string, *TableMetadata) {
	for _, schemaName := range s.searchPath {
		schema := s.db.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		table := schema.GetTable(name)
		if table != nil {
			return schema.proto.Name, table
		}
	}
	return "", nil
}

// SearchIndex searches for an index in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (s *DatabaseSearcher) SearchIndex(name string) (string, *IndexMetadata) {
	for _, schemaName := range s.searchPath {
		schema := s.db.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		index := schema.GetIndex(name)
		if index != nil {
			return schema.proto.Name, index
		}
	}
	return "", nil
}

// SearchView searches for a view in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (s *DatabaseSearcher) SearchView(name string) (string, *ViewMetadata) {
	for _, schemaName := range s.searchPath {
		schema := s.db.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		view := schema.GetView(name)
		if view != nil {
			return schema.proto.Name, view
		}
	}
	return "", nil
}

// SearchExternalTable searches for an external table in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (s *DatabaseSearcher) SearchExternalTable(name string) (string, *ExternalTableMetadata) {
	for _, schemaName := range s.searchPath {
		schema := s.db.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		externalTable := schema.GetExternalTable(name)
		if externalTable != nil {
			return schema.proto.Name, externalTable
		}
	}
	return "", nil
}

// SearchSequence searches for a sequence in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (s *DatabaseSearcher) SearchSequence(name string) (string, *SequenceMetadata) {
	for _, schemaName := range s.searchPath {
		schema := s.db.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		sequence := schema.GetSequence(name)
		if sequence != nil {
			return schema.proto.Name, sequence
		}
	}
	return "", nil
}

// SearchMaterializedView searches for a materialized view in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (s *DatabaseSearcher) SearchMaterializedView(name string) (string, *MaterializedViewMetadata) {
	for _, schemaName := range s.searchPath {
		schema := s.db.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		materializedView := schema.GetMaterializedView(name)
		if materializedView != nil {
			return schema.proto.Name, materializedView
		}
	}
	return "", nil
}

// SearchFunctions searches for functions in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (s *DatabaseSearcher) SearchFunctions(name string) ([]string, []*FunctionMetadata) {
	var schemas []string
	var funcs []*FunctionMetadata
	for _, schemaName := range s.searchPath {
		schema := s.db.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		for _, function := range schema.ListFunctions() {
			if s.db.isDetailCaseSensitive {
				if function.proto.Name == name {
					schemas = append(schemas, schema.proto.Name)
					funcs = append(funcs, function)
				}
			} else {
				if strings.EqualFold(function.proto.Name, name) {
					schemas = append(schemas, schema.proto.Name)
					funcs = append(funcs, function)
				}
			}
		}
	}
	return schemas, funcs
}

// SearchObject searches for any database object in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (s *DatabaseSearcher) SearchObject(name string) (string, string) {
	for _, schemaName := range s.searchPath {
		schema := s.db.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		if schema.GetTable(name) != nil || schema.GetView(name) != nil || schema.GetMaterializedView(name) != nil || schema.GetFunction(name) != nil || schema.GetProcedure(name) != nil || schema.GetPackage(name) != nil || schema.GetSequence(name) != nil || schema.GetExternalTable(name) != nil {
			return schema.proto.Name, name
		}
	}
	return "", ""
}

// SearchTable searches for a table in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) SearchTable(searchPath []string, name string) (string, *TableMetadata) {
	for _, schemaName := range searchPath {
		schema := d.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		table := schema.GetTable(name)
		if table != nil {
			return schema.proto.Name, table
		}
	}
	return "", nil
}

// SearchIndex searches for an index in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) SearchIndex(searchPath []string, name string) (string, *IndexMetadata) {
	for _, schemaName := range searchPath {
		schema := d.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		index := schema.GetIndex(name)
		if index != nil {
			return schema.proto.Name, index
		}
	}
	return "", nil
}

// SearchView searches for a view in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) SearchView(searchPath []string, name string) (string, *ViewMetadata) {
	for _, schemaName := range searchPath {
		schema := d.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		view := schema.GetView(name)
		if view != nil {
			return schema.proto.Name, view
		}
	}
	return "", nil
}

// SearchExternalTable searches for an external table in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) SearchExternalTable(searchPath []string, name string) (string, *ExternalTableMetadata) {
	for _, schemaName := range searchPath {
		schema := d.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		externalTable := schema.GetExternalTable(name)
		if externalTable != nil {
			return schema.proto.Name, externalTable
		}
	}
	return "", nil
}

// SearchSequence searches for a sequence in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) SearchSequence(searchPath []string, name string) (string, *SequenceMetadata) {
	for _, schemaName := range searchPath {
		schema := d.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		sequence := schema.GetSequence(name)
		if sequence != nil {
			return schema.proto.Name, sequence
		}
	}
	return "", nil
}

// SearchMaterializedView searches for a materialized view in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) SearchMaterializedView(searchPath []string, name string) (string, *MaterializedViewMetadata) {
	for _, schemaName := range searchPath {
		schema := d.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		materializedView := schema.GetMaterializedView(name)
		if materializedView != nil {
			return schema.proto.Name, materializedView
		}
	}
	return "", nil
}

// SearchFunctions searches for functions in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) SearchFunctions(searchPath []string, name string) ([]string, []*FunctionMetadata) {
	var schemas []string
	var funcs []*FunctionMetadata
	for _, schemaName := range searchPath {
		schema := d.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		for _, function := range schema.ListFunctions() {
			if d.isDetailCaseSensitive {
				if function.proto.Name == name {
					schemas = append(schemas, schema.proto.Name)
					funcs = append(funcs, function)
				}
			} else {
				if strings.EqualFold(function.proto.Name, name) {
					schemas = append(schemas, schema.proto.Name)
					funcs = append(funcs, function)
				}
			}
		}
	}
	return schemas, funcs
}

// SearchObject searches for any database object in the search path.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) SearchObject(searchPath []string, name string) (string, string) {
	for _, schemaName := range searchPath {
		schema := d.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		if schema.GetTable(name) != nil || schema.GetView(name) != nil || schema.GetMaterializedView(name) != nil || schema.GetFunction(name) != nil || schema.GetProcedure(name) != nil || schema.GetPackage(name) != nil || schema.GetSequence(name) != nil || schema.GetExternalTable(name) != nil {
			return schema.proto.Name, name
		}
	}
	return "", ""
}
