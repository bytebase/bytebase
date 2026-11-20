package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*BuiltinPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &BuiltinPriorBackupCheckAdvisor{})
}

// BuiltinPriorBackupCheckAdvisor is the advisor checking for disallow mix DDL and DML.
type BuiltinPriorBackupCheckAdvisor struct {
}

// Check checks for disallow mix DDL and DML.
func (*BuiltinPriorBackupCheckAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	schemaName := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
	facts := v1review.PriorBackupFacts{
		TotalSize:      checkCtx.StatementsTotalSize,
		MaxSize:        common.MaxSheetCheckSize,
		Database:       schemaName,
		DatabaseExists: checkCtx.OriginalMetadata.GetSchemaMetadata(schemaName) != nil,
	}

	var adviceList []*storepb.Advice
	for _, f := range pgv1review.PriorBackupCheck(nodes, facts) {
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
			stmt := stmts[f.Statement]
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("Data change can only run DML, \"%s\" is not DML", statementTrimmedText(stmt)),
				Code:    code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: &storepb.Position{
					Line:   statementFindingLine(stmt),
					Column: 0,
				},
			})
		case v1review.PriorBackupDataModifyingCTE:
			stmt := stmts[f.Statement]
			content := fmt.Sprintf("Prior backup does not cover rows changed by data-modifying WITH query %s", f.Value)
			switch nodes[f.Statement].(type) {
			case *ast.UpdateStmt, *ast.DeleteStmt:
				content = fmt.Sprintf("Prior backup does not support data-modifying WITH query %s; the task will fail unless prior backup is disabled or skipped", f.Value)
			default:
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: content,
				Code:    code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: &storepb.Position{
					Line:   statementFindingLine(stmt),
					Column: 0,
				},
			})
		case v1review.PriorBackupDatabaseMissing:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("Need schema %q to do prior backup but it does not exist", f.Name),
				Code:          code.SchemaNotExists.Int32(),
				StartPosition: nil,
			})
		case v1review.PriorBackupMixedDMLOnTable:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("The statement type is not the same for all statements on the same table %q", f.Table),
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: nil,
			})
		default:
		}
	}

	return adviceList, nil
}
