package pg

import (
	"context"

	pgcompletion "github.com/bytebase/omni/pg/completion"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(store.Engine_POSTGRES, Completion)
	base.RegisterCompleteFunc(store.Engine_SNOWFLAKE, Completion)
	base.RegisterCompleteFunc(store.Engine_COCKROACHDB, Completion)
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

	return base.DeduplicateCandidates(candidates, func(c pgcompletion.Candidate) (base.Candidate, bool) {
		typ := convertCandidateType(c.Type)
		if typ == base.CandidateTypeNone {
			return base.Candidate{}, false
		}
		return base.Candidate{
			Text:       c.Text,
			Type:       typ,
			Definition: c.Definition,
			Comment:    c.Comment,
		}, true
	}, base.CompareCandidateByTypeText), nil
}

func convertCandidateType(t pgcompletion.CandidateType) base.CandidateType {
	switch t {
	case pgcompletion.CandidateKeyword, pgcompletion.CandidateType_:
		return base.CandidateTypeKeyword
	case pgcompletion.CandidateSchema:
		return base.CandidateTypeSchema
	case pgcompletion.CandidateTable:
		return base.CandidateTypeTable
	case pgcompletion.CandidateView:
		return base.CandidateTypeView
	case pgcompletion.CandidateMaterializedView:
		return base.CandidateTypeMaterializedView
	case pgcompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case pgcompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case pgcompletion.CandidateSequence:
		return base.CandidateTypeSequence
	case pgcompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case pgcompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	default:
		return base.CandidateTypeNone
	}
}
