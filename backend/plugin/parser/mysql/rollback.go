package mysql

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterTransformDMLToSelect(store.Engine_MYSQL, TransformDMLToSelect)
}

func TransformDMLToSelect(statement string, sourceDatabase string, targetDatabase string, tableSuffix string) ([]base.RollbackStatement, error) {
	tableStatementMap, err := prepareTransformation(sourceDatabase, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}

	return generateSQL(tableStatementMap, targetDatabase, tableSuffix)
}

func prepareTransformation(databaseName, statement string) (map[string][]*tableStatement, error) {
	list, err := SplitSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split sql")
	}

	result := make(map[string][]*tableStatement)
	tableStatementTypeMap := make(map[string]StatementType)

	for _, sql := range list {
		if len(sql.Text) == 0 || sql.Empty {
			continue
		}
		parseResult, isDML, isDDL, err := getSQLType(sql.Text)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get sql type")
		}
		if isDDL {
			return nil, errors.New("cannot transform mixed DDL and DML statements")
		}
		if !isDML {
			continue
		}

		tables, err := extractTables(databaseName, parseResult)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract tables")
		}
		for _, table := range tables {
			if table.StatementType == StatementTypeUnknown {
				return nil, errors.Errorf("unknown statement type for table %q", table.Table)
			}

			tp, exists := tableStatementTypeMap[table.Table]
			if exists && tp != table.StatementType {
				return nil, errors.Errorf("cannot transform mixed DML statements for table %q", table.Table)
			}

			if !exists {
				tableStatementTypeMap[table.Table] = table.StatementType
			}

			result[table.Table] = append(result[table.Table], &tableStatement{
				tree:  parseResult,
				table: &TableReference{Table: table.Table, Alias: table.Alias},
			})
		}
	}

	return result, nil
}

func getSQLType(statement string) (*ParseResult, bool, bool, error) {
	listener := &StatementTypeChecker{}

	stmts, err := ParseMySQL(statement)
	if err != nil {
		return nil, false, false, errors.Wrap(err, "failed to parse sql")
	}

	if len(stmts) != 1 {
		return nil, false, false, errors.New("statement is not single sql")
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, stmts[0].Tree)
	return stmts[0], listener.IsDML, listener.IsDDL, nil
}

func generateSQL(tableStatementMap map[string][]*tableStatement, databaseName string, tableSuffix string) ([]base.RollbackStatement, error) {
	var result []base.RollbackStatement
	for tableName, tableStatements := range tableStatementMap {
		targetTable := fmt.Sprintf("%s%s", tableName, tableSuffix)
		var buf strings.Builder
		if _, err := buf.WriteString(fmt.Sprintf("CREATE TABLE `%s`.`%s` AS ", databaseName, targetTable)); err != nil {
			return nil, errors.Wrap(err, "failed to write create table statement")
		}
		for i, tableStatement := range tableStatements {
			if i > 0 {
				if _, err := buf.WriteString(" UNION "); err != nil {
					return nil, errors.Wrap(err, "failed to write union all statement")
				}
			}
			tableName := tableStatement.table.Table
			if len(tableStatement.table.Alias) > 0 {
				tableName = tableStatement.table.Alias
			}
			if _, err := buf.WriteString(fmt.Sprintf("SELECT `%s`.* FROM ", tableName)); err != nil {
				return nil, errors.Wrap(err, "failed to write select statement")
			}

			if err := extractSuffixSelectStatement(tableStatement.tree, &buf); err != nil {
				return nil, errors.Wrap(err, "failed to extract suffix select statement")
			}
		}
		if err := buf.WriteByte(';'); err != nil {
			return nil, errors.Wrap(err, "failed to write semicolon")
		}
		result = append(result, base.RollbackStatement{
			Statement: buf.String(),
			TableName: targetTable,
		})
	}
	return result, nil
}

func extractSuffixSelectStatement(parseResult *ParseResult, buf *strings.Builder) error {
	listener := &suffixSelectStatementListener{
		buf: buf,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)
	return listener.err
}

type suffixSelectStatementListener struct {
	*parser.BaseMySQLParserListener

	buf *strings.Builder
	err error
}

func (l *suffixSelectStatementListener) EnterDeleteStatement(ctx *parser.DeleteStatementContext) {
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

type tableStatement struct {
	tree  *ParseResult
	table *TableReference
}

type StatementType int

const (
	StatementTypeUnknown StatementType = iota
	StatementTypeUpdate
	StatementTypeInsert
	StatementTypeDelete
)

type TableReference struct {
	Table         string
	Alias         string
	StatementType StatementType
}

func extractTables(databaseName string, parseResult *ParseResult) ([]*TableReference, error) {
	listener := &tableReferenceListener{
		databaseName: databaseName,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)

	return listener.tables, listener.err
}

type tableReferenceListener struct {
	*parser.BaseMySQLParserListener

	databaseName string
	tables       []*TableReference
	err          error
}

func (l *tableReferenceListener) EnterDeleteStatement(ctx *parser.DeleteStatementContext) {
	if _, ok := ctx.GetParent().(*parser.SimpleStatementContext); !ok {
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

		l.tables = append(l.tables, &TableReference{
			Table:         table,
			Alias:         alias,
			StatementType: StatementTypeDelete,
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
			l.tables = append(l.tables, singleTable)
		}
	}
}

func (l *tableReferenceListener) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if _, ok := ctx.GetParent().(*parser.SimpleStatementContext); !ok {
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
		l.tables = append(l.tables, singleTable)
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
	table := &TableReference{
		Table: tableName,
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
