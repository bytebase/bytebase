package plsql

import (
	oracleast "github.com/bytebase/omni/oracle/ast"
	oracleparser "github.com/bytebase/omni/oracle/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_ORACLE, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
func validateQuery(statement string) (bool, bool, error) {
	for _, segment := range oracleparser.Split(statement) {
		if segment.Kind == oracleparser.SegmentSQLPlusCommand && !segment.Empty() {
			return false, false, nil
		}
	}

	list, err := ParsePLSQLOmni(statement)
	if err != nil {
		return false, false, convertOmniError(err, base.Statement{Text: statement})
	}
	if list == nil || len(list.Items) == 0 {
		return false, false, nil
	}

	for _, item := range list.Items {
		raw, ok := item.(*oracleast.RawStmt)
		if !ok || raw.Stmt == nil {
			return false, false, nil
		}
		switch raw.Stmt.(type) {
		case *oracleast.SelectStmt, *oracleast.ExplainPlanStmt:
		default:
			return false, false, nil
		}
	}

	return true, true, nil
}
