package mysql

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementDisallowMixDML, &StatementDisallowMixDMLAdvisor{})
}

type StatementDisallowMixDMLAdvisor struct {
}

func (*StatementDisallowMixDMLAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementDisallowMixDMLChecker{
		level:             level,
		title:             string(ctx.Rule.Type),
		dmlStatementCount: make(map[table]map[string]int),
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	for table, dmlCount := range checker.dmlStatementCount {
		if len(dmlCount) > 1 {
			content := "Found"
			for _, t := range []string{"DELETE", "INSERT", "UPDATE"} {
				count, ok := dmlCount[t]
				if ok {
					content += fmt.Sprintf(" %d %s,", count, t)
				}
			}
			content = strings.TrimSuffix(content, ",")
			content += fmt.Sprintf(" on table `%s`.`%s`, disallow mixing different types of DML statements", table.database, table.table)
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementDisallowMixDML,
				Title:   checker.title,
				Content: content,
			})
		}
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type statementDisallowMixDMLChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string

	dmlStatementCount map[table]map[string]int
}

type table struct {
	database string
	table    string
}

func (c *statementDisallowMixDMLChecker) EnterInsertStatement(ctx *mysql.InsertStatementContext) {
	if ctx.TableRef() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	table := table{
		database: databaseName,
		table:    tableName,
	}
	if _, ok := c.dmlStatementCount[table]; !ok {
		c.dmlStatementCount[table] = make(map[string]int)
	}
	c.dmlStatementCount[table]["INSERT"]++
}

func (c *statementDisallowMixDMLChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	var allTables []table
	for _, tableRefCtx := range ctx.TableReferenceList().AllTableReference() {
		tables, err := extractTableReference(tableRefCtx)
		if err != nil {
			c.adviceList = append(c.adviceList, advisor.Advice{
				Status:  c.level,
				Code:    advisor.Internal,
				Title:   c.title,
				Content: fmt.Sprintf("Failed to extract table reference: %v", err),
				Line:    tableRefCtx.GetStart().GetLine() + c.baseLine,
			})
			continue
		}
		allTables = append(allTables, tables...)
	}
	for _, table := range allTables {
		if _, ok := c.dmlStatementCount[table]; !ok {
			c.dmlStatementCount[table] = make(map[string]int)
		}
		c.dmlStatementCount[table]["UPDATE"]++
	}
}

func (c *statementDisallowMixDMLChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	var allTables []table
	if ctx.TableRef() != nil {
		tables, err := extractTableRef(ctx.TableRef())
		if err != nil {
			c.adviceList = append(c.adviceList, advisor.Advice{
				Status:  c.level,
				Code:    advisor.Internal,
				Title:   c.title,
				Content: fmt.Sprintf("Failed to extract table reference: %v", err),
				Line:    ctx.GetStart().GetLine() + c.baseLine,
			})
		} else {
			allTables = append(allTables, tables...)
		}
	}
	if ctx.TableReferenceList() != nil {
		tables, err := extractTableReferenceList(ctx.TableReferenceList())
		if err != nil {
			c.adviceList = append(c.adviceList, advisor.Advice{
				Status:  c.level,
				Code:    advisor.Internal,
				Title:   c.title,
				Content: fmt.Sprintf("Failed to extract table reference: %v", err),
				Line:    ctx.GetStart().GetLine() + c.baseLine,
			})
		} else {
			allTables = append(allTables, tables...)
		}
	}
	for _, table := range allTables {
		if _, ok := c.dmlStatementCount[table]; !ok {
			c.dmlStatementCount[table] = make(map[string]int)
		}
		c.dmlStatementCount[table]["DELETE"]++
	}
}

func extractTableReference(ctx mysql.ITableReferenceContext) ([]table, error) {
	if ctx.TableFactor() == nil {
		return nil, nil
	}
	res, err := extractTableFactor(ctx.TableFactor())
	if err != nil {
		return nil, err
	}
	for _, joinedTableCtx := range ctx.AllJoinedTable() {
		tables, err := extractJoinedTable(joinedTableCtx)
		if err != nil {
			return nil, err
		}
		res = append(res, tables...)
	}

	return res, nil
}

func extractTableRef(ctx mysql.ITableRefContext) ([]table, error) {
	if ctx == nil {
		return nil, nil
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx)
	return []table{
		{
			database: databaseName,
			table:    tableName,
		},
	}, nil
}

func extractTableReferenceList(ctx mysql.ITableReferenceListContext) ([]table, error) {
	var res []table
	for _, tableRefCtx := range ctx.AllTableReference() {
		tables, err := extractTableReference(tableRefCtx)
		if err != nil {
			return nil, err
		}
		res = append(res, tables...)
	}
	return res, nil
}

func extractTableReferenceListParens(ctx mysql.ITableReferenceListParensContext) ([]table, error) {
	if ctx.TableReferenceList() != nil {
		return extractTableReferenceList(ctx.TableReferenceList())
	}
	if ctx.TableReferenceListParens() != nil {
		return extractTableReferenceListParens(ctx.TableReferenceListParens())
	}
	return nil, nil
}

func extractTableFactor(ctx mysql.ITableFactorContext) ([]table, error) {
	switch {
	case ctx.SingleTable() != nil:
		return extractSingleTable(ctx.SingleTable())
	case ctx.SingleTableParens() != nil:
		return extractSingleTableParens(ctx.SingleTableParens())
	case ctx.DerivedTable() != nil:
		return nil, nil
	case ctx.TableReferenceListParens() != nil:
		return extractTableReferenceListParens(ctx.TableReferenceListParens())
	case ctx.TableFunction() != nil:
		return nil, nil
	default:
		return nil, nil
	}
}

func extractSingleTable(ctx mysql.ISingleTableContext) ([]table, error) {
	return extractTableRef(ctx.TableRef())
}

func extractSingleTableParens(ctx mysql.ISingleTableParensContext) ([]table, error) {
	if ctx.SingleTable() != nil {
		return extractSingleTable(ctx.SingleTable())
	}
	if ctx.SingleTableParens() != nil {
		return extractSingleTableParens(ctx.SingleTableParens())
	}
	return nil, nil
}

func extractJoinedTable(ctx mysql.IJoinedTableContext) ([]table, error) {
	if ctx.TableFactor() != nil {
		return extractTableFactor(ctx.TableFactor())
	}
	if ctx.TableReference() != nil {
		return extractTableReference(ctx.TableReference())
	}
	return nil, nil
}
