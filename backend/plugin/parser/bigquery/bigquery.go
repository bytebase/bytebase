// Package bigquery registers the BigQuery engine handlers on omni's bigquery
// tree: the query-span extractor, the splitter, the diagnostics adapter, the
// query-type mapping and the result limit.
//
// Masking note: the query-span extractor is masking-critical (bytebase's
// masker applies per-output-column maskers positionally and fails OPEN, so
// under-attributed lineage returns sensitive columns unmasked). The
// differential corpus (test-data/query-span/standard.yaml, recorded from the
// legacy ANTLR resolver) and the leak-pin tests in this package pin it.
package bigquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/bigquery/analysis"
	"github.com/bytebase/omni/bigquery/ast"
	"github.com/bytebase/omni/bigquery/diagnostics"
	"github.com/bytebase/omni/bigquery/parser"
	"github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:          SplitSQL,
		Parse:          ParseStatements,
		StatementTypes: GetStatementTypes,
		Diagnose: func(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
			return Diagnose(statement), nil
		},
		ValidateQuery: analysis.ValidateQuery,
		QuerySpan:     GetQuerySpan,
		ResultLimit:   statementWithResultLimit,
	}, storepb.Engine_BIGQUERY)
}

// SplitSQL splits the input into multiple SQL statements using the omni
// GoogleSQL splitter (block-aware: BEGIN/END, IF, LOOP, CASE bodies stay
// whole), re-attaching trailing ';' delimiters when present.
//
// Each statement's Text starts at its own first token.
func SplitSQL(statement string) ([]base.Statement, error) {
	segs := parser.Split(statement)
	return base.BuildStatements(statement, segs, func(s parser.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, true), nil
}

// AST wraps one omni GoogleSQL statement node so it satisfies base.AST. Node is
// nil when omni could not parse the statement.
type AST = base.NodeAST[ast.Node]

// ParseStatements splits the input and parses each
// statement into an omni AST node.
//
// A statement omni cannot parse carries an AST with a nil Node rather than no
// AST so base.ExtractASTs preserves it for UNSPECIFIED classification.
func ParseStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		if file, err := parser.Parse(stmt.Text); err == nil && file != nil && len(file.Stmts) > 0 {
			asts := make([]base.AST, len(file.Stmts))
			for i, node := range file.Stmts {
				asts[i] = &AST{Node: node, StartPosition: stmt.Start}
			}
			return asts, nil
		}
		return []base.AST{&AST{StartPosition: stmt.Start}}, nil
	})
}

// GetStatementTypes maps each parsed statement onto the storepb
// statement type that approval rules and risk levels are written against.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	types := make([]storepb.StatementType, 0, len(asts))
	for _, a := range asts {
		node, ok := a.(*AST)
		if !ok || node == nil || node.Node == nil {
			types = append(types, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED)
			continue
		}
		types = append(types, base.OmniStatementType(analysis.StatementTypeOf(node.Node)))
	}
	return types, nil
}

// Diagnose returns syntax diagnostics for the given statement,
// surfacing lex/parse errors from the omni parser via the omni diagnostics
// analyzer. Mirrors the Trino cutover (#20517).
//
// omni's Analyze emits a "<X> statement parsing is not yet supported" stub
// diagnostic for the handful of valid-but-not-yet-modeled statement forms (e.g.
// IMPORT MODULE, EXPORT METADATA) — recognized statement-leading keywords whose
// body the omni parser doesn't build yet. The legacy ANTLR parser parsed them
// without error, so flagging them would be a false positive on valid SQL; only
// those stubs are suppressed and every genuine lex/parse error (including
// "unknown or unsupported statement …") passes through.
func Diagnose(statement string) []base.Diagnostic {
	diags := diagnostics.Analyze(statement)
	filtered := make([]diagnostics.Diagnostic, 0, len(diags))
	for _, d := range diags {
		if !isValidButUnimplementedStub(d.Message) {
			filtered = append(filtered, d)
		}
	}
	return base.ConvertOffsetDiagnostics(statement, filtered, func(d diagnostics.Diagnostic) (int, string) {
		return d.Range.Start.Offset, d.Message
	})
}

// isValidButUnimplementedStub reports whether msg is an omni "not yet supported"
// stub for a statement form that is valid SQL (so it must not be flagged
// as a syntax error). The genuine-unknown case ("unknown or unsupported
// statement starting with …") and all lexer errors are intentionally NOT matched
// here.
func isValidButUnimplementedStub(msg string) bool {
	return strings.HasSuffix(msg, "statement parsing is not yet supported")
}

// GetQuerySpan is the registered query-span handler.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, defaultDatabase, _ string, _ bool) (*base.QuerySpan, error) {
	getMeta := base.QuerySpanProtoMetadataFunc(ctx, gCtx, true)
	span, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: defaultDatabase,
		// The written dataset is the DATABASE (default fill-in, single
		// unnamed metadata schema), system queries keep their table-level
		// resources, SET classifies as Select (the legacy listener's "treat
		// SAFE SET as select"), and set-operation star-merge names are
		// upper-cased (the legacy BigQuery extractor's rendering).
		SetStatementIsSelect:   true,
		UppercaseStarMergeName: true,
		GetDatabaseMetadata: func(database string) (*metadata.DatabaseSchemaMetadata, error) {
			if getMeta == nil {
				return nil, &analysis.ResourceNotFoundError{Database: &database}
			}
			return getMeta(database)
		},
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		return nil, err
	}
	return base.OmniQuerySpan(span.Type, span.SourceColumns, nil, span.Results, span.NotFoundError), nil
}

// statementWithResultLimit implements base.ResultLimitFunc for BigQuery, which
// takes no engineVersion. The caller only applies this to a statement it has
// already confirmed is a SELECT, so this wraps unconditionally.
func statementWithResultLimit(statement string, limit int, _ string) string {
	limitPart := ""
	if limit > 0 {
		limitPart = fmt.Sprintf(" LIMIT %d", limit)
	}
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result%s;", base.TrimStatement(statement), limitPart)
}
