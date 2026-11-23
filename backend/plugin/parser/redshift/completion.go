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

	return base.ConvertOmniCandidates(candidates, func(c redshiftcompletion.Candidate) (string, redshiftcompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidate), nil
}
