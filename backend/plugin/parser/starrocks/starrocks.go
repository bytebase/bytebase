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
	base.Register(base.Engine{
		Split:           SplitSQL,
		Parse:           parseStarRocksStatements,
		StatementRanges: GetStatementRanges,
		Diagnose:        Diagnose,
		ValidateQuery:   validateQuery,
		QuerySpan:       GetQuerySpan,
		Explain:         explainStatement,
		ResultLimit:     statementWithResultLimit,
		ExtractChanged:  extractChangedResources,
	}, storepb.Engine_STARROCKS)
}

type astNode = base.NodeAST[ast.Node]

// SplitSQL splits the input into multiple SQL statements using omni's analysis
// package, then attaches 1-based line and rune column positions.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.OmniStatements(statement, analysis.SplitStatements(statement)), nil
}

func parseStarRocksStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseSegmentedStatements(statement, SplitSQL, parseSegment)
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

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	rawRanges := analysis.StatementRanges(statement)
	if len(rawRanges) == 0 {
		return nil, nil
	}
	return base.MapByteRangesToLSP(statement, rawRanges), nil
}

// Diagnose returns syntax diagnostics for the given StarRocks statement,
// delegating parse and stub filtering to omni's analysis package.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.ConvertOffsetDiagnostics(statement, analysis.Diagnose(statement), func(d analysis.SyntaxError) (int, string) {
		return d.Offset, d.Message
	}), nil
}

func validateQuery(statement string) (bool, bool, error) {
	readOnly, returnsData, err := analysis.ValidateQuery(statement)
	if err != nil {
		return false, false, convertParseError(statement, err, nil)
	}
	return readOnly, returnsData, nil
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

func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	planned, err := analysis.ExplainStatement(statement)
	if err != nil {
		return "", convertParseError(statement, err, nil)
	}
	return planned, nil
}

func statementWithResultLimit(statement string, limit int, _ string) string {
	return analysis.RewriteResultLimit(statement, limit)
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
