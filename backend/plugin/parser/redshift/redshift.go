package redshift

import (
	"context"

	omniredshift "github.com/bytebase/omni/redshift"
	"github.com/bytebase/omni/redshift/analysis"
	"github.com/bytebase/omni/redshift/ast"
	redshiftcompletion "github.com/bytebase/omni/redshift/completion"
	omniredshiftparser "github.com/bytebase/omni/redshift/parser"
	pkgerrors "github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.Register(base.Engine{
		Split:           SplitSQL,
		Parse:           parseRedshiftStatements,
		StatementTypes:  GetStatementTypes,
		StatementRanges: GetStatementRanges,
		Diagnose:        Diagnose,
		Complete:        Completion,
		ValidateQuery:   ValidateSQLForEditor,
		QuerySpan:       GetQuerySpan,
		Explain:         explainStatement,
		ResultLimit:     statementWithResultLimit,
		ExtractChanged:  extractChangedResources,
	}, storepb.Engine_REDSHIFT)
}

// AST wraps an omni AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, omniredshift.Split(statement), func(s omniredshift.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, false), nil
}

func parseRedshiftStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		if stmt.Empty {
			return nil, nil
		}
		parsed, err := omniredshift.Parse(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		asts := make([]base.AST, 0, len(parsed))
		for _, parsedStmt := range parsed {
			if parsedStmt.Empty() {
				continue
			}
			asts = append(asts, &AST{
				Node:          parsedStmt.AST,
				Text:          stmt.Text,
				StartPosition: stmt.Start,
			})
		}
		return asts, nil
	})
}

func getStatementTypesWithPosition(asts []base.AST) ([]base.StatementTypeWithPosition, error) {
	return base.CollectSingleStatementTypesWithPosition(asts, "expected Redshift omni AST", false, func(a *AST) (storepb.StatementType, string, bool) {
		if a == nil || a.Node == nil {
			return 0, "", false
		}
		return base.OmniStatementType(analysis.StatementTypeOf(a.Node)), a.Text, true
	})
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := getStatementTypesWithPosition(asts)
	if err != nil {
		return nil, err
	}
	return base.ExtractTypesFromPositions(results), nil
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	rawRanges := analysis.StatementRanges(statement)
	if len(rawRanges) == 0 {
		return nil, nil
	}
	return base.MapByteRangesToLSP(statement, rawRanges), nil
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	var syntaxErr *base.SyntaxError
	if diag := analysis.Diagnose(statement); diag != nil {
		position := base.ByteOffsetToRunePosition(statement, diag.Offset)
		syntaxErr = &base.SyntaxError{
			Position:   position,
			Message:    base.FormatLineSyntaxError(position, diag.Message),
			RawMessage: diag.Message,
		}
	}
	return base.DiagnoseSyntaxError(statement, syntaxErr), nil
}

// Completion returns Redshift auto-completion candidates backed by omni's
// session-aware Redshift completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, _ := base.CompletionMetadataLoaders(cCtx)
	candidates := redshiftcompletion.CompleteSession(ctx, redshiftcompletion.SessionConfig{
		DefaultDatabase: cCtx.DefaultDatabase,
		DefaultSchema:   cCtx.DefaultSchema,
		LoadDatabase:    loadDB,
	}, statement, caretLine, caretOffset)

	return base.ConvertOmniCandidates(candidates, func(c redshiftcompletion.Candidate) (string, redshiftcompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidate), nil
}

// ValidateSQLForEditor validates the SQL statement for SQL editor.
func ValidateSQLForEditor(statement string) (bool, bool, error) {
	readOnly, returnsData, err := analysis.ValidateQuery(statement)
	if err != nil {
		return false, false, convertError(err, base.Statement{
			Text:  statement,
			Start: &storepb.Position{Line: 1, Column: 1},
		})
	}
	return readOnly, returnsData, nil
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, _ bool) (*base.QuerySpan, error) {
	meta, err := base.RequireDatabaseMetadata(ctx, gCtx, database)
	if err != nil {
		return nil, err
	}
	searchPath := meta.GetSearchPath()
	if schema != "" {
		searchPath = []string{schema}
	}

	resolved, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		SearchPath:          searchPath,
		GetDatabaseMetadata: base.QuerySpanProtoMetadataFunc(ctx, gCtx, false),
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		err = convertError(err, stmt)
		return nil, pkgerrors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	span := base.OmniQuerySpan(resolved.Type, resolved.SourceColumns, resolved.PredicateColumns, resolved.Results, resolved.NotFoundError)
	span.FunctionNotSupportedError = base.OmniFunctionNotSupportedError(resolved.FunctionNotSupportedError)
	return span, nil
}

// explainStatement implements base.ExplainStatement. Redshift EXPLAIN has no
// machine-readable format, so every request gets the engine's default plan.
func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	explained, err := analysis.ExplainStatement(statement)
	return base.ExplainStatementResult(explained, err, nil)
}

func statementWithResultLimit(statement string, limit int, _ string) string {
	return analysis.StatementWithResultLimit(statement, limit)
}

func extractChangedResources(database string, schema string, dbMetadata *model.DatabaseMetadata, _ []base.AST, statement string) (*base.ChangeSummary, error) {
	changes, err := analysis.ExtractChangedResources(statement, database, schema)
	if err != nil {
		return nil, err
	}
	summary := &base.ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
	}
	base.AppendStatementChanges(summary, *changes, false)
	return summary, nil
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *omniredshiftparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniredshiftparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineSyntaxError(pos, pe.Message)
		},
	)
}
