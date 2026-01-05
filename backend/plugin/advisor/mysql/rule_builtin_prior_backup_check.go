package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
}

type StatementPriorBackupCheckAdvisor struct {
}

func (*StatementPriorBackupCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}

	var adviceList []*storepb.Advice
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	if checkCtx.StatementsTotalSize > common.MaxSheetCheckSize {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %d bytes for backup", common.MaxSheetCheckSize),
			Code:          code.BuiltinPriorBackupCheck.Int32(),
			StartPosition: nil,
		})
	}

	for _, stmt := range checkCtx.ParsedStatements {
		checker := &mysqlparser.StatementTypeChecker{}
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)

		if checker.IsDDL {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       "Prior backup cannot deal with mixed DDL and DML statements",
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: common.ConvertANTLRLineToPosition(stmt.BaseLine()),
			})
		}
	}

	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_MYSQL)
	if !advisor.DatabaseExists(ctx, checkCtx, databaseName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("Need database %q to do prior backup but it does not exist", databaseName),
			Code:          code.DatabaseNotExists.Int32(),
			StartPosition: nil,
		})
	}

	tableReferences, err := prepareTransformation(checkCtx.DBSchema.Name, checkCtx.ParsedStatements)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}
	groupByTable := make(map[string][]*mysqlparser.TableReference)
	for _, table := range tableReferences {
		key := fmt.Sprintf("%s.%s", table.Database, table.Table)
		groupByTable[key] = append(groupByTable[key], table)
	}

	for key, list := range groupByTable {
		stmtType := mysqlparser.StatementTypeUnknown
		for _, item := range list {
			if stmtType == mysqlparser.StatementTypeUnknown {
				stmtType = item.StatementType
			}
			if stmtType != item.StatementType {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Prior backup cannot handle mixed DML statements on the same table %q", key),
					Code:          code.BuiltinPriorBackupCheck.Int32(),
					StartPosition: nil,
				})
				break
			}
		}
	}

	return adviceList, nil
}

func prepareTransformation(databaseName string, parsedStatements []base.ParsedStatement) ([]*mysqlparser.TableReference, error) {
	var result []*mysqlparser.TableReference
	for i, stmt := range parsedStatements {
		if stmt.AST == nil {
			continue
		}
		ast, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		tables, err := mysqlparser.ExtractTables(databaseName, ast, i)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract tables")
		}
		for _, table := range tables {
			result = append(result, &mysqlparser.TableReference{
				Database:      table.Table.Database,
				Table:         table.Table.Table,
				Alias:         table.Table.Alias,
				StatementType: table.Table.StatementType,
			})
		}
	}

	return result, nil
}
