package pg

import (
	"context"
	"fmt"
	"slices"
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
	maxTableNameLength = 63
)

type StatementType int

const (
	StatementTypeUnknown StatementType = iota
	StatementTypeUpdate
	StatementTypeInsert
	StatementTypeDelete
)

type TableReference struct {
	Database      string
	Schema        string
	Table         string
	Alias         string
	StatementType StatementType
}

func (t *TableReference) String() string {
	if t.Database != "" {
		return fmt.Sprintf(`"%s"."%s"."%s"`, t.Database, t.Schema, t.Table)
	}

	if t.Schema != "" {
		return fmt.Sprintf(`"%s"."%s"`, t.Schema, t.Table)
	}

	return fmt.Sprintf(`"%s"`, t.Table)
}

type statementInfo struct {
	offset    int
	statement string
	tree      antlr.ParserRuleContext
	table     *TableReference
	baseLine  int
}

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_POSTGRES, TransformDMLToSelect)
}

func TransformDMLToSelect(ctx context.Context, tCtx base.TransformContext, statement string, _ string, targetSchema string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(ctx, tCtx, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare transformation")
	}

	return generateSQL(statementInfoList, targetSchema, tablePrefix)
}

func generateSQL(statementInfoList []statementInfo, targetSchema string, tablePrefix string) ([]base.BackupStatement, error) {
	groupByTable := make(map[string][]statementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s", item.table.Schema, item.table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	// Check if the statement type is the same for all statements on the same table.
	for key, list := range groupByTable {
		statementType := StatementTypeUnknown
		for _, item := range list {
			if statementType == StatementTypeUnknown {
				statementType = item.table.StatementType
			}
			if statementType != item.table.StatementType {
				return nil, errors.Errorf("The statement type is not the same for all statements on the same table %q", key)
			}
		}
	}

	var result []base.BackupStatement
	for key, list := range groupByTable {
		backupStatement, err := generateSQLForTable(list, targetSchema, tablePrefix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate SQL for table %q", key)
		}
		result = append(result, *backupStatement)
	}

	slices.SortFunc(result, func(a, b base.BackupStatement) int {
		if a.StartPosition.Line != b.StartPosition.Line {
			if a.StartPosition.Line < b.StartPosition.Line {
				return -1
			}
			return 1
		}
		if a.StartPosition.Column != b.StartPosition.Column {
			if a.StartPosition.Column < b.StartPosition.Column {
				return -1
			}
			return 1
		}
		if a.SourceTableName < b.SourceTableName {
			return -1
		}
		if a.SourceTableName > b.SourceTableName {
			return 1
		}
		return 0
	})

	return result, nil
}

func generateSQLForTable(statementInfoList []statementInfo, targetSchema string, tablePrefix string) (*base.BackupStatement, error) {
	table := statementInfoList[0].table

	targetTable := fmt.Sprintf("%s_%s_%s", tablePrefix, table.Table, table.Schema)
	targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)
	var buf strings.Builder
	if _, err := fmt.Fprintf(&buf, `CREATE TABLE "%s"."%s" AS`+"\n", targetSchema, targetTable); err != nil {
		return nil, errors.Wrap(err, "failed to write to buffer")
	}

	for i, item := range statementInfoList {
		if i != 0 {
			if _, err := buf.WriteString("\n  UNION\n"); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		}
		if table.Alias != "" {
			if _, err := fmt.Fprintf(&buf, `  SELECT "%s".* `, table.Alias); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		} else {
			if _, err := fmt.Fprintf(&buf, `  SELECT %s.* `, table.String()); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		}

		if err := writeSuffixSelectClause(&buf, item.tree); err != nil {
			return nil, errors.Wrap(err, "failed to write string with new line")
		}
	}

	if _, err := buf.WriteString(";"); err != nil {
		return nil, errors.Wrap(err, "failed to write to buffer")
	}

	return &base.BackupStatement{
		Statement:       buf.String(),
		SourceSchema:    table.Schema,
		SourceTableName: table.Table,
		TargetTableName: targetTable,
		StartPosition: &storepb.Position{
			Line:   int32(statementInfoList[0].tree.GetStart().GetLine() + statementInfoList[0].baseLine),
			Column: int32(statementInfoList[0].tree.GetStart().GetColumn()),
		},
		EndPosition: common.ConvertANTLRTokenToExclusiveEndPosition(
			int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetLine()+statementInfoList[len(statementInfoList)-1].baseLine),
			int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetColumn()),
			statementInfoList[len(statementInfoList)-1].tree.GetStop().GetText(),
		),
	}, nil
}

