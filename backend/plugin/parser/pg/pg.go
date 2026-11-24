package pg

import (
	"context"

	lsp "github.com/bytebase/lsp-protocol"
	omnipg "github.com/bytebase/omni/pg"
	"github.com/bytebase/omni/pg/analysis"
	"github.com/bytebase/omni/pg/ast"
	pgcompletion "github.com/bytebase/omni/pg/completion"
	omniparser "github.com/bytebase/omni/pg/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:                SplitSQL,
		Parse:                parsePgStatements,
		StatementTypes:       getStatementTypesForRegistry,
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
	}, storepb.Engine_POSTGRES)
	// CockroachDB shares PostgreSQL grammar for split, diagnose, completion,
	// query validation, query span, explain, and result limit; parser/cockroachdb
	// registers its own Parse, StatementTypes, and ExtractChanged.
	base.Register(base.Engine{
		Split:         SplitSQL,
		Diagnose:      Diagnose,
		Complete:      Completion,
		ValidateQuery: validateQuery,
		QuerySpan:     GetQuerySpan,
		Explain:       cockroachExplainStatement,
		ResultLimit:   statementWithResultLimit,
	}, storepb.Engine_COCKROACHDB)
	// Snowflake reuses PostgreSQL completion.
	base.Register(base.Engine{Complete: Completion}, storepb.Engine_SNOWFLAKE)
}

// AST wraps an omni AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// It uses omni's lexical splitter which handles quotes, dollar-quotes,
// comments, and BEGIN ATOMIC blocks without requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, omnipg.Split(statement), func(s omnipg.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, false), nil
}

func parsePgStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		parsed, err := omnipg.Parse(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		asts := make([]base.AST, len(parsed))
		for i, os := range parsed {
			asts[i] = &AST{Node: os.AST, Text: stmt.Text, StartPosition: stmt.Start}
		}
		return asts, nil
	})
}

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition

// GetStatementTypes returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypes(asts []base.AST) ([]StatementTypeWithPosition, error) {
	return base.CollectStatementTypesWithPosition(asts, func(unifiedAST base.AST) ([]storepb.StatementType, string, bool, error) {
		wrapped, ok := unifiedAST.(*AST)
		if !ok {
			return nil, "", false, errors.New("expected AST for PostgreSQL")
		}
		if wrapped.Node == nil {
			return nil, "", false, nil
		}
		return base.OmniStatementTypes(analysis.StatementTypes(wrapped.Node)), wrapped.Text, true, nil
	})
}

func getStatementTypesForRegistry(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := GetStatementTypes(asts)
	if err != nil {
		return nil, err
	}
	return base.ExtractTypesFromPositions(results), nil
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	var ranges []base.Range
	for _, r := range analysis.StatementRanges(statement) {
		ranges = append(ranges, lsp.Range{
			Start: lsp.Position{Line: r.Start.Line, Character: r.Start.Character},
			End:   lsp.Position{Line: r.End.Line, Character: r.End.Character},
		})
	}
	return ranges, nil
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.DiagnoseSyntaxError(statement, convertDiagnoseError(analysis.Diagnose(statement))), nil
}

// convertDiagnoseError positions the error in the text omni parsed.
func convertDiagnoseError(err *analysis.DiagnoseError) *base.SyntaxError {
	if err == nil {
		return nil
	}
	if !err.ParseError {
		return base.SyntaxErrorAtOffset(err.Text, -1, err.Message)
	}
	return base.SyntaxErrorAtOffset(err.Text, err.Offset, err.Message)
}

// Completion returns PostgreSQL auto-completion candidates backed by omni's
// session-aware PostgreSQL completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, _ := base.CompletionMetadataLoaders(cCtx)
	candidates := pgcompletion.CompleteSession(ctx, pgcompletion.SessionConfig{
		DefaultDatabase: cCtx.DefaultDatabase,
		DefaultSchema:   cCtx.DefaultSchema,
		LoadDatabase:    loadDB,
	}, statement, caretLine, caretOffset)

	return base.ConvertOmniCandidates(candidates, func(c pgcompletion.Candidate) (string, pgcompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidateByTypeText), nil
}

// validateQuery reports whether the statement may run in the SQL editor's
// read-only mode and whether every statement returns data.
func validateQuery(statement string) (bool, bool, error) {
	stmts, err := omnipg.Parse(statement)
	if err != nil {
		return false, false, err
	}
	nodes := make([]ast.Node, 0, len(stmts))
	for _, stmt := range stmts {
		nodes = append(nodes, stmt.AST)
	}
	readOnly, returnsData := analysis.ValidateQuery(nodes)
	return readOnly, returnsData, nil
}

func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	var formatName string
	switch format {
	case base.ExplainFormatJSON:
		formatName = "json"
	case base.ExplainFormatXML:
		formatName = "xml"
	case base.ExplainFormatYAML:
		formatName = "yaml"
	default:
	}
	explain, err := analysis.ExplainStatement(statement, formatName)
	return base.ExplainStatementResult(explain, err, nil)
}

func cockroachExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	explain, err := analysis.CockroachExplainStatement(statement, format == base.ExplainFormatJSON)
	return base.ExplainStatementResult(explain, err, nil)
}

// DescribeExplain returns the output format of an EXPLAIN statement and
// whether it executes the statement it plans.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	return analysis.DescribeExplain(statement)
}

func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "WITH result AS (\n%s\n) SELECT * FROM result LIMIT %d;")
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, false,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineSyntaxError(pos, pe.Message)
		},
	)
}
