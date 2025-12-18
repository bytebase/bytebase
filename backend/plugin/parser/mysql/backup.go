package mysql

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
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

type StatementInfo struct {
	Offset        int
	Statement     string
	Tree          antlr.ParserRuleContext
	Table         *TableReference
	StartPosition *store.Position
	EndPosition   *store.Position
}

func prepareTransformation(databaseName, statement string) ([]StatementInfo, error) {
	list, err := SplitSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split sql")
	}

	var result []StatementInfo

	for i, item := range list {
		if len(item.Text) == 0 || item.Empty {
			continue
		}
		parseResult, err := ParseMySQL(item.Text)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse sql")
		}

		for _, sql := range parseResult {
			// After splitting the SQL, we should have only one statement in the list.
			// The FOR loop is just for safety.
			// So we can use the i as the offset.
			tables, err := ExtractTables(databaseName, sql, i)
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract tables")
			}
			for _, table := range tables {
				result = append(result, StatementInfo{
					Offset:        i,
					Statement:     table.Statement,
					Table:         table.Table,
					Tree:          table.Tree,
					StartPosition: item.Start,
					EndPosition:   item.End,
				})
			}
		}
	}

	return result, nil
}

func generateSQL(ctx context.Context, tCtx base.TransformContext, statementInfoList []StatementInfo, databaseName string, tablePrefix string) ([]base.BackupStatement, error) {
	groupByTable := make(map[string][]StatementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s", item.Table.Database, item.Table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	// Check if the statement type is the same for all statements in one table.
	for key, list := range groupByTable {
		stmtType := StatementTypeUnknown
		for _, item := range list {
			if stmtType == StatementTypeUnknown {
				stmtType = item.Table.StatementType
			}
			if stmtType != item.Table.StatementType {
				return nil, errors.Errorf("prior backup cannot handle mixed DML statements on the same table %s", key)
			}
		}
	}

	var result []base.BackupStatement

	for key, list := range groupByTable {
		backupStatement, err := generateSQLForTable(ctx, tCtx, list, databaseName, tablePrefix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate SQL for table %s", key)
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

func generateSQLForTable(ctx context.Context, tCtx base.TransformContext, statementInfoList []StatementInfo, databaseName string, tablePrefix string) (*base.BackupStatement, error) {
	table := statementInfoList[0].Table

	generatedColumns, normalColumns, err := classifyColumns(ctx, tCtx.GetDatabaseMetadataFunc, tCtx.ListDatabaseNamesFunc, tCtx.IsCaseSensitive, tCtx.InstanceID, table)
	if err != nil {
		return nil, errors.Wrap(err, "failed to classify columns")
	}

	targetTable := fmt.Sprintf("%s_%s_%s", tablePrefix, table.Table, table.Database)
	targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)
	var buf strings.Builder
	if _, err := buf.WriteString(fmt.Sprintf("CREATE TABLE `%s`.`%s` LIKE `%s`.`%s`;\n", databaseName, targetTable, table.Database, table.Table)); err != nil {
		return nil, errors.Wrap(err, "failed to write create table statement")
	}

	if _, err := buf.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s`", databaseName, targetTable)); err != nil {
		return nil, errors.Wrap(err, "failed to write insert into statement")
	}
	if len(generatedColumns) > 0 {
		if _, err := buf.WriteString(" ("); err != nil {
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
		if _, err := buf.WriteString(")"); err != nil {
			return nil, errors.Wrap(err, "failed to write insert into statement")
		}
	}
	for i, item := range statementInfoList {
		if i != 0 {
			// We assume that the source table has a primary key.
			// If we have multiple statements, we need to use UNION DISTINCT to avoid duplicate rows.
			if _, err := buf.WriteString("\n  UNION DISTINCT\n"); err != nil {
				return nil, errors.Wrap(err, "failed to write union all statement")
			}
		}
		tableNameOrAlias := item.Table.Table
		if len(item.Table.Alias) > 0 {
			tableNameOrAlias = item.Table.Alias
		}
		if _, err := buf.WriteString("  "); err != nil {
			return nil, errors.Wrap(err, "failed to write space")
		}
		cteString := extractCTE(item.Tree)
		if len(cteString) > 0 {
			if _, err := buf.WriteString(fmt.Sprintf("%s ", cteString)); err != nil {
				return nil, errors.Wrap(err, "failed to write cte")
			}
		}
		if len(generatedColumns) == 0 {
			if _, err := buf.WriteString(fmt.Sprintf("SELECT `%s`.* FROM ", tableNameOrAlias)); err != nil {
				return nil, errors.Wrap(err, "failed to write select statement")
			}
		} else {
			if _, err := buf.WriteString("SELECT "); err != nil {
				return nil, errors.Wrap(err, "failed to write select statement")
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
		if err := extractSuffixSelectStatement(item.Tree, &buf); err != nil {
			return nil, errors.Wrap(err, "failed to extract suffix select statement")
		}
	}

	if err := buf.WriteByte(';'); err != nil {
		return nil, errors.Wrap(err, "failed to write semicolon")
	}

	return &base.BackupStatement{
		Statement:       buf.String(),
		SourceTableName: table.Table,
		TargetTableName: targetTable,
		StartPosition:   statementInfoList[0].StartPosition,
		EndPosition:     statementInfoList[len(statementInfoList)-1].EndPosition,
	}, nil
}

func extractCTE(ctx antlr.ParserRuleContext) string {
	switch node := ctx.(type) {
	case *parser.UpdateStatementContext:
		if node.WithClause() != nil {
			return node.GetParser().GetTokenStream().GetTextFromRuleContext(node.WithClause())
		}
	case *parser.DeleteStatementContext:
		if node.WithClause() != nil {
			return node.GetParser().GetTokenStream().GetTextFromRuleContext(node.WithClause())
		}
	}

	return ""
}

func classifyColumns(ctx context.Context, getDatabaseMetadataFunc base.GetDatabaseMetadataFunc, listDatabaseNamesFunc base.ListDatabaseNamesFunc, isCaseSensitive bool, instanceID string, table *TableReference) ([]string, []string, error) {
	if getDatabaseMetadataFunc == nil {
		return nil, nil, errors.New("GetDatabaseMetadataFunc is not set")
	}

	var dbMetadata *model.DatabaseMetadata
	allDatabaseNames, err := listDatabaseNamesFunc(ctx, instanceID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to list databases names")
	}
	if !isCaseSensitive {
		for _, db := range allDatabaseNames {
			if strings.EqualFold(db, table.Database) {
				_, dbMetadata, err = getDatabaseMetadataFunc(ctx, instanceID, db)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	} else {
		for _, db := range allDatabaseNames {
			if db == table.Database {
				_, dbMetadata, err = getDatabaseMetadataFunc(ctx, instanceID, db)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	}
	if dbMetadata == nil {
		slog.Debug("failed to get database metadata", slog.String("instanceID", instanceID), slog.String("database", table.Database))
		return nil, nil, errors.Errorf("failed to get database metadata for InstanceID %q, Database %q", instanceID, table.Database)
	}

	emptySchema := ""
	schema := dbMetadata.GetSchemaMetadata(emptySchema)
	if schema == nil {
		return nil, nil, errors.New("failed to get schema metadata")
	}

	var tableSchema *model.TableMetadata
	if !isCaseSensitive {
		for _, tableName := range schema.ListTableNames() {
			if strings.EqualFold(tableName, table.Table) {
				tableSchema = schema.GetTable(tableName)
				break
			}
		}
	} else {
		tableSchema = schema.GetTable(table.Table)
	}
	if tableSchema == nil {
		return nil, nil, errors.Errorf("table %s not found in schema", table.Table)
	}

	var generatedColumns, normalColumns []string
	for _, column := range tableSchema.GetProto().GetColumns() {
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

func ExtractTables(databaseName string, ast *base.ANTLRAST, offset int) ([]StatementInfo, error) {
	listener := &tableReferenceListener{
		databaseName: databaseName,
		offset:       offset,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, ast.Tree)

	return listener.tables, listener.err
}

type tableReferenceListener struct {
	*parser.BaseMySQLParserListener

	databaseName string
	offset       int
	tables       []StatementInfo
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

	cteMap := make(map[string]bool)
	if ctx.WithClause() != nil {
		for _, cte := range ctx.WithClause().AllCommonTableExpression() {
			tableName := NormalizeMySQLIdentifier(cte.Identifier())
			cteMap[tableName] = true
		}
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

		l.tables = append(l.tables, StatementInfo{
			Offset:    l.offset,
			Statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			Tree:      ctx,
			Table: &TableReference{
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

			if len(database) == 0 && cteMap[table] {
				continue
			}

			singleTable, ok := singleTables.singleTables[table]
			if !ok {
				l.err = errors.Errorf("cannot extract reference table: no matched table %q in referenced table list", table)
				return
			}

			singleTable.StatementType = StatementTypeDelete
			l.tables = append(l.tables, StatementInfo{
				Offset:    l.offset,
				Statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
				Tree:      ctx,
				Table:     singleTable,
			})
		}
	}
}

func (l *tableReferenceListener) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	cteMap := make(map[string]bool)

	if ctx.WithClause() != nil {
		for _, cte := range ctx.WithClause().AllCommonTableExpression() {
			tableName := NormalizeMySQLIdentifier(cte.Identifier())
			cteMap[tableName] = true
		}
	}

	listener := &updateTableListener{
		tables: make(map[string]bool),
		cteMap: cteMap,
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
		l.tables = append(l.tables, StatementInfo{
			Offset:    l.offset,
			Statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			Tree:      ctx,
			Table:     singleTable,
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
	cteMap map[string]bool
}

func (l *updateTableListener) EnterUpdateElement(ctx *parser.UpdateElementContext) {
	_, table, _ := NormalizeMySQLColumnRef(ctx.ColumnRef())
	if _, exists := l.cteMap[table]; !exists {
		l.tables[table] = true
	}
}
