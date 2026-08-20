package trino

import (
	"github.com/bytebase/omni/trino/ast"
	"github.com/bytebase/omni/trino/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TRINO, validateQuery)
}

// validateQuery reports whether the given statement is valid for the SQL editor,
// which only permits read-only queries.
//
// It returns (canRunInReadOnly, returnsData, error):
//   - canRunInReadOnly: every statement can run in read-only mode;
//   - returnsData: every statement returns data;
//   - error: a syntax error if the statement is invalid.
//
// EXPLAIN ANALYZE is special-cased exactly as the legacy plugin did: because it
// executes the inner query, it is only accepted when getQueryType reports it as
// read-only (base.Select); it then counts as read-only and data-returning.
// changesSession reports whether a statement rebinds the connection rather
// than reading from it.
func changesSession(node ast.Node) bool {
	switch node.(type) {
	case *parser.SetSessionStmt, *parser.ResetSessionStmt,
		*parser.SetSessionAuthorizationStmt, *parser.ResetSessionAuthorizationStmt,
		*parser.SetPathStmt, *parser.SetRoleStmt, *parser.SetTimeZoneStmt,
		*parser.UseStmt:
		return true
	default:
		return false
	}
}

func validateQuery(statement string) (bool, bool, error) {
	parsed, err := parseTrinoSQL(statement)
	if err != nil {
		return false, false, err
	}

	allReadOnly := true
	allReturnData := true

	for _, p := range parsed {
		queryType, isAnalyze := getQueryType(p.Node())

		if isAnalyze {
			// EXPLAIN ANALYZE runs the query. Only allow it when the type is a
			// read-only SELECT; in that case it is read-only and returns data.
			if queryType != base.Select {
				return false, false, nil
			}
			continue
		}

		readOnly := queryType == base.Select ||
			queryType == base.Explain ||
			queryType == base.SelectInfoSchema
		// USE, SET ROLE, SET PATH, SET SESSION and SET SESSION AUTHORIZATION
		// read nothing and return nothing; what they do is rebind the
		// connection's catalog, schema, role or identity, so every statement
		// after them resolves somewhere the caller did not ask for. Reporting
		// them as not-data-returning is how every other engine reports its SET
		// family, and it is what a read-only caller is held to.
		returnsData := readOnly && !changesSession(p.Node())

		if !readOnly {
			allReadOnly = false
		}
		if !returnsData {
			allReturnData = false
		}
		if !allReadOnly {
			break
		}
	}

	return allReadOnly, allReturnData, nil
}
