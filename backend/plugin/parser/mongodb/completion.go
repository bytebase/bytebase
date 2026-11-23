package mongodb

import (
	"context"

	omnicompletion "github.com/bytebase/omni/mongo/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MONGODB, Completion)
}

// Completion is the entry point for MongoDB code completion.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDatabase, _ := base.CompletionMetadataLoaders(cCtx)

	candidates := omnicompletion.CompleteSession(ctx, omnicompletion.SessionConfig{
		DefaultDatabase: cCtx.DefaultDatabase,
		LoadDatabase:    loadDatabase,
	}, statement, caretLine, caretOffset)

	return base.DeduplicateCandidates(candidates, func(c omnicompletion.Candidate) (base.Candidate, bool) {
		return base.Candidate{
			Type:       candidateTypeToBase(c.Type),
			Text:       c.Text,
			Definition: c.Definition,
			Comment:    c.Comment,
		}, true
	}, base.CompareCandidate), nil
}

// candidateTypeToBase maps omni completion candidate types to Bytebase base types.
func candidateTypeToBase(t omnicompletion.CandidateType) base.CandidateType {
	switch t {
	case omnicompletion.CandidateCollection:
		return base.CandidateTypeTable
	case omnicompletion.CandidateKeyword, omnicompletion.CandidateShowTarget:
		return base.CandidateTypeKeyword
	case omnicompletion.CandidateMethod,
		omnicompletion.CandidateCursorMethod,
		omnicompletion.CandidateAggStage,
		omnicompletion.CandidateQueryOperator,
		omnicompletion.CandidateBSONHelper,
		omnicompletion.CandidateDbMethod,
		omnicompletion.CandidateRsMethod,
		omnicompletion.CandidateShMethod:
		return base.CandidateTypeFunction
	default:
		return base.CandidateTypeFunction
	}
}
