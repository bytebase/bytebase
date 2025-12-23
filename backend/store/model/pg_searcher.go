package model

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// DatabaseSearcher provides a fluent interface for searching database objects with a specific search path.
// This helper avoids repetitive searchPath parameter passing when performing multiple searches.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
type DatabaseSearcher struct {
	db         *DatabaseMetadata
	searchPath []string
}

// NewSearcher creates a searcher for database objects.
// If schemaName is provided (non-empty), it searches only in that schema.
// Otherwise, it uses the provided fallbackSearchPath (e.g., from UI-selected schema).
// If fallbackSearchPath is also empty, defaults to ["public"] for PostgreSQL.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func (d *DatabaseMetadata) NewSearcher(schemaName string, fallbackSearchPath []string) *DatabaseSearcher {
	searchPath := fallbackSearchPath
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
func (s *DatabaseSearcher) SearchView(name string) (string, *storepb.ViewMetadata) {
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
func (s *DatabaseSearcher) SearchSequence(name string) (string, *storepb.SequenceMetadata) {
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
func (s *DatabaseSearcher) SearchMaterializedView(name string) (string, *storepb.MaterializedViewMetadata) {
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
func (s *DatabaseSearcher) SearchFunctions(name string) ([]string, []*storepb.FunctionMetadata) {
	var schemas []string
	var funcs []*storepb.FunctionMetadata
	for _, schemaName := range s.searchPath {
		schema := s.db.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		for _, function := range schema.GetProto().GetFunctions() {
			if s.db.isDetailCaseSensitive {
				if function.Name == name {
					schemas = append(schemas, schema.proto.Name)
					funcs = append(funcs, function)
				}
			} else {
				if strings.EqualFold(function.Name, name) {
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
func (d *DatabaseMetadata) SearchView(searchPath []string, name string) (string, *storepb.ViewMetadata) {
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
func (d *DatabaseMetadata) SearchSequence(searchPath []string, name string) (string, *storepb.SequenceMetadata) {
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
func (d *DatabaseMetadata) SearchMaterializedView(searchPath []string, name string) (string, *storepb.MaterializedViewMetadata) {
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
func (d *DatabaseMetadata) SearchFunctions(searchPath []string, name string) ([]string, []*storepb.FunctionMetadata) {
	var schemas []string
	var funcs []*storepb.FunctionMetadata
	for _, schemaName := range searchPath {
		schema := d.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		for _, function := range schema.GetProto().GetFunctions() {
			if d.isDetailCaseSensitive {
				if function.Name == name {
					schemas = append(schemas, schema.proto.Name)
					funcs = append(funcs, function)
				}
			} else {
				if strings.EqualFold(function.Name, name) {
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

// normalizeSearchPath normalizes the search path string into a slice of strings.
// NOTE: This is primarily designed for PostgreSQL's search_path concept.
func normalizeSearchPath(searchPath string) []string {
	if searchPath == "" {
		return []string{}
	}

	// Split the search path by comma and trim spaces.
	parts := strings.Split(searchPath, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	// Remove empty parts.
	var result []string
	for _, part := range parts {
		schema := strings.TrimSpace(part)
		if part == "\"$user\"" {
			continue
		}
		if strings.HasPrefix(part, "\"") && strings.HasSuffix(part, "\"") {
			// Remove the quotes from the schema name.
			schema = strings.Trim(schema, "\"")
		} else if strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'") {
			// Remove the single quotes from the schema name.
			schema = strings.Trim(schema, "'")
		} else {
			// For non-quoted schema names, we just return the lower string for PostgreSQL.
			schema = strings.ToLower(schema)
		}
		schema = strings.TrimSpace(schema)
		if isSystemPath(schema) {
			// Skip system schemas.
			continue
		}
		if schema != "" {
			result = append(result, schema)
		}
	}

	return result
}

// isSystemPath checks if a path is a PostgreSQL system schema.
// NOTE: This is primarily designed for PostgreSQL.
func isSystemPath(path string) bool {
	// PostgreSQL system schemas.
	systemSchemas := []string{"pg_catalog", "information_schema", "pg_toast", "pg_temp_1", "pg_temp_2", "pg_global", "$user"}
	for _, schema := range systemSchemas {
		if strings.EqualFold(path, schema) {
			return true
		}
	}
	return false
}
