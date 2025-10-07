package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"
)

func (l *querySpanExtractor) EnterQualified_name(qn *postgresql.Qualified_nameContext) {
	if l.err != nil {
		return
	}
	resource := base.ColumnResource{
		Database: l.defaultDatabase,
	}
	var directions []string
	qnLength := 1
	directions = append(directions, NormalizePostgreSQLColid(qn.Colid()))
	if qn.Indirection() != nil {
		allIndirectionElements := qn.Indirection().AllIndirection_el()
		for _, el := range allIndirectionElements {
			if el.Attr_name() != nil {
				directions = append(directions, normalizePostgreSQLCollabel(el.Attr_name().Collabel()))
				qnLength++
				break
			}
			continue
		}
	}

	switch qnLength {
	case 1:
		resource.Table = directions[0]
	case 2:
		resource.Schema = directions[0]
		resource.Table = directions[1]
	case 3:
		resource.Database = directions[0]
		resource.Schema = directions[1]
		resource.Table = directions[2]
	default:
		l.err = errors.Errorf("improper qualified name (too many dotted names): %s", qn.GetParser().GetTokenStream().GetTextFromInterval(qn.GetSourceInterval()))
		return
	}

	if !isSystemResource(resource) {
		searchPath := l.searchPath
		if resource.Schema != "" {
			searchPath = []string{resource.Schema}
		}

		databaseMetadata, err := l.getDatabaseMetadata(l.defaultDatabase)
		if err != nil {
			l.err = errors.Wrapf(err, "failed to get database metadata for database: %s", l.defaultDatabase)
		}
		// Access pseudo table or table/view we do not sync, return directly.
		if databaseMetadata == nil {
			return
		}
		schemaName, name := databaseMetadata.SearchObject(searchPath, resource.Table)
		if schemaName == "" && name == "" {
			return
		}
		resource.Schema = schemaName
	}

	l.accessTables = append(l.accessTables, resource)
}

// isMixedQuery checks whether the query accesses the user table and system table at the same time.
func isMixedQuery(m base.SourceColumnSet) (bool, bool) {
	hasSystem, hasUser := false, false
	for table := range m {
		if isSystemResource(table) {
			hasSystem = true
		} else {
			hasUser = true
		}
	}

	if hasSystem && hasUser {
		return false, true
	}

	return !hasUser && hasSystem, false
}

func isSystemResource(resource base.ColumnResource) bool {
	// User can access the system table/view by name directly without database/schema name.
	// For example: `SELECT * FROM pg_database`, which will access the system table `pg_database`.
	// Additionally, user can create a table/view with the same name with system table/view and access them
	// by specify the schema name, for example:
	// `CREATE TABLE pg_database(id INT); SELECT * FROM public.pg_database;` which will access the user table `pg_database`.
	if IsSystemSchema(resource.Schema) {
		return true
	}
	if resource.Schema == "" && IsSystemView(resource.Table) {
		return true
	}
	if resource.Schema == "" && IsSystemTable(resource.Table) {
		return true
	}
	return false
}
