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
