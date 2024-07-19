package tsql

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_MSSQL, TransformDMLToSelect)
}

const (
	// The default schema is 'dbo' for MSSQL.
	// TODO(zp): We should support default schema in the future.
	defaultSchema      = "dbo"
	maxTableNameLength = 128
)

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
	tree      antlr.Tree
	table     *TableReference
	line      int
}

func TransformDMLToSelect(_ base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(sourceDatabase, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}

	return generateSQL(statementInfoList, targetDatabase, tablePrefix)
}

func generateSQL(statementInfoList []statementInfo, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	var result []base.BackupStatement
	offsetLength := 1
	if len(statementInfoList) > 1 {
		offsetLength = base.GetOffsetLength(statementInfoList[len(statementInfoList)-1].offset)
	}
	for _, statementInfo := range statementInfoList {
		table := statementInfo.table
		targetTable := fmt.Sprintf("%s_%0*d_%s", tablePrefix, offsetLength, statementInfo.offset, table.Table)
		targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)
		topClause, fromClause, err := extractSuffixSelectStatement(statementInfo.tree)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract suffix select statement")
		}
		var buf strings.Builder
		if len(table.Alias) == 0 {
			if _, err := buf.WriteString(fmt.Sprintf(`SELECT "%s"."%s"."%s".* `, table.Database, table.Schema, table.Table)); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
		} else {
			if _, err := buf.WriteString(fmt.Sprintf(`SELECT "%s".* `, table.Alias)); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
		}
		if len(topClause) > 0 {
			if _, err := buf.WriteString(topClause); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
			if _, err := buf.WriteString(" "); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
		}
		if _, err := buf.WriteString(fmt.Sprintf(`INTO "%s"."%s"."%s" `, targetDatabase, defaultSchema, targetTable)); err != nil {
			return nil, errors.Wrap(err, "failed to write buffer")
		}
		if len(fromClause) > 0 {
			if _, err := buf.WriteString(fromClause); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
		}
		if _, err := buf.WriteString(";"); err != nil {
			return nil, errors.Wrap(err, "failed to write buffer")
		}
		result = append(result, base.BackupStatement{
			Statement:    buf.String(),
			TableName:    targetTable,
			OriginalLine: statementInfo.line,
		})
	}
	return result, nil
}

func extractSuffixSelectStatement(tree antlr.Tree) (string, string, error) {
	extractor := &suffixSelectStatementExtractor{}
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree)
	return extractor.topClause, extractor.fromClause, extractor.err
}

type suffixSelectStatementExtractor struct {
	*parser.BaseTSqlParserListener

	topClause  string
	fromClause string
	err        error
}

func (e *suffixSelectStatementExtractor) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if e.err != nil {
		return
	}

	if IsTopLevel(ctx.GetParent()) && ctx.Ddl_object() != nil {
		if ctx.CURRENT() != nil {
			e.err = errors.New("UPDATE statement with CURSOR clause is not supported")
			return
		}

		if ctx.TOP() != nil {
			if ctx.PERCENT() != nil {
				e.topClause = ctx.GetParser().GetTokenStream().GetTextFromTokens(ctx.TOP().GetSymbol(), ctx.PERCENT().GetSymbol())
			} else {
				e.topClause = ctx.GetParser().GetTokenStream().GetTextFromTokens(ctx.TOP().GetSymbol(), ctx.RR_BRACKET().GetSymbol())
			}
		}

		var buf strings.Builder
		if _, err := buf.WriteString("FROM "); err != nil {
			e.err = errors.Wrap(err, "failed to write buffer")
			return
		}
		if ctx.Table_sources() == nil {
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Ddl_object())); err != nil {
				e.err = errors.Wrap(err, "failed to write buffer")
				return
			}
		} else {
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Table_sources())); err != nil {
				e.err = errors.Wrap(err, "failed to write buffer")
				return
			}
		}
		if _, err := buf.WriteString(" "); err != nil {
			e.err = errors.Wrap(err, "failed to write buffer")
			return
		}
		var start, stop int
		if ctx.WHERE() != nil {
			start = ctx.WHERE().GetSymbol().GetTokenIndex()
		} else if ctx.For_clause() != nil {
			start = ctx.For_clause().GetStart().GetTokenIndex()
		} else if ctx.Option_clause() != nil {
			start = ctx.Option_clause().GetStart().GetTokenIndex()
		} else if ctx.SEMI() != nil {
			start = ctx.SEMI().GetSymbol().GetTokenIndex()
		} else {
			return
		}

		if ctx.SEMI() != nil {
			stop = ctx.SEMI().GetSymbol().GetTokenIndex() - 1
		} else {
			stop = ctx.GetStop().GetTokenIndex()
		}
		if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.NewInterval(start, stop))); err != nil {
			e.err = errors.Wrap(err, "failed to write buffer")
			return
		}
		e.fromClause = buf.String()
	}
}

func (e *suffixSelectStatementExtractor) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if e.err != nil {
		return
	}

	if IsTopLevel(ctx.GetParent()) {
		if ctx.CURRENT() != nil {
			e.err = errors.New("DELETE statement with CURSOR clause is not supported")
			return
		}

		if ctx.TOP() != nil {
			if ctx.DECIMAL() != nil {
				e.topClause = "TOP DECIMAL"
			} else {
				if ctx.PERCENT() != nil {
					e.topClause = ctx.GetParser().GetTokenStream().GetTextFromTokens(ctx.TOP().GetSymbol(), ctx.PERCENT().GetSymbol())
				} else {
					e.topClause = ctx.GetParser().GetTokenStream().GetTextFromTokens(ctx.TOP().GetSymbol(), ctx.RR_BRACKET().GetSymbol())
				}
			}
		}

		var buf strings.Builder
		if _, err := buf.WriteString("FROM "); err != nil {
			e.err = errors.Wrap(err, "failed to write buffer")
			return
		}
		if ctx.From_table_sources() == nil {
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Delete_statement_from())); err != nil {
				e.err = errors.Wrap(err, "failed to write buffer")
				return
			}
		} else {
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.From_table_sources().Table_sources())); err != nil {
				e.err = errors.Wrap(err, "failed to write buffer")
				return
			}
		}

		if _, err := buf.WriteString(" "); err != nil {
			e.err = errors.Wrap(err, "failed to write buffer")
			return
		}
		var start, stop int
		if ctx.WHERE() != nil {
			start = ctx.WHERE().GetSymbol().GetTokenIndex()
		} else if ctx.For_clause() != nil {
			start = ctx.For_clause().GetStart().GetTokenIndex()
		} else if ctx.Option_clause() != nil {
			start = ctx.Option_clause().GetStart().GetTokenIndex()
		} else if ctx.SEMI() != nil {
			start = ctx.SEMI().GetSymbol().GetTokenIndex()
		} else {
			return
		}

		if ctx.SEMI() != nil {
			stop = ctx.SEMI().GetSymbol().GetTokenIndex() - 1
		} else {
			stop = ctx.GetStop().GetTokenIndex()
		}
		if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.NewInterval(start, stop))); err != nil {
			e.err = errors.Wrap(err, "failed to write buffer")
			return
		}
		e.fromClause = buf.String()
	}
}

func prepareTransformation(databaseName, statement string) ([]statementInfo, error) {
	parseResult, err := ParseTSQL(statement)
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
		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
			line:      ctx.GetStart().GetLine(),
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

		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
			line:      ctx.GetStart().GetLine(),
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
	name, err := NormalizeFullTableName(ctx)
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
