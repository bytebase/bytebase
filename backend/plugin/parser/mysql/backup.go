package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterTransformDMLToSelect(store.Engine_MYSQL, TransformDMLToSelect)
}

const (
	maxTableNameLength = 64
)

func TransformDMLToSelect(ctx context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(sourceDatabase, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}

	return generateSQL(ctx, tCtx, statementInfoList, targetDatabase, tablePrefix)
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
	list, err := SplitSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split sql")
	}

	var result []statementInfo

	for i, sql := range list {
		if len(sql.Text) == 0 || sql.Empty {
			continue
		}
		parseResult, err := ParseMySQL(sql.Text)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse sql")
		}

		for _, sql := range parseResult {
			// After splitting the SQL, we should have only one statement in the list.
			// The FOR loop is just for safety.
			// So we can use the i as the offset.
			tables, err := extractTables(databaseName, sql, i)
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract tables")
			}
			result = append(result, tables...)
		}
	}

	return result, nil
}

func generateSQL(ctx context.Context, tCtx base.TransformContext, statementInfoList []statementInfo, databaseName string, tablePrefix string) ([]base.BackupStatement, error) {
	var result []base.BackupStatement
	offsetLength := 1
	if len(statementInfoList) > 1 {
		offsetLength = base.GetOffsetLength(statementInfoList[len(statementInfoList)-1].offset)
	}

	for _, statementInfo := range statementInfoList {
		table := statementInfo.table
		targetTable := fmt.Sprintf("%s_%0*d_%s", tablePrefix, offsetLength, statementInfo.offset, table.Table)
		targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)
		// If enforce_gtid_consistency = true on MySQL 5.6+, we cannot run CREATE TABLE .. AS SELECT.
		// So we need to create the table first and then run INSERT INTO .. SELECT.
		var buf strings.Builder
		if _, err := buf.WriteString(fmt.Sprintf("CREATE TABLE `%s`.`%s` LIKE `%s`.`%s`;\n", databaseName, targetTable, table.Database, table.Table)); err != nil {
			return nil, errors.Wrap(err, "failed to write create table statement")
		}
		generatedColumns, normalColumns, err := classifyColumns(ctx, tCtx, table)
		if err != nil {
			return nil, errors.Wrap(err, "failed to classify columns")
		}
		tableNameOrAlias := table.Table
		if len(table.Alias) > 0 {
			tableNameOrAlias = table.Alias
		}
		if len(generatedColumns) == 0 {
			if _, err := buf.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` SELECT `%s`.* FROM ", databaseName, targetTable, tableNameOrAlias)); err != nil {
				return nil, errors.Wrap(err, "failed to write insert into statement")
			}
		} else {
			if _, err := buf.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` (", databaseName, targetTable)); err != nil {
				return nil, errors.Wrap(err, "failed to write insert into statement")
			}
			for i, column := range normalColumns {
				if i > 0 {
					if err := buf.WriteByte(','); err != nil {
						return nil, errors.Wrap(err, "failed to write comma")
					}
				}
				if _, err := buf.WriteString(fmt.Sprintf("`%s`", column)); err != nil {
					return nil, errors.Wrap(err, "failed to write column")
				}
			}
			if _, err := buf.WriteString(") SELECT "); err != nil {
				return nil, errors.Wrap(err, "failed to write select")
			}
			for i, column := range normalColumns {
				if i > 0 {
					if err := buf.WriteByte(','); err != nil {
						return nil, errors.Wrap(err, "failed to write comma")
					}
				}
				if _, err := buf.WriteString(fmt.Sprintf("`%s`.`%s`", tableNameOrAlias, column)); err != nil {
					return nil, errors.Wrap(err, "failed to write column")
				}
			}
			if _, err := buf.WriteString(" FROM "); err != nil {
				return nil, errors.Wrap(err, "failed to write from")
			}
		}
		if err := extractSuffixSelectStatement(statementInfo.tree, &buf); err != nil {
			return nil, errors.Wrap(err, "failed to extract suffix select statement")
		}
		if err := buf.WriteByte(';'); err != nil {
			return nil, errors.Wrap(err, "failed to write semicolon")
		}
		result = append(result, base.BackupStatement{
			Statement:       buf.String(),
			SourceTableName: table.Table,
			TargetTableName: targetTable,
			StartPosition: &store.Position{
				Line:   int32(statementInfo.tree.GetStart().GetLine()),
				Column: int32(statementInfo.tree.GetStart().GetColumn()),
			},
			EndPosition: &store.Position{
				Line:   int32(statementInfo.tree.GetStop().GetLine()),
				Column: int32(statementInfo.tree.GetStop().GetColumn()),
			},
		})
	}
	return result, nil
}