func writeSuffixSelectClause(buf *strings.Builder, tree antlr.Tree) error {
	extractor := &suffixSelectClauseExtractor{
		buf: buf,
	}
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree)
	return extractor.err
}

type suffixSelectClauseExtractor struct {
	*parser.BasePostgreSQLParserListener

	buf *strings.Builder
	err error
}

func (e *suffixSelectClauseExtractor) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if e.err != nil || !isTopLevel(ctx.GetParent()) {
		return
	}

	if _, err := fmt.Fprintf(e.buf, "FROM %s", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Relation_expr_opt_alias())); err != nil {
		e.err = errors.Wrap(err, "failed to write to buffer")
	}

	if ctx.From_clause() != nil {
		if _, err := fmt.Fprintf(e.buf, ", %s", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.From_clause().From_list())); err != nil {
			e.err = errors.Wrap(err, "failed to write to buffer")
		}
	}

	if ctx.Where_or_current_clause() != nil {
		if _, err := fmt.Fprintf(e.buf, " %s", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Where_or_current_clause())); err != nil {
			e.err = errors.Wrap(err, "failed to write to buffer")
		}
	}
}

func (e *suffixSelectClauseExtractor) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if e.err != nil || !isTopLevel(ctx.GetParent()) {
		return
	}

	if _, err := fmt.Fprintf(e.buf, "FROM %s", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Relation_expr_opt_alias())); err != nil {
		e.err = errors.Wrap(err, "failed to write to buffer")
	}

	if ctx.Using_clause() != nil {
		if _, err := fmt.Fprintf(e.buf, ", %s", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Using_clause().From_list())); err != nil {
			e.err = errors.Wrap(err, "failed to write to buffer")
		}
	}

	if ctx.Where_or_current_clause() != nil {
		if _, err := fmt.Fprintf(e.buf, " %s", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Where_or_current_clause())); err != nil {
			e.err = errors.Wrap(err, "failed to write to buffer")
		}
	}
}

func prepareTransformation(ctx context.Context, tCtx base.TransformContext, statement string) ([]statementInfo, error) {
	parseResults, err := ParsePostgreSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}

	if tCtx.GetDatabaseMetadataFunc == nil {
		return nil, errors.New("GetDatabaseMetadataFunc is not set in TransformContext")
	}

	_, metadata, err := tCtx.GetDatabaseMetadataFunc(ctx, tCtx.InstanceID, tCtx.DatabaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database metadata")
	}

	extractor := &dmlExtractor{
		metadata:   metadata,
		searchPath: metadata.GetSearchPath(),
	}

	// Walk all parse results to extract DML statements
	for _, parseResult := range parseResults {
		extractor.currentBaseLine = base.GetLineOffset(parseResult.StartPosition)
		antlr.ParseTreeWalkerDefault.Walk(extractor, parseResult.Tree)
		if extractor.err != nil {
			return nil, extractor.err
		}
	}

	return extractor.dmls, nil
}

type dmlExtractor struct {
	*parser.BasePostgreSQLParserListener

	metadata        *model.DatabaseMetadata
	searchPath      []string
	dmls            []statementInfo
	offset          int
	err             error
	currentBaseLine int
}

