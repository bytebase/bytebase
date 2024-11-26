package oracle

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	plsql "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	maxMixedDMLCount = 5
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleBuiltinPriorBackupCheck, &StatementPriorBackupCheckAdvisor{})
}

type StatementPriorBackupCheckAdvisor struct {
}

func (*StatementPriorBackupCheckAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	if ctx.PreUpdateBackupDetail == nil || ctx.ChangeType != storepb.PlanCheckRunConfig_DML {
		return nil, nil
	}

	var adviceList []*storepb.Advice
	stmtList, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(ctx.Rule.Type)

	if !advisor.DatabaseExists(ctx, extractDatabaseName(ctx.PreUpdateBackupDetail.Database)) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: fmt.Sprintf("Need database %q to do prior backup but it does not exist", ctx.PreUpdateBackupDetail.Database),
			Code:    advisor.DatabaseNotExists.Int32(),
			StartPosition: &storepb.Position{
				Line: 0,
			},
		})
		return adviceList, nil
	}

	checker := &statementDisallowMixDMLChecker{}
	antlr.ParseTreeWalkerDefault.Walk(checker, stmtList)

	if checker.hasDDL {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: "Prior backup cannot deal with mixed DDL and DML statements",
			Code:    int32(advisor.BuiltinPriorBackupCheck),
			StartPosition: &storepb.Position{
				Line: 0,
			},
		})
	}

	if len(checker.updateStatements)+len(checker.deleteStatements) > maxMixedDMLCount && !updateForOneTableWithUnique(ctx.DBSchema, checker) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: fmt.Sprintf("Prior backup cannot deal with mixed DML more than %d statements, otherwise statements need be UPDATE for one table with PRIMARY KEY or UNIQUE KEY in WHERE clause", maxMixedDMLCount),
			Code:    int32(advisor.BuiltinPriorBackupCheck),
			StartPosition: &storepb.Position{
				Line: 0,
			},
		})
	}

	return adviceList, nil
}

type tableRef struct {
	database string
	table    string
	alias    string
}

func updateForOneTableWithUnique(dbSchema *storepb.DatabaseSchemaMetadata, checker *statementDisallowMixDMLChecker) bool {
	if len(checker.deleteStatements) > 0 {
		return false
	}

	var table *tableRef
	for _, update := range checker.updateStatements {
		extractor := &tableExtractor{
			databaseName: dbSchema.Name,
		}
		antlr.ParseTreeWalkerDefault.Walk(extractor, update)

		if table == nil {
			table = extractor.table
		} else if !equalTable(table, extractor.table) {
			return false
		}
		if !hasUniqueInWhereClause(dbSchema, table, update) {
			return false
		}
	}

	return true
}

