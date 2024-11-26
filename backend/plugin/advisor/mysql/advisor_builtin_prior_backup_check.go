package mysql

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLBuiltinPriorBackupCheck, &StatementPriorBackupCheckAdvisor{})
}

type StatementPriorBackupCheckAdvisor struct {
}

func (*StatementPriorBackupCheckAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	if ctx.PreUpdateBackupDetail == nil || ctx.ChangeType != storepb.PlanCheckRunConfig_DML {
		return nil, nil
	}

	var adviceList []*storepb.Advice
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(ctx.Rule.Type)

	if len(ctx.Statements) > common.MaxSheetCheckSize {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %d bytes for backup", common.MaxSheetCheckSize),
			Code:    advisor.BuiltinPriorBackupCheck.Int32(),
			StartPosition: &storepb.Position{
				Line: 1,
			},
		})
	}

	for _, stmt := range stmtList {
		checker := &mysqlparser.StatementTypeChecker{}
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)

		if checker.IsDDL {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: "Prior backup cannot deal with mixed DDL and DML statements",
				Code:    advisor.BuiltinPriorBackupCheck.Int32(),
				StartPosition: &storepb.Position{
					Line: int32(stmt.BaseLine) + 1,
				},
			})
		}
	}

	databaseName := extractDatabaseName(ctx.PreUpdateBackupDetail.Database)
	if !advisor.DatabaseExists(ctx, databaseName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: fmt.Sprintf("Need database %q to do prior backup but it does not exist", ctx.PreUpdateBackupDetail.Database),
			Code:    advisor.DatabaseNotExists.Int32(),
			StartPosition: &storepb.Position{
				Line: 1,
			},
		})
	}

	tableReferences, err := prepareTransformation(databaseName, stmtList)
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Prior backup cannot handle mixed DML statements on the same table %q", key),
					Code:    advisor.BuiltinPriorBackupCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: 1,
					},
				})
				break
			}
		}
	}

	return adviceList, nil
}

func prepareTransformation(databaseName string, parseResult []*mysqlparser.ParseResult) ([]*mysqlparser.TableReference, error) {
	var result []*mysqlparser.TableReference
	for i, sql := range parseResult {
		tables, err := mysqlparser.ExtractTables(databaseName, sql, i)
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

func extractDatabaseName(databaseUID string) string {
	segments := strings.Split(databaseUID, "/")
	return segments[len(segments)-1]
}
