package plsql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	maxTableNameLengthAfter12_2  = 128
	maxTableNameLengthBefore12_2 = 30
)

func init() {
	base.RegisterTransformDMLToSelect(store.Engine_ORACLE, TransformDMLToSelect)
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
	baseLine  int
}

// TransformDMLToSelect transforms DML statement to SELECT statement.
// For Oracle, we only consider the managed on schema mode.
func TransformDMLToSelect(_ context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(sourceDatabase, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}

	return generateSQL(tCtx, statementInfoList, targetDatabase, tablePrefix)
}

func generateSQL(ctx base.TransformContext, statementInfoList []statementInfo, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
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
				return nil, errors.Errorf("prior backup cannot handle statements with different types on the same table: %s", key)
			}
		}
	}

	var result []base.BackupStatement
	for key, list := range groupByTable {
		backupStatement, err := generateSQLForTable(ctx, list, targetDatabase, tablePrefix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate SQL for table: %s", key)
		}
		result = append(result, *backupStatement)
	}

	slices.SortFunc(result, func(i, j base.BackupStatement) int {
		if i.StartPosition.Line != j.StartPosition.Line {
			if i.StartPosition.Line < j.StartPosition.Line {
				return -1
			}
			return 1
		}
		if i.StartPosition.Column != j.StartPosition.Column {
			if i.StartPosition.Column < j.StartPosition.Column {
				return -1
			}
			return 1
		}
		if i.SourceTableName < j.SourceTableName {
			return -1
		}
		if i.SourceTableName > j.SourceTableName {
			return 1
		}
		return 0
	})

	return result, nil
}

func generateSQLForTable(ctx base.TransformContext, statementInfoList []statementInfo, targetDatabase string, tablePrefix string) (*base.BackupStatement, error) {
	table := statementInfoList[0].table

	version, ok := ctx.Version.(*Version)
	if !ok {
		version = &Version{
			First:  11,
			Second: 0,
		}
	}

	targetTable := fmt.Sprintf("%s_%s_%s", tablePrefix, table.Table, table.Schema)
	if version.GTE(&Version{First: 12, Second: 2}) {
		targetTable, _ = common.TruncateString(targetTable, maxTableNameLengthAfter12_2)
	} else {
		targetTable, _ = common.TruncateString(targetTable, maxTableNameLengthBefore12_2)
	}

	var buf strings.Builder
	if _, err := buf.WriteString(fmt.Sprintf(`CREATE TABLE "%s"."%s" AS`+"\n", targetDatabase, targetTable)); err != nil {
		return nil, errors.Wrap(err, "failed to write to buffer")
	}
	for i, info := range statementInfoList {
		if i != 0 {
			if _, err := buf.WriteString("\n  UNION\n"); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		}
		t := info.table
		if t.Alias != "" {
			if _, err := buf.WriteString(fmt.Sprintf(`  SELECT "%s".* FROM `, t.Alias)); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		} else {
			if t.HasSchema {
				if _, err := buf.WriteString(fmt.Sprintf(`  SELECT "%s"."%s".* FROM `, t.Schema, t.Table)); err != nil {
					return nil, errors.Wrap(err, "failed to write to buffer")
				}
			} else {
				if _, err := buf.WriteString(fmt.Sprintf(`  SELECT "%s".* FROM `, t.Table)); err != nil {
					return nil, errors.Wrap(err, "failed to write to buffer")
				}
			}
		}
		if err := writeSuffixSelectClause(&buf, info.tree); err != nil {
			return nil, errors.Wrap(err, "failed to write suffix select clause")
		}
	}

	if _, err := buf.WriteString(";"); err != nil {
		return nil, errors.Wrap(err, "failed to write to buffer")
	}

	return &base.BackupStatement{
		Statement:       buf.String(),
		SourceSchema:    table.Schema,
		SourceTableName: table.Table,
		TargetTableName: targetTable,
		StartPosition: &store.Position{
			Line:   int32(statementInfoList[0].tree.GetStart().GetLine() + statementInfoList[0].baseLine),
			Column: int32(statementInfoList[0].tree.GetStart().GetColumn()),
		},
		EndPosition: &store.Position{
			Line:   int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetLine() + statementInfoList[len(statementInfoList)-1].baseLine),
			Column: int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetColumn()),
		},
	}, nil
}

