package snowflake

import (
	"context"

	sfcompletion "github.com/bytebase/omni/snowflake/completion"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Completion returns Snowflake auto-completion candidates from omni's
// Snowflake completer, which reads Snowflake's own grammar.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, _ := base.CompletionMetadataLoaders(cCtx)
	candidates := sfcompletion.CompleteSession(ctx, sfcompletion.SessionConfig{
		DefaultDatabase: cCtx.DefaultDatabase,
		DefaultSchema:   cCtx.DefaultSchema,
		LoadDatabase:    loadDB,
	}, statement, caretLine, caretOffset)

	return base.ConvertOmniCandidates(candidates, func(c sfcompletion.Candidate) (string, sfcompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidateByTypeText), nil
}
