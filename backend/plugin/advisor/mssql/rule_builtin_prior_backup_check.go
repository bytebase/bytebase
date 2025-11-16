package mssql

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

const (
	// The default schema is 'dbo' for MSSQL.
	// TODO(zp): We should support default schema in the future.
	defaultSchema = "dbo"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.BuiltinRulePriorBackupCheck, &StatementPriorBackupCheckAdvisor{})
}

type StatementPriorBackupCheckAdvisor struct {
}

func (*StatementPriorBackupCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup || checkCtx.ChangeType != storepb.PlanCheckRunConfig_DML {
		return nil, nil
	}

	var adviceList []*storepb.Advice
	stmtList, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(checkCtx.Rule.Type)

	if len(checkCtx.Statements) > common.MaxSheetCheckSize {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("The size of statements in the sheet exceeds the limit of %d", common.MaxSheetCheckSize),
			Code:          code.BuiltinPriorBackupCheck.Int32(),
			StartPosition: common.ConvertANTLRLineToPosition(1),
		})
	}

	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_MSSQL)
	if !advisor.DatabaseExists(ctx, checkCtx, databaseName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("Need database %q to do prior backup but it does not exist", databaseName),
			Code:          code.DatabaseNotExists.Int32(),
			StartPosition: common.ConvertANTLRLineToPosition(1),
		})
		return adviceList, nil
	}

	// Use the refactored rule for DDL/DML checking
	rule := NewStatementDisallowMixDMLRule(level, title)
	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, stmtList)

	if rule.hasDDL {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       "Prior backup cannot deal with mixed DDL and DML statements",
			Code:          int32(code.BuiltinPriorBackupCheck),
			StartPosition: common.ConvertANTLRLineToPosition(1),
		})
	}

	statementInfoList, err := prepareTransformation(checkCtx.DBSchema.Name, checkCtx.Statements)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare transformation")
	}

	groupByTable := make(map[string][]statementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s.%s", item.table.Database, item.table.Schema, item.table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	// Check if the statement type is the same for all statements on the same table.
	for key, list := range groupByTable {
		statementType := StatementTypeUnknown
		for _, item := range list {
			if statementType == StatementTypeUnknown {
				statementType = item.table.StatementType
			}
			if statementType != item.table.StatementType {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("The statement type is not the same for all statements on the same table %q", key),
					Code:          code.BuiltinPriorBackupCheck.Int32(),
					StartPosition: common.ConvertANTLRLineToPosition(1),
				})
				break
			}
		}
	}

	return adviceList, nil
}

// StatementDisallowMixDMLRule is the rule for checking mixed DDL and DML statements.
type StatementDisallowMixDMLRule struct {
	BaseRule
	updateStatements []*parser.Update_statementContext
	deleteStatements []*parser.Delete_statementContext
	hasDDL           bool
}

// NewStatementDisallowMixDMLRule creates a new StatementDisallowMixDMLRule.
func NewStatementDisallowMixDMLRule(level storepb.Advice_Status, title string) *StatementDisallowMixDMLRule {
	return &StatementDisallowMixDMLRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*StatementDisallowMixDMLRule) Name() string {
	return "StatementDisallowMixDMLRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementDisallowMixDMLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Ddl_clause":
		r.enterDDLClause(ctx.(*parser.Ddl_clauseContext))
	case NodeTypeUpdateStatement:
		r.enterUpdateStatement(ctx.(*parser.Update_statementContext))
	case NodeTypeDeleteStatement:
		r.enterDeleteStatement(ctx.(*parser.Delete_statementContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementDisallowMixDMLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *StatementDisallowMixDMLRule) enterDDLClause(_ *parser.Ddl_clauseContext) {
	r.hasDDL = true
}

func (r *StatementDisallowMixDMLRule) enterUpdateStatement(ctx *parser.Update_statementContext) {
	if tsqlparser.IsTopLevel(ctx.GetParent()) {
		r.updateStatements = append(r.updateStatements, ctx)
	}
}

func (r *StatementDisallowMixDMLRule) enterDeleteStatement(ctx *parser.Delete_statementContext) {
	if tsqlparser.IsTopLevel(ctx.GetParent()) {
		r.deleteStatements = append(r.deleteStatements, ctx)
	}
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
	parseResult, err := tsqlparser.ParseTSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}

	extractor := &dmlExtractor{
		databaseName: databaseName,
	}
	antlr.ParseTreeWalkerDefault.Walk(extractor, parseResult.Tree)
	return extractor.dmls, nil
}

type dmlExtractor struct {
	*parser.BaseTSqlParserListener

	databaseName string
	dmls         []statementInfo
	offset       int
}

func IsTopLevel(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}
	switch ctx := ctx.(type) {
	case *parser.Dml_clauseContext,
		*parser.Sql_clausesContext,
		*parser.Batch_without_goContext:
		return IsTopLevel(ctx.GetParent())
	case *parser.Tsql_fileContext:
		return true
	default:
		return false
	}
}

