package mssql

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	tsql "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// The default schema is 'dbo' for MSSQL.
	// TODO(zp): We should support default schema in the future.
	defaultSchema    = "dbo"
	maxMixedDMLCount = 5
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLBuiltinPriorBackupCheck, &StatementPriorBackupCheckAdvisor{})
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
			Content: fmt.Sprintf("Prior backup is feasible only with up to %d statements that are either UPDATE or DELETE, or if all UPDATEs target the same table with a PRIMARY or UNIQUE KEY in the WHERE clause", maxMixedDMLCount),
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
	schema   string
	table    string
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

func hasUniqueInWhereClause(dbSchema *storepb.DatabaseSchemaMetadata, table *tableRef, update *tsql.Update_statementContext) bool {
	if update.Search_condition() == nil {
		return false
	}

	if dbSchema == nil {
		return false
	}

	list := extractColumnsInEqualCondition(dbSchema.Name, table, update.Search_condition())
	columnMap := make(map[string]bool)
	for _, column := range list {
		columnMap[strings.ToLower(column)] = true
	}

	for _, schema := range dbSchema.Schemas {
		if !strings.EqualFold(schema.Name, table.schema) {
			continue
		}
		for _, t := range schema.Tables {
			if !strings.EqualFold(t.Name, table.table) {
				continue
			}
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

	return false
}

func extractColumnsInEqualCondition(database string, table *tableRef, ctx antlr.ParserRuleContext) []string {
	if ctx == nil {
		return nil
	}

	switch n := ctx.(type) {
	case *tsql.Search_conditionContext:
		switch {
		case n.AND() != nil:
			return append(extractColumnsInEqualCondition(database, table, n.Search_condition(0)), extractColumnsInEqualCondition(database, table, n.Search_condition(1))...)
		case n.OR() != nil:
			return nil
		case len(n.AllNOT()) > 0:
			return nil
		default:
			if n.Predicate() != nil {
				return extractColumnsInEqualCondition(database, table, n.Predicate())
			}
			if len(n.AllSearch_condition()) == 1 {
				return extractColumnsInEqualCondition(database, table, n.Search_condition(0))
			}
		}
	case *tsql.PredicateContext:
		if n.Comparison_operator() == nil {
			return nil
		}
		if n.Comparison_operator().GetText() != "=" {
			return nil
		}
		if n.Subquery() != nil {
			return nil
		}
		if len(n.AllExpression()) != 2 {
			return nil
		}
		if isConstant(n.Expression(0)) {
			return extractColumnsInEqualCondition(database, table, n.Expression(1))
		}
		if isConstant(n.Expression(1)) {
			return extractColumnsInEqualCondition(database, table, n.Expression(0))
		}
	case *tsql.ExpressionContext:
		return extractColumnsInEqualCondition(database, table, n.Full_column_name())
	case *tsql.Full_column_nameContext:
		if n.Full_table_name() != nil {
			databaseName, schemaName, tableName := extractFullTableName(n.Full_table_name(), database, defaultSchema)
			if !equalTable(table, &tableRef{
				database: databaseName,
				schema:   schemaName,
				table:    tableName,
			}) {
				return nil
			}
			if n.Id_() == nil {
				return nil
			}
			_, columnName := tsqlparser.NormalizeTSQLIdentifier(n.Id_())
			return []string{columnName}
		}
	}

	return nil
}

func isConstant(ctx antlr.ParserRuleContext) bool {
	if ctx == nil {
		return false
	}
	switch n := ctx.(type) {
	case *tsql.ExpressionContext:
		return isConstant(n.Primitive_expression())
	case *tsql.Primitive_expressionContext:
		return isConstant(n.Primitive_constant())
	case *tsql.Primitive_constantContext:
		return true
	case *tsql.Unary_operator_expressionContext:
		return isConstant(n.Expression())
	}
	return false
}

func equalTable(a, b *tableRef) bool {
	if a == nil || b == nil {
		return false
	}

	return a.database == b.database && a.schema == b.schema && a.table == b.table
}

type tableExtractor struct {
	*tsql.BaseTSqlParserListener

	databaseName string
	table        *tableRef
}

func (e *tableExtractor) EnterFull_table_name(ctx *tsql.Full_table_nameContext) {
	databaseName, schemaName, tableName := extractFullTableName(ctx, e.databaseName, defaultSchema)
	table := tableRef{
		database: databaseName,
		schema:   schemaName,
		table:    tableName,
	}
	e.table = &table
}

func extractFullTableName(ctx tsql.IFull_table_nameContext, defaultDatabase string, defaultSchema string) (string, string, string) {
	name, err := tsqlparser.NormalizeFullTableName(ctx)
	if err != nil {
		slog.Debug("Failed to normalize full table name", "error", err)
		return defaultDatabase, defaultSchema, ""
	}
	schemaName := defaultSchema
	if name.Schema != "" {
		schemaName = name.Schema
	}
	databaseName := defaultDatabase
	if name.Database != "" {
		databaseName = name.Database
	}
	return databaseName, schemaName, name.Table
}

type statementDisallowMixDMLChecker struct {
	*tsql.BaseTSqlParserListener

	updateStatements []*tsql.Update_statementContext
	deleteStatements []*tsql.Delete_statementContext
	hasDDL           bool
}

func (l *statementDisallowMixDMLChecker) EnterDdl_clause(_ *tsql.Ddl_clauseContext) {
	l.hasDDL = true
}

func (l *statementDisallowMixDMLChecker) EnterUpdate_statement(ctx *tsql.Update_statementContext) {
	if tsqlparser.IsTopLevel(ctx.GetParent()) {
		l.updateStatements = append(l.updateStatements, ctx)
	}
}

func (l *statementDisallowMixDMLChecker) EnterDelete_statement(ctx *tsql.Delete_statementContext) {
	if tsqlparser.IsTopLevel(ctx.GetParent()) {
		l.deleteStatements = append(l.deleteStatements, ctx)
	}
}

func extractDatabaseName(databaseUID string) string {
	segments := strings.Split(databaseUID, "/")
	return segments[len(segments)-1]
}
