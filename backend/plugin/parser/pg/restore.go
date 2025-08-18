package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	maxCommentLength = 1000
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_POSTGRES, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractSingleSQL(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	if len(originalSQL) == 0 {
		return "", errors.Errorf("no original SQL")
	}

	tree, err := ParsePostgreSQL(statement)
	if err != nil {
		return "", err
	}

	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}

	prependStatements, err := getPrependStatements(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to get prepend statements")
	}

	return doGenerate(ctx, rCtx, sqlForComment, tree, backupItem, prependStatements)
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, tree *ParseResult, backupItem *storepb.PriorBackupDetail_Item, prependStatements string) (string, error) {
	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}

	if rCtx.GetDatabaseMetadataFunc == nil {
		return "", errors.Errorf("GetDatabaseMetadataFunc is required")
	}

	_, metadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, sourceDatabase)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database metadata for %s", sourceDatabase)
	}

	if metadata == nil {
		return "", errors.Errorf("database metadata not found for %s", sourceDatabase)
	}

	schema := backupItem.SourceTable.Schema
	if schema == "" {
		schema = "public"
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
		ctx:            ctx,
		rCtx:           rCtx,
		backupSchema:   backupItem.TargetTable.Schema,
		backupTable:    backupItem.TargetTable.Table,
		originalSchema: schema,
		originalTable:  backupItem.SourceTable.Table,
		table:          tableMetadata,
		isFirst:        true,
	}
	antlr.ParseTreeWalkerDefault.Walk(g, tree.Tree)
	if g.err != nil {
		return "", g.err
	}

	if len(prependStatements) > 0 {
		return fmt.Sprintf("%s\n/*\nOriginal SQL:\n%s\n*/\n%s", prependStatements, sqlForComment, g.result), nil
	}
	return fmt.Sprintf("/*\nOriginal SQL:\n%s\n*/\n%s", sqlForComment, g.result), nil
}

type generator struct {
	*parser.BasePostgreSQLParserListener

	backupSchema   string
	backupTable    string
	originalSchema string
	originalTable  string
	table          *model.TableMetadata

	isFirst bool
	ctx     context.Context
	rCtx    base.RestoreContext
	result  string
	err     error
}

func (g *generator) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if isTopLevel(ctx.GetParent()) && g.isFirst {
		g.isFirst = false
		g.result = fmt.Sprintf(`INSERT INTO "%s"."%s" SELECT * FROM "%s"."%s";`, g.originalSchema, g.originalTable, g.backupSchema, g.backupTable)
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

func (g *generator) findDisjointUniqueKey(fields []string) (string, error) {
	columnMap := make(map[string]bool)
	for _, field := range fields {
		columnMap[field] = true
	}
	pk := g.table.GetPrimaryKey()
	if pk != nil {
		if disjoint(pk.GetProto().Expressions, columnMap) {
			return pk.GetProto().Name, nil
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
			return index.Name, nil
		}
	}

	return "", errors.Errorf("no disjoint unique key found for %s.%s", g.originalSchema, g.originalTable)
}

func (g *generator) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if isTopLevel(ctx.GetParent()) && g.isFirst {
		g.isFirst = false

		l := &setFieldListener{}
		antlr.ParseTreeWalkerDefault.Walk(l, ctx)

		uk, err := g.findDisjointUniqueKey(l.result)
		if err != nil {
			g.err = err
			return
		}

		var buf strings.Builder
		if _, err := fmt.Fprintf(&buf, `INSERT INTO "%s"."%s" SELECT * FROM "%s"."%s" ON CONFLICT ON CONSTRAINT "%s" DO UPDATE SET `, g.originalSchema, g.originalTable, g.backupSchema, g.backupTable, uk); err != nil {
			g.err = errors.Wrapf(err, "failed to generate update statement")
			return
		}
		for i, field := range l.result {
			if i > 0 {
				if _, err := fmt.Fprint(&buf, ", "); err != nil {
					g.err = errors.Wrapf(err, "failed to generate update statement")
					return
				}
			}
			// The field is written by user and no need to escape.
			if _, err := fmt.Fprintf(&buf, `"%s" = EXCLUDED."%s"`, field, field); err != nil {
				g.err = errors.Wrapf(err, "failed to generate update statement")
				return
			}
		}
		if _, err := fmt.Fprint(&buf, `;`); err != nil {
			g.err = errors.Wrapf(err, "failed to generate update statement")
			return
		}
		g.result = buf.String()
	}
}

type setFieldListener struct {
	*parser.BasePostgreSQLParserListener

	result []string
}

func (l *setFieldListener) EnterSet_clause(ctx *parser.Set_clauseContext) {
	if ctx.Set_target() != nil {
		names := normalizePostgreSQLSetTarget(ctx.Set_target())
		if len(names) > 0 {
			l.result = append(l.result, names[len(names)-1])
		}
	}
}

func extractSingleSQL(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}

	tree, err := ParsePostgreSQL(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse statement")
	}

	l := &originalSQLExtractor{
		startPos: backupItem.StartPosition,
		endPos:   backupItem.EndPosition,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, tree.Tree)
	return strings.Join(l.originalSQL, ";\n"), nil
}

type originalSQLExtractor struct {
	*parser.BasePostgreSQLParserListener

	originalSQL []string
	startPos    *storepb.Position
	endPos      *storepb.Position
}

func (l *originalSQLExtractor) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		if inRange(&storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: int32(ctx.GetStart().GetColumn()),
		}, &storepb.Position{
			Line:   int32(ctx.GetStop().GetLine()),
			Column: int32(ctx.GetStop().GetColumn()),
		}, l.startPos, l.endPos) {
			l.originalSQL = append(l.originalSQL, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
		}
	}
}

func (l *originalSQLExtractor) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		if inRange(&storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: int32(ctx.GetStart().GetColumn()),
		}, &storepb.Position{
			Line:   int32(ctx.GetStop().GetLine()),
			Column: int32(ctx.GetStop().GetColumn()),
		}, l.startPos, l.endPos) {
			l.originalSQL = append(l.originalSQL, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
		}
	}
}

func inRange(start, end, targetStart, targetEnd *storepb.Position) bool {
	if start.Line < targetStart.Line || (start.Line == targetStart.Line && start.Column < targetStart.Column) {
		return false
	}
	if end.Line > targetEnd.Line || (end.Line == targetEnd.Line && end.Column > targetEnd.Column) {
		return false
	}
	return true
}

func getPrependStatements(statement string) (string, error) {
	nodes, err := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	for _, node := range nodes {
		if n, ok := node.(*ast.VariableSetStmt); ok {
			if n.Name == "role" {
				return n.Text(), nil
			}
		}
	}

	return "", nil
}
