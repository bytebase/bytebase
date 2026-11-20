package mysql

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mysql/ast"
	"github.com/bytebase/omni/mysql/catalog"
	omnicompletion "github.com/bytebase/omni/mysql/completion"
	mysqlparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MYSQL, Completion)
	base.RegisterCompleteFunc(storepb.Engine_CLICKHOUSE, Completion)
	base.RegisterCompleteFunc(storepb.Engine_STARROCKS, Completion)
	base.RegisterCompleteFunc(storepb.Engine_DORIS, Completion)
}

// Completion provides auto-complete candidates for MySQL statements using the
// omni MySQL completion engine.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	return base.CompleteMySQLFamily(
		cCtx,
		statement,
		caretLine,
		caretOffset,
		currentStatement,
		mysqlparser.IsReservedKeyword,
		func(stmt string, pos int) []base.Candidate {
			cat := catalog.New()
			base.PopulateMySQLFamilyCatalog(
				ctx,
				cCtx,
				stmt,
				pos,
				hasDatabaseRule,
				func(name string) {
					_ = cat.DefineDatabase(&ast.CreateDatabaseStmt{Name: name, IfNotExists: true})
				},
				func(name string, meta *metadatapb.DatabaseSchemaMetadata) {
					_, _ = cat.LoadMetadataInto(ctx, meta, name)
				},
				cat.SetCurrentDatabase,
			)
			raw := omnicompletion.Complete(stmt, pos, cat)
			out := make([]base.Candidate, len(raw))
			for i, c := range raw {
				out[i] = base.Candidate{
					Type:       candidateTypeToBase(c.Type),
					Text:       c.Text,
					Definition: c.Definition,
					Comment:    c.Comment,
				}
			}
			return out
		},
	), nil
}

func hasDatabaseRule(sql string, offset int) bool {
	cs := mysqlparser.Collect(sql, offset)
	return cs != nil && cs.HasRule("database_ref")
}

func currentStatement(statement string, pos int) (string, int) {
	for _, seg := range mysqlparser.Split(statement) {
		if pos >= seg.ByteStart && pos <= seg.ByteEnd {
			return seg.Text, pos - seg.ByteStart
		}
	}
	return statement, pos
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
