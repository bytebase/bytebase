package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
}

type StatementPriorBackupCheckAdvisor struct {
}

func (*StatementPriorBackupCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	var nodes []ast.Node
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok || node == nil {
			continue
		}
		nodes = append(nodes, node)
	}

	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_MSSQL)
	schema := v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}
	facts := v1review.PriorBackupFacts{
		TotalSize:      checkCtx.StatementsTotalSize,
		MaxSize:        common.MaxSheetCheckSize,
		Database:       databaseName,
		DatabaseExists: advisor.DatabaseExists(ctx, checkCtx, databaseName),
	}

	var adviceList []*storepb.Advice
	for _, f := range mssqlv1review.PriorBackupCheck(nodes, schema, facts) {
		switch f.Code {
		case v1review.PriorBackupScriptTooLarge:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("The size of statements in the sheet exceeds the limit of %s", f.Value),
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: &storepb.Position{Line: 1},
			})
		case v1review.PriorBackupDatabaseMissing:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("Need database %q to do prior backup but it does not exist", f.Name),
				Code:          code.DatabaseNotExists.Int32(),
				StartPosition: &storepb.Position{Line: 1},
			})
		case v1review.PriorBackupDDLInBatch:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       "Prior backup cannot deal with mixed DDL and DML statements",
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: &storepb.Position{Line: 1},
			})
		case v1review.PriorBackupMixedDMLOnTable:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("The statement type is not the same for all statements on the same table %q", f.Table),
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: &storepb.Position{Line: 1},
			})
		default:
		}
	}

	return adviceList, nil
}
