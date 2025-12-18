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
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	maxCommentLength = 1000
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_POSTGRES, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	if len(originalSQL) == 0 {
		return "", errors.Errorf("no original SQL")
	}

	parseResults, err := ParsePostgreSQL(statement)
	if err != nil {
		return "", err
	}

	// Find the parse result that contains the statement at the backup position
	var targetResult *base.ANTLRAST
	for _, parseResult := range parseResults {
		// Walk the tree to find if this parse result contains the target statement
		finder := &statementAtPositionFinder{
			startPos: backupItem.StartPosition,
			endPos:   backupItem.EndPosition,
			baseLine: base.GetLineOffset(parseResult.StartPosition),
		}
		antlr.ParseTreeWalkerDefault.Walk(finder, parseResult.Tree)
		if finder.found {
			targetResult = parseResult
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

	prependStatements, err := getPrependStatements(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to get prepend statements")
	}

	return doGenerate(ctx, rCtx, sqlForComment, targetResult, backupItem, prependStatements)
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, tree *base.ANTLRAST, backupItem *storepb.PriorBackupDetail_Item, prependStatements string) (string, error) {
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
	schemaMetadata := metadata.GetSchemaMetadata(schema)
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

func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}

	parseResults, err := ParsePostgreSQL(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse statement")
	}

	l := &originalSQLExtractor{
		startPos: backupItem.StartPosition,
		endPos:   backupItem.EndPosition,
	}

	// Walk all parse results to find statements within the specified position range
	for _, parseResult := range parseResults {
		l.baseLine = base.GetLineOffset(parseResult.StartPosition)
		antlr.ParseTreeWalkerDefault.Walk(l, parseResult.Tree)
	}

	return strings.Join(l.originalSQL, ";\n"), nil
}

type originalSQLExtractor struct {
	*parser.BasePostgreSQLParserListener

	originalSQL []string
	startPos    *storepb.Position
	endPos      *storepb.Position
	baseLine    int
}

func (l *originalSQLExtractor) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		if inRange(&storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()) + int32(l.baseLine),
			Column: int32(ctx.GetStart().GetColumn()),
		}, &storepb.Position{
			Line:   int32(ctx.GetStop().GetLine()) + int32(l.baseLine),
			Column: int32(ctx.GetStop().GetColumn()),
		}, l.startPos, l.endPos) {
			l.originalSQL = append(l.originalSQL, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
		}
	}
}

func (l *originalSQLExtractor) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		if inRange(&storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()) + int32(l.baseLine),
			Column: int32(ctx.GetStart().GetColumn()),
		}, &storepb.Position{
			Line:   int32(ctx.GetStop().GetLine()) + int32(l.baseLine),
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
	// Parse with ANTLR
	parseResults, err := ParsePostgreSQL(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	// Create listener to find SET role statements across all parsed statements
	listener := &setRoleListener{}
	for _, parseResult := range parseResults {
		antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)
		// If we found a SET role statement, return it immediately
		if listener.setRoleText != "" {
			break
		}
	}

	return listener.setRoleText, nil
}

// setRoleListener detects SET role statements
type setRoleListener struct {
	*parser.BasePostgreSQLParserListener
	setRoleText string
}

// EnterVariablesetstmt handles SET statements
func (l *setRoleListener) EnterVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	// Only process if we haven't found a SET role statement yet
	if l.setRoleText != "" {
		return
	}

	// Check if this is SET role
	// Structure: VariablesetstmtContext -> Set_rest -> Set_rest_more -> Generic_set -> Var_name
	if ctx.Set_rest() != nil {
		setRest := ctx.Set_rest()
		if setRest.Set_rest_more() != nil {
			setRestMore := setRest.Set_rest_more()
			if setRestMore.Generic_set() != nil {
				genericSet := setRestMore.Generic_set()
				if genericSet.Var_name() != nil {
					varName := genericSet.Var_name().GetText()
					if strings.EqualFold(varName, "role") {
						// Found SET role statement, capture the full text
						l.setRoleText = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
					}
				}
			}
		}
	}
}

// statementAtPositionFinder finds if a parse tree contains a statement at the given position.
type statementAtPositionFinder struct {
	*parser.BasePostgreSQLParserListener
	startPos *storepb.Position
	endPos   *storepb.Position
	baseLine int
	found    bool
}

func (f *statementAtPositionFinder) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if isTopLevel(ctx.GetParent()) && inRange(&storepb.Position{
		Line:   int32(ctx.GetStart().GetLine()) + int32(f.baseLine),
		Column: int32(ctx.GetStart().GetColumn()),
	}, &storepb.Position{
		Line:   int32(ctx.GetStop().GetLine()) + int32(f.baseLine),
		Column: int32(ctx.GetStop().GetColumn()),
	}, f.startPos, f.endPos) {
		f.found = true
	}
}

func (f *statementAtPositionFinder) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if isTopLevel(ctx.GetParent()) && inRange(&storepb.Position{
		Line:   int32(ctx.GetStart().GetLine()) + int32(f.baseLine),
		Column: int32(ctx.GetStart().GetColumn()),
	}, &storepb.Position{
		Line:   int32(ctx.GetStop().GetLine()) + int32(f.baseLine),
		Column: int32(ctx.GetStop().GetColumn()),
	}, f.startPos, f.endPos) {
		f.found = true
	}
}
