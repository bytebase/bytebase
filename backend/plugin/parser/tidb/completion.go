package tidb

import (
	"context"

	omnicompletion "github.com/bytebase/omni/tidb/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_TIDB, Completion)
}

// Completion provides auto-complete candidates for TiDB statements using the
// omni TiDB completion engine.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, listDBs := base.MySQLFamilyDatabaseLoaders(cCtx)
	raw := omnicompletion.CompleteSession(ctx, omnicompletion.SessionConfig{
		ReadOnly:          cCtx.Scene == base.SceneTypeQuery,
		DefaultDatabase:   cCtx.DefaultDatabase,
		LoadDatabase:      loadDB,
		ListDatabaseNames: listDBs,
	}, statement, caretLine, caretOffset)
	return base.ConvertOmniCandidates(raw, func(c omnicompletion.Candidate) (string, omnicompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidate), nil
}
