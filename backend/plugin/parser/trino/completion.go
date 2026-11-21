package trino

import (
	"context"

	"github.com/bytebase/omni/trino/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_TRINO, Completion)
}

// Completion returns auto-completion candidates for a Trino statement at the
// given caret position, backed by omni's session-aware Trino completer.
//
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
			Type: convertCandidateType(c.Type),
		})
	}
	return result, nil
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
