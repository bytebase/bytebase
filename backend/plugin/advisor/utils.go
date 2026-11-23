package advisor

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// explainBudget caps the statements a row limit rule EXPLAINs at common.MaximumLintExplainSize and
// counts the statements it leaves unchecked.
type explainBudget struct {
	used         int
	skipped      int
	firstSkipped *storepb.Position
}

// spend reports whether the statement at position may be explained, recording it as skipped
// otherwise.
func (b *explainBudget) spend(position *storepb.Position) bool {
	if b.used < common.MaximumLintExplainSize {
		b.used++
		return true
	}
	if b.skipped == 0 {
		b.firstSkipped = position
	}
	b.skipped++
	return false
}

// appendSkippedAdvice appends one warning covering the skipped statements, if any. It is a warning
// whatever the rule's level.
func (b *explainBudget) appendSkippedAdvice(adviceList []*storepb.Advice, title string, adviceCode code.Code) []*storepb.Advice {
	if b.skipped == 0 {
		return adviceList
	}
	return append(adviceList, &storepb.Advice{
		Status:        storepb.Advice_WARNING,
		Code:          adviceCode.Int32(),
		Title:         title,
		Content:       fmt.Sprintf("Only the first %d statements were estimated; %d more were not checked against the row limit.", common.MaximumLintExplainSize, b.skipped),
		StartPosition: b.firstSkipped,
	})
}

// ContainsDDL checks if any of the parsed statements is a DDL statement.
// When DDLs and DMLs are mixed, DML statements often reference objects created
// by DDL statements, causing false positives in dry run checks.
// BYT-8855
func ContainsDDL(engine storepb.Engine, parsedStatements []base.ParsedStatement) bool {
	asts := base.ExtractASTs(parsedStatements)
	types, err := base.GetStatementTypes(engine, asts)
	if err != nil {
		return false
	}
	for _, t := range types {
		switch t {
		case storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
			storepb.StatementType_INSERT,
			storepb.StatementType_UPDATE,
			storepb.StatementType_DELETE,
			storepb.StatementType_MERGE:
		default:
			return true
		}
	}
	return false
}

func DatabaseExists(ctx context.Context, checkCtx Context, database string) bool {
	if checkCtx.ListDatabaseNamesFunc == nil {
		return false
	}

	names, err := checkCtx.ListDatabaseNamesFunc(ctx, checkCtx.InstanceID)
	if err != nil {
		slog.Debug("failed to list databases", slog.String("instance", checkCtx.InstanceID), log.BBError(err))
		return false
	}

	return slices.Contains(names, database)
}
