package tsql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/tsql"

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
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	if len(originalSQL) == 0 {
		return "", errors.Errorf("no original SQL")
	}

	antlrASTs, err := ParseTSQL(statement)
	if err != nil {
		return "", err
	}

	// Find the AST that contains the statement at the backup position
	var targetResult *base.ANTLRAST
	for _, ast := range antlrASTs {
		// Walk the tree to find if this AST contains the target statement
		finder := &statementAtPositionFinder{
			startPos: backupItem.StartPosition,
			endPos:   backupItem.EndPosition,
			baseLine: base.GetLineOffset(ast.StartPosition),
		}
		antlr.ParseTreeWalkerDefault.Walk(finder, ast.Tree)
		if finder.found {
			targetResult = ast
			break
		}
	}

	if targetResult == nil {
		return "", errors.Errorf("could not find statement at position (line %d:%d - %d:%d)",
			backupItem.StartPosition.Line, backupItem.StartPosition.Column,
			backupItem.EndPosition.Line, backupItem.EndPosition.Column)
	}

	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}
	return doGenerate(ctx, rCtx, sqlForComment, targetResult, backupItem)
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
	schemaMetadata := metadata.GetSchemaMetadata(schema)
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
	antlr.ParseTreeWalkerDefault.Walk(g, ast.Tree)
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

// hasIdentityColumn checks if the table has any IDENTITY columns
func (g *generator) hasIdentityColumn() bool {
	if g.table == nil {
		return false
	}
	for _, col := range g.table.GetProto().GetColumns() {
		if col.IsIdentity {
			return true
		}
	}
	return false
}

func (g *generator) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if IsTopLevel(ctx.GetParent()) && g.isFirst {
		g.isFirst = false

		// Check if the table has IDENTITY columns
		hasIdentity := g.hasIdentityColumn()

		if hasIdentity {
			// For tables with IDENTITY columns, we need to enable IDENTITY_INSERT
			// and use explicit column lists
			var buf strings.Builder

			// Build column list
			var columnList strings.Builder
			for i, column := range g.table.GetProto().GetColumns() {
				if i > 0 {
					columnList.WriteString(", ")
				}
				fmt.Fprintf(&columnList, "[%s]", column.Name)
			}

			fmt.Fprintf(&buf, "SET IDENTITY_INSERT [%s].[%s].[%s] ON;\n",
				g.originalDatabase, g.originalSchema, g.originalTable)
			fmt.Fprintf(&buf, "INSERT INTO [%s].[%s].[%s] (%s) SELECT %s FROM [%s].[dbo].[%s];\n",
				g.originalDatabase, g.originalSchema, g.originalTable,
				columnList.String(),
				columnList.String(),
				g.backupDatabase, g.backupTable)
			fmt.Fprintf(&buf, "SET IDENTITY_INSERT [%s].[%s].[%s] OFF;\n",
				g.originalDatabase, g.originalSchema, g.originalTable)
			fmt.Fprintf(&buf, "EXEC('DBCC CHECKIDENT (''[%s].[%s].[%s]'', RESEED)');",
				g.originalDatabase, g.originalSchema, g.originalTable)

			g.result = buf.String()
		} else {
			// Simple INSERT for tables without IDENTITY columns
			g.result = fmt.Sprintf(`INSERT INTO [%s].[%s].[%s] SELECT * FROM [%s].[dbo].[%s];`,
				g.originalDatabase, g.originalSchema, g.originalTable,
				g.backupDatabase, g.backupTable)
		}
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

		// Check if the table has IDENTITY columns
		hasIdentity := g.hasIdentityColumn()

		if hasIdentity {
			// Enable IDENTITY_INSERT for MERGE statement
			if _, err := fmt.Fprintf(&buf, "SET IDENTITY_INSERT [%s].[%s].[%s] ON;\n",
				g.originalDatabase, g.originalSchema, g.originalTable); err != nil {
				g.err = err
				return
			}
		}

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
		for i, column := range g.table.GetProto().GetColumns() {
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
		for i, column := range g.table.GetProto().GetColumns() {
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

		// Check if we need to disable IDENTITY_INSERT and reseed
		if hasIdentity {
			if _, err := fmt.Fprintf(&buf, "\n\nSET IDENTITY_INSERT [%s].[%s].[%s] OFF;\nEXEC('DBCC CHECKIDENT (''[%s].[%s].[%s]'', RESEED)');",
				g.originalDatabase, g.originalSchema, g.originalTable,
				g.originalDatabase, g.originalSchema, g.originalTable); err != nil {
				g.err = err
				return
			}
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

func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}

	parseResults, err := ParseTSQL(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse statement")
	}

	l := &originalSQLExtractor{
		startPos: backupItem.StartPosition,
		endPos:   backupItem.EndPosition,
	}

	// Walk all ASTs to find statements within the specified position range
	for _, ast := range parseResults {
		l.baseLine = base.GetLineOffset(ast.StartPosition)
		antlr.ParseTreeWalkerDefault.Walk(l, ast.Tree)
	}

	if len(l.originalSQL) == 0 {
		return "", nil
	}

	// Join with semicolons and add a trailing semicolon
	return strings.Join(l.originalSQL, ";\n") + ";", nil
}

// originalSQLExtractor extracts original SQL statements within a position range.
type originalSQLExtractor struct {
	*parser.BaseTSqlParserListener

	originalSQL []string
	startPos    *storepb.Position
	endPos      *storepb.Position
	baseLine    int
}

func (l *originalSQLExtractor) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if IsTopLevel(ctx.GetParent()) {
		if inRange(&storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()) + int32(l.baseLine),
			Column: int32(ctx.GetStart().GetColumn()),
		}, &storepb.Position{
			Line:   int32(ctx.GetStop().GetLine()) + int32(l.baseLine),
			Column: int32(ctx.GetStop().GetColumn()),
		}, l.startPos, l.endPos) {
			sql := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
			// Strip trailing semicolon to avoid double semicolons when joining
			sql = strings.TrimSuffix(sql, ";")
			l.originalSQL = append(l.originalSQL, sql)
		}
	}
}

