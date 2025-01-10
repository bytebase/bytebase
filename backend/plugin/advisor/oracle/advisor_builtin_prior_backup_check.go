package oracle

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	plsql "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

	databaseName := extractDatabaseName(ctx.PreUpdateBackupDetail.Database)
	if !advisor.DatabaseExists(ctx, databaseName) {
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

	statementInfoList, err := prepareTransformation(ctx.DBSchema.Name, ctx.Statements)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}

	groupByTable := make(map[string][]statementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s", item.table.Schema, item.table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	// Check if the statement type is the same for all statements in the group.
	for key, list := range groupByTable {
		statementType := StatementTypeUnknown
		for _, item := range list {
			if statementType == StatementTypeUnknown {
				statementType = item.table.StatementType
			}
			if statementType != item.table.StatementType {
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

type StatementType int

const (
	StatementTypeUnknown StatementType = iota
	StatementTypeUpdate
	StatementTypeInsert
	StatementTypeDelete
)

type TableReference struct {
	Database      string
	HasSchema     bool
	Schema        string
	Table         string
	Alias         string
	StatementType StatementType
}

type statementInfo struct {
	offset    int
	statement string
	tree      antlr.ParserRuleContext
	table     *TableReference
}

func prepareTransformation(databaseName, statement string) ([]statementInfo, error) {
	tree, _, err := plsqlparser.ParsePLSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse PLSQL")
	}

	extractor := &dmlExtractor{
		databaseName: databaseName,
	}
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree)
	return extractor.dmls, nil
}

func IsTopLevelStatement(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}
	switch ctx := ctx.(type) {
	case *plsql.Unit_statementContext, *plsql.Sql_scriptContext:
		return true
	case *plsql.Data_manipulation_language_statementsContext:
		return IsTopLevelStatement(ctx.GetParent())
	default:
		return false
	}
}

type dmlExtractor struct {
	*plsql.BasePlSqlParserListener

	databaseName string
	dmls         []statementInfo
	offset       int
}

func (e *dmlExtractor) ExitUnit_statement(_ *plsql.Unit_statementContext) {
	e.offset++
}

func (e *dmlExtractor) ExitSql_plus_command(_ *plsql.Sql_plus_commandContext) {
	e.offset++
}

func (e *dmlExtractor) EnterDelete_statement(ctx *plsql.Delete_statementContext) {
	if IsTopLevelStatement(ctx.GetParent()) {
		extractor := &tableExtractor{
			databaseName: e.databaseName,
		}
		antlr.ParseTreeWalkerDefault.Walk(extractor, ctx)
		extractor.table.StatementType = StatementTypeDelete

		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     extractor.table,
		})
	}
}

func (e *dmlExtractor) EnterUpdate_statement(ctx *plsql.Update_statementContext) {
	if IsTopLevelStatement(ctx.GetParent()) {
		extractor := &tableExtractor{
			databaseName: e.databaseName,
		}
		antlr.ParseTreeWalkerDefault.Walk(extractor, ctx)
		extractor.table.StatementType = StatementTypeUpdate

		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     extractor.table,
		})
	}
}

type tableExtractor struct {
	*plsql.BasePlSqlParserListener

	databaseName string
	table        *TableReference
}

func (e *tableExtractor) EnterGeneral_table_ref(ctx *plsql.General_table_refContext) {
	dmlTableExpr := ctx.Dml_table_expression_clause()
	if dmlTableExpr != nil && dmlTableExpr.Tableview_name() != nil {
		_, schemaName, tableName := plsqlparser.NormalizeTableViewName("", dmlTableExpr.Tableview_name())
		e.table = &TableReference{
			Database:  schemaName,
			HasSchema: true,
			Schema:    schemaName,
			Table:     tableName,
		}
		if schemaName == "" {
			e.table.Schema = e.databaseName
			e.table.HasSchema = false
		}
		if ctx.Table_alias() != nil {
			e.table.Alias = plsqlparser.NormalizeTableAlias(ctx.Table_alias())
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
