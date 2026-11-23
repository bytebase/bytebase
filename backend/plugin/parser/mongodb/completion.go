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
		typ := base.OmniCandidateType(c.Type)
		if typ == base.CandidateTypeNone {
			typ = base.CandidateTypeFunction
		}
		return base.Candidate{
			Type:       typ,
			Text:       c.Text,
			Definition: c.Definition,
			Comment:    c.Comment,
		}, true
	}, base.CompareCandidate), nil
}
