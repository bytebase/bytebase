package trino

import (
	"context"
	"unicode/utf8"

	"github.com/bytebase/omni/trino/catalog"
	"github.com/bytebase/omni/trino/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_TRINO, Completion)
}

// Completion returns auto-completion candidates for a Trino statement at the
// given caret position. It is backed by the omni Trino completer
// (github.com/bytebase/omni/trino/completion), which tokenizes the input up to
// the caret, classifies the completion context, and produces candidates from a
// three-level Trino catalog (catalog -> schema -> table/view -> column) plus a
// curated keyword set.
//
// The bytebase metadata (database -> schema -> table -> column) maps onto the
// Trino catalog model as catalog=database, so a Company.dbo.Employees object is
// reachable as the Trino name Company.dbo.Employees.
//
// Divergence from the legacy ANTLR (CodeCompletionCore) completer: the omni
// completer is intentionally NOT a c3 port (see the omni completion package
// doc). It emits {Text, Type} candidates only — it does not populate the
// Definition ("catalog.schema.table | type, NOT NULL") or Priority fields the
// legacy completer attached, and its candidate *set* in column contexts is the
// in-scope FROM columns (resolved via query-span analysis) rather than the
// legacy completer's "every column in the default schema, ranked by priority".
// Those fields are therefore left empty here.
func Completion(_ context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	cat := buildCompletionCatalog(cCtx)

	byteOffset := caretToByteOffset(statement, caretLine, caretOffset)
	cands := completion.Complete(statement, byteOffset, cat)

	result := make([]base.Candidate, 0, len(cands))
	for _, c := range cands {
		result = append(result, base.Candidate{
			Text: c.Text,
			Type: convertCandidateType(c.Type),
		})
	}
	return result, nil
}

// buildCompletionCatalog constructs an omni Trino catalog from the completion
// context's metadata. Each bytebase database becomes a Trino catalog; its
// schemas, tables/views and columns are copied in. The session's current
// catalog/schema are set from the context defaults so unqualified names resolve
// there. Returns nil when no metadata is available (omni Complete then offers
// only keywords and statement-derived CTE names).
func buildCompletionCatalog(cCtx base.CompletionContext) *catalog.Catalog {
	if cCtx.Metadata == nil || cCtx.ListDatabaseNames == nil {
		return nil
	}
	ctx := context.Background()
	names, err := cCtx.ListDatabaseNames(ctx, cCtx.InstanceID)
	if err != nil {
		return nil
	}

	cat := catalog.New()
	loaded := false
	for _, dbName := range names {
		_, meta, err := cCtx.Metadata(ctx, cCtx.InstanceID, dbName)
		if err != nil || meta == nil {
			continue
		}
		database := cat.EnsureCatalog(catalog.Normalize(dbName))
		for _, schemaName := range meta.ListSchemaNames() {
			schemaMeta := meta.GetSchemaMetadata(schemaName)
			if schemaMeta == nil {
				continue
			}
			sc := database.EnsureSchema(catalog.Normalize(schemaName))
			for _, tableName := range schemaMeta.ListTableNames() {
				tableMeta := schemaMeta.GetTable(tableName)
				if tableMeta == nil {
					continue
				}
				sc.AddTable(catalog.Normalize(tableName), columnsOf(tableMeta.GetProto().GetColumns())...)
			}
			for _, viewName := range schemaMeta.ListViewNames() {
				viewMeta := schemaMeta.GetView(viewName)
				if viewMeta == nil {
					continue
				}
				sc.AddView(catalog.Normalize(viewName), columnsOf(viewMeta.GetColumns())...)
			}
			loaded = true
		}
	}
	if !loaded {
		return nil
	}

	if cCtx.DefaultDatabase != "" {
		cat.SetCurrentCatalog(catalog.Normalize(cCtx.DefaultDatabase))
	}
	if cCtx.DefaultSchema != "" {
		cat.SetCurrentSchema(catalog.Normalize(cCtx.DefaultSchema))
	}
	return cat
}

// columnsOf converts storepb columns into omni catalog columns (names
// normalized).
func columnsOf(columns []*storepb.ColumnMetadata) []*catalog.Column {
	out := make([]*catalog.Column, 0, len(columns))
	for _, c := range columns {
		if c == nil {
			continue
		}
		out = append(out, catalog.NewColumn(catalog.Normalize(c.Name), c.Type, c.Nullable))
	}
	return out
}

// convertCandidateType maps an omni completion candidate type onto the bytebase
// base.CandidateType. A Trino catalog corresponds to a bytebase DATABASE.
func convertCandidateType(t completion.CandidateType) base.CandidateType {
	switch t {
	case completion.CandidateKeyword:
		return base.CandidateTypeKeyword
	case completion.CandidateCatalog:
		return base.CandidateTypeDatabase
	case completion.CandidateSchema:
		return base.CandidateTypeSchema
	case completion.CandidateTable:
		return base.CandidateTypeTable
	case completion.CandidateView:
		return base.CandidateTypeView
	case completion.CandidateColumn:
		return base.CandidateTypeColumn
	default:
		return base.CandidateTypeNone
	}
}

// caretToByteOffset converts a (1-based line, 0-based column) caret position
// into a byte offset into statement. The column is interpreted as a UTF-16
// code-unit offset (the monaco/LSP convention the caller uses), matching how
// the completion test derives the caret from a "|" marker.
func caretToByteOffset(statement string, caretLine, caretColumn int) int {
	line := 1
	col := 0 // UTF-16 code units consumed on the current line
	for i := 0; i < len(statement); {
		if line == caretLine && col >= caretColumn {
			return i
		}
		r, size := utf8.DecodeRuneInString(statement[i:])
		if r == '\n' {
			if line == caretLine {
				// Caret column is past the end of this line; clamp to line end.
				return i
			}
			line++
			col = 0
		} else if r <= 0xFFFF {
			col++
		} else {
			col += 2 // surrogate pair
		}
		i += size
	}
	return len(statement)
}
