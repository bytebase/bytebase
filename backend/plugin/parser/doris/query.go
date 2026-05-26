package doris

import (
	"github.com/bytebase/omni/doris/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_STARROCKS, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_DORIS, validateQuery)
}

// validateQuery reports whether the given statement is a read-only query
// suitable for the SQL editor / data-query path.
//
// Each top-level statement in the input must classify as either a SELECT,
// a SELECT-from-info-schema (SHOW, DESCRIBE, EXPLAIN, HELP), or one of the
// EXPLAIN-on-DML cases that omni's Classify already maps to SelectInfoSchema.
// Anything else (DML without EXPLAIN, DDL, transaction control, etc.) is
// rejected.
//
// The (bool, bool, error) return shape matches the bytebase QueryValidator
// contract: (isReadOnly, isExplicitReadOnly, syntaxError).
func validateQuery(statement string) (bool, bool, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return false, false, err
	}
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}
		qt := analysis.Classify(stmt.Text)
		switch qt {
		case analysis.QueryTypeSelect, analysis.QueryTypeSelectInfoSchema:
			// Read-only — SELECT / SHOW / DESCRIBE / EXPLAIN / HELP and CTE
			// SELECT (WITH ... SELECT). EXPLAIN-on-DML is also classified
			// as SelectInfoSchema because EXPLAIN is the leading keyword.
		default:
			return false, false, nil
		}
	}
	return true, true, nil
}
