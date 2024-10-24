package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	maxMixedDMLCount = 5
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

	if !databaseExists(ctx.Context, ctx.Driver, extractDatabaseName(ctx.PreUpdateBackupDetail.Database)) {
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

	// Do not allow mixed DML on the same table.
	checker := &statementDisallowMixDMLChecker{}
	for _, stmt := range stmtList {
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	if len(checker.updateStatements)+len(checker.deleteStatements) > maxMixedDMLCount && !updateForOneTableWithUnique(ctx.DBSchema, checker) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: fmt.Sprintf("Prior backup is feasible only with up to %d statements that are either UPDATE or DELETE, or if all UPDATEs target the same table with a PRIMARY or UNIQUE KEY in the WHERE clause", maxMixedDMLCount),
			Code:    advisor.BuiltinPriorBackupCheck.Int32(),
			StartPosition: &storepb.Position{
				Line: 1,
			},
		})
	}

	return adviceList, nil
}

func updateForOneTableWithUnique(dbSchema *storepb.DatabaseSchemaMetadata, checker *statementDisallowMixDMLChecker) bool {
	if len(checker.deleteStatements) > 0 {
		return false
	}

	var table *table
	for _, update := range checker.updateStatements {
		for _, tableRefCtx := range update.TableReferenceList().AllTableReference() {
			tables, err := extractTableReference(tableRefCtx)
			if err != nil {
				slog.Debug("failed to extract table reference", "err", err)
				return false
			}
			if len(tables) != 1 {
				return false
			}
			if table == nil {
				table = &tables[0]
			} else if !equalTable(table, &tables[0]) {
				return false
			}
			if !hasUniqueInWhereClause(dbSchema, update, table) {
				return false
			}
		}
	}

	return true
}

func hasUniqueInWhereClause(dbSchema *storepb.DatabaseSchemaMetadata, update *mysql.UpdateStatementContext, table *table) bool {
	if update.WhereClause() == nil {
		return false
	}
	list := extractColumnsInEqualCondition(table, update.WhereClause().Expr())
	columnMap := make(map[string]bool)
	for _, column := range list {
		columnMap[strings.ToLower(column)] = true
	}

	if dbSchema == nil {
		return false
	}

	for _, schema := range dbSchema.Schemas {
		for _, t := range schema.Tables {
			if strings.EqualFold(t.Name, table.table) {
				for _, index := range t.Indexes {
					if index.Unique || index.Primary {
						exists := true
						for _, column := range index.Expressions {
							if !columnMap[strings.ToLower(column)] {
								exists = false
								break
							}
						}
						if exists {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func extractColumnsInEqualCondition(table *table, node antlr.ParserRuleContext) []string {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *mysql.ExprIsContext:
		return extractColumnsInEqualCondition(table, n.BoolPri())
	case *mysql.ExprAndContext:
		return append(extractColumnsInEqualCondition(table, n.Expr(0)), extractColumnsInEqualCondition(table, n.Expr(1))...)
	case *mysql.PrimaryExprCompareContext:
		if n.CompOp().EQUAL_OPERATOR() == nil {
			return nil
		}
		if isConstant(n.Predicate()) {
			return extractColumnsInEqualCondition(table, n.BoolPri())
		} else if isConstant(n.BoolPri()) {
			return extractColumnsInEqualCondition(table, n.Predicate())
		}
	case *mysql.PrimaryExprPredicateContext:
		return extractColumnsInEqualCondition(table, n.Predicate())
	case *mysql.PredicateContext:
		if n.PredicateOperations() != nil || n.MEMBER_SYMBOL() != nil || n.LIKE_SYMBOL() != nil {
			return nil
		}
		return extractColumnsInEqualCondition(table, n.BitExpr(0))
	case *mysql.BitExprContext:
		return extractColumnsInEqualCondition(table, n.SimpleExpr())
	case *mysql.SimpleExprColumnRefContext:
		databaseName, tableName, columnName := mysqlparser.NormalizeMySQLColumnRef(n.ColumnRef())
		if databaseName != "" && table.database != "" && !strings.EqualFold(databaseName, table.database) {
			return nil
		}
		if tableName != "" && table.table != "" && !strings.EqualFold(tableName, table.table) {
			return nil
		}
		return []string{columnName}
	}

	return nil
}

func isConstant(node antlr.ParserRuleContext) bool {
	if node == nil {
		return false
	}
	switch n := node.(type) {
	case *mysql.ExprIsContext:
		return isConstant(n.BoolPri())
	case *mysql.PrimaryExprPredicateContext:
		return isConstant(n.Predicate())
	case *mysql.PredicateContext:
		if n.PredicateOperations() != nil || n.MEMBER_SYMBOL() != nil || n.LIKE_SYMBOL() != nil {
			return false
		}
		return isConstant(n.BitExpr(0))
	case *mysql.BitExprContext:
		return isConstant(n.SimpleExpr())
	case *mysql.SimpleExprLiteralContext:
		return true
	default:
		return false
	}
}

func equalTable(a, b *table) bool {
	if a == nil || b == nil {
		return false
	}
	return a.database == b.database && a.table == b.table
}

func extractDatabaseName(databaseUID string) string {
	segments := strings.Split(databaseUID, "/")
	return segments[len(segments)-1]
}

func databaseExists(ctx context.Context, driver *sql.DB, database string) bool {
	if driver == nil {
		return false
	}
	var count int
	if err := driver.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", database).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

type statementDisallowMixDMLChecker struct {
	*mysql.BaseMySQLParserListener

	updateStatements []*mysql.UpdateStatementContext
	deleteStatements []*mysql.DeleteStatementContext
}

type table struct {
	database string
	table    string
}

func (c *statementDisallowMixDMLChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	c.updateStatements = append(c.updateStatements, ctx)
}

func (c *statementDisallowMixDMLChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	c.deleteStatements = append(c.deleteStatements, ctx)
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
