package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableDisallowDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableDisallowDDL, &TableDisallowDDLAdvisor{})
}

// TableDisallowDDLAdvisor is the advisor checking for disallow DDL on specific tables.
type TableDisallowDDLAdvisor struct {
}

func (*TableDisallowDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &tableDisallowDDLChecker{
		level:        level,
		title:        string(checkCtx.Rule.Type),
		disallowList: payload.List,
	}
	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.adviceList, nil
}

type tableDisallowDDLChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	level      storepb.Advice_Status
	title      string
	adviceList []*storepb.Advice
	// disallowList is the list of table names that disallow DDL.
	disallowList []string
}

func (checker *tableDisallowDDLChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDDLChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if tableName == "" {
		return
	}
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDDLChecker) EnterDropTable(ctx *mysql.DropTableContext) {
	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		if tableName == "" {
			continue
		}
		checker.checkTableName(tableName, ctx.GetStart().GetLine())
	}
}

func (checker *tableDisallowDDLChecker) EnterRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	for _, renamePair := range ctx.AllRenamePair() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
		if tableName == "" {
			continue
		}
		checker.checkTableName(tableName, ctx.GetStart().GetLine())
	}
}

func (checker *tableDisallowDDLChecker) EnterTruncateTableStatement(ctx *mysql.TruncateTableStatementContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDDLChecker) checkTableName(tableName string, line int) {
	for _, disallow := range checker.disallowList {
		if tableName == disallow {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.TableDisallowDDL.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("DDL is disallowed on table %s.", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
			return
		}
	}
}
