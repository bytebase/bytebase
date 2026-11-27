package doris

import (
	"github.com/bytebase/omni/doris/analysis"
	"github.com/bytebase/omni/doris/ast"
	doriscompletion "github.com/bytebase/omni/doris/completion"
	"github.com/bytebase/omni/doris/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, doriscompletion.Engine)
	engine.Parse = parseDorisStatements
	engine.ExtractChanged = extractChangedResources
	base.Register(engine, storepb.Engine_DORIS)
}

type astNode = base.NodeAST[ast.Node]

// parseDorisStatements is not omni's because it keeps a segment holding only
// whitespace the lexer does not skip, such as U+00A0, as a statement without an
// AST, which an omni ParsedStatement cannot express.
func parseDorisStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseSegmentedStatements(statement, func(script string) ([]base.Statement, error) {
		return base.SplitMultiSQL(storepb.Engine_DORIS, script)
	}, parseSegment)
}

func parseSegment(stmt base.Statement) (*astNode, error) {
	// Parse each segment on its own so ParseError byte offsets align with stmt.Text
	// before convertParseError shifts them into script coordinates.
	file, err := parser.Parse(stmt.Text)
	if err != nil {
		return nil, convertParseError(stmt.Text, err, stmt.Start)
	}
	var node ast.Node
	if file != nil && len(file.Stmts) > 0 {
		node = file.Stmts[0]
	}
	return &astNode{Node: node, Text: stmt.Text, StartPosition: stmt.Start}, nil
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractTypedChangedResources(dbMetadata, asts, "expected astNode for Doris", true, func(parsed *astNode) (analysis.StatementChanges, bool) {
		if parsed.Node == nil {
			return analysis.StatementChanges{}, false
		}
		return analysis.ExtractChangedResources(parsed.Node, currentDatabase, parsed.Text), true
	})
}

func convertParseError(statement string, err error, basePos *storepb.Position) error {
	return base.ConvertParseError(statement, err, basePos, func(pe *parser.ParseError) (int, string, string) {
		return pe.Position, pe.Error(), pe.Message
	})
}
