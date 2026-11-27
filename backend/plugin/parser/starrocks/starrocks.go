package starrocks

import (
	"github.com/bytebase/omni/starrocks/analysis"
	"github.com/bytebase/omni/starrocks/ast"
	starrockscompletion "github.com/bytebase/omni/starrocks/completion"
	"github.com/bytebase/omni/starrocks/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, starrockscompletion.Engine)
	engine.Parse = parseStarRocksStatements
	base.Register(engine, storepb.Engine_STARROCKS)
}

type astNode = base.NodeAST[ast.Node]

// parseStarRocksStatements is not omni's because it keeps a segment holding only
// whitespace the lexer does not skip, such as U+00A0, as a statement without an
// AST, which an omni ParsedStatement cannot express.
func parseStarRocksStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseSegmentedStatements(statement, func(script string) ([]base.Statement, error) {
		return base.SplitMultiSQL(storepb.Engine_STARROCKS, script)
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

func convertParseError(statement string, err error, basePos *storepb.Position) error {
	return base.ConvertParseError(statement, err, basePos, func(pe *parser.ParseError) (int, string, string) {
		return pe.Position, pe.Error(), pe.Message
	})
}
