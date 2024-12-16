package plsql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	maxTableNameLengthAfter12_2  = 128
	maxTableNameLengthBefore12_2 = 30
	maxMixedDMLCount             = 5
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
	if len(statementInfoList) <= maxMixedDMLCount {
		return generateSQLForMixedDML(ctx, statementInfoList, targetDatabase, tablePrefix)
	}
	return generateSQLForSingleTable(ctx, statementInfoList, targetDatabase, tablePrefix)
}

func generateSQLForSingleTable(ctx base.TransformContext, statementInfoList []statementInfo, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	table := statementInfoList[0].table

	for _, item := range statementInfoList {
		if !equalTable(table, item.table) {
			return nil, errors.Errorf("prior backup cannot handle statements on different tables more than %d", maxMixedDMLCount)
		}
	}

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
			if _, err := buf.WriteString("\n  UNION ALL\n"); err != nil {
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

	return []base.BackupStatement{
		{
			Statement:       buf.String(),
			SourceSchema:    table.Schema,
			SourceTableName: table.Table,
			TargetTableName: targetTable,
			StartPosition: &store.Position{
				Line:   int32(statementInfoList[0].tree.GetStart().GetLine()),
				Column: int32(statementInfoList[0].tree.GetStart().GetColumn()),
			},
			EndPosition: &store.Position{
				Line:   int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetLine()),
				Column: int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetColumn()),
			},
		},
	}, nil
}

func equalTable(a, b *TableReference) bool {
	if a == nil || b == nil {
		return false
	}

	return a.Database == b.Database && a.Schema == b.Schema && a.Table == b.Table
}

func generateSQLForMixedDML(ctx base.TransformContext, statementInfoList []statementInfo, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	var result []base.BackupStatement
	offsetLength := 1
	if len(statementInfoList) > 1 {
		offsetLength = getOffsetLength(statementInfoList[len(statementInfoList)-1].offset)
	}

	version, ok := ctx.Version.(*Version)
	if !ok {
		version = &Version{
			First:  11,
			Second: 0,
		}
	}

	for _, info := range statementInfoList {
		table := info.table
		targetTable := fmt.Sprintf("%s_%0*d_%s", tablePrefix, offsetLength, info.offset, table.Table)
		if version.GTE(&Version{First: 12, Second: 2}) {
			targetTable, _ = common.TruncateString(targetTable, maxTableNameLengthAfter12_2)
		} else {
			targetTable, _ = common.TruncateString(targetTable, maxTableNameLengthBefore12_2)
		}
		var buf strings.Builder
		if _, err := buf.WriteString(fmt.Sprintf(`CREATE TABLE "%s"."%s" AS SELECT `, targetDatabase, targetTable)); err != nil {
			return nil, errors.Wrap(err, "failed to write to buffer")
		}
		if table.Alias != "" {
			if _, err := buf.WriteString(fmt.Sprintf(`"%s".* FROM `, table.Alias)); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		} else {
			if table.HasSchema {
				if _, err := buf.WriteString(fmt.Sprintf(`"%s"."%s".* FROM `, table.Schema, table.Table)); err != nil {
					return nil, errors.Wrap(err, "failed to write to buffer")
				}
			} else {
				if _, err := buf.WriteString(fmt.Sprintf(`"%s".* FROM `, table.Table)); err != nil {
					return nil, errors.Wrap(err, "failed to write to buffer")
				}
			}
		}

		if err := writeSuffixSelectClause(&buf, info.tree); err != nil {
			return nil, errors.Wrap(err, "failed to write suffix select clause")
		}

		if _, err := buf.WriteString(";"); err != nil {
			return nil, errors.Wrap(err, "failed to write to buffer")
		}

		result = append(result, base.BackupStatement{
			Statement:       buf.String(),
			SourceSchema:    table.Schema,
			SourceTableName: table.Table,
			TargetTableName: targetTable,
			StartPosition: &store.Position{
				Line:   int32(info.tree.GetStart().GetLine()),
				Column: int32(info.tree.GetStart().GetColumn()),
			},
			EndPosition: &store.Position{
				Line:   int32(info.tree.GetStop().GetLine()),
				Column: int32(info.tree.GetStop().GetColumn()),
			},
		})
	}

	return result, nil
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

func getOffsetLength(total int) int {
	length := 1
	for {
		if total < 10 {
			return length
		}
		total /= 10
		length++
	}
}

func prepareTransformation(databaseName, statement string) ([]statementInfo, error) {
	tree, _, err := ParsePLSQL(statement)
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

		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     extractor.table,
		})
	}
}

func (e *dmlExtractor) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if IsTopLevelStatement(ctx.GetParent()) {
		extractor := &tableExtractor{
			databaseName: e.databaseName,
		}
		antlr.ParseTreeWalkerDefault.Walk(extractor, ctx)

		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     extractor.table,
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
