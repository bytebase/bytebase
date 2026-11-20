package starrocks

import (
	"github.com/bytebase/omni/starrocks/analysis"
	"github.com/bytebase/omni/starrocks/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_STARROCKS, validateQuery)
}

func validateQuery(statement string) (bool, bool, error) {
	readOnly, returnsData, err := analysis.ValidateQuery(statement)
	if err != nil {
		return false, false, convertParseError(statement, err, nil)
	}
	if !readOnly {
		return false, returnsData, nil
	}
	// Work around omni/starrocks/analysis.ValidateQuery accepting EXPLAIN over
	// queries with INTO OUTFILE so EXPLAIN cannot bypass the export gate.
	parsed, err := parseStarRocksSQL(statement)
	if err != nil {
		return false, false, err
	}
	for _, p := range parsed {
		if exp, ok := p.Node().(*ast.ExplainStmt); ok && writesOutfile(exp.Query) {
			return false, false, nil
		}
	}
	return readOnly, returnsData, nil
}

func writesOutfile(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.SelectStmt:
		return n.Into != nil
	case *ast.SetOpStmt:
		return n.Into != nil || writesOutfile(n.Left) || writesOutfile(n.Right)
	case *ast.ParenSelect:
		return n.Into != nil || writesOutfile(n.Sel)
	default:
		return false
	}
}