func (e *dmlExtractor) ExitBatch(ctx *parser.Batch_without_goContext) {
	if len(ctx.AllSql_clauses()) == 0 {
		e.offset++
	}
}

func (e *dmlExtractor) ExitSql_clauses(ctx *parser.Sql_clausesContext) {
	if IsTopLevel(ctx.GetParent()) {
		e.offset++
	}
}

func (e *dmlExtractor) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if IsTopLevel(ctx.GetParent()) && ctx.Ddl_object() != nil {
		extractor := &tableExtractor{
			databaseName: e.databaseName,
		}
		antlr.ParseTreeWalkerDefault.Walk(extractor, ctx.Ddl_object())

		table := extractor.table
		if extractor.table != nil && ctx.Table_sources() != nil && table.Database == e.databaseName && table.Schema == defaultSchema {
			table = extractPhysicalTable(ctx.Table_sources(), extractor.table)
		}
		table.StatementType = StatementTypeUpdate
		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
		})
	}
}

func (e *dmlExtractor) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if IsTopLevel(ctx.GetParent()) {
		extractor := &tableExtractor{
			databaseName: e.databaseName,
		}
		antlr.ParseTreeWalkerDefault.Walk(extractor, ctx.Delete_statement_from())

		table := extractor.table
		if extractor.table != nil && ctx.From_table_sources() != nil && table.Database == e.databaseName && table.Schema == defaultSchema {
			table = extractPhysicalTable(ctx.From_table_sources().Table_sources(), extractor.table)
		}
		table.StatementType = StatementTypeDelete
		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
		})
	}
}

func extractPhysicalTable(ctx antlr.Tree, table *TableReference) *TableReference {
	if ctx == nil || table == nil {
		return table
	}

	extractor := &physicalTableExtractor{
		table: table,
	}
	antlr.ParseTreeWalkerDefault.Walk(extractor, ctx)
	if extractor.result != nil {
		return extractor.result
	}
	return table
}

type physicalTableExtractor struct {
	*parser.BaseTSqlParserListener

	table  *TableReference
	result *TableReference
}

func (e *physicalTableExtractor) EnterTable_source_item(ctx *parser.Table_source_itemContext) {
	if ctx.As_table_alias() != nil && ctx.Full_table_name() != nil {
		alias := unquote(ctx.As_table_alias().Table_alias().GetText())
		if alias == e.table.Table {
			databaseName, schemaName, tableName := extractFullTableName(ctx.Full_table_name(), e.table.Database, e.table.Schema)
			e.result = &TableReference{
				Database:      databaseName,
				Schema:        schemaName,
				Table:         tableName,
				Alias:         alias,
				StatementType: e.table.StatementType,
			}
		}
	}
}

func unquote(name string) string {
	if len(name) < 2 {
		return name
	}
	if name[0] == '[' && name[len(name)-1] == ']' {
		return name[1 : len(name)-1]
	}

	if len(name) > 3 && name[0] == 'N' && name[1] == '\'' && name[len(name)-1] == '\'' {
		return name[2 : len(name)-1]
	}
	return name
}

type tableExtractor struct {
	*parser.BaseTSqlParserListener

	databaseName string
	table        *TableReference
}

func (e *tableExtractor) EnterFull_table_name(ctx *parser.Full_table_nameContext) {
	databaseName, schemaName, tableName := extractFullTableName(ctx, e.databaseName, defaultSchema)
	table := TableReference{
		Database: databaseName,
		Schema:   schemaName,
		Table:    tableName,
	}
	e.table = &table
}

func extractFullTableName(ctx parser.IFull_table_nameContext, defaultDatabase string, defaultSchema string) (string, string, string) {
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
