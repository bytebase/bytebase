package doris

import (
	"context"

	omnicompletion "github.com/bytebase/omni/doris/completion"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Completion returns Doris auto-completion candidates from omni's Doris
// completer, which reads Doris's own grammar.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, listDBs := base.CompletionMetadataLoaders(cCtx)
	candidates := omnicompletion.CompleteSession(ctx, omnicompletion.SessionConfig{
		DefaultDatabase:   cCtx.DefaultDatabase,
		LoadDatabase:      loadDB,
		ListDatabaseNames: listDBs,
	}, statement, caretLine, caretOffset)

	return base.ConvertOmniCandidates(candidates, func(c omnicompletion.Candidate) (string, omnicompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidateByTypeText), nil
}
