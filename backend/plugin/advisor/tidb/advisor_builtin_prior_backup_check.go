package tidb

import (
	"context"
	"fmt"

	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
}

// StatementPriorBackupCheckAdvisor flags inputs incompatible with the
// prior-backup workflow. The judgment lives in omni
// (tidb/v1review.PriorBackupCheck); this adapter resolves the backup database
// existence and renders the findings as advice.
type StatementPriorBackupCheckAdvisor struct{}

// Check evaluates the prior-backup compatibility of the reviewed
// statements. Gated on `checkCtx.EnablePriorBackup`.
func (*StatementPriorBackupCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	defaultDB := checkCtx.DBSchema.GetName()
	if defaultDB == "" {
		defaultDB = checkCtx.CurrentDatabase
	}

	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_TIDB)
	schema := v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}
	facts := v1review.PriorBackupFacts{
		TotalSize:      checkCtx.StatementsTotalSize,
		MaxSize:        common.MaxSheetCheckSize,
		Database:       databaseName,
		DatabaseExists: advisor.DatabaseExists(ctx, checkCtx, databaseName),
	}

	var adviceList []*storepb.Advice
	for _, f := range tidbv1review.PriorBackupCheck(stmtNodes(stmts), defaultDB, schema, facts) {
		switch f.Code {
		case v1review.PriorBackupScriptTooLarge:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %s bytes for backup", f.Value),
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: nil,
			})
		case v1review.PriorBackupMixedDDLAndDML:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       "Prior backup cannot deal with mixed DDL and DML statements",
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: base.ConvertANTLRLineToPosition(stmts[f.Statement].FirstTokenLine()),
			})
		case v1review.PriorBackupDatabaseMissing:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("Need database %q to do prior backup but it does not exist", f.Name),
				Code:          code.DatabaseNotExists.Int32(),
				StartPosition: nil,
			})
		case v1review.PriorBackupMixedDMLOnTable:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("Prior backup cannot handle mixed DML statements on the same table %q", f.Table),
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: nil,
			})
		case v1review.PriorBackupTooManyDMLStatements:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("Prior backup cannot handle more than %s DML statements across different tables", f.Value),
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: nil,
			})
		default:
		}
	}

	return adviceList, nil
}
