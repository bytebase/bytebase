package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	maxCommentLength = 1000
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MYSQL, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	parseResult, err := ParseMySQL(originalSQL)
	if err != nil {
		return "", err
	}

	if len(parseResult) == 0 {
		return "", errors.Errorf("no parse result")
	}

	// We only need the first parse result.
	// There are two cases:
	// 1. The statement is a single SQL statement.
	// 2. The statement is a multi SQL statement, but all SQL statements' backup is in the same table.
	//    So we only need to restore the first SQL statement.
	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}
	return doGenerate(ctx, rCtx, sqlForComment, parseResult[0], backupItem)
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, ast *base.ANTLRAST, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}
	_, targetDatabase, err := common.GetInstanceDatabaseID(backupItem.TargetTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get target database ID for %s", backupItem.TargetTable.Database)
	}
	generatedColumns, normalColumns, err := classifyColumns(ctx, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc, rCtx.IsCaseSensitive, rCtx.InstanceID, &TableReference{
		Database: sourceDatabase,
		Table:    backupItem.SourceTable.Table,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to classify columns for %s.%s", backupItem.SourceTable.Database, backupItem.SourceTable.Table)
	}

	g := &generator{
		ctx:              ctx,
		rCtx:             rCtx,
		backupDatabase:   targetDatabase,
		backupTable:      backupItem.TargetTable.Table,
		originalDatabase: sourceDatabase,
		originalTable:    backupItem.SourceTable.Table,
		generatedColumns: generatedColumns,
		normalColumns:    normalColumns,
	}
	var buf strings.Builder
	antlr.ParseTreeWalkerDefault.Walk(g, ast.Tree)
	if g.err != nil {
		return "", g.err
	}
	if _, err := fmt.Fprintf(&buf, "/*\nOriginal SQL:\n%s\n*/\n%s", sqlForComment, g.result); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}

	list, err := SplitSQL(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to split sql")
	}

	start := 0
	end := len(list) - 1
	for i, item := range list {
		if equalOrLess(item.Start, backupItem.StartPosition) {
			start = i
		}
	}

	for i := len(list) - 1; i >= 0; i-- {
		if equalOrGreater(list[i].Start, backupItem.EndPosition) {
			end = i
		}
	}

	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}

	var result []string
	// We only need statements that contain the source table.
	for i := start; i <= end; i++ {
		parseResult, err := ParseMySQL(list[i].Text)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse sql")
		}
		containsSourceTable := false
		for _, sql := range parseResult {
			tables, err := ExtractTables(sourceDatabase, sql, i)
			if err != nil {
				return "", errors.Wrap(err, "failed to extract tables")
			}
			for _, table := range tables {
				if table.Table.Database == sourceDatabase && table.Table.Table == backupItem.SourceTable.Table {
					containsSourceTable = true
					break
				}
			}
			if containsSourceTable {
				break
			}
		}
		if containsSourceTable {
			result = append(result, list[i].Text)
		}
	}
	return strings.Join(result, ""), nil
}

func equalOrLess(a, b *storepb.Position) bool {
	if a.Line < b.Line {
		return true
	}
	if a.Line == b.Line && a.Column <= b.Column {
		return true
	}
	return false
}

func equalOrGreater(a, b *storepb.Position) bool {
	if a.Line > b.Line {
		return true
	}
	if a.Line == b.Line && a.Column >= b.Column {
		return true
	}
	return false
}

type generator struct {
	*parser.BaseMySQLParserListener

	ctx  context.Context
	rCtx base.RestoreContext

	backupDatabase   string
	backupTable      string
	originalDatabase string
	originalTable    string
	generatedColumns []string
	normalColumns    []string
	result           string
	err              error
}

func (g *generator) EnterDeleteStatement(ctx *parser.DeleteStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if len(g.generatedColumns) == 0 {
		g.result = fmt.Sprintf("INSERT INTO `%s`.`%s` SELECT * FROM `%s`.`%s`;", g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable)
	} else {
		var quotedColumns []string
		for _, column := range g.normalColumns {
			quotedColumns = append(quotedColumns, fmt.Sprintf("`%s`", column))
		}
		quotedColumnList := strings.Join(quotedColumns, ", ")
		g.result = fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) SELECT %s FROM `%s`.`%s`;", g.originalDatabase, g.originalTable, quotedColumnList, quotedColumnList, g.backupDatabase, g.backupTable)
	}
}

