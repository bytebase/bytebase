// Package googlesql holds the omni-backed implementation shared by the two
// GoogleSQL engine plugins (bigquery and spanner): the query-span extractor,
// the splitter, the diagnostics adapter, and the query-type mapping. One
// omni grammar serves both dialects; the per-engine packages register the
// base.* handlers with a Config carrying the few dialect deltas.
//
// Masking note: the query-span extractor is masking-critical (bytebase's
// masker applies per-output-column maskers positionally and fails OPEN, so
// under-attributed lineage returns sensitive columns unmasked). Keeping ONE
// implementation here — rather than a copy per engine — is deliberate: the
// behavior was validated against legacy-resolver differential corpora plus
// leak-pin tests in both engine packages, and a single code path cannot drift
// between engines.
package googlesql

import (
	"context"
	"strings"

	"github.com/bytebase/omni/googlesql/analysis"
	"github.com/bytebase/omni/googlesql/ast"
	"github.com/bytebase/omni/googlesql/diagnostics"
	"github.com/bytebase/omni/googlesql/parser"
	"github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Config carries the dialect deltas between the BigQuery and Spanner plugins.
// Every flag mirrors a documented legacy-extractor behavior difference; see the
// per-engine packages for the values and the corpus/leak-pin tests pinning them.
type Config struct {
	// Dialect selects omni's per-dialect analysis behavior (system schemas,
	// name-part bucketing, default join type).
	Dialect analysis.Dialect

	// IgnoreWrittenDatabase pins every resource's Database to the default
	// database. Spanner: the legacy resolver silently ignored the db part of a
	// 3-part db.schema.table and recorded the session database. BigQuery keeps
	// the written dataset (Database) with default fill-in.
	IgnoreWrittenDatabase bool

	// SchemaScopedMetadata looks tables up under the metadata schema named by
	// the reference (case-SENSITIVE, "" = default schema) within the default
	// database — the Spanner named-schema model. BigQuery instead resolves the
	// written dataset as the DATABASE and uses its single unnamed schema.
	SchemaScopedMetadata bool

	// SystemOnlyEmptySpan early-returns an EMPTY SelectInfoSchema span (no
	// source columns, no results, no metadata access) for a query reading ONLY
	// system tables — the legacy spanner extractor's exact behavior. BigQuery
	// instead keeps the system tables as table-level resources.
	SystemOnlyEmptySpan bool

	// SetStatementIsSelect classifies a SET statement as base.Select — BOTH
	// legacy queryTypeListeners' "treat SAFE SET as select" case (omni
	// classifies SET as Unknown, which the new-ACL access check would reject as
	// a disallowed query type).
	SetStatementIsSelect bool

	// UppercaseStarMergeName upper-cases a set-operation star-merge output name
	// (the legacy BigQuery extractor's rendering). The legacy spanner extractor
	// passed a LEFT-star-derived name through in the star column's metadata
	// case instead.
	UppercaseStarMergeName bool

	// CanonicalBaseFieldName renders a base-table FIELD passthrough (the JOIN
	// ... USING coalesced key, marked BaseFieldName by omni) in the field's
	// METADATA case — the legacy spanner resolver named it after the left
	// PhysicalTable's field, not the written USING token.
	CanonicalBaseFieldName bool

	// ContiguousSplitText makes each split statement's Text run contiguously
	// from the END of the previous statement (inter-statement whitespace and
	// comments attach to the FOLLOWING statement) — the legacy spanner
	// parse-tree splitter's convention. BigQuery starts each statement's Text
	// at its own first token.
	ContiguousSplitText bool
}

// Handlers bundles the registered engine handlers Register returns, so the
// engine package can re-export each with its own declaration for its tests
// (the differential corpus and the leak pins drive the exact functions
// production uses).
type Handlers struct {
	SplitSQL     base.SplitMultiSQLFunc
	GetQuerySpan base.GetQuerySpanFunc
}

// Register registers the splitter, query-validator, diagnose, query-span,
// statement-parser, and statement-type handlers for one GoogleSQL engine with
// the given dialect configuration.
func Register(engine storepb.Engine, cfg Config) Handlers {
	h := Handlers{
		SplitSQL: func(statement string) ([]base.Statement, error) {
			return SplitSQL(statement, cfg)
		},
		GetQuerySpan: func(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
			return getQuerySpan(ctx, cfg, database, gCtx, stmt.Text)
		},
	}
	base.Register(base.Engine{
		Split: h.SplitSQL,
		Parse: func(statement string) ([]base.ParsedStatement, error) {
			return ParseStatements(statement, cfg)
		},
		StatementTypes: GetStatementTypes,
		Diagnose: func(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
			return Diagnose(statement), nil
		},
		ValidateQuery: func(statement string) (bool, bool, error) {
			return analysis.ValidateQuery(statement, cfg.Dialect)
		},
		QuerySpan: h.GetQuerySpan,
	}, engine)
	return h
}

// SplitSQL splits the input into multiple SQL statements using the omni
// GoogleSQL splitter (block-aware: BEGIN/END, IF, LOOP, CASE bodies stay
// whole), re-attaching trailing ';' delimiters when present.
//
// When cfg.ContiguousSplitText is true (Spanner), each statement's Text runs
// contiguously from the end of the previous statement so inter-statement
// whitespace and comments attach to the following statement.
func SplitSQL(statement string, cfg Config) ([]base.Statement, error) {
	segs := parser.Split(statement)
	if !cfg.ContiguousSplitText {
		return base.BuildStatements(statement, segs, func(s parser.Segment) (int, int, bool) {
			return s.ByteStart, s.ByteEnd, s.Empty()
		}, true), nil
	}
	prevEnd := 0
	return base.BuildStatements(statement, segs, func(s parser.Segment) (int, int, bool) {
		start := prevEnd
		end := s.ByteEnd
		if end < len(statement) && statement[end] == ';' {
			end++
		}
		prevEnd = end
		return start, end, s.Empty()
	}, false), nil
}

// AST wraps one omni GoogleSQL statement node so it satisfies base.AST. Node is
// nil when omni could not parse the statement.
type AST = base.NodeAST[ast.Node]

// ParseStatements splits the input with the dialect's splitter and parses each
// statement into an omni AST node.
//
// A statement omni cannot parse carries an AST with a nil Node rather than no
// AST so base.ExtractASTs preserves it for UNSPECIFIED classification.
func ParseStatements(statement string, cfg Config) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, func(s string) ([]base.Statement, error) {
		return SplitSQL(s, cfg)
	}, func(stmt base.Statement) ([]base.AST, error) {
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

// GetStatementTypes maps each parsed GoogleSQL statement onto the storepb
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

// Diagnose returns syntax diagnostics for the given GoogleSQL statement,
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
// stub for a statement form that is valid GoogleSQL (so it must not be flagged
// as a syntax error). The genuine-unknown case ("unknown or unsupported
// statement starting with …") and all lexer errors are intentionally NOT matched
// here.
func isValidButUnimplementedStub(msg string) bool {
	return strings.HasSuffix(msg, "statement parsing is not yet supported")
}

func getQuerySpan(ctx context.Context, cfg Config, defaultDatabase string, gCtx base.GetQuerySpanContext, statement string) (*base.QuerySpan, error) {
	getMeta := base.QuerySpanProtoMetadataFunc(ctx, gCtx, true)
	span, err := analysis.ResolveQuerySpan(statement, analysis.ResolveOptions{
		Dialect:                cfg.Dialect,
		DefaultDatabase:        defaultDatabase,
		IgnoreWrittenDatabase:  cfg.IgnoreWrittenDatabase,
		SchemaScopedMetadata:   cfg.SchemaScopedMetadata,
		SystemOnlyEmptySpan:    cfg.SystemOnlyEmptySpan,
		SetStatementIsSelect:   cfg.SetStatementIsSelect,
		UppercaseStarMergeName: cfg.UppercaseStarMergeName,
		CanonicalBaseFieldName: cfg.CanonicalBaseFieldName,
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
