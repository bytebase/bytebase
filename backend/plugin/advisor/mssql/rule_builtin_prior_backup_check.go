package mssql

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/omni/mssql/ast"
	parser "github.com/bytebase/parser/tsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
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
			Content:       fmt.Sprintf("The size of statements in the sheet exceeds the limit of %d", common.MaxSheetCheckSize),
			Code:          code.BuiltinPriorBackupCheck.Int32(),
			StartPosition: &storepb.Position{Line: 1},
		})
	}

	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_MSSQL)
	if !advisor.DatabaseExists(ctx, checkCtx, databaseName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("Need database %q to do prior backup but it does not exist", databaseName),
			Code:          code.DatabaseNotExists.Int32(),
			StartPosition: &storepb.Position{Line: 1},
		})
		return adviceList, nil
	}

	// Use omni rule for DDL detection.
	ddlRule := &statementDisallowMixDMLOmniRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: title},
	}
	RunOmniRules(checkCtx.ParsedStatements, []OmniRule{ddlRule})

	if ddlRule.hasDDL {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       "Prior backup cannot deal with mixed DDL and DML statements",
			Code:          int32(code.BuiltinPriorBackupCheck),
			StartPosition: &storepb.Position{Line: 1},
		})
	}

	statementInfoList := prepareTransformation(checkCtx.DBSchema.Name, checkCtx.ParsedStatements)

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
					StartPosition: &storepb.Position{Line: 1},
				})
				break
			}
		}
	}

	return adviceList, nil
}

// statementDisallowMixDMLOmniRule uses omni AST to detect DDL statements.
type statementDisallowMixDMLOmniRule struct {
	OmniBaseRule
	hasDDL bool
}

func (*statementDisallowMixDMLOmniRule) Name() string {
	return "StatementDisallowMixDMLOmniRule"
}

func (r *statementDisallowMixDMLOmniRule) OnStatement(node ast.Node) {
	if r.hasDDL {
		return
	}
	switch node.(type) {
	case *ast.CreateTableStmt,
		*ast.AlterTableStmt,
		*ast.DropStmt,
		*ast.CreateIndexStmt,
		*ast.CreateViewStmt,
		*ast.CreateFunctionStmt,
		*ast.CreateProcedureStmt,
		*ast.CreateSchemaStmt,
		*ast.CreateDatabaseStmt,
		*ast.CreateTriggerStmt,
		*ast.CreateTypeStmt,
		*ast.AlterIndexStmt,
		*ast.AlterSchemaStmt,
		*ast.AlterDatabaseStmt:
		r.hasDDL = true
	default:
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

func prepareTransformation(databaseName string, parsedStatements []base.ParsedStatement) []statementInfo {
	extractor := &dmlExtractor{
		databaseName: databaseName,
	}

	for _, stmt := range parsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(extractor, antlrAST.Tree)
	}

	return extractor.dmls
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
