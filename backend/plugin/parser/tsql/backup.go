package tsql

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
	tree      antlr.ParserRuleContext
	table     *TableReference
}

func TransformDMLToSelect(_ context.Context, _ base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(sourceDatabase, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}

	return generateSQL(statementInfoList, targetDatabase, tablePrefix)
}

// generateBackupTableSQL creates backup table without IDENTITY properties
// We use dynamic SQL to handle IDENTITY columns at runtime by casting them to their base types
func generateBackupTableSQL(statementInfoList []statementInfo, table *TableReference, targetDatabase string, targetTable string) (string, error) {
	// Since we can't easily determine IDENTITY columns at parse time,
	// we'll use a dynamic SQL approach that handles them at runtime

	var buf strings.Builder

	// Create a more robust solution using dynamic SQL to handle IDENTITY columns
	// This creates the backup table and removes IDENTITY properties
	if _, err := buf.WriteString(fmt.Sprintf(`-- Create backup table without IDENTITY columns
DECLARE @sql NVARCHAR(MAX);
DECLARE @cols NVARCHAR(MAX);

-- Get column list with IDENTITY columns cast to their base types
SELECT @cols = STRING_AGG(
    CASE
        WHEN COLUMNPROPERTY(OBJECT_ID('[%s].[%s].[%s]'), c.name, 'IsIdentity') = 1
        THEN 'CAST(' + QUOTENAME(c.name) + ' AS ' +
             TYPE_NAME(c.user_type_id) +
             CASE
                 WHEN TYPE_NAME(c.user_type_id) IN ('varchar', 'char', 'nvarchar', 'nchar', 'varbinary', 'binary')
                 THEN '(' + CASE WHEN c.max_length = -1 THEN 'MAX' ELSE CAST(c.max_length AS VARCHAR(10)) END + ')'
                 WHEN TYPE_NAME(c.user_type_id) IN ('decimal', 'numeric')
                 THEN '(' + CAST(c.precision AS VARCHAR(10)) + ',' + CAST(c.scale AS VARCHAR(10)) + ')'
                 ELSE ''
             END + ') AS ' + QUOTENAME(c.name)
        ELSE QUOTENAME(c.name)
    END, ', ') WITHIN GROUP (ORDER BY c.column_id)
FROM sys.columns c
WHERE c.object_id = OBJECT_ID('[%s].[%s].[%s]');

-- Create the backup table using the modified column list
SET @sql = 'SELECT ' + @cols + ' INTO [%s].[%s].[%s] FROM (`,
		table.Database, table.Schema, table.Table,
		table.Database, table.Schema, table.Table,
		targetDatabase, defaultSchema, targetTable)); err != nil {
		return "", errors.Wrap(err, "failed to write buffer")
	}

	// Add the SELECT statements
	for i, item := range statementInfoList {
		if i > 0 {
			if _, err := buf.WriteString("\n  UNION\n"); err != nil {
				return "", errors.Wrap(err, "failed to write buffer")
			}
		}
		topClause, fromClause, err := extractSuffixSelectStatement(item.tree)
		if err != nil {
			return "", errors.Wrap(err, "failed to extract suffix select statement")
		}
		if len(item.table.Alias) == 0 {
			if _, err := buf.WriteString(fmt.Sprintf(`  SELECT [%s].[%s].[%s].* `,
				item.table.Database, item.table.Schema, item.table.Table)); err != nil {
				return "", errors.Wrap(err, "failed to write buffer")
			}
		} else {
			if _, err := buf.WriteString(fmt.Sprintf(`  SELECT [%s].* `, item.table.Alias)); err != nil {
				return "", errors.Wrap(err, "failed to write buffer")
			}
		}
		if len(topClause) > 0 {
			if _, err := buf.WriteString(topClause); err != nil {
				return "", errors.Wrap(err, "failed to write buffer")
			}
			if _, err := buf.WriteString(" "); err != nil {
				return "", errors.Wrap(err, "failed to write buffer")
			}
		}
		if len(fromClause) > 0 {
			// Escape single quotes in the FROM clause by doubling them for SQL Server
			escapedFromClause := strings.ReplaceAll(fromClause, "'", "''")
			if _, err := buf.WriteString(escapedFromClause); err != nil {
				return "", errors.Wrap(err, "failed to write buffer")
			}
		}
	}

	if _, err := buf.WriteString(`) AS backup_data';

EXEC sp_executesql @sql;`); err != nil {
		return "", errors.Wrap(err, "failed to write buffer")
	}

	// The dynamic SQL creates and populates the table in one step
	return buf.String(), nil
}

func generateSQL(statementInfoList []statementInfo, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
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
				return nil, errors.Errorf("prior backup cannot handle mixed DMLs on the same table %s", key)
			}
		}
	}

	var result []base.BackupStatement
	for key, list := range groupByTable {
		backupStatement, err := generateSQLForTable(list, targetDatabase, tablePrefix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate SQL for table %s", key)
		}
		result = append(result, *backupStatement)
	}

	slices.SortFunc(result, func(a, b base.BackupStatement) int {
		if a.StartPosition.Line != b.StartPosition.Line {
			if a.StartPosition.Line < b.StartPosition.Line {
				return -1
			}
			return 1
		}
		if a.StartPosition.Column != b.StartPosition.Column {
			if a.StartPosition.Column < b.StartPosition.Column {
				return -1
			}
			return 1
		}
		if a.SourceTableName < b.SourceTableName {
			return -1
		}
		if a.SourceTableName > b.SourceTableName {
			return 1
		}
		return 0
	})

	return result, nil
}

func generateSQLForTable(statementInfoList []statementInfo, targetDatabase string, tablePrefix string) (*base.BackupStatement, error) {
	table := statementInfoList[0].table

	targetTable := fmt.Sprintf("%s_%s_%s", tablePrefix, table.Table, table.Database)
	targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)

	// Generate dynamic SQL that handles IDENTITY columns at runtime
	createSQL, err := generateBackupTableSQL(statementInfoList, table, targetDatabase, targetTable)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate backup table SQL")
	}

	return &base.BackupStatement{
		Statement:       createSQL,
		SourceSchema:    table.Schema,
		SourceTableName: table.Table,
		TargetTableName: targetTable,
		StartPosition: &storepb.Position{
			Line:   int32(statementInfoList[0].tree.GetStart().GetLine()),
			Column: int32(statementInfoList[0].tree.GetStart().GetColumn()),
		},
		EndPosition: &storepb.Position{
			Line:   int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetLine()),
			Column: int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetColumn()),
		},
	}, nil
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