func classifyColumns(ctx context.Context, tCtx base.TransformContext, table *TableReference) ([]string, []string, error) {
	if tCtx.GetDatabaseMetadataFunc == nil {
		return nil, nil, errors.New("GetDatabaseMetadataFunc is not set")
	}

	_, metadata, err := tCtx.GetDatabaseMetadataFunc(ctx, tCtx.InstanceID, table.Database)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get database metadata for InstanceID %q, Database %q", tCtx.InstanceID, table.Database)
	}

	schemaMetadata := metadata.GetSchema("")
	if schemaMetadata == nil {
		return nil, nil, errors.New("failed to get schema metadata")
	}

	tableMetadata := schemaMetadata.GetTable(table.Table)
	if tableMetadata == nil {
		return nil, nil, errors.New("failed to get table metadata for table " + table.Table)
	}

	var generatedColumns, normalColumns []string
	for _, column := range tableMetadata.GetColumns() {
		if column.GetGeneration() != nil {
			generatedColumns = append(generatedColumns, column.GetName())
		} else {
			normalColumns = append(normalColumns, column.GetName())
		}
	}

	return generatedColumns, normalColumns, nil
}

func extractSuffixSelectStatement(tree antlr.Tree, buf *strings.Builder) error {
	listener := &suffixSelectStatementListener{
		buf: buf,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.err
}

type suffixSelectStatementListener struct {
	*parser.BaseMySQLParserListener

	buf *strings.Builder
	err error
}

func (l *suffixSelectStatementListener) EnterDeleteStatement(ctx *parser.DeleteStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	if ctx.TableRef() != nil {
		// Single table delete statement.
		if _, err := l.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromTokens(
			ctx.TableRef().GetStart(),
			ctx.GetStop(),
		)); err != nil {
			l.err = errors.Wrap(err, "failed to write suffix select statement")
			return
		}
	}

	if ctx.TableAliasRefList() != nil {
		// Multi table delete statement.
		if _, err := l.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromTokens(
			ctx.TableReferenceList().GetStart(),
			ctx.GetStop(),
		)); err != nil {
			l.err = errors.Wrap(err, "failed to write suffix select statement")
			return
		}
	}
}

func (l *suffixSelectStatementListener) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	if _, err := l.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.TableReferenceList())); err != nil {
		l.err = errors.Wrap(err, "failed to write suffix select statement")
		return
	}

	if ctx.WhereClause() != nil {
		if err := l.buf.WriteByte(' '); err != nil {
			l.err = errors.Wrap(err, "failed to write suffix select statement")
			return
		}
		if _, err := l.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.WhereClause())); err != nil {
			l.err = errors.Wrap(err, "failed to write suffix select statement")
			return
		}
	}

	if ctx.OrderClause() != nil {
		if err := l.buf.WriteByte(' '); err != nil {
			l.err = errors.Wrap(err, "failed to write suffix select statement")
			return
		}
		if _, err := l.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.OrderClause())); err != nil {
			l.err = errors.Wrap(err, "failed to write suffix select statement")
			return
		}
	}

	if ctx.SimpleLimitClause() != nil {
		if err := l.buf.WriteByte(' '); err != nil {
			l.err = errors.Wrap(err, "failed to write suffix select statement")
			return
		}
		if _, err := l.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.SimpleLimitClause())); err != nil {
			l.err = errors.Wrap(err, "failed to write suffix select statement")
			return
		}
	}
}

func extractTables(databaseName string, parseResult *ParseResult, offset int) ([]statementInfo, error) {
	listener := &tableReferenceListener{
		databaseName: databaseName,
		offset:       offset,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)

	return listener.tables, listener.err
}

type tableReferenceListener struct {
	*parser.BaseMySQLParserListener

	databaseName string
	offset       int
	tables       []statementInfo
	err          error
}

