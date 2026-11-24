// Package trino provides a SQL parser adapter for Trino, backed by the omni
// hand-written Trino parser (github.com/bytebase/omni/trino/*).
package trino

import (
	"context"
	stderrors "errors"

	"github.com/bytebase/omni/trino/analysis"
	"github.com/bytebase/omni/trino/ast"
	"github.com/bytebase/omni/trino/completion"
	"github.com/bytebase/omni/trino/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:         SplitSQL,
		Parse:         parseTrinoStatements,
		Diagnose:      Diagnose,
		Complete:      Completion,
		ValidateQuery: validateQuery,
		QuerySpan:     GetQuerySpan,
		Explain:       explainStatement,
	}, storepb.Engine_TRINO)
}

type astNode = base.NodeAST[ast.Node]

// SplitSQL splits the input into multiple SQL statements using the omni Trino
// splitter, re-attaching trailing ';' delimiters when present.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, parser.Split(statement), func(s parser.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, true), nil
}

func parseTrinoStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseSegmentedStatements(statement, SplitSQL, parseSegment)
}

func parseTrinoSQL(statement string) ([]*astNode, error) {
	return base.ParseNonEmptySegments(statement, SplitSQL, parseSegment)
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
	return &astNode{Node: node, StartPosition: stmt.Start}, nil
}

// Diagnose returns syntax diagnostics for the given Trino statement,
// delegating parse and stub filtering to omni's analysis package.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.ConvertOffsetDiagnostics(statement, analysis.Diagnose(statement), func(d parser.Diagnostic) (int, string) {
		return d.Loc.Start, d.Msg
	}), nil
}

// Completion returns auto-completion candidates for a Trino statement at the
// given caret position, backed by omni's session-aware Trino completer.
// Bytebase metadata (database -> schema -> table -> column) maps onto the Trino
// catalog model as catalog=database.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadCatalog, listCatalogs := base.CompletionMetadataLoaders(cCtx)
	cands := completion.CompleteSession(ctx, completion.SessionConfig{
		DefaultCatalog:   cCtx.DefaultDatabase,
		DefaultSchema:    cCtx.DefaultSchema,
		LoadCatalog:      loadCatalog,
		ListCatalogNames: listCatalogs,
	}, statement, caretLine, caretOffset)

	result := make([]base.Candidate, 0, len(cands))
	for _, c := range cands {
		result = append(result, base.Candidate{
			Text: c.Text,
			Type: base.OmniCandidateType(c.Type),
		})
	}
	return result, nil
}

func validateQuery(statement string) (bool, bool, error) {
	readOnly, returnsData, err := analysis.ValidateQuery(statement)
	if err != nil {
		return false, false, convertParseError(statement, err, nil)
	}
	return readOnly, returnsData, nil
}

// GetQuerySpan gets the query span for a Trino statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	resolved, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		ListDatabases:       base.QuerySpanListDatabasesFunc(ctx, gCtx, false),
		GetDatabaseMetadata: base.QuerySpanProtoMetadataFunc(ctx, gCtx, true),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	span := base.OmniQuerySpan(resolved.Type, resolved.SourceColumns, resolved.PredicateColumns, resolved.Results, nil)
	if resolved.NotFoundError != nil {
		converted := base.OmniNotFoundError(resolved.NotFoundError)
		var nf *base.ResourceNotFoundError
		if stderrors.As(converted, &nf) {
			span.NotFoundError = nf
		} else {
			span.NotFoundError = &base.ResourceNotFoundError{Err: resolved.NotFoundError}
		}
	}
	return span, nil
}

// explainStatement implements base.ExplainStatement. Trino spells its structured
// plan (FORMAT JSON); any other format the caller names gets the engine's own
// text plan (the driver capability table refuses the rest).
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	explain, err := analysis.ExplainStatement(statement, format == base.ExplainFormatJSON)
	return base.ExplainStatementResult(explain, err, func(e error) error {
		return convertParseError(statement, e, nil)
	})
}

func convertParseError(statement string, err error, basePos *storepb.Position) error {
	return base.ConvertParseError(statement, err, basePos, func(pe *parser.ParseError) (int, string, string) {
		return pe.Position, pe.Error(), pe.Message
	})
}
