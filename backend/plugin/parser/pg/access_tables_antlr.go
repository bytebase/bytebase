package pg

import (
	"context"

	"github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// antlrAccessTableExtractor is the ANTLR-based access table extractor.
// Used by query_span_extractor.go which still relies on ANTLR parse trees.
type antlrAccessTableExtractor struct {
	*postgresql.BasePostgreSQLParserListener
	err                    error
	defaultDatabase        string
	searchPath             []string
	accessTables           []base.ColumnResource
	getDatabaseMetadata    base.GetDatabaseMetadataFunc
	ctx                    context.Context
	instanceID             string
	skipMetadataValidation bool
}

func (a *antlrAccessTableExtractor) EnterQualified_name(qn *postgresql.Qualified_nameContext) {
	if a.err != nil {
		return
	}
	resource := base.ColumnResource{
		Database: a.defaultDatabase,
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
		a.err = errors.Errorf("improper qualified name (too many dotted names): %s", qn.GetParser().GetTokenStream().GetTextFromInterval(qn.GetSourceInterval()))
		return
	}

	if a.skipMetadataValidation {
		if resource.Schema == "" && !isSystemResource(resource) {
			if len(a.searchPath) > 0 {
				resource.Schema = a.searchPath[0]
			}
		}
		a.accessTables = append(a.accessTables, resource)
		return
	}

	if !isSystemResource(resource) {
		searchPath := a.searchPath
		if resource.Schema != "" {
			searchPath = []string{resource.Schema}
		}

		_, databaseMetadata, err := a.getDatabaseMetadata(a.ctx, a.instanceID, a.defaultDatabase)
		if err != nil {
			a.err = errors.Wrapf(err, "failed to get database metadata for database: %s", a.defaultDatabase)
			return
		}
		if databaseMetadata == nil {
			return
		}
		schemaName, name := databaseMetadata.SearchObject(searchPath, resource.Table)
		if schemaName == "" && name == "" {
			return
		}
		resource.Schema = schemaName
	}

	a.accessTables = append(a.accessTables, resource)
}
