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
	return base.MapMySQLFamilyCandidates(raw, candidateToBase), nil
}

func candidateToBase(c omnicompletion.Candidate) base.Candidate {
	return base.Candidate{
		Type:       candidateTypeToBase(c.Type),
		Text:       c.Text,
		Definition: c.Definition,
		Comment:    c.Comment,
	}
}

func candidateTypeToBase(t omnicompletion.CandidateType) base.CandidateType {
	switch t {
	case omnicompletion.CandidateKeyword, omnicompletion.CandidateType_:
		return base.CandidateTypeKeyword
	case omnicompletion.CandidateDatabase:
		return base.CandidateTypeDatabase
	case omnicompletion.CandidateTable:
		return base.CandidateTypeTable
	case omnicompletion.CandidateView:
		return base.CandidateTypeView
	case omnicompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case omnicompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case omnicompletion.CandidateProcedure:
		return base.CandidateTypeRoutine
	case omnicompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case omnicompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	case omnicompletion.CandidateEvent:
		return base.CandidateTypeEvent
	case omnicompletion.CandidateCharset:
		return base.CandidateTypeCharset
	case omnicompletion.CandidateEngine:
		return base.CandidateTypeEngine
	case omnicompletion.CandidateVariable:
		return base.CandidateTypeSystemVar
	default:
		return base.CandidateTypeNone
	}
}
