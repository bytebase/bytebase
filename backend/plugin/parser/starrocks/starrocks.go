package starrocks

import (
	"context"

	"github.com/bytebase/omni/starrocks/analysis"
	"github.com/bytebase/omni/starrocks/ast"
	"github.com/bytebase/omni/starrocks/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	engine := base.OmniEngine(analysis.Engine)
	engine.Parse = parseStarRocksStatements
	engine.Complete = Completion
	engine.QuerySpan = GetQuerySpan
	engine.ExtractChanged = extractChangedResources
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

func GetQuerySpan(
	_ context.Context,
	_ base.GetQuerySpanContext,
	stmt base.Statement,
	database, _ string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	resolved, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		IgnoreCaseSensitive: ignoreCaseSensitive,
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		return nil, convertParseError(stmt.Text, err, stmt.Start)
	}

	return base.OmniTableQuerySpan(resolved.Type, resolved.SourceColumns), nil
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractTypedChangedResources(dbMetadata, asts, "expected astNode for StarRocks", true, func(parsed *astNode) (analysis.StatementChanges, bool) {
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
