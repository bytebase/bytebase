package googlesql

import (
	"github.com/bytebase/omni/googlesql/analysis"
	"github.com/bytebase/omni/googlesql/ast"
	"github.com/bytebase/omni/googlesql/parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ClassifyStatement classifies a single GoogleSQL statement via the omni
// analysis classifier and maps the result onto bytebase's base.QueryType.
// Config.SetStatementIsSelect re-applies the legacy spanner queryTypeListener's
// "treat SAFE SET as select" case (omni classifies SET as Unknown).
func ClassifyStatement(statement string, cfg Config) base.QueryType {
	qt := MapQueryType(analysis.ClassifySQL(statement, cfg.Dialect))
	if qt == base.QueryTypeUnknown && cfg.SetStatementIsSelect && IsSetStatement(statement) {
		return base.Select
	}
	return qt
}

// IsSetStatement reports whether the statement parses to a single SET statement.
func IsSetStatement(statement string) bool {
	file, errs := parser.Parse(statement)
	if len(errs) > 0 || file == nil || len(file.Stmts) != 1 {
		return false
	}
	_, ok := file.Stmts[0].(*ast.SetStmt)
	return ok
}

// MapQueryType maps an omni analysis.QueryType onto bytebase's base.QueryType.
// The two enums are defined to correspond 1:1 (omni's QueryType doc states the
// values mirror base.QueryType so this switch is total); an unexpected value
// falls back to QueryTypeUnknown.
func MapQueryType(t analysis.QueryType) base.QueryType {
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
