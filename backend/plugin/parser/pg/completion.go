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

	return base.ConvertOmniCandidates(candidates, func(c pgcompletion.Candidate) (string, pgcompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidateByTypeText), nil
}
