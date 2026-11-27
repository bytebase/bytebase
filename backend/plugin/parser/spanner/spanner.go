// Package spanner registers omni's Spanner analysis engine with the query-span
// extractor and the statement-type mapping.
//
// Masking note: the query-span extractor is masking-critical (bytebase's
// masker applies per-output-column maskers positionally and fails OPEN, so
// under-attributed lineage returns sensitive columns unmasked). The
// differential corpus (test-data/query-span/standard.yaml, recorded from the
// legacy ANTLR resolver) and the leak-pin tests in this package pin it.
package spanner

import (
	"github.com/bytebase/omni/spanner/analysis"
	"github.com/bytebase/omni/spanner/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine)
	engine.StatementTypes = GetStatementTypes
	base.Register(engine, storepb.Engine_SPANNER)
}

// GetStatementTypes maps each parsed statement onto the storepb
// statement type that approval rules and risk levels are written against. A
// statement omni cannot parse or classify is UNSPECIFIED rather than dropped, so
// base.IsAllDML does not read it as DML.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	types := make([]storepb.StatementType, 0, len(asts))
	for _, a := range asts {
		node, ok := base.ExtractNode[ast.Node](a)
		if !ok || node == nil {
			types = append(types, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED)
			continue
		}
		types = append(types, base.OmniStatementType(analysis.StatementTypeOf(node)))
	}
	return types, nil
}