func (g *generator) hasDisjointUniqueKey(updateColumns []string) (bool, error) {
	columnMap := make(map[string]bool)
	for _, column := range updateColumns {
		columnMap[strings.ToLower(column)] = true
	}

	if g.rCtx.GetDatabaseMetadataFunc == nil {
		return false, errors.Errorf("GetDatabaseMetadataFunc is nil")
	}

	_, metadata, err := g.rCtx.GetDatabaseMetadataFunc(g.ctx, g.rCtx.InstanceID, g.originalDatabase)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get database metadata for %s", g.originalDatabase)
	}

	if metadata == nil {
		return false, errors.Errorf("database metadata is nil for %s", g.originalDatabase)
	}

	schema := metadata.GetSchemaMetadata("")
	if schema == nil {
		return false, errors.Errorf("schema is nil for %s", g.originalDatabase)
	}

	tableMetadata := schema.GetTable(g.originalTable)
	if tableMetadata == nil {
		return false, errors.Errorf("table metadata is nil for %s.%s", g.originalDatabase, g.originalTable)
	}

	for _, index := range tableMetadata.GetProto().Indexes {
		if !index.Primary && !index.Unique {
			continue
		}
		if disjoint(index.Expressions, columnMap) {
			return true, nil
		}
	}

	return false, nil
}

func disjoint(a []string, b map[string]bool) bool {
	for _, item := range a {
		if _, ok := b[strings.ToLower(item)]; ok {
			return false
		}
	}
	return true
}

func (g *generator) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	singleTables := &singleTableListener{
		databaseName: g.originalDatabase,
		singleTables: make(map[string]*TableReference),
	}

	antlr.ParseTreeWalkerDefault.Walk(singleTables, ctx.TableReferenceList())

	updateItems := &updateItemListener{
		database:      g.originalDatabase,
		normalColumns: g.normalColumns,
	}
	for _, table := range singleTables.singleTables {
		if strings.EqualFold(table.Table, g.originalTable) {
			updateItems.table = table
			break
		}
	}
	antlr.ParseTreeWalkerDefault.Walk(updateItems, ctx.UpdateList())

	has, err := g.hasDisjointUniqueKey(updateItems.result)
	if err != nil {
		g.err = err
		return
	}
	if !has {
		g.err = errors.Errorf("no disjoint unique key found for %s.%s", g.originalDatabase, g.originalTable)
		return
	}

	var buf strings.Builder
	if len(g.generatedColumns) == 0 {
		if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s` SELECT * FROM `%s`.`%s` ON DUPLICATE KEY UPDATE ", g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable); err != nil {
			g.err = err
			return
		}
	} else {
		var quotedColumns []string
		for _, column := range g.normalColumns {
			quotedColumns = append(quotedColumns, fmt.Sprintf("`%s`", column))
		}
		quotedColumnList := strings.Join(quotedColumns, ", ")
		if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s` (%s) SELECT %s FROM `%s`.`%s` ON DUPLICATE KEY UPDATE ", g.originalDatabase, g.originalTable, quotedColumnList, quotedColumnList, g.backupDatabase, g.backupTable); err != nil {
			g.err = err
			return
		}
	}

	for i, field := range updateItems.result {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				g.err = err
				return
			}
		}

		if _, err := fmt.Fprintf(&buf, "`%s` = VALUES(`%s`)", field, field); err != nil {
			g.err = err
			return
		}
	}
	if _, err := buf.WriteString(";"); err != nil {
		g.err = err
		return
	}
	g.result = buf.String()
}

type updateItemListener struct {
	*parser.BaseMySQLParserListener
	normalColumns []string
	database      string
	table         *TableReference
	result        []string
}

func (l *updateItemListener) EnterUpdateElement(ctx *parser.UpdateElementContext) {
	database, table, column := NormalizeMySQLColumnRef(ctx.ColumnRef())

	if database != "" && !strings.EqualFold(database, l.database) {
		return
	}

	if table == "" {
		for _, c := range l.normalColumns {
			if strings.EqualFold(c, column) {
				l.result = append(l.result, column)
				return
			}
		}
		return
	}

	if l.table.Alias == table || strings.EqualFold(l.table.Table, table) {
		l.result = append(l.result, column)
		return
	}
}