func (l *originalSQLExtractor) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if IsTopLevel(ctx.GetParent()) {
		if inRange(&storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()) + int32(l.baseLine),
			Column: int32(ctx.GetStart().GetColumn()),
		}, &storepb.Position{
			Line:   int32(ctx.GetStop().GetLine()) + int32(l.baseLine),
			Column: int32(ctx.GetStop().GetColumn()),
		}, l.startPos, l.endPos) {
			sql := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
			// Strip trailing semicolon to avoid double semicolons when joining
			sql = strings.TrimSuffix(sql, ";")
			l.originalSQL = append(l.originalSQL, sql)
		}
	}
}

// statementAtPositionFinder finds if a parse tree contains a statement at the given position.
type statementAtPositionFinder struct {
	*parser.BaseTSqlParserListener
	startPos *storepb.Position
	endPos   *storepb.Position
	baseLine int
	found    bool
}

func (f *statementAtPositionFinder) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if IsTopLevel(ctx.GetParent()) && inRange(&storepb.Position{
		Line:   int32(ctx.GetStart().GetLine()) + int32(f.baseLine),
		Column: int32(ctx.GetStart().GetColumn()),
	}, &storepb.Position{
		Line:   int32(ctx.GetStop().GetLine()) + int32(f.baseLine),
		Column: int32(ctx.GetStop().GetColumn()),
	}, f.startPos, f.endPos) {
		f.found = true
	}
}

func (f *statementAtPositionFinder) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if IsTopLevel(ctx.GetParent()) && inRange(&storepb.Position{
		Line:   int32(ctx.GetStart().GetLine()) + int32(f.baseLine),
		Column: int32(ctx.GetStart().GetColumn()),
	}, &storepb.Position{
		Line:   int32(ctx.GetStop().GetLine()) + int32(f.baseLine),
		Column: int32(ctx.GetStop().GetColumn()),
	}, f.startPos, f.endPos) {
		f.found = true
	}
}

// inRange checks if a statement's position range is within the target range.
func inRange(start, end, targetStart, targetEnd *storepb.Position) bool {
	if start.Line < targetStart.Line || (start.Line == targetStart.Line && start.Column < targetStart.Column) {
		return false
	}
	if end.Line > targetEnd.Line || (end.Line == targetEnd.Line && end.Column > targetEnd.Column) {
		return false
	}
	return true
}
