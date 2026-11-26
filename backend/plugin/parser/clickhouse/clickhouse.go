// Package clickhouse registers ClickHouse's parser functions, each backed by
// omni's ClickHouse packages.
package clickhouse

import (
	"context"

	chanalysis "github.com/bytebase/omni/clickhouse/analysis"
	chcompletion "github.com/bytebase/omni/clickhouse/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(chanalysis.Engine)
	engine.Complete = completion
	base.Register(engine, storepb.Engine_CLICKHOUSE)
}

func completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, listNames := base.CompletionMetadataLoaders(cCtx)
	candidates := chcompletion.CompleteSession(ctx, chcompletion.SessionConfig{
		DefaultDatabase:   cCtx.DefaultDatabase,
		LoadDatabase:      loadDB,
		ListDatabaseNames: listNames,
	}, statement, caretLine, caretOffset)
	return base.ConvertOmniCandidates(candidates, func(c chcompletion.Candidate) (string, chcompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidateByTypeText), nil
}