func writeSuffixSelectClause(buf *strings.Builder, tree antlr.Tree) error {
	extractor := &suffixSelectClauseExtractor{
		buf: buf,
	}
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree)
	return extractor.err
}

type suffixSelectClauseExtractor struct {
	*parser.BasePlSqlParserListener

	buf *strings.Builder
	err error
}

func (e *suffixSelectClauseExtractor) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if e.err != nil || !IsTopLevelStatement(ctx.GetParent()) {
		return
	}

	if _, err := e.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.General_table_ref())); err != nil {
		e.err = errors.Wrap(err, "failed to write to buffer")
		return
	}

	if ctx.Where_clause() != nil {
		if _, err := e.buf.WriteString(" "); err != nil {
			e.err = errors.Wrap(err, "failed to write to buffer")
			return
		}
		if _, err := e.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Where_clause())); err != nil {
			e.err = errors.Wrap(err, "failed to write to buffer")
			return
		}
	}
}

func (e *suffixSelectClauseExtractor) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if e.err != nil || !IsTopLevelStatement(ctx.GetParent()) {
		return
	}

	if _, err := e.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.General_table_ref())); err != nil {
		e.err = errors.Wrap(err, "failed to write to buffer")
		return
	}

	if ctx.Where_clause() != nil {
		if _, err := e.buf.WriteString(" "); err != nil {
			e.err = errors.Wrap(err, "failed to write to buffer")
			return
		}
		if _, err := e.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Where_clause())); err != nil {
			e.err = errors.Wrap(err, "failed to write to buffer")
			return
		}
	}
}

func prepareTransformation(databaseName, statement string) ([]statementInfo, error) {
	results, err := ParsePLSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse PLSQL")
	}
	if len(results) == 0 {
		return nil, errors.New("no parse results")
	}

	extractor := &dmlExtractor{
		databaseName: databaseName,
	}

	// Walk each ANTLRAST tree to extract DML statements
	for _, result := range results {
		extractor.baseLine = base.GetLineOffset(result.StartPosition)
		antlr.ParseTreeWalkerDefault.Walk(extractor, result.Tree)
	}

	return extractor.dmls, nil
}

func IsTopLevelStatement(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}
	switch ctx := ctx.(type) {
	case *parser.Unit_statementContext, *parser.Sql_scriptContext:
		return true
	case *parser.Data_manipulation_language_statementsContext:
		return IsTopLevelStatement(ctx.GetParent())
	default:
		return false
	}
}

type dmlExtractor struct {
	*parser.BasePlSqlParserListener

	databaseName string
	dmls         []statementInfo
	offset       int
	baseLine     int
}

func (e *dmlExtractor) ExitUnit_statement(_ *parser.Unit_statementContext) {
	e.offset++
}

func (e *dmlExtractor) ExitSql_plus_command(_ *parser.Sql_plus_commandContext) {
	e.offset++
}

func (e *dmlExtractor) EnterDelete_statement(ctx *parser.Delete_statementContext) {
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
			baseLine:  e.baseLine,
		})
	}
}

func (e *dmlExtractor) EnterUpdate_statement(ctx *parser.Update_statementContext) {
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
			baseLine:  e.baseLine,
		})
	}
}

type tableExtractor struct {
	*parser.BasePlSqlParserListener

	databaseName string
	table        *TableReference
}

func (e *tableExtractor) EnterGeneral_table_ref(ctx *parser.General_table_refContext) {
	dmlTableExpr := ctx.Dml_table_expression_clause()
	if dmlTableExpr != nil && dmlTableExpr.Tableview_name() != nil {
		_, schemaName, tableName := NormalizeTableViewName("", dmlTableExpr.Tableview_name())
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
			e.table.Alias = NormalizeTableAlias(ctx.Table_alias())
		}
	}
}
