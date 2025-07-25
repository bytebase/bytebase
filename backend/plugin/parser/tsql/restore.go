package tsql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	maxCommentLength = 1000
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MSSQL, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractSingleSQL(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	parseResult, err := ParseTSQL(statement)
	if err != nil {
		return "", err
	}

	if parseResult.Tree == nil {
		return "", errors.Errorf("no parse result")
	}

	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}
	return doGenerate(ctx, rCtx, sqlForComment, parseResult, backupItem)
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, tree *ParseResult, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
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

	if metadata == nil {
		return "", errors.Errorf("database metadata is nil")
	}

	schema := backupItem.SourceTable.Schema
	if schema == "" {
		schema = "dbo"
	}
	schemaMetadata := metadata.GetSchema(schema)
	if schemaMetadata == nil {
		return "", errors.Errorf("schema metadata not found for %s", schema)
	}

	tableMetadata := schemaMetadata.GetTable(backupItem.SourceTable.Table)
	if tableMetadata == nil {
		return "", errors.Errorf("table metadata not found for %s.%s", schema, backupItem.SourceTable.Table)
	}

	g := &generator{
		isFirst:          true,
		ctx:              ctx,
		rCtx:             rCtx,
		backupDatabase:   targetDatabase,
		backupTable:      backupItem.TargetTable.Table,
		originalDatabase: sourceDatabase,
		originalSchema:   schema,
		originalTable:    backupItem.SourceTable.Table,
		pk:               tableMetadata.GetPrimaryKey(),
		table:            tableMetadata,
	}
	antlr.ParseTreeWalkerDefault.Walk(g, tree.Tree)
	if g.err != nil {
		return "", g.err
	}
	return fmt.Sprintf("/*\nOriginal SQL:\n%s\n*/\n%s", sqlForComment, g.result), nil
}

type generator struct {
	*parser.BaseTSqlParserListener

	ctx  context.Context
	rCtx base.RestoreContext

	backupDatabase   string
	backupTable      string
	originalDatabase string
	originalSchema   string
	originalTable    string
	pk               *model.IndexMetadata
	table            *model.TableMetadata

	isFirst bool
	result  string
	err     error
}

func (g *generator) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if IsTopLevel(ctx.GetParent()) && g.isFirst {
		g.isFirst = false
		g.result = fmt.Sprintf(`INSERT INTO [%s].[%s].[%s] SELECT * FROM [%s].[dbo].[%s];`, g.originalDatabase, g.originalSchema, g.originalTable, g.backupDatabase, g.backupTable)
	}
}

func disjoint(a []string, b map[string]bool) bool {
	for _, item := range a {
		if _, ok := b[item]; ok {
			return false
		}
	}
	return true
}

func (g *generator) findDisjointUniqueKey(updateColumns []string) ([]string, error) {
	columnMap := make(map[string]bool)
	for _, column := range updateColumns {
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
		if disjoint(index.Expressions, columnMap) {
			return index.Expressions, nil
		}
	}
	return nil, errors.Errorf("no disjoint unique key found for %s.%s.%s", g.originalDatabase, g.originalSchema, g.originalTable)
}

func (g *generator) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if IsTopLevel(ctx.GetParent()) && g.isFirst {
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
		if _, err := fmt.Fprintf(&buf, "MERGE INTO [%s].[%s].[%s] AS t\nUSING [%s].[dbo].[%s] AS b\n  ON", g.originalDatabase, g.originalSchema, g.originalTable, g.backupDatabase, g.backupTable); err != nil {
			g.err = err
			return
		}
		for i, column := range uk {
			if i > 0 {
				if _, err := fmt.Fprint(&buf, " AND"); err != nil {
					g.err = err
					return
				}
			}
			if _, err := fmt.Fprintf(&buf, " t.[%s] = b.[%s]", column, column); err != nil {
				g.err = err
				return
			}
		}
		if _, err := fmt.Fprint(&buf, "\nWHEN MATCHED THEN\n  UPDATE SET"); err != nil {
			g.err = err
			return
		}
		for i, field := range l.result {
			if i > 0 {
				if _, err := fmt.Fprint(&buf, ","); err != nil {
					g.err = err
					return
				}
			}
			if _, err := fmt.Fprintf(&buf, " t.[%s] = b.[%s]", field, field); err != nil {
				g.err = err
				return
			}
		}
		if _, err := fmt.Fprint(&buf, "\nWHEN NOT MATCHED THEN\n INSERT ("); err != nil {
			g.err = err
			return
		}
		for i, column := range g.table.GetColumns() {
			if i > 0 {
				if _, err := fmt.Fprint(&buf, ", "); err != nil {
					g.err = err
					return
				}
			}
			if _, err := fmt.Fprintf(&buf, "[%s]", column.Name); err != nil {
				g.err = err
				return
			}
		}
		if _, err := fmt.Fprint(&buf, ") VALUES ("); err != nil {
			g.err = err
			return
		}
		for i, column := range g.table.GetColumns() {
			if i > 0 {
				if _, err := fmt.Fprint(&buf, ", "); err != nil {
					g.err = err
					return
				}
			}
			if _, err := fmt.Fprintf(&buf, "b.[%s]", column.Name); err != nil {
				g.err = err
				return
			}
		}
		if _, err := fmt.Fprint(&buf, ");"); err != nil {
			g.err = err
			return
		}
		g.result = buf.String()
	}
}

type updateElemListener struct {
	*parser.BaseTSqlParserListener

	result []string
	err    error
}

func (l *updateElemListener) EnterUpdate_elem(ctx *parser.Update_elemContext) {
	if l.err != nil {
		return
	}
	if ctx.Full_column_name() != nil {
		_, columnName, err := NormalizeFullColumnName(ctx.Full_column_name())
		if err != nil {
			l.err = err
			return
		}
		l.result = append(l.result, columnName)
	}
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
		if equalOrLess(item.Start, backupItem.StartPosition) {
			start = i
		}
	}

	for i := len(list) - 1; i >= 0; i-- {
		if equalOrGreater(list[i].Start, backupItem.EndPosition) {
			end = i
		}
	}
	var result []string
	for i := start; i <= end; i++ {
		result = append(result, list[i].Text)
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
