package tidb

import (
	"context"

	"github.com/bytebase/omni/tidb/analysis"
	"github.com/bytebase/omni/tidb/ast"
	omnicompletion "github.com/bytebase/omni/tidb/completion"
	omniparser "github.com/bytebase/omni/tidb/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.Register(base.Engine{
		Split:                SplitSQL,
		Parse:                parseTiDBStatements,
		StatementTypes:       GetStatementTypes,
		StatementRanges:      GetStatementRanges,
		Diagnose:             Diagnose,
		Complete:             Completion,
		ValidateQuery:        validateQuery,
		QuerySpan:            GetQuerySpan,
		Explain:              explainStatement,
		ResultLimit:          statementWithResultLimit,
		ExtractChanged:       extractChangedResources,
		TransformDMLToSelect: TransformDMLToSelect,
		GenerateRestoreSQL:   GenerateRestoreSQL,
	}, storepb.Engine_TIDB)
}

// AST wraps an omni/tidb AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, omniparser.Split(statement), func(s omniparser.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, true), nil
}

func parseTiDBStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseMultiNodeStatements(statement, func(s string) ([]base.Statement, error) {
		return base.SplitMultiSQL(storepb.Engine_TIDB, s)
	}, func(stmt base.Statement) ([]base.AST, error) {
		if stmt.Empty {
			return nil, nil
		}
		list, err := omniparser.Parse(stmt.Text)
		if err != nil {
			return nil, convertParseError(err, stmt)
		}
		if list == nil || len(list.Items) == 0 {
			return nil, nil
		}
		asts := make([]base.AST, len(list.Items))
		for i, node := range list.Items {
			asts[i] = &AST{
				Node:          node,
				Text:          stmt.Text,
				StartPosition: stmt.Start,
			}
		}
		return asts, nil
	})
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected TiDB AST", analysis.StatementTypeOf)
}

// GetStatementRanges returns LSP-compatible ranges (0-based line, UTF-16 character)
// for each top-level SQL statement in statement. The ranges come from a lexical
// split, so the editor still gets them while the user is mid-edit.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	return base.MapByteRangesToLSP(statement, analysis.StatementRanges(statement)), nil
}

// Diagnose returns the script's first syntax error as a diagnostic.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	syntaxError, err := analysis.Diagnose(statement)
	if err != nil {
		return nil, err
	}
	if syntaxError == nil {
		return base.DiagnoseSyntaxError(statement, nil), nil
	}
	return base.DiagnoseSyntaxError(statement, toSyntaxError(statement, syntaxError)), nil
}

// toSyntaxError places omni's error the way convertParseError places a parse
// error from the split statement that holds it.
func toSyntaxError(statement string, e *analysis.SyntaxError) *base.SyntaxError {
	stmt := base.NewStatementFromRange(statement, base.NewByteOffsetPositionMapper(statement), e.ByteStart, e.ByteEnd, false)
	pos := base.ByteOffsetToRunePosition(stmt.Text, e.Offset)
	if stmt.Start != nil {
		pos.Line += stmt.Start.Line - 1
	}
	return &base.SyntaxError{
		Position:   pos,
		Message:    base.FormatLineColSyntaxError(pos, e.Message, e.RelatedText),
		RawMessage: e.Message,
	}
}

// Completion provides auto-complete candidates for TiDB statements using the
// omni TiDB completion engine.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, listDBs := base.MySQLFamilyDatabaseLoaders(cCtx)
	raw := omnicompletion.CompleteSession(ctx, omnicompletion.SessionConfig{
		ReadOnly:          cCtx.Scene == base.SceneTypeQuery,
		DefaultDatabase:   cCtx.DefaultDatabase,
		LoadDatabase:      loadDB,
		ListDatabaseNames: listDBs,
	}, statement, caretLine, caretOffset)
	return base.ConvertOmniCandidates(raw, func(c omnicompletion.Candidate) (string, omnicompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidate), nil
}

func validateQuery(statement string) (bool, bool, error) {
	stmts, err := base.ParseStatements(storepb.Engine_TIDB, statement)
	if err != nil {
		return false, false, err
	}
	var nodes []ast.Node
	for _, stmt := range stmts {
		node, ok := GetNode(stmt.AST)
		if !ok {
			// An empty or comment-only statement carries no AST.
			continue
		}
		nodes = append(nodes, node)
	}
	readOnly, returnsData := analysis.ValidateQuery(nodes)
	return readOnly, returnsData, nil
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	_, getDB := base.UnnamedSchemaQuerySpanCatalog(ctx, gCtx, wrapDatabase)
	querySpan, err := base.OmniResolvedSpanResult(analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: database,
		Catalog:         analysis.Catalog{GetDatabase: getDB},
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

// explainStatement implements base.ExplainStatement. Only the JSON format has
// a reader downstream; any other format gets the engine's default plan.
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	explain, err := analysis.ExplainStatement(statement, format == base.ExplainFormatJSON)
	return base.ExplainStatementResult(explain, err, func(e error) error {
		return convertParseError(e, base.Statement{Text: statement})
	})
}

// DescribeExplain returns the output format of an EXPLAIN statement and whether
// it executes the statement it plans.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	return analysis.DescribeExplain(statement)
}

// statementWithResultLimit implements base.ResultLimitFunc for TiDB, which
// takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "WITH result AS (%s) SELECT * FROM result LIMIT %d;")
}

func extractChangedResources(database string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractTypedChangedResources(dbMetadata, asts, "expected TiDB AST", false, func(wrapped *AST) (analysis.StatementChanges, bool) {
		return analysis.ExtractChangedResources(wrapped.Node, database, wrapped.Text), true
	})
}

func convertParseError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, false,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, pe.RelatedText)
		},
	)
}
