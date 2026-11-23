package plsql

import (
	"context"

	oraclecompletion "github.com/bytebase/omni/oracle/completion"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(store.Engine_ORACLE, Completion)
}

// Completion returns Oracle auto-completion candidates backed by omni's
// session-aware Oracle completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadSchema, listSchemas := base.CompletionMetadataLoaders(cCtx)
	raw := oraclecompletion.CompleteSession(ctx, oraclecompletion.SessionConfig{
		ReadOnly:        cCtx.Scene == base.SceneTypeQuery,
		DefaultSchema:   cCtx.DefaultDatabase,
		LoadSchema:      loadSchema,
		ListSchemaNames: listSchemas,
	}, statement, caretLine, caretOffset)
	if raw == nil {
		return nil, nil
	}
	return base.DeduplicateCandidates(raw, func(c oraclecompletion.Candidate) (base.Candidate, bool) {
		typ := base.OmniCandidateType(c.Type)
		if c.Type == oraclecompletion.CandidateVariable {
			// Oracle package variables are user variables rather than MySQL-style system variables.
			typ = base.CandidateTypeUserVar
		}
		if typ == base.CandidateTypeNone {
			return base.Candidate{}, false
		}
		return base.Candidate{
			Type:       typ,
			Text:       c.Text,
			Definition: c.Definition,
			Comment:    c.Comment,
		}, true
	}, base.CompareCandidate), nil
}
