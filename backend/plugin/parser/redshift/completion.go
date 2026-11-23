package redshift

import (
	"context"

	redshiftcompletion "github.com/bytebase/omni/redshift/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_REDSHIFT, Completion)
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

	return base.DeduplicateCandidates(candidates, func(c redshiftcompletion.Candidate) (base.Candidate, bool) {
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
	}, base.CompareCandidate), nil
}

func convertCandidateType(t redshiftcompletion.CandidateType) base.CandidateType {
	switch t {
	case redshiftcompletion.CandidateKeyword, redshiftcompletion.CandidateType_:
		return base.CandidateTypeKeyword
	case redshiftcompletion.CandidateSchema:
		return base.CandidateTypeSchema
	case redshiftcompletion.CandidateTable:
		return base.CandidateTypeTable
	case redshiftcompletion.CandidateView:
		return base.CandidateTypeView
	case redshiftcompletion.CandidateMaterializedView:
		return base.CandidateTypeMaterializedView
	case redshiftcompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case redshiftcompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case redshiftcompletion.CandidateSequence:
		return base.CandidateTypeSequence
	case redshiftcompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case redshiftcompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	default:
		return base.CandidateTypeNone
	}
}
