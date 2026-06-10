package bigquery

import (
	"github.com/bytebase/omni/googlesql/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// classifyStatement classifies a single BigQuery (GoogleSQL) statement via the
// omni analysis classifier and maps the result onto bytebase's base.QueryType.
//
// omni's analysis.ClassifySQL reproduces the legacy queryTypeListener's rules
// for the BigQuery dialect:
//   - a query_statement is Select, promoted to SelectInfoSchema when it reads
//     exclusively from INFORMATION_SCHEMA (the legacy allSystems case);
//   - INSERT/UPDATE/DELETE/MERGE/TRUNCATE (and, in the legacy listener, CALL) are
//     DML;
//   - every CREATE/ALTER/DROP/GRANT/REVOKE form is DDL;
//   - EXPLAIN is Explain;
//   - everything unrecognized is Unknown.
//
// The omni QueryType values map 1:1 to base.QueryType (see mapQueryType).
func classifyStatement(statement string) base.QueryType {
	return mapQueryType(analysis.ClassifySQL(statement, analysis.DialectBigQuery))
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
