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
	return base.DeduplicateCandidates(raw, func(c mssqlcompletion.Candidate) (base.Candidate, bool) {
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

func convertCandidateType(t mssqlcompletion.CandidateType) base.CandidateType {
	switch t {
	case mssqlcompletion.CandidateKeyword, mssqlcompletion.CandidateType_:
		return base.CandidateTypeKeyword
	case mssqlcompletion.CandidateDatabase:
		return base.CandidateTypeDatabase
	case mssqlcompletion.CandidateSchema:
		return base.CandidateTypeSchema
	case mssqlcompletion.CandidateTable:
		return base.CandidateTypeTable
	case mssqlcompletion.CandidateView:
		return base.CandidateTypeView
	case mssqlcompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case mssqlcompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case mssqlcompletion.CandidateProcedure:
		return base.CandidateTypeRoutine
	case mssqlcompletion.CandidateSequence:
		return base.CandidateTypeSequence
	case mssqlcompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case mssqlcompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	default:
		return base.CandidateTypeNone
	}
}
