package tsql

import (
	"context"

	mssqlcompletion "github.com/bytebase/omni/mssql/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MSSQL, Completion)
}

// Completion returns T-SQL auto-completion candidates backed by omni's
// session-aware MSSQL completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, listDBs := base.CompletionMetadataLoaders(cCtx)
	raw := mssqlcompletion.CompleteSession(ctx, mssqlcompletion.SessionConfig{
		DefaultDatabase:   cCtx.DefaultDatabase,
		DefaultSchema:     cCtx.DefaultSchema,
		LoadDatabase:      loadDB,
		ListDatabaseNames: listDBs,
	}, statement, caretLine, caretOffset)
	return base.ConvertOmniCandidates(raw, func(c mssqlcompletion.Candidate) (string, mssqlcompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidate), nil
}
