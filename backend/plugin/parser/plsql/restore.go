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
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_ORACLE, GenerateRestoreSQL)
}

const (
	maxCommentLength = 1000
)

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractSQL(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	tree, _, err := ParsePLSQL(originalSQL)
	if err != nil {
		return "", err
	}

	if tree == nil {
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
	return doGenerate(ctx, rCtx, sqlForComment, tree, backupItem)
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, tree antlr.Tree, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}
	_, targetDatabase, err := common.GetInstanceDatabaseID(backupItem.TargetTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get target database ID for %s", backupItem.TargetTable.Database)
	}

	if rCtx.GetDatabaseMetadataFunc == nil {
		return "", errors.Errorf("GetDatabaseMetadataFunc is required")
	}

	_, metadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, sourceDatabase)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database metadata for %s", sourceDatabase)
	}

	schemaMetadata := metadata.GetSchema("")
	if schemaMetadata == nil {
		return "", errors.Errorf("no schema metadata for %s", sourceDatabase)
	}

	tableMetadata := schemaMetadata.GetTable(backupItem.SourceTable.Table)
	if tableMetadata == nil {
		return "", errors.Errorf("no table metadata for %s.%s", sourceDatabase, backupItem.SourceTable.Table)
	}

	g := &generator{
		ctx:              ctx,
		rCtx:             rCtx,
		backupDatabase:   targetDatabase,
		backupTable:      backupItem.TargetTable.Table,
		originalDatabase: sourceDatabase,
		originalTable:    backupItem.SourceTable.Table,
		pk:               tableMetadata.GetPrimaryKey(),
		table:            tableMetadata,
		isFirst:          true,
	}
	antlr.ParseTreeWalkerDefault.Walk(g, tree)
	if g.err != nil {
		return "", g.err
	}
	return fmt.Sprintf("/*\nOriginal SQL:\n%s\n*/\n%s", sqlForComment, g.result), nil
}

type generator struct {
	*parser.BasePlSqlParserListener

	backupDatabase   string
	backupTable      string
	originalDatabase string
	originalTable    string
	pk               *model.IndexMetadata
	table            *model.TableMetadata

	isFirst bool
	ctx     context.Context
	rCtx    base.RestoreContext
	result  string
	err     error
}

func (g *generator) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if !IsTopLevelStatement(ctx.GetParent()) || !g.isFirst {
		return
	}

	g.isFirst = false
	g.result = fmt.Sprintf(`INSERT INTO "%s"."%s" SELECT * FROM "%s"."%s";`, g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable)
}

func disjoint(a []string, b map[string]bool) bool {
	for _, item := range a {
		if _, ok := b[item]; ok {
			return false
		}
	}
	return true
}

func (g *generator) findDisjointUniqueKey(columns []string) ([]string, error) {
	columnMap := make(map[string]bool)
	for _, column := range columns {
		columnMap[column] = true
	}
	if g.pk != nil {
		if disjoint(g.pk.GetProto().Expressions, columnMap) {
			return g.pk.GetProto().Expressions, nil
		}
	}
	for _, index := range g.table.GetProto().Indexes {
		if index.Primary {
			continue
		}
		if !index.Unique {
			continue
		}
		if strings.Contains(index.Type, "FUNCTION-BASED") {
			continue
		}
		if disjoint(index.Expressions, columnMap) {
			return index.Expressions, nil
		}
	}
	return nil, errors.Errorf("no disjoint unique key found for %s.%s", g.originalDatabase, g.originalTable)
}

