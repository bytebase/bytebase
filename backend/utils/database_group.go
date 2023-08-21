package utils

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/store"
)

// ConvertDatabaseToParserEngineType converts a database type to a parser engine type.
func ConvertDatabaseToParserEngineType(engine db.Type) (parser.EngineType, error) {
	switch engine {
	case db.Oracle:
		return parser.Oracle, nil
	case db.MSSQL:
		return parser.MSSQL, nil
	case db.Postgres:
		return parser.Postgres, nil
	case db.Redshift:
		return parser.Redshift, nil
	case db.MySQL:
		return parser.MySQL, nil
	case db.TiDB:
		return parser.TiDB, nil
	case db.MariaDB:
		return parser.MariaDB, nil
	case db.OceanBase:
		return parser.OceanBase, nil
	}
	return parser.EngineType("UNKNOWN"), errors.Errorf("unsupported engine type %q", engine)
}

// GetStatementsAndSchemaGroupsFromSchemaGroups takes in a statement template and a list of schema groups, returns a list of expanded(rendered) statements and schema group names.
func GetStatementsAndSchemaGroupsFromSchemaGroups(statement string, parserEngineType parser.EngineType, schemaGroupParent string, schemaGroups []*store.SchemaGroupMessage, schemaGroupMatchedTables map[string][]string) ([]string, []string, error) {
	flush := func(emptyStatementBuilder *strings.Builder, statementBuilder *strings.Builder, schemaGroup *store.SchemaGroupMessage, matchedTables []string) ([]string, []string) {
		if statementBuilder.Len() == 0 {
			return nil, nil
		}
		var resultStatements, schemaGroupNames []string
		if len(matchedTables) > 0 {
			for _, tableName := range matchedTables {
				statement := emptyStatementBuilder.String() +
					strings.ReplaceAll(statementBuilder.String(), schemaGroup.Placeholder, tableName)
				resultStatements = append(resultStatements, statement)
				schemaGroupNames = append(schemaGroupNames, fmt.Sprintf("%s/%s%s", schemaGroupParent, common.SchemaGroupNamePrefix, schemaGroup.ResourceID))
			}
		} else {
			statement := emptyStatementBuilder.String() + statementBuilder.String()
			resultStatements = append(resultStatements, statement)
			schemaGroupNames = append(schemaGroupNames, "")
		}
		emptyStatementBuilder.Reset()
		statementBuilder.Reset()
		return resultStatements, schemaGroupNames
	}

	singleStatements, err := parser.SplitMultiSQL(parserEngineType, statement)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to split sql")
	}
	if len(singleStatements) == 0 {
		return nil, nil, errors.Errorf("no sql statement found")
	}

	var resultStatements, resultSchemaGroupNames []string
	var emptyStatementBuilder, statementBuilder strings.Builder

	var preMatch, curMatch *store.SchemaGroupMessage
	for _, singleStatement := range singleStatements {
		if singleStatement.Empty {
			_, _ = emptyStatementBuilder.WriteString(singleStatement.Text)
			continue
		}
		for _, schemaGroup := range schemaGroups {
			if strings.Contains(singleStatement.Text, schemaGroup.Placeholder) {
				curMatch = schemaGroup
				break
			}
		}

		// discard statement that matches the placeholder but has no matched tables
		if curMatch != nil && len(schemaGroupMatchedTables[curMatch.ResourceID]) == 0 {
			curMatch = nil
			continue
		}

		if preMatch == nil && curMatch != nil {
			statements, schemaGroupNames := flush(&emptyStatementBuilder, &statementBuilder, nil, nil)
			resultStatements = append(resultStatements, statements...)
			resultSchemaGroupNames = append(resultSchemaGroupNames, schemaGroupNames...)
		}
		if preMatch != nil && curMatch == nil {
			statements, schemaGroupNames := flush(&emptyStatementBuilder, &statementBuilder, preMatch, schemaGroupMatchedTables[preMatch.ResourceID])
			resultStatements = append(resultStatements, statements...)
			resultSchemaGroupNames = append(resultSchemaGroupNames, schemaGroupNames...)
		}
		if preMatch != nil && curMatch != nil && preMatch.ResourceID != curMatch.ResourceID {
			statements, schemaGroupNames := flush(&emptyStatementBuilder, &statementBuilder, preMatch, schemaGroupMatchedTables[preMatch.ResourceID])
			resultStatements = append(resultStatements, statements...)
			resultSchemaGroupNames = append(resultSchemaGroupNames, schemaGroupNames...)
		}

		_, _ = statementBuilder.WriteString(singleStatement.Text)
		_, _ = statementBuilder.WriteString("\n")

		preMatch = curMatch
		curMatch = nil
	}

	if preMatch != nil {
		statements, schemaGroupNames := flush(&emptyStatementBuilder, &statementBuilder, preMatch, schemaGroupMatchedTables[preMatch.ResourceID])
		resultStatements = append(resultStatements, statements...)
		resultSchemaGroupNames = append(resultSchemaGroupNames, schemaGroupNames...)
	} else {
		statements, schemaGroupNames := flush(&emptyStatementBuilder, &statementBuilder, nil, nil)
		resultStatements = append(resultStatements, statements...)
		resultSchemaGroupNames = append(resultSchemaGroupNames, schemaGroupNames...)
	}

	if emptyStatementBuilder.Len() > 0 && len(resultStatements) > 0 {
		resultStatements[len(resultStatements)-1] += emptyStatementBuilder.String()
	}

	return resultStatements, resultSchemaGroupNames, nil
}
