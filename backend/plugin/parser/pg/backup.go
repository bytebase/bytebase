package pg

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/postgresql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storebp "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	defaultSchema      = "public"
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
}

func init() {
	base.RegisterTransformDMLToSelect(storebp.Engine_POSTGRES, TransformDMLToSelect)
}

func TransformDMLToSelect(_ context.Context, _ base.TransformContext, statement string, _ string, targetSchema string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(statement)
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

	sort.Slice(result, func(i, j int) bool {
		if result[i].StartPosition.Line != result[j].StartPosition.Line {
			return result[i].StartPosition.Line < result[j].StartPosition.Line
		}
		if result[i].StartPosition.Column != result[j].StartPosition.Column {
			return result[i].StartPosition.Column < result[j].StartPosition.Column
		}
		return result[i].SourceTableName < result[j].SourceTableName
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
		StartPosition: &storebp.Position{
			Line:   int32(statementInfoList[0].tree.GetStart().GetLine()),
			Column: int32(statementInfoList[0].tree.GetStart().GetColumn()),
		},
		EndPosition: &storebp.Position{
			Line:   int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetLine()),
			Column: int32(statementInfoList[len(statementInfoList)-1].tree.GetStop().GetColumn()),
		},
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

func prepareTransformation(statement string) ([]statementInfo, error) {
	tree, err := ParsePostgreSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}

	extractor := &dmlExtractor{}
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree.Tree)
	return extractor.dmls, nil
}

type dmlExtractor struct {
	*parser.BasePostgreSQLParserListener

	dmls   []statementInfo
	offset int
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

func (e *dmlExtractor) ExitStmt(ctx *parser.StmtContext) {
	if isTopLevel(ctx) {
		e.offset++
	}
}

func (e *dmlExtractor) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		table := extractTableReference(ctx.Relation_expr_opt_alias())
		if table == nil {
			return
		}
		table.StatementType = StatementTypeUpdate
		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
		})
	}
}

func (e *dmlExtractor) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		table := extractTableReference(ctx.Relation_expr_opt_alias())
		if table == nil {
			return
		}
		table.StatementType = StatementTypeDelete
		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
		})
	}
}

func extractTableReference(ctx parser.IRelation_expr_opt_aliasContext) *TableReference {
	if ctx == nil {
		return nil
	}

	table := TableReference{}

	relationExpr := ctx.Relation_expr()
	if relationExpr == nil {
		return nil
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
		table.Schema = defaultSchema
		table.Table = list[0]
	default:
		slog.Debug("Invalid table name", log.BBError(errors.Errorf("Invalid table name: %v", list)))
		return nil
	}

	if ctx.Colid() != nil {
		table.Alias = NormalizePostgreSQLColid(ctx.Colid())
	}
	return &table
}