func hasUniqueInWhereClause(dbSchema *storepb.DatabaseSchemaMetadata, table *tableRef, update plsql.IUpdate_statementContext) bool {
	if update.Where_clause() == nil || update.Where_clause().Expression() == nil {
		return false
	}

	if dbSchema == nil {
		return false
	}

	list := extractColumnsInEqualCondition(dbSchema.Name, table, update.Where_clause().Expression())
	columnMap := make(map[string]bool)
	for _, column := range list {
		columnMap[column] = true
	}

	for _, schema := range dbSchema.Schemas {
		for _, t := range schema.Tables {
			if t.Name == table.table {
				for _, index := range t.Indexes {
					if index.Unique || index.Primary {
						exists := true
						for _, column := range index.Expressions {
							if !columnMap[column] {
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

func extractColumnsInEqualCondition(databaseName string, table *tableRef, ctx antlr.ParserRuleContext) []string {
	if ctx == nil {
		return nil
	}

	switch n := ctx.(type) {
	case *plsql.ExpressionContext:
		return extractColumnsInEqualCondition(databaseName, table, n.Logical_expression())
	case *plsql.Logical_expressionContext:
		switch {
		case n.Unary_logical_expression() != nil:
			return extractColumnsInEqualCondition(databaseName, table, n.Unary_logical_expression())
		case n.AND() != nil:
			return append(extractColumnsInEqualCondition(databaseName, table, n.Logical_expression(0)), extractColumnsInEqualCondition(databaseName, table, n.Logical_expression(1))...)
		default:
			return nil
		}
	case *plsql.Unary_logical_expressionContext:
		if len(n.AllNOT()) > 0 {
			return nil
		}
		return extractColumnsInEqualCondition(databaseName, table, n.Multiset_expression())
	case *plsql.Multiset_expressionContext:
		if n.Concatenation() != nil {
			return nil
		}
		return extractColumnsInEqualCondition(databaseName, table, n.Relational_expression())
	case *plsql.Relational_expressionContext:
		if n.Compound_expression() != nil {
			return extractColumnsInEqualCondition(databaseName, table, n.Compound_expression())
		}
		if len(n.AllRelational_expression()) != 2 {
			return nil
		}
		if n.Relational_operator() != nil && n.Relational_operator().GetText() == "=" {
			if isConstant(n.Relational_expression(1)) {
				return extractColumnsInEqualCondition(databaseName, table, n.Relational_expression(0))
			}
			if isConstant(n.Relational_expression(0)) {
				return extractColumnsInEqualCondition(databaseName, table, n.Relational_expression(1))
			}
		}
		return nil
	case *plsql.Compound_expressionContext:
		switch {
		case n.NOT() != nil, n.IN() != nil, n.BETWEEN() != nil, n.LIKE() != nil:
			return nil
		default:
			return extractColumnsInEqualCondition(databaseName, table, n.Concatenation(0))
		}
	case *plsql.ConcatenationContext:
		if n.GetOp() != nil {
			return nil
		}
		if len(n.AllBAR()) > 0 {
			return nil
		}
		return extractColumnsInEqualCondition(databaseName, table, n.Model_expression())
	case *plsql.Model_expressionContext:
		if n.Model_expression_element() != nil {
			return nil
		}
		return extractColumnsInEqualCondition(databaseName, table, n.Unary_expression())
	case *plsql.Unary_expressionContext:
		if n.PLUS_SIGN() != nil || n.MINUS_SIGN() != nil {
			return extractColumnsInEqualCondition(databaseName, table, n.Unary_expression())
		}
		if n.Atom() != nil {
			return extractColumnsInEqualCondition(databaseName, table, n.Atom())
		}
		return nil
	case *plsql.AtomContext:
		return extractColumnsInEqualCondition(databaseName, table, n.General_element())
	case *plsql.General_elementContext:
		if table == nil {
			return nil
		}
		ids := plsqlparser.NormalizeGeneralElement(n)
		switch len(ids) {
		case 0:
			return nil
		case 1:
			return []string{ids[0]}
		case 2:
			if ids[0] != table.table && ids[0] != table.alias {
				return nil
			}
			return []string{ids[1]}
		default:
			// more than 2
			if ids[len(ids)-3] != table.database || (ids[len(ids)-2] != table.table && ids[len(ids)-2] != table.alias) {
				return nil
			}
			return []string{ids[len(ids)-1]}
		}
	}
	return nil
}

func isConstant(ctx antlr.ParserRuleContext) bool {
	if ctx == nil {
		return false
	}
	switch n := ctx.(type) {
	case *plsql.Relational_expressionContext:
		return isConstant(n.Compound_expression())
	case *plsql.Compound_expressionContext:
		switch {
		case n.NOT() != nil, n.IN() != nil, n.BETWEEN() != nil, n.LIKE() != nil:
			return false
		default:
			return isConstant(n.Concatenation(0))
		}
	case *plsql.ConcatenationContext:
		if n.GetOp() != nil {
			return false
		}
		if len(n.AllBAR()) > 0 {
			return false
		}
		return isConstant(n.Model_expression())
	case *plsql.Model_expressionContext:
		if n.Model_expression_element() != nil {
			return false
		}
		return isConstant(n.Unary_expression())
	case *plsql.Unary_expressionContext:
		if n.PLUS_SIGN() != nil || n.MINUS_SIGN() != nil {
			return isConstant(n.Unary_expression())
		}
		if n.Atom() != nil {
			return isConstant(n.Atom())
		}
		return false
	case *plsql.AtomContext:
		return n.Constant_without_variable() != nil
	}
	return false
}

func equalTable(a, b *tableRef) bool {
	return a.database == b.database && a.table == b.table
}

type tableExtractor struct {
	*plsql.BasePlSqlParserListener

	databaseName string
	table        *tableRef
}

func (e *tableExtractor) EnterGeneral_table_ref(ctx *plsql.General_table_refContext) {
	dmlTableExpr := ctx.Dml_table_expression_clause()
	if dmlTableExpr != nil && dmlTableExpr.Tableview_name() != nil {
		_, schemaName, tableName := plsqlparser.NormalizeTableViewName("", dmlTableExpr.Tableview_name())
		e.table = &tableRef{
			database: schemaName,
			table:    tableName,
		}
		if ctx.Table_alias() != nil {
			e.table.alias = plsqlparser.NormalizeTableAlias(ctx.Table_alias())
		}
	}
}

type statementDisallowMixDMLChecker struct {
	*plsql.BasePlSqlParserListener

	updateStatements []plsql.IUpdate_statementContext
	deleteStatements []plsql.IDelete_statementContext
	hasDDL           bool
}

func (l *statementDisallowMixDMLChecker) EnterUnit_statement(ctx *plsql.Unit_statementContext) {
	if dml := ctx.Data_manipulation_language_statements(); dml != nil {
		if update := dml.Update_statement(); update != nil {
			l.updateStatements = append(l.updateStatements, update)
		} else if d := dml.Delete_statement(); d != nil {
			l.deleteStatements = append(l.deleteStatements, d)
		}
	} else {
		l.hasDDL = true
	}
}

func extractDatabaseName(databaseUID string) string {
	segments := strings.Split(databaseUID, "/")
	return segments[len(segments)-1]
}