func (g *generator) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if !IsTopLevelStatement(ctx.GetParent()) || !g.isFirst {
		return
	}

	g.isFirst = false

	l := &updateElemListener{}
	antlr.ParseTreeWalkerDefault.Walk(l, ctx)
	if l.err != nil {
		g.err = l.err
		return
	}

	uk, err := g.findDisjointUniqueKey(l.result)
	if err != nil {
		g.err = err
		return
	}

	var buf strings.Builder
	if _, err := fmt.Fprintf(&buf, "MERGE INTO \"%s\".\"%s\" t\nUSING \"%s\".\"%s\" b\n  ON(", g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable); err != nil {
		g.err = err
		return
	}
	for i, column := range uk {
		if i > 0 {
			if _, err := fmt.Fprintf(&buf, " AND"); err != nil {
				g.err = err
				return
			}
		}
		if _, err := fmt.Fprintf(&buf, " t.\"%s\" = b.\"%s\"", column, column); err != nil {
			g.err = err
			return
		}
	}
	if _, err := fmt.Fprintf(&buf, ")\nWHEN MATCHED THEN\n  UPDATE SET"); err != nil {
		g.err = err
		return
	}
	for i, field := range l.result {
		if i > 0 {
			if _, err := fmt.Fprintf(&buf, ","); err != nil {
				g.err = err
				return
			}
		}
		// The field is written by user and no need to escape.
		if _, err := fmt.Fprintf(&buf, " t.\"%s\" = b.\"%s\"", field, field); err != nil {
			g.err = err
			return
		}
	}
	if _, err := fmt.Fprintf(&buf, "\nWHEN NOT MATCHED THEN\n INSERT ("); err != nil {
		g.err = err
		return
	}
	for i, column := range g.table.GetColumns() {
		if i > 0 {
			if _, err := fmt.Fprintf(&buf, ", "); err != nil {
				g.err = err
				return
			}
		}
		if _, err := fmt.Fprintf(&buf, "\"%s\"", column.Name); err != nil {
			g.err = err
			return
		}
	}
	if _, err := fmt.Fprintf(&buf, ") VALUES ("); err != nil {
		g.err = err
		return
	}
	for i, column := range g.table.GetColumns() {
		if i > 0 {
			if _, err := fmt.Fprintf(&buf, ", "); err != nil {
				g.err = err
				return
			}
		}
		if _, err := fmt.Fprintf(&buf, "b.\"%s\"", column.Name); err != nil {
			g.err = err
			return
		}
	}
	if _, err := fmt.Fprintf(&buf, ");"); err != nil {
		g.err = err
		return
	}
	g.result = buf.String()
}

type updateElemListener struct {
	*parser.BasePlSqlParserListener

	result []string
	err    error
}

func (l *updateElemListener) EnterColumn_based_update_set_clause(ctx *parser.Column_based_update_set_clauseContext) {
	if l.err != nil {
		return
	}
	if ctx.Column_name() != nil {
		_, _, columnName, err := plsqlNormalizeColumnName("", ctx.Column_name())
		if err != nil {
			l.err = errors.Wrapf(err, "failed to normalize column name")
			return
		}
		l.result = append(l.result, columnName)
		return
	}
	if ctx.Paren_column_list() != nil {
		for _, column := range ctx.Paren_column_list().Column_list().AllColumn_name() {
			_, _, columnName, err := plsqlNormalizeColumnName("", column)
			if err != nil {
				l.err = errors.Wrapf(err, "failed to normalize column name")
				return
			}
			l.result = append(l.result, columnName)
		}
	}
}

func extractSQL(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.New("backup item is nil")
	}

	list, err := SplitSQL(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to split SQL")
	}

	start := 0
	end := len(list) - 1
	for i, item := range list {
		if equalOrLess(&storepb.Position{
			Line:   int32(item.FirstStatementLine),
			Column: int32(item.FirstStatementColumn),
		}, backupItem.StartPosition) {
			start = i
		}
	}

	for i := len(list) - 1; i >= 0; i-- {
		if equalOrGreater(&storepb.Position{
			Line:   int32(list[i].FirstStatementLine),
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
	for i := start; i <= end; i++ {
		containsSourceTable := false
		tables, err := prepareTransformation(sourceDatabase, list[i].Text)
		if err != nil {
			return "", errors.Wrap(err, "failed to prepare transformation")
		}
		for _, table := range tables {
			if table.table.Schema == sourceDatabase && table.table.Table == backupItem.SourceTable.Table {
				containsSourceTable = true
				break
			}
		}
		if containsSourceTable {
			result = append(result, list[i].Text)
		}
	}
	return strings.Join(result, ";\n"), nil
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