func isTopLevel(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}

	switch ctx := ctx.(type) {
	case *parser.RootContext, *parser.StmtblockContext:
		return true
	case *parser.StmtmultiContext, *parser.StmtContext:
		return isTopLevel(ctx.GetParent())
	default:
		return false
	}
}

func (e *dmlExtractor) EnterVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	setRest := ctx.Set_rest()
	if setRest == nil {
		return
	}
	setRestMore := setRest.Set_rest_more()
	if setRestMore == nil {
		return
	}
	genericSet := setRestMore.Generic_set()
	if genericSet == nil {
		return
	}
	varName := genericSet.Var_name()
	if varName == nil {
		return
	}
	if len(varName.AllColid()) != 1 {
		return
	}
	name := NormalizePostgreSQLColid(varName.Colid(0))
	if !strings.EqualFold(name, "search_path") {
		return
	}
	var searchPath []string
	for _, value := range genericSet.Var_list().AllVar_value() {
		valueText := value.GetText()
		if strings.HasPrefix(valueText, "\"") && strings.HasSuffix(valueText, "\"") {
			// Remove the quotes from the schema name.
			valueText = strings.Trim(valueText, "\"")
		} else if strings.HasPrefix(valueText, "'") && strings.HasSuffix(valueText, "'") {
			// Remove the quotes from the schema name.
			valueText = strings.Trim(valueText, "'")
		} else {
			// For non-quoted schema names, we just return the lower string for PostgreSQL.
			valueText = strings.ToLower(valueText)
		}
		searchPath = append(searchPath, strings.TrimSpace(valueText))
	}
	e.searchPath = searchPath
}

func (e *dmlExtractor) ExitStmt(ctx *parser.StmtContext) {
	if isTopLevel(ctx) {
		e.offset++
	}
}

func (e *dmlExtractor) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		table, err := e.extractTableReference(ctx.Relation_expr_opt_alias())
		if err != nil {
			e.err = errors.Wrapf(err, "failed to extract table reference from update statement at offset %d", e.offset)
			return
		}
		if table == nil {
			return
		}
		table.StatementType = StatementTypeUpdate
		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
			baseLine:  e.currentBaseLine,
		})
	}
}

func (e *dmlExtractor) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		table, err := e.extractTableReference(ctx.Relation_expr_opt_alias())
		if err != nil {
			e.err = errors.Wrapf(err, "failed to extract table reference from delete statement at offset %d", e.offset)
			return
		}
		if table == nil {
			return
		}
		table.StatementType = StatementTypeDelete
		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
			baseLine:  e.currentBaseLine,
		})
	}
}

func (e *dmlExtractor) extractTableReference(ctx parser.IRelation_expr_opt_aliasContext) (*TableReference, error) {
	if ctx == nil {
		return nil, nil
	}

	table := TableReference{}

	relationExpr := ctx.Relation_expr()
	if relationExpr == nil {
		return nil, nil
	}

	list := NormalizePostgreSQLQualifiedName(relationExpr.Qualified_name())
	switch len(list) {
	case 3:
		table.Database = list[0]
		table.Schema = list[1]
		table.Table = list[2]
	case 2:
		table.Schema = list[0]
		table.Table = list[1]
	case 1:
		// TODO: remove it in the future.
		// Handle the case where the search path is not synchronized with the metadata.
		if len(e.searchPath) == 0 {
			e.searchPath = []string{"public"}
		}
		schemaName, _ := e.metadata.SearchObject(e.searchPath, list[0])
		if schemaName == "" {
			return nil, errors.Errorf("Table %q not found in metadata with search path %v", list[0], e.searchPath)
		}
		table.Schema = schemaName
		table.Table = list[0]
	default:
		return nil, errors.Errorf("Invalid table name: %v", list)
	}

	if ctx.Colid() != nil {
		table.Alias = NormalizePostgreSQLColid(ctx.Colid())
	}
	return &table, nil
}
