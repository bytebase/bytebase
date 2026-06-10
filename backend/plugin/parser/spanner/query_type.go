package spanner

import (
	"github.com/bytebase/omni/googlesql/analysis"
	"github.com/bytebase/omni/googlesql/ast"
	"github.com/bytebase/omni/googlesql/parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// classifyStatement classifies a single Spanner (GoogleSQL) statement via the
// omni analysis classifier and maps the result onto bytebase's base.QueryType.
//
// omni's analysis.ClassifySQL reproduces the legacy queryTypeListener's rules
// for the Spanner dialect (system schemas = INFORMATION_SCHEMA and SPANNER_SYS),
// with one legacy-spanner special case re-applied here: the legacy listener
// classified a SET statement as base.Select ("treat SAFE SET as select") while
// omni classifies it Unknown.
func classifyStatement(statement string) base.QueryType {
	qt := mapQueryType(analysis.ClassifySQL(statement, analysis.DialectSpanner))
	if qt == base.QueryTypeUnknown && isSetStatement(statement) {
		return base.Select
	}
	return qt
}

// isSetStatement reports whether the statement parses to a single SET statement
// (the legacy spanner queryTypeListener's Set_statement → Select case).
func isSetStatement(statement string) bool {
	file, errs := parser.Parse(statement)
	if len(errs) > 0 || file == nil || len(file.Stmts) != 1 {
		return false
	}
	_, ok := file.Stmts[0].(*ast.SetStmt)
	return ok
}

// mapQueryType maps an omni analysis.QueryType onto bytebase's base.QueryType.
// The two enums are defined to correspond 1:1 (omni's QueryType doc states the
// values mirror base.QueryType so this switch is total); an unexpected value
// falls back to QueryTypeUnknown.
func mapQueryType(t analysis.QueryType) base.QueryType {
	switch t {
	case analysis.Select:
		return base.Select
	case analysis.Explain:
		return base.Explain
	case analysis.SelectInfoSchema:
		return base.SelectInfoSchema
	case analysis.DDL:
		return base.DDL
	case analysis.DML:
		return base.DML
	case analysis.Unknown:
		return base.QueryTypeUnknown
	default:
		return base.QueryTypeUnknown
	}
}
