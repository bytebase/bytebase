package trino

import (
	"strings"

	"github.com/bytebase/omni/trino/analysis"
	"github.com/bytebase/omni/trino/ast"
	"github.com/bytebase/omni/trino/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TRINO, validateQuery)
}

func validateQuery(statement string) (bool, bool, error) {
	readOnly, returnsData, err := analysis.ValidateQuery(statement)
	if err != nil {
		return false, false, convertParseError(statement, err, nil)
	}
	if !readOnly {
		return false, returnsData, nil
	}
	// Work around omni classifying WithFunctionStmt as Select in ValidateQuery while
	// spanQueryType classifies it as DDL and omni's Trino driver rowCap leaves it uncapped.
	if containsASCIIFold(statement, "function") {
		parsed, err := parseTrinoSQL(statement)
		if err != nil {
			return false, false, err
		}
		for _, p := range parsed {
			if hasWithFunction(p.Node()) {
				return false, false, nil
			}
		}
	}
	return readOnly, returnsData, nil
}

func hasWithFunction(node ast.Node) bool {
	switch n := node.(type) {
	case *parser.WithFunctionStmt:
		return true
	case *parser.ExplainStmt:
		return n.Statement != nil && hasWithFunction(n.Statement)
	default:
		return false
	}
}

// containsASCIIFold reports whether s contains substr under ASCII case-folding without allocating.
func containsASCIIFold(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if strings.EqualFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}
