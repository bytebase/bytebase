package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	maxCommentLength = 1000
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MYSQL, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractSingleSQL(statement, backupItem)
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

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, parseResult *ParseResult, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}
	_, targetDatabase, err := common.GetInstanceDatabaseID(backupItem.TargetTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get target database ID for %s", backupItem.TargetTable.Database)
	}
	generatedColumns, normalColumns, normalColumnsExceptPrimary, err := classifyColumns(ctx, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc, rCtx.IsCaseSensitive, rCtx.InstanceID, &TableReference{
		Database: sourceDatabase,
		Table:    backupItem.SourceTable.Table,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to classify columns for %s.%s", backupItem.SourceTable.Database, backupItem.SourceTable.Table)
	}

	g := &generator{
		ctx:                        ctx,
		rCtx:                       rCtx,
		backupDatabase:             targetDatabase,
		backupTable:                backupItem.TargetTable.Table,
		originalDatabase:           sourceDatabase,
		originalTable:              backupItem.SourceTable.Table,
		generatedColumns:           generatedColumns,
		normalColumns:              normalColumns,
		normalColumnsExceptPrimary: normalColumnsExceptPrimary,
	}
	var buf strings.Builder
	antlr.ParseTreeWalkerDefault.Walk(g, parseResult.Tree)
	if g.err != nil {
		return "", g.err
	}
	if _, err := fmt.Fprintf(&buf, "/*\nOriginal SQL:\n%s\n*/\n%s", sqlForComment, g.result); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func extractSingleSQL(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
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
		if equalOrLess(&storepb.Position{
			Line:   int32(item.FirstStatementLine + 1),
			Column: int32(item.FirstStatementColumn),
		}, backupItem.StartPosition) {
			start = i
		}
	}

	for i := len(list) - 1; i >= 0; i-- {
		if equalOrGreater(&storepb.Position{
			Line:   int32(list[i].FirstStatementLine + 1),
			Column: int32(list[i].FirstStatementColumn),
		}, backupItem.EndPosition) {
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

	backupDatabase             string
	backupTable                string
	originalDatabase           string
	originalTable              string
	generatedColumns           []string
	normalColumns              []string
	normalColumnsExceptPrimary []string
	result                     string
	err                        error
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

func (g *generator) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if len(g.normalColumns) == len(g.normalColumnsExceptPrimary) {
		// No primary key, we can't do ON DUPLICATE KEY UPDATE.
		g.err = errors.Errorf("primary key not found for %s.%s", g.originalDatabase, g.originalTable)
		return
	}

	singleTables := &singleTableListener{
		databaseName: g.originalDatabase,
		singleTables: make(map[string]*TableReference),
	}

	antlr.ParseTreeWalkerDefault.Walk(singleTables, ctx.TableReferenceList())

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

	for i, field := range g.normalColumnsExceptPrimary {
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
