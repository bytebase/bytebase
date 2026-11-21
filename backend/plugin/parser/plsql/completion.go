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
		typ := convertCandidateType(c.Type)
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

func convertCandidateType(t oraclecompletion.CandidateType) base.CandidateType {
	switch t {
	case oraclecompletion.CandidateKeyword, oraclecompletion.CandidateType_:
		return base.CandidateTypeKeyword
	case oraclecompletion.CandidateSchema:
		return base.CandidateTypeSchema
	case oraclecompletion.CandidateTable, oraclecompletion.CandidateSynonym:
		return base.CandidateTypeTable
	case oraclecompletion.CandidateView:
		return base.CandidateTypeView
	case oraclecompletion.CandidateMaterializedView:
		return base.CandidateTypeMaterializedView
	case oraclecompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case oraclecompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case oraclecompletion.CandidateProcedure:
		return base.CandidateTypeRoutine
	case oraclecompletion.CandidateSequence:
		return base.CandidateTypeSequence
	case oraclecompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case oraclecompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	case oraclecompletion.CandidateVariable:
		return base.CandidateTypeUserVar
	default:
		return base.CandidateTypeNone
	}
}