func isTopLevel(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}
	switch ctx := ctx.(type) {
	case *parser.SimpleStatementContext:
		return isTopLevel(ctx.GetParent())
	case *parser.QueryContext, *parser.ScriptContext:
		return true
	default:
		return false
	}
}

func (l *tableReferenceListener) EnterDeleteStatement(ctx *parser.DeleteStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.TableRef() != nil {
		// Single table delete statement.
		database, table := NormalizeMySQLTableRef(ctx.TableRef())
		if len(database) > 0 && database != l.databaseName {
			l.err = errors.Errorf("database is not matched: %s != %s", database, l.databaseName)
			return
		}

		alias := ""

		if ctx.TableAlias() != nil {
			alias = NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
		}

		if len(database) == 0 {
			database = l.databaseName
		}

		l.tables = append(l.tables, statementInfo{
			offset:    l.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table: &TableReference{
				Database:      database,
				Table:         table,
				Alias:         alias,
				StatementType: StatementTypeDelete,
			},
		})
		return
	}

	if ctx.TableAliasRefList() != nil {
		// Multi table delete statement.
		singleTables := &singleTableListener{
			databaseName: l.databaseName,
			singleTables: make(map[string]*TableReference),
		}

		antlr.ParseTreeWalkerDefault.Walk(singleTables, ctx.TableReferenceList())

		for _, tableRef := range ctx.TableAliasRefList().AllTableRefWithWildcard() {
			database, table := NormalizeMySQLTableRefWithWildcard(tableRef)
			if len(database) > 0 && database != l.databaseName {
				l.err = errors.Errorf("database is not matched: %s != %s", database, l.databaseName)
				return
			}

			singleTable, ok := singleTables.singleTables[table]
			if !ok {
				l.err = errors.Errorf("cannot extract reference table: no matched table %q in referenced table list", table)
				return
			}

			singleTable.StatementType = StatementTypeDelete
			l.tables = append(l.tables, statementInfo{
				offset:    l.offset,
				statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
				tree:      ctx,
				table:     singleTable,
			})
		}
	}
}

func (l *tableReferenceListener) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	listener := &updateTableListener{
		tables: make(map[string]bool),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, ctx.UpdateList())

	singleTables := &singleTableListener{
		databaseName: l.databaseName,
		singleTables: make(map[string]*TableReference),
	}

	antlr.ParseTreeWalkerDefault.Walk(singleTables, ctx.TableReferenceList())

	if len(singleTables.singleTables) == 1 {
		// We only allow users do not specify table alias when there is only one table in the update statement.
		// TODO: Support other cases.
		if _, exists := listener.tables[""]; exists {
			delete(listener.tables, "")
			for tableName := range singleTables.singleTables {
				listener.tables[tableName] = true
			}
		}
	}

	for table := range listener.tables {
		singleTable, ok := singleTables.singleTables[table]
		if !ok {
			l.err = errors.Errorf("cannot extract reference table: no matched updated table %q in referenced table list", table)
			return
		}

		singleTable.StatementType = StatementTypeUpdate
		l.tables = append(l.tables, statementInfo{
			offset:    l.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     singleTable,
		})
	}
}

type singleTableListener struct {
	*parser.BaseMySQLParserListener

	databaseName string
	singleTables map[string]*TableReference
	err          error
}

func (l *singleTableListener) EnterSingleTable(ctx *parser.SingleTableContext) {
	if l.err != nil {
		return
	}
	database, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	if len(database) > 0 && database != l.databaseName {
		l.err = errors.Errorf("database is not matched: %s != %s", database, l.databaseName)
	}
	if len(database) == 0 {
		database = l.databaseName
	}
	table := &TableReference{
		Database: database,
		Table:    tableName,
	}

	if ctx.TableAlias() != nil {
		table.Alias = NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
		l.singleTables[table.Alias] = table
	} else {
		l.singleTables[table.Table] = table
	}
}

type updateTableListener struct {
	*parser.BaseMySQLParserListener

	tables map[string]bool
}

func (l *updateTableListener) EnterUpdateElement(ctx *parser.UpdateElementContext) {
	_, table, _ := NormalizeMySQLColumnRef(ctx.ColumnRef())
	l.tables[table] = true
}
