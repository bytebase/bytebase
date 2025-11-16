package oracle

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.BuiltinRulePriorBackupCheck, &StatementPriorBackupCheckAdvisor{})
}

type StatementPriorBackupCheckAdvisor struct {
}

func (*StatementPriorBackupCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup || checkCtx.ChangeType != storepb.PlanCheckRunConfig_DML {
		return nil, nil
	}

	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewStatementPriorBackupCheckRule(ctx, level, string(checkCtx.Rule.Type), checkCtx)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
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
	results, err := plsqlparser.ParsePLSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse PLSQL")
	}
	if len(results) == 0 {
		return nil, errors.New("no parse results")
	}

	extractor := &dmlExtractor{
		databaseName: databaseName,
	}

	// Walk each parse result tree to extract DML statements
	for _, result := range results {
		antlr.ParseTreeWalkerDefault.Walk(extractor, result.Tree)
	}

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

// StatementPriorBackupCheckRule is the rule implementation for prior backup checks.
type StatementPriorBackupCheckRule struct {
	BaseRule

	ctx      context.Context
	checkCtx advisor.Context

	updateStatements []plsql.IUpdate_statementContext
	deleteStatements []plsql.IDelete_statementContext
	hasDDL           bool
}

// NewStatementPriorBackupCheckRule creates a new StatementPriorBackupCheckRule.
func NewStatementPriorBackupCheckRule(ctx context.Context, level storepb.Advice_Status, title string, checkCtx advisor.Context) *StatementPriorBackupCheckRule {
	return &StatementPriorBackupCheckRule{
		BaseRule: NewBaseRule(level, title, 0),
		ctx:      ctx,
		checkCtx: checkCtx,
	}
}

// Name returns the rule name.
func (*StatementPriorBackupCheckRule) Name() string {
	return "builtin.prior-backup-check"
}

// OnEnter is called when the parser enters a rule context.
func (r *StatementPriorBackupCheckRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Unit_statement" {
		r.handleUnitStatement(ctx.(*plsql.Unit_statementContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (r *StatementPriorBackupCheckRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Sql_script" {
		r.handleSQLScriptExit()
	}
	return nil
}

func (r *StatementPriorBackupCheckRule) handleUnitStatement(ctx *plsql.Unit_statementContext) {
	if dml := ctx.Data_manipulation_language_statements(); dml != nil {
		if update := dml.Update_statement(); update != nil {
			r.updateStatements = append(r.updateStatements, update)
		} else if d := dml.Delete_statement(); d != nil {
			r.deleteStatements = append(r.deleteStatements, d)
		}
	} else {
		r.hasDDL = true
	}
}

func (r *StatementPriorBackupCheckRule) handleSQLScriptExit() {
	var adviceList []*storepb.Advice

	if len(r.checkCtx.Statements) > common.MaxSheetCheckSize {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        r.level,
			Title:         r.title,
			Content:       fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %d bytes for backup", common.MaxSheetCheckSize),
			Code:          code.BuiltinPriorBackupCheck.Int32(),
			StartPosition: nil,
		})
	}

	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_ORACLE)
	if !advisor.DatabaseExists(r.ctx, r.checkCtx, databaseName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        r.level,
			Title:         r.title,
			Content:       fmt.Sprintf("Need database %q to do prior backup but it does not exist", databaseName),
			Code:          code.DatabaseNotExists.Int32(),
			StartPosition: nil,
		})
		r.adviceList = append(r.adviceList, adviceList...)
		return
	}

	if r.hasDDL {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        r.level,
			Title:         r.title,
			Content:       "Prior backup cannot deal with mixed DDL and DML statements",
			Code:          int32(code.BuiltinPriorBackupCheck),
			StartPosition: nil,
		})
	}

	statementInfoList, err := prepareTransformation(r.checkCtx.DBSchema.Name, r.checkCtx.Statements)
	if err != nil {
		return
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
					Status:        r.level,
					Title:         r.title,
					Content:       fmt.Sprintf("Prior backup cannot handle mixed DML statements on the same table %q", key),
					Code:          code.BuiltinPriorBackupCheck.Int32(),
					StartPosition: nil,
				})
				break
			}
		}
	}

	r.adviceList = append(r.adviceList, adviceList...)
}
