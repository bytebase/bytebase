package trino

import (
	"context"
	"strings"

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
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	cat := buildCompletionCatalog(ctx, cCtx, statement)

	byteOffset := base.CaretToByteOffset(statement, caretLine, caretOffset)
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
// there. Returns nil when no catalogs are available (omni Complete then offers
// only keywords and statement-derived CTE names).
//
// Every catalog name is registered cheaply so catalog-level completion can list
// them all, but full schema/table/column metadata is loaded only for catalogs
// the current statement actually needs (the default catalog and any catalog
// named in the statement). Trino instances routinely federate many catalogs and
// the LSP invokes completion on every keystroke, so eagerly loading all of them
// would fan out into hundreds of metadata fetches and stall completion.
func buildCompletionCatalog(ctx context.Context, cCtx base.CompletionContext, statement string) *catalog.Catalog {
	if cCtx.Metadata == nil || cCtx.ListDatabaseNames == nil {
		return nil
	}
	names, err := cCtx.ListDatabaseNames(ctx, cCtx.InstanceID)
	if err != nil || len(names) == 0 {
		return nil
	}

	lowerStmt := strings.ToLower(statement)
	defaultDB := catalog.Normalize(cCtx.DefaultDatabase)

	cat := catalog.New()
	for _, dbName := range names {
		norm := catalog.Normalize(dbName)
		// Register the catalog name (cheap) so catalog-level completion lists it.
		cat.EnsureCatalog(norm)
		if !catalogNeeded(norm, defaultDB, lowerStmt) {
			continue
		}
		_, meta, err := cCtx.Metadata(ctx, cCtx.InstanceID, dbName)
		if err != nil || meta == nil {
			continue
		}
		cat.LoadMetadata(meta.GetProto())
	}

	if cCtx.DefaultDatabase != "" {
		cat.SetCurrentCatalog(catalog.Normalize(cCtx.DefaultDatabase))
	}
	if cCtx.DefaultSchema != "" {
		cat.SetCurrentSchema(catalog.Normalize(cCtx.DefaultSchema))
	}
	return cat
}

// catalogNeeded reports whether the current completion statement needs this
// catalog's full schema/table/column metadata loaded: true for the session
// default catalog and for any catalog whose name appears in the statement text
// (a qualified reference such as catalog.schema.table). The reference check is a
// case-insensitive substring, which errs toward loading an occasional extra
// catalog rather than missing a referenced one.
func catalogNeeded(normName, defaultDB, lowerStmt string) bool {
	if normName == "" {
		return false
	}
	if defaultDB != "" && normName == defaultDB {
		return true
	}
	// A catalog name containing a double quote is rewritten inside a quoted
	// reference (each " is escaped as ""), so its normalized form is not a literal
	// substring of the statement and the check below would miss it. Such names are
	// vanishingly rare; load them unconditionally rather than drop completion. A
	// double quote is the only identifier character Trino re-spells when quoting —
	// every other character appears verbatim, so the substring check is reliable.
	if strings.Contains(normName, `"`) {
		return true
	}
	return strings.Contains(lowerStmt, strings.ToLower(normName))
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
