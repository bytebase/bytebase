package pg

import (
	"fmt"
	"log/slog"
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
	maxTableNameLength = 63
)

type TableReference struct {
	Database string
	Schema   string
	Table    string
	Alias    string
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
	tree      antlr.Tree
	table     *TableReference
	line      int
}

func init() {
	base.RegisterTransformDMLToSelect(storebp.Engine_POSTGRES, TransformDMLToSelect)
}

func TransformDMLToSelect(statement string, _ string, targetSchema string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare transformation")
	}

	return generateSQL(statementInfoList, targetSchema, tablePrefix)
}

func generateSQL(statementInfoList []statementInfo, targetSchema string, tablePrefix string) ([]base.BackupStatement, error) {
	var result []base.BackupStatement
	offsetLength := 1
	if len(statementInfoList) > 1 {
		offsetLength = base.GetOffsetLength(statementInfoList[len(statementInfoList)-1].offset)
	}

	for _, info := range statementInfoList {
		table := info.table
		targetTable := fmt.Sprintf("%s_%0*d_%s", tablePrefix, offsetLength, info.offset, table.Table)
		targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)
		var buf strings.Builder
		if _, err := fmt.Fprintf(&buf, `CREATE TABLE "%s"."%s" AS SELECT `, targetSchema, targetTable); err != nil {
			return nil, errors.Wrap(err, "failed to write to buffer")
		}
		if table.Alias != "" {
			if _, err := fmt.Fprintf(&buf, `"%s".* `, table.Alias); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		} else {
			if _, err := fmt.Fprintf(&buf, `%s.* `, table.String()); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		}

		if err := writeSuffixSelectClause(&buf, info.tree); err != nil {
			return nil, errors.Wrap(err, "failed to write suffix select clause")
		}

		if _, err := buf.WriteString(";"); err != nil {
			return nil, errors.Wrap(err, "failed to write to buffer")
		}

		result = append(result, base.BackupStatement{
			Statement:    buf.String(),
			TableName:    targetTable,
			OriginalLine: info.line,
		})
	}

	return result, nil
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

		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
			line:      ctx.GetStart().GetLine(),
		})
	}
}

func (e *dmlExtractor) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		table := extractTableReference(ctx.Relation_expr_opt_alias())
		if table == nil {
			return
		}

		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
			line:      ctx.GetStart().GetLine(),
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
